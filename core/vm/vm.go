package vm

import (
	"fmt"
	"log"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vm/mul"
	"github.com/republicprotocol/oro-go/core/vm/open"
	"github.com/republicprotocol/oro-go/core/vm/process"
	"github.com/republicprotocol/oro-go/core/vm/rng"
	"github.com/republicprotocol/oro-go/core/vss/pedersen"
)

type VM struct {
	index          uint64
	processes      map[process.ID]process.Process
	processIntents map[task.MessageID]process.Intent

	rng  task.Task
	mul  task.Task
	open task.Task
}

func New(scheme pedersen.Pedersen, index, n, k uint64, cap int) task.Task {
	return newTask(newVM(scheme, index, n, k, cap), cap)
}

func newTask(vm *VM, cap int) task.Task {
	return task.New(cap, vm, vm.rng, vm.mul, vm.open)
}

func newVM(scheme pedersen.Pedersen, index, n, k uint64, cap int) VM {
	return &VM{
		index: index,

		rng:            rng.New(scheme, index, n, k, n-k, cap),
		mul:            mul.New(n, k, cap),
		open:           open.New(n, k, cap),
		processes:      map[process.ID]process.Process{},
		processIntents: map[task.MessageID]process.Intent{},
	}
}

func (vm *VM) Reduce(message task.Message) task.Message {
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

func (vm *VM) exec(exec Exec) task.Message {
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
		vm.processIntents[pidToMsgid(proc.ID, proc.PC)] = intent
		vm.rng.Send(rng.NewSignalGenerateRn(pidToMsgid(proc.ID, proc.PC)))

	case process.IntentToMultiply:
		vm.processIntents[pidToMsgid(proc.ID, proc.PC)] = intent
		vm.mul.Send(mul.NewSignalMul(task.MessageID(pidToMsgid(proc.ID, proc.PC)), intent.X, intent.Y, intent.Rho, intent.Sigma))

	case process.IntentToOpen:
		vm.processIntents[pidToMsgid(proc.ID, proc.PC)] = intent
		vm.open.Send(open.NewSignal(task.MessageID(pidToMsgid(proc.ID, proc.PC)), intent.Value))

	case process.IntentToError:
		return task.NewError(intent)

	default:
		panic("unimplemented")
	}

	return nil
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

	vm.exec(NewExec(vm.processes[msgidToPid(message.MessageID)]))
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

	vm.exec(NewExec(vm.processes[msgidToPid(message.MessageID)]))
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

	vm.exec(NewExec(vm.processes[msgidToPid(message.MessageID)]))
}

func pidToMsgid(pid process.ID, pc process.PC) task.MessageID {
	id := task.MessageID{}
	copy(id[:32], pid[:32])
	id[32] = byte(pc)
	id[33] = byte(pc >> 8)
	id[34] = byte(pc >> 16)
	id[35] = byte(pc >> 24)
	id[36] = byte(pc >> 32)
	id[37] = byte(pc >> 40)
	id[38] = byte(pc >> 48)
	id[39] = byte(pc >> 56)
	return id
}

func msgidToPid(msgid task.MessageID) process.ID {
	pid := process.ID{}
	copy(pid[:32], msgid[:32])
	return pid
}
