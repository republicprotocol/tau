package vm

import (
	"fmt"
	"log"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vm/mul"
	"github.com/republicprotocol/oro-go/core/vm/open"
	"github.com/republicprotocol/oro-go/core/vm/process"
	"github.com/republicprotocol/oro-go/core/vm/rng"
	"github.com/republicprotocol/oro-go/core/vss/pedersen"
)

type VM struct {
	io task.IO

	index uint64

	rng            task.Task
	mul            task.Task
	open           task.Task
	processes      map[process.ID]process.Process
	processIntents map[[32]byte]process.Intent
}

func New(scheme pedersen.Pedersen, index, n, k uint64, cap int) VM {
	vm := VM{
		io: task.NewIO(cap),

		index: index,

		rng:            rng.New(scheme, index, n, k, n-k, cap),
		mul:            mul.New(n, k, cap),
		open:           open.New(n, k, cap),
		processes:      map[process.ID]process.Process{},
		processIntents: map[[32]byte]process.Intent{},
	}
	return vm
}

func (vm *VM) Channel() task.Channel {
	return vm.io.Channel()
}

func (vm *VM) Run(done <-chan struct{}) {
	defer log.Printf("[info] (vm) terminating")

	co.ParBegin(
		func() {
			vm.runBackgroundGoroutines(done)
		},
		func() {
			for {
				message, ok := vm.io.Flush(done, vm.rng.Channel(), vm.mul.Channel(), vm.open.Channel())
				if !ok {
					return
				}
				if message != nil {
					vm.recv(message)
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

func (vm *VM) recv(message task.Message) {
	switch message := message.(type) {

	case Exec:
		vm.exec(message)

	case RemoteProcedureCall:
		vm.invoke(message)

	case rng.RnShares:
		vm.recvInternalRnShares(message)

	case rng.Result:
		vm.recvInternalRngResult(message)

	case mul.OpenMul:
		vm.recvInternalOpenMul(message)

	case mul.Result:
		vm.recvInternalMulResult(message)

	case open.Open:
		vm.recvInternalOpen(message)

	case open.Result:
		vm.recvInternalOpenResult(message)

	case task.Error:
		log.Printf("[error] (vm) %v", message.Error())

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (vm *VM) exec(exec Exec) {
	proc := exec.proc
	vm.processes[proc.ID] = proc

	ret := proc.Exec()
	vm.processes[proc.ID] = proc

	if ret.IsReady() {
		vm.io.Write(task.NewError(fmt.Errorf("process is ready after execution = %v", proc.ID)))
		return
	}
	if ret.IsTerminated() {
		result, err := proc.Stack.Pop()
		if err != nil {
			vm.io.Write(task.NewError(err))
			return
		}
		vm.io.Write(NewResult(result.(process.Value)))
		return
	}
	if ret.Intent() == nil {
		log.Printf("[debug] (vm %v) process is waiting = %v", vm.index, proc.ID)
		return
	}

	switch intent := ret.Intent().(type) {
	case process.IntentToGenerateRn:
		vm.processIntents[proc.Nonce()] = intent
		vm.rng.Channel().Send(rng.NewSignalGenerateRn(task.MessageID(proc.Nonce())))

	case process.IntentToMultiply:
		vm.processIntents[proc.Nonce()] = intent
		vm.mul.Channel().Send(mul.NewSignalMul(task.MessageID(proc.Nonce()), intent.X, intent.Y, intent.Rho, intent.Sigma))

	case process.IntentToOpen:
		vm.processIntents[proc.Nonce()] = intent
		vm.open.Channel().Send(open.NewSignal(task.MessageID(proc.Nonce()), intent.Value))

	case process.IntentToError:
		vm.io.Write(task.NewError(intent))
		return

	default:
		panic("unimplemented")
	}
}

func (vm *VM) invoke(message RemoteProcedureCall) {
	switch message := message.Message.(type) {

	case rng.RnShares, rng.ProposeRnShare:
		vm.rng.Channel().Send(message)

	case mul.OpenMul:
		vm.mul.Channel().Send(message)

	case open.Open:
		vm.open.Channel().Send(message)

	default:
		panic(fmt.Sprintf("unexpected rpc type %T", message))
	}
}

func (vm *VM) recvInternalRnShares(message rng.RnShares) {
	vm.io.Write(NewRemoteProcedureCall(message))
}

func (vm *VM) recvInternalRngResult(message rng.Result) {
	intent, ok := vm.processIntents[message.MessageID]
	if !ok {
		return
	}

	switch intent := intent.(type) {
	case process.IntentToGenerateRn:

		select {
		case intent.Rho <- message.Rho:
		default:
			log.Printf("[error] (vm, rng, ρ) unavailable intent")
		}

		select {
		case intent.Sigma <- message.Sigma:
		default:
			log.Printf("[error] (vm, rng, σ) unavailable intent")
		}
	default:
		// FIXME: Handle intent transitioning correctly.
		log.Printf("[error] (vm, rng) unexpected intent type %T", intent)
		return
	}

	delete(vm.processIntents, message.MessageID)

	pid := [31]byte{}
	copy(pid[:], message.MessageID[1:])
	vm.exec(NewExec(vm.processes[process.ID(pid)]))
}

func (vm *VM) recvInternalOpenMul(message mul.OpenMul) {
	vm.io.Write(NewRemoteProcedureCall(message))
}

func (vm *VM) recvInternalMulResult(message mul.Result) {
	intent, ok := vm.processIntents[message.MessageID]
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
		// FIXME: Handle intent transitioning correctly.
		log.Printf("[error] (vm, mul) <%v> unexpected intent type %T", vm.index, intent)
		return
	}

	delete(vm.processIntents, message.MessageID)

	pid := [31]byte{}
	copy(pid[:], message.MessageID[1:])
	vm.exec(NewExec(vm.processes[process.ID(pid)]))
}

func (vm *VM) recvInternalOpen(message open.Open) {
	vm.io.Write(NewRemoteProcedureCall(message))
}

func (vm *VM) recvInternalOpenResult(message open.Result) {
	intent, ok := vm.processIntents[(message.MessageID)]
	if !ok {
		return
	}

	switch intent := intent.(type) {
	case process.IntentToOpen:
		select {
		case intent.Ret <- message.Value:
		default:
			log.Printf("[error] (vm, open) unavailable intent")
			return
		}
	default:
		log.Printf("[error] (vm, open) unexpected intent type %T", intent)
		return
	}

	delete(vm.processIntents, message.MessageID)

	pid := [31]byte{}
	copy(pid[:], message.MessageID[1:])
	vm.exec(NewExec(vm.processes[process.ID(pid)]))
}
