package vm

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vm/asm"
	"github.com/republicprotocol/oro-go/core/vm/mul"
	"github.com/republicprotocol/oro-go/core/vm/open"
	"github.com/republicprotocol/oro-go/core/vm/proc"
	"github.com/republicprotocol/oro-go/core/vm/rng"
	"github.com/republicprotocol/oro-go/core/vss/pedersen"
)

type VM struct {
	index   uint64
	procs   map[proc.ID]proc.Proc
	intents map[proc.IntentID]proc.Intent

	rng  task.Task
	mul  task.Task
	open task.Task
}

func New(scheme pedersen.Pedersen, index, n, k uint64, cap int) task.Task {
	rng := rng.New(scheme, index, n, k, cap)
	mul := mul.New(index, n, k, cap)
	open := open.New(index, n, k, cap)
	vm := newVM(scheme, index, rng, mul, open)
	return task.New(task.NewIO(cap), vm, vm.rng, vm.mul, vm.open)
}

func newVM(scheme pedersen.Pedersen, index uint64, rng, mul, open task.Task) *VM {
	return &VM{
		index: index,

		rng:     rng,
		mul:     mul,
		open:    open,
		procs:   map[proc.ID]proc.Proc{},
		intents: map[proc.IntentID]proc.Intent{},
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

	case mul.BroadcastMulShares:
		return vm.recvInternalOpenBroadcastMulShares(message)

	case mul.Result:
		return vm.recvInternalMulResult(message)

	case open.BroadcastShares:
		return vm.recvInternalOpenBroadcastShares(message)

	case open.Result:
		return vm.recvInternalOpenResult(message)

	case task.Error:
		return task.NewError(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (vm *VM) exec(exec Exec) task.Message {
	process := exec.process
	vm.procs[process.ID] = process

	intent := process.Exec()
	vm.procs[process.ID] = process

	return vm.execIntent(process, intent)
}

func (vm *VM) execIntent(process proc.Proc, intent proc.Intent) task.Message {

	switch state := intent.State().(type) {
	case *asm.InstGenerateRnState:
		vm.intents[intent.IID()] = intent
		vm.rng.Send(rng.NewGenerateRn(iidToMsgid(intent.IID()), state.Num))

	case *asm.InstGenerateRnZeroState:
		vm.intents[intent.IID()] = intent
		vm.rng.Send(rng.NewGenerateRnZero(iidToMsgid(intent.IID()), state.Num))

	case *asm.InstGenerateRnTupleState:
		vm.intents[intent.IID()] = intent
		vm.rng.Send(rng.NewGenerateRnTuple(iidToMsgid(intent.IID()), state.Num))

	case *asm.InstMulState:
		vm.intents[intent.IID()] = intent
		vm.mul.Send(mul.NewMul(iidToMsgid(intent.IID()), state.Xs, state.Ys, state.Rhos, state.Sigmas))

	case *asm.InstOpenState:
		vm.intents[intent.IID()] = intent
		vm.open.Send(open.NewOpen(iidToMsgid(intent.IID()), state.Shares))

	case *asm.InstExitState:
		return NewResult(state.Values)

	default:
		panic(fmt.Sprintf("unexpected intent type %T", intent))
	}
	return nil
}

func (vm *VM) invoke(message RemoteProcedureCall) task.Message {
	switch message := message.Message.(type) {

	case rng.RnShares, rng.ProposeRnShare:
		vm.rng.Send(message)

	case mul.BroadcastMulShares:
		vm.mul.Send(message)

	case open.BroadcastShares:
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

	switch state := intent.State().(type) {
	case *asm.InstGenerateRnState:
		state.Sigmas = message.Sigmas

	case *asm.InstGenerateRnZeroState:
		state.Sigmas = message.Sigmas

	case *asm.InstGenerateRnTupleState:
		state.Rhos = message.Rhos
		state.Sigmas = message.Sigmas

	default:
		panic(fmt.Sprintf("unexpected intent type %T", intent))
	}
	delete(vm.intents, msgidToIID(message.MessageID))

	return vm.exec(NewExec(vm.procs[msgidToPid(message.MessageID)]))
}

func (vm *VM) recvInternalOpenBroadcastMulShares(message mul.BroadcastMulShares) task.Message {
	return NewRemoteProcedureCall(message)
}

func (vm *VM) recvInternalMulResult(message mul.Result) task.Message {
	intent, ok := vm.intents[msgidToIID(message.MessageID)]
	if !ok {
		return nil
	}

	switch state := intent.State().(type) {
	case *asm.InstMulState:
		state.Results = message.Shares

	default:
		return task.NewError(fmt.Errorf("unexpected intent type %T", intent))
	}
	delete(vm.intents, msgidToIID(message.MessageID))

	return vm.exec(NewExec(vm.procs[msgidToPid(message.MessageID)]))
}

func (vm *VM) recvInternalOpenBroadcastShares(message open.BroadcastShares) task.Message {
	return NewRemoteProcedureCall(message)
}

func (vm *VM) recvInternalOpenResult(message open.Result) task.Message {
	intent, ok := vm.intents[msgidToIID(message.MessageID)]
	if !ok {
		return nil
	}

	switch state := intent.State().(type) {
	case *asm.InstOpenState:
		state.Results = message.Values

	default:
		return task.NewError(fmt.Errorf("unexpected intent type %T", intent))
	}
	delete(vm.intents, msgidToIID(message.MessageID))

	return vm.exec(NewExec(vm.procs[msgidToPid(message.MessageID)]))
}

func iidToMsgid(iid proc.IntentID) task.MessageID {
	id := task.MessageID{}
	copy(id[:40], iid[:40])
	return id
}

func msgidToIID(msgid task.MessageID) proc.IntentID {
	iid := proc.IntentID{}
	copy(iid[:40], msgid[:40])
	return iid
}

func msgidToPid(msgid task.MessageID) proc.ID {
	pid := proc.ID{}
	copy(pid[:32], msgid[:32])
	return pid
}

type Exec struct {
	process proc.Proc
}

func NewExec(process proc.Proc) Exec {
	return Exec{
		process,
	}
}

func (message Exec) IsMessage() {
}

type Result struct {
	Values []asm.Value
}

func NewResult(values []asm.Value) Result {
	return Result{values}
}

func (message Result) IsMessage() {
}

type RemoteProcedureCall struct {
	Message task.Message
}

func NewRemoteProcedureCall(message task.Message) RemoteProcedureCall {
	return RemoteProcedureCall{
		message,
	}
}

func (message RemoteProcedureCall) IsMessage() {
}
