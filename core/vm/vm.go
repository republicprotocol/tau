package vm

import (
	"log"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/process"
	"github.com/republicprotocol/smpc-go/core/vm/mul"
	"github.com/republicprotocol/smpc-go/core/vm/open"
	"github.com/republicprotocol/smpc-go/core/vm/rng"
	"github.com/republicprotocol/smpc-go/core/vm/task"
	"github.com/republicprotocol/smpc-go/core/vss/pedersen"
)

type VM struct {
	io             task.IO
	ioExternal     task.IO
	processes      map[process.ID]process.Process
	processIntents map[process.ID]process.Intent

	rng  task.Task
	mul  task.Task
	open task.Task
}

func New(r, w buffer.ReaderWriter, addr, leader uint64, ped pedersen.Pedersen, n, k uint, cap int) VM {
	return VM{
		io:             task.NewIO(buffer.New(cap), r.Reader(), w.Writer()),
		ioExternal:     task.NewIO(buffer.New(cap), w.Reader(), r.Writer()),
		processes:      map[process.ID]process.Process{},
		processIntents: map[process.ID]process.Intent{},

		rng:  rng.New(buffer.NewReaderWriter(cap), buffer.NewReaderWriter(cap), rng.Address(addr), rng.Address(leader), ped, n, k, n-k, cap),
		mul:  mul.New(buffer.NewReaderWriter(cap), buffer.NewReaderWriter(cap), n, k, cap),
		open: open.New(buffer.NewReaderWriter(cap), buffer.NewReaderWriter(cap), n, k, cap),
	}
}

func (vm *VM) IO() task.IO {
	return vm.ioExternal
}

func (vm *VM) Run(done <-chan struct{}) {
	defer log.Printf("[info] (vm) terminating")

	co.ParBegin(
		func() {
			vm.runBackgroundGoroutines(done)
		},
		func() {
			for {
				ok := task.Select(
					done,
					vm.recvMessage,
					vm.io,
					vm.rng.IO(),
					vm.mul.IO(),
					vm.open.IO(),
				)
				if !ok {
					return
				}
			}
		})
}

func (vm *VM) runBackgroundGoroutines(done <-chan struct{}) {
	co.ParBegin(
		func() {
			vm.rng.Run(done)
		},
		func() {
			vm.mul.Run(done)
		},
		func() {
			vm.open.Run(done)
		})
}

func (vm *VM) recvMessage(message buffer.Message) {
	switch message := message.(type) {

	case Exec:
		vm.exec(message)

	case RemoteProcedureCall:
		vm.invoke(message)

	case rng.ProposeRn:
		vm.proposeRn(message)

	case rng.LocalRnShares:
		vm.handleRngLocalRnShares(message)

	case rng.ProposeGlobalRnShare:
		vm.handleRngProposeGlobalRnShare(message)

	case rng.GlobalRnShare:
		vm.handleRngResult(message)

	case mul.Open:
		vm.handleMulOpen(message)

	case mul.Result:
		vm.handleMulResult(message)

	case open.Result:
		vm.handleOpenResult(message)

	default:
		log.Printf("[error] (vm) unexpected message type %T", message)
	}
}

func (vm *VM) exec(exec Exec) {
	proc := exec.proc
	vm.processes[proc.ID] = proc

	ret := proc.Exec()
	vm.processes[proc.ID] = proc

	if ret.IsReady() {
		log.Printf("[error] (vm) process is ready after execution = %v", proc.ID)
		return
	}
	if ret.IsTerminated() {
		log.Printf("[debug] (vm) process is terminated = %v", proc.ID)
		return
	}
	if ret.Intent() == nil {
		log.Printf("[debug] (vm) process is waiting = %v", proc.ID)
		return
	}

	switch intent := ret.Intent().(type) {
	case process.IntentToGenerateRn:
		vm.processIntents[proc.ID] = intent
		vm.rng.IO().Send(rng.NewGenerateRn(rng.Nonce(proc.ID)))

	case process.IntentToMultiply:
		vm.processIntents[proc.ID] = intent
		vm.mul.IO().Send(mul.NewMultiply(mul.Nonce(proc.ID), intent.X, intent.Y, intent.Rho, intent.Sigma))

	case process.IntentToOpen:
		vm.processIntents[proc.ID] = intent
		vm.open.IO().Send(open.NewOpen(open.Nonce(proc.ID), intent.Value))
		vm.io.Send(open.NewOpen(open.Nonce(proc.ID), intent.Value))

	case process.IntentToError:
		log.Printf("[error] (vm) %v", intent.Error())

	default:
		panic("unimplemented")
	}
}

func (vm *VM) invoke(message RemoteProcedureCall) {
	switch message := message.Message.(type) {

	case rng.ProposeRn, rng.LocalRnShares, rng.ProposeGlobalRnShare:
		vm.rng.IO().Send(message)

	case mul.Open:
		vm.mul.IO().Send(message)

	case open.Open:
		vm.open.IO().Send(message)

	default:
		panic("unimplemented")
	}
}

func (vm *VM) proposeRn(message rng.ProposeRn) {
	vm.io.Send(NewRemoteProcedureCall(message))
}

func (vm *VM) handleRngLocalRnShares(message rng.LocalRnShares) {
	vm.io.Send(NewRemoteProcedureCall(message))
}

func (vm *VM) handleRngProposeGlobalRnShare(message rng.ProposeGlobalRnShare) {
	vm.io.Send(NewRemoteProcedureCall(message))
}

func (vm *VM) handleRngResult(message rng.GlobalRnShare) {
	intent, ok := vm.processIntents[process.ID(message.Nonce)]
	if !ok {
		return
	}

	switch intent := intent.(type) {
	case process.IntentToGenerateRn:

		select {
		case intent.Rho <- message.RhoShare:
		default:
			log.Printf("[error] (vm, rng, ρ) unavailable intent")
		}

		select {
		case intent.Sigma <- message.SigmaShare:
		default:
			log.Printf("[error] (vm, rng, σ) unavailable intent")
		}
	default:
		log.Printf("[error] (vm, rng) unexpected intent type %T", intent)
	}

	vm.exec(NewExecMessage(vm.processes[process.ID(message.Nonce)]))
}

func (vm *VM) handleMulOpen(message mul.Open) {
	vm.io.Send(NewRemoteProcedureCall(message))
}

func (vm *VM) handleMulResult(message mul.Result) {
	intent, ok := vm.processIntents[process.ID(message.Nonce)]
	if !ok {
		return
	}

	switch intent := intent.(type) {
	case process.IntentToMultiply:
		select {
		case intent.Ret <- message.Share:
		default:
			log.Printf("[error] (vm, mul) unavailable intent")
		}
	default:
		log.Printf("[error] (vm, mul) unexpected intent type %T", intent)
	}

	vm.exec(NewExecMessage(vm.processes[process.ID(message.Nonce)]))
}

func (vm *VM) handleOpenResult(message open.Result) {
	intent, ok := vm.processIntents[process.ID(message.Nonce)]
	if !ok {
		return
	}

	switch intent := intent.(type) {
	case process.IntentToOpen:
		select {
		case intent.Ret <- message.Value:
		default:
			log.Printf("[error] (vm, open) unavailable intent")
		}
	default:
		log.Printf("[error] (vm, open) unexpected intent type %T", intent)
	}

	vm.exec(NewExecMessage(vm.processes[process.ID(message.Nonce)]))
}
