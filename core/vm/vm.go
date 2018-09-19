package vm

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vm/mul"
	"github.com/republicprotocol/oro-go/core/vm/open"
	"github.com/republicprotocol/oro-go/core/vm/process"
	"github.com/republicprotocol/oro-go/core/vm/rng"
	"github.com/republicprotocol/oro-go/core/vss/pedersen"
)

type VM struct {
	index   uint64
	procs   map[process.ID]process.Process
	intents map[process.IntentID]process.Intent

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

		rng:     rng,
		mul:     mul,
		open:    open,
		procs:   map[process.ID]process.Process{},
		intents: map[process.IntentID]process.Intent{},
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
		return task.NewError(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (vm *VM) exec(exec Exec) task.Message {
	proc := exec.proc
	vm.procs[proc.ID] = proc

	ret := proc.Exec()
	vm.procs[proc.ID] = proc

	if ret.IsReady() {
		return task.NewError(fmt.Errorf("process %v is ready after execution", proc.ID))
	}
	if ret.Intent() == nil {
		return task.NewError(fmt.Errorf("process %v has no intent after execution", proc.ID))
	}

	switch intent := ret.Intent().(type) {
	case process.IntentToError:
		return task.NewError(intent)

	case process.IntentToExit:
		return NewResult(intent.Values)

	case process.IntentToAwait:
		vm.intents[intent.IntentID()] = intent
		return vm.execAwaitIntent(proc, intent)

	default:
		vm.intents[intent.IntentID()] = intent
		return vm.execAsyncIntent(proc, intent)
	}
}

func (vm *VM) execAsyncIntent(proc process.Process, intent process.Intent) task.Message {
	switch intent := intent.(type) {
	case process.IntentToGenerateRn:
		vm.rng.Send(rng.NewGenerateRn(iidToMsgid(intent.IntentID())))

	case process.IntentToGenerateRnZero:
		vm.rng.Send(rng.NewGenerateRnZero(iidToMsgid(intent.IntentID())))

	case process.IntentToGenerateRnTuple:
		vm.rng.Send(rng.NewGenerateRnTuple(iidToMsgid(intent.IntentID())))

	case process.IntentToMultiply:
		vm.mul.Send(mul.NewSignalMul(iidToMsgid(intent.IntentID()), intent.X, intent.Y, intent.Rho, intent.Sigma))

	case process.IntentToOpen:
		vm.open.Send(open.NewSignal(iidToMsgid(intent.IntentID()), intent.Value))

	default:
		panic(fmt.Sprintf("unexpected intent type %T", intent))
	}
	return nil
}

func (vm *VM) execAwaitIntent(proc process.Process, intent process.IntentToAwait) task.Message {
	vm.intents[intent.IntentID()] = intent
	messages := []task.Message{}
	for _, intent := range intent.Intents {
		if intent != nil {
			if message := vm.execAsyncIntent(proc, intent); message != nil {
				messages = append(messages, message)
			}
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return task.NewMessageBatch(messages)
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
	intent, ok := vm.intents[msgidToIID(message.MessageID)]
	if !ok {
		return nil
	}

	switch intent := intent.(type) {
	case process.IntentToGenerateRn:

		select {
		case intent.Sigma <- message.Sigma.Share():
		default:
			return task.NewError(fmt.Errorf("unavailable intent"))
		}

	case process.IntentToGenerateRnZero:

		select {
		case intent.Sigma <- message.Sigma.Share():
		default:
			return task.NewError(fmt.Errorf("unavailable intent"))
		}

	case process.IntentToGenerateRnTuple:

		select {
		case intent.Rho <- message.Rho.Share():
		default:
			return task.NewError(fmt.Errorf("unavailable intent"))
		}

		select {
		case intent.Sigma <- message.Sigma.Share():
		default:
			return task.NewError(fmt.Errorf("unavailable intent"))
		}

	default:
		panic(fmt.Sprintf("unexpected intent type %T", intent))
	}

	delete(vm.intents, msgidToIID(message.MessageID))

	return vm.exec(NewExec(vm.procs[msgidToPid(message.MessageID)]))
}

func (vm *VM) recvInternalOpenMul(message mul.OpenMul) task.Message {
	return NewRemoteProcedureCall(message)
}

func (vm *VM) recvInternalMulResult(message mul.Result) task.Message {
	intent, ok := vm.intents[msgidToIID(message.MessageID)]
	if !ok {
		return nil
	}

	switch intent := intent.(type) {
	case process.IntentToMultiply:
		select {
		case intent.Ret <- message.Share:
		default:
			return task.NewError(fmt.Errorf("unavailable intent"))
		}
	default:
		return task.NewError(fmt.Errorf("unexpected intent type %T", intent))
	}

	delete(vm.intents, msgidToIID(message.MessageID))

	return vm.exec(NewExec(vm.procs[msgidToPid(message.MessageID)]))
}

func (vm *VM) recvInternalOpen(message open.Open) task.Message {
	return NewRemoteProcedureCall(message)
}

func (vm *VM) recvInternalOpenResult(message open.Result) task.Message {
	intent, ok := vm.intents[msgidToIID(message.MessageID)]
	if !ok {
		return nil
	}

	switch intent := intent.(type) {
	case process.IntentToOpen:
		select {
		case intent.Ret <- message.Value:
		default:
			return task.NewError(fmt.Errorf("unavailable intent"))
		}
	default:
		return task.NewError(fmt.Errorf("unexpected intent type %T", intent))
	}

	delete(vm.intents, msgidToIID(message.MessageID))

	return vm.exec(NewExec(vm.procs[msgidToPid(message.MessageID)]))
}

func iidToMsgid(iid process.IntentID) task.MessageID {
	id := task.MessageID{}
	copy(id[:40], iid[:40])
	return id
}

func msgidToIID(msgid task.MessageID) process.IntentID {
	iid := process.IntentID{}
	copy(iid[:40], msgid[:40])
	return iid
}

func msgidToPid(msgid task.MessageID) process.ID {
	pid := process.ID{}
	copy(pid[:32], msgid[:32])
	return pid
}
