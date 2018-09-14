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
	rng := rng.New(scheme, index, n, k, n-k, cap)
	mul := mul.New(n, k, cap)
	open := open.New(n, k, cap)
	vm := newVM(scheme, index, rng, mul, open)
	return task.New(task.NewIO(cap), vm, vm.rng, vm.mul, vm.open)
}

func newVM(scheme pedersen.Pedersen, index uint64, rng, mul, open task.Task) *VM {
	return &VM{
		index: index,

		rng:            rng,
		mul:            mul,
		open:           open,
		processes:      map[process.ID]process.Process{},
		processIntents: map[task.MessageID]process.Intent{},
	}
}

func (vm *VM) Reduce(message task.Message) task.Message {
	switch message := message.(type) {

	case Exec:
		return vm.exec(message)

	case RemoteProcedureCall:
		return vm.invoke(message)

	case rng.RnShares:
		return vm.recvInternalRnShares(message)

	case rng.ProposeRnShare:
		return vm.recvInternalRngProposeRnShare(message)

	case rng.Result:
		return vm.recvInternalRngResult(message)

	case mul.OpenMul:
		return vm.recvInternalOpenMul(message)

	case mul.Result:
		return vm.recvInternalMulResult(message)

	case open.Open:
		return vm.recvInternalOpen(message)

	case open.Result:
		return vm.recvInternalOpenResult(message)

	case task.Error:
		return task.NewError(fmt.Errorf("[error] (vm) %v", message.Error()))

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
		return task.NewError(fmt.Errorf("process %v is ready after execution", proc.ID))
	}
	if ret.IsTerminated() {
		result, err := proc.Stack.Pop()
		if err != nil {
			return task.NewError(err)
		}
		return NewResult(result.(process.Value))
	}
	if ret.Intent() == nil {
		log.Printf("[debug] (vm %v) process is waiting = %v", vm.index, proc.ID)
		return nil
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

func (vm *VM) invoke(message RemoteProcedureCall) task.Message {
	switch message := message.Message.(type) {

	case rng.RnShares, rng.ProposeRnShare:
		vm.rng.Send(message)

	case mul.OpenMul:
		vm.mul.Send(message)

	case open.Open:
		vm.open.Send(message)

	default:
		panic(fmt.Sprintf("unexpected rpc type %T", message))
	}

	return nil
}

func (vm *VM) recvInternalRnShares(message rng.RnShares) task.Message {
	return NewRemoteProcedureCall(message)
}

func (vm *VM) recvInternalRngProposeRnShare(message rng.ProposeRnShare) task.Message {
	return NewRemoteProcedureCall(message)
}

func (vm *VM) recvInternalRngResult(message rng.Result) task.Message {
	intent, ok := vm.processIntents[message.MessageID]
	if !ok {
		return nil
	}

	switch intent := intent.(type) {
	case process.IntentToGenerateRn:

		select {
		case intent.Rho <- message.Rho:
		default:
			return task.NewError(fmt.Errorf("(vm, rng, ρ) unavailable intent"))
		}

		select {
		case intent.Sigma <- message.Sigma:
		default:
			return task.NewError(fmt.Errorf("[error] (vm, rng, σ) unavailable intent"))
		}

	default:
		// FIXME: Handle intent transitioning correctly.
		return task.NewError(fmt.Errorf("[error] (vm, rng) unexpected intent type %T", intent))
	}

	delete(vm.processIntents, message.MessageID)

	return vm.exec(NewExec(vm.processes[msgidToPid(message.MessageID)]))
}

func (vm *VM) recvInternalOpenMul(message mul.OpenMul) task.Message {
	return NewRemoteProcedureCall(message)
}

func (vm *VM) recvInternalMulResult(message mul.Result) task.Message {
	intent, ok := vm.processIntents[message.MessageID]
	if !ok {
		return nil
	}

	switch intent := intent.(type) {
	case process.IntentToMultiply:
		select {
		case intent.Ret <- message.Share:
		default:
			return task.NewError(fmt.Errorf("[error] (vm, mul) unavailable intent"))
		}
	default:
		// FIXME: Handle intent transitioning correctly.
		return task.NewError(fmt.Errorf("[error] (vm, mul) <%v> unexpected intent type %T", vm.index, intent))
	}

	delete(vm.processIntents, message.MessageID)

	return vm.exec(NewExec(vm.processes[msgidToPid(message.MessageID)]))
}

func (vm *VM) recvInternalOpen(message open.Open) task.Message {
	return NewRemoteProcedureCall(message)
}

func (vm *VM) recvInternalOpenResult(message open.Result) task.Message {
	intent, ok := vm.processIntents[(message.MessageID)]
	if !ok {
		return nil
	}

	switch intent := intent.(type) {
	case process.IntentToOpen:
		select {
		case intent.Ret <- message.Value:
		default:
			return task.NewError(fmt.Errorf("[error] (vm, open) unavailable intent"))
		}
	default:
		return task.NewError(fmt.Errorf("[error] (vm, open) unexpected intent type %T", intent))
	}

	delete(vm.processIntents, message.MessageID)

	return vm.exec(NewExec(vm.processes[msgidToPid(message.MessageID)]))
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
