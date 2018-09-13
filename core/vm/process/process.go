package process

import (
	"log"

	"github.com/republicprotocol/oro-go/core/stack"
	"github.com/republicprotocol/oro-go/core/vss/algebra"

	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type Return struct {
	intent     Intent
	ready      bool
	terminated bool
}

func Ready() Return {
	return Return{
		intent:     nil,
		ready:      true,
		terminated: false,
	}
}

func NotReady(intent Intent) Return {
	return Return{
		intent:     intent,
		ready:      false,
		terminated: false,
	}
}

func Terminated() Return {
	return Return{
		intent:     nil,
		ready:      false,
		terminated: true,
	}
}

func (ret Return) Intent() Intent {
	return ret.intent
}

func (ret Return) IsReady() bool {
	return ret.ready
}

func (ret Return) IsTerminated() bool {
	return ret.terminated
}

type ID [32]byte

type Process struct {
	ID
	stack.Stack
	Memory
	Code
	PC
}

func New(id ID, stack stack.Stack, mem Memory, code Code) Process {
	expandMacros(&code)
	return Process{
		ID:     id,
		Stack:  stack,
		Memory: mem,
		Code:   code,
		PC:     0,
	}
}

func (proc *Process) Exec() Return {
	ret := Ready()

	for ret.IsReady() {
		if proc.PC == PC(len(proc.Code)) {
			return Terminated()
		}

		switch inst := proc.Code[proc.PC].(type) {
		case instPush:
			ret = proc.execInstPush(inst)
		case instCopy:
			ret = proc.execInstCopy(inst)
		case instStore:
			ret = proc.execInstStore(inst)
		case instLoad:
			ret = proc.execInstLoad(inst)
		case instLoadStack:
			ret = proc.execInstLoadStack(inst)
		case instAdd:
			ret = proc.execInstAdd(inst)
		case instNeg:
			ret = proc.execInstNeg(inst)
		case instSub:
			ret = proc.execInstSub(inst)
		case instGenerateRn:
			ret = proc.execInstGenerateRn(inst)
		case instMul:
			ret = proc.execInstMul(inst)
		case instOpen:
			ret = proc.execInstOpen(inst)

		default:
			ret = NotReady(ErrorUnexpectedInst(inst, proc.PC))
		}
	}

	return ret
}

func (proc *Process) execInstPush(inst instPush) Return {
	if err := proc.Stack.Push(inst.value); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	proc.PC++
	return Ready()
}

func (proc *Process) execInstCopy(inst instCopy) Return {
	values := make([]stack.Element, inst.depth)
	for i := range values {
		value, err := proc.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
		values[len(values)-i-1] = value
	}

	values = append(values, values...)
	for _, value := range values {
		if err := proc.Stack.Push(value); err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
	}

	proc.PC++
	return Ready()
}

func (proc *Process) execInstStore(inst instStore) Return {
	value, err := proc.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}
	proc.Memory[inst.addr] = value.(Value)
	proc.PC++
	return Ready()
}

func (proc *Process) execInstLoad(inst instLoad) Return {
	value, ok := proc.Memory[inst.addr]
	if !ok {
		return NotReady(ErrorInvalidMemoryAddr(inst.addr, proc.PC))
	}
	if err := proc.Stack.Push(value); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}
	proc.PC++
	return Ready()
}

func (proc *Process) execInstLoadStack(inst instLoadStack) Return {
	value, err := proc.Stack.Peek(int(inst.offset))
	if err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}
	if err := proc.Stack.Push(value); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}
	proc.PC++
	return Ready()
}

func (proc *Process) execInstAdd(inst instAdd) Return {
	rhs, err := proc.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	lhs, err := proc.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	ret := Value(nil)
	switch lhs := lhs.(type) {
	case ValuePublic:
		ret = lhs.Add(rhs.(Value))
	case ValuePrivate:
		ret = lhs.Add(rhs.(Value))
	default:
		return NotReady(ErrorUnexpectedTypeConversion(lhs, nil, proc.PC))
	}
	if err := proc.Stack.Push(ret); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	proc.PC++
	return Ready()
}

func (proc *Process) execInstNeg(inst instNeg) Return {
	lhs, err := proc.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	ret := Value(nil)
	switch lhs := lhs.(type) {
	case ValuePublic:
		ret = lhs.Neg()
	case ValuePrivate:
		ret = lhs.Neg()
	default:
		return NotReady(ErrorUnexpectedTypeConversion(lhs, nil, proc.PC))
	}
	if err := proc.Stack.Push(ret); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	proc.PC++
	return Ready()
}

func (proc *Process) execInstSub(inst instSub) Return {
	rhs, err := proc.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	lhs, err := proc.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	ret := Value(nil)
	switch lhs := lhs.(type) {
	case ValuePublic:
		ret = lhs.Sub(rhs.(Value))
	case ValuePrivate:
		ret = lhs.Sub(rhs.(Value))
	default:
		return NotReady(ErrorUnexpectedTypeConversion(lhs, nil, proc.PC))
	}
	if err := proc.Stack.Push(ret); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	proc.PC++
	return Ready()
}

func (proc *Process) execInstGenerateRn(inst instGenerateRn) Return {
	if inst.ρCh == nil || inst.σCh == nil {
		ρCh := make(chan shamir.Share, 1)
		σCh := make(chan shamir.Share, 1)
		inst.ρCh = ρCh
		inst.σCh = σCh
		proc.Code[proc.PC] = inst
		return NotReady(GenerateRn(ρCh, σCh))
	}

	if !inst.ρReady {
		select {
		case ρ := <-inst.ρCh:
			inst.ρReady = true
			inst.ρ = ρ
			proc.Code[proc.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	if !inst.σReady {
		select {
		case σ := <-inst.σCh:
			inst.σReady = true
			inst.σ = σ
			proc.Code[proc.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	if err := proc.Push(ValuePrivateRn{
		Rho:   inst.ρ,
		Sigma: inst.σ,
	}); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	proc.PC++
	return Ready()
}

func (proc *Process) execInstMul(inst instMul) Return {
	if inst.retCh == nil {

		rnValue, err := proc.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
		rn, ok := rnValue.(ValuePrivateRn)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(rnValue, ValuePrivateRn{}, proc.PC))
		}

		yValue, err := proc.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
		y, ok := yValue.(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(yValue, ValuePrivate{}, proc.PC))
		}

		xValue, err := proc.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
		x, ok := xValue.(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(xValue, ValuePrivate{}, proc.PC))
		}

		retCh := make(chan shamir.Share, 1)
		inst.retCh = retCh
		proc.Code[proc.PC] = inst
		return NotReady(Multiply(x.Share, y.Share, rn.Rho, rn.Sigma, retCh))
	}

	if !inst.retReady {
		select {
		case ret := <-inst.retCh:
			inst.retReady = true
			inst.ret = ret
			proc.Code[proc.PC] = inst
		default:
			log.Printf("[error] (proc) still waiting")
			return NotReady(nil)
		}
	}

	proc.Push(ValuePrivate{
		Share: inst.ret,
	})

	proc.PC++
	return Ready()
}

func (proc *Process) execInstOpen(inst instOpen) Return {
	if inst.retCh == nil {

		value, err := proc.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
		v, ok := value.(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(value, ValuePrivate{}, proc.PC))
		}

		retCh := make(chan algebra.FpElement, 1)
		inst.retCh = retCh
		proc.Code[proc.PC] = inst
		return NotReady(Open(v.Share, retCh))
	}

	if !inst.retReady {
		select {
		case ret := <-inst.retCh:
			inst.retReady = true
			inst.ret = ret
			proc.Code[proc.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	proc.Push(ValuePublic{
		Value: inst.ret,
	})

	proc.PC++
	return Ready()
}

func expandMacros(code *Code) {
	for i := 0; i < len(*code); i++ {
		if inst, ok := (*code)[i].(instMacro); ok {
			temp := append(inst.code, (*code)[i+1:]...)
			*code = append((*code)[:i], temp...)
			i--
		}
	}
}
