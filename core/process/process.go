package process

import (
	"github.com/republicprotocol/smpc-go/core/vss/algebra"

	"github.com/republicprotocol/smpc-go/core/vss/shamir"
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
	Stack
	Memory
	Code
	PC
}

func New(id ID, stack Stack, mem Memory, code Code) Process {
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
		case InstPush:
			ret = proc.execInstPush(inst)
		case InstAdd:
			ret = proc.execInstAdd(inst)
		case InstRand:
			ret = proc.execInstRand(inst)
		case InstMul:
			ret = proc.execInstMul(inst)
		case InstOpen:
			ret = proc.execInstOpen(inst)
		default:
			ret = NotReady(ErrorUnexpectedInst(inst, proc.PC))
		}
	}

	return ret
}

func (proc *Process) execInstPush(inst InstPush) Return {
	if err := proc.Stack.Push(inst.Value); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	proc.PC++
	return Ready()
}

func (proc *Process) execInstAdd(inst InstAdd) Return {
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
		ret = lhs.Add(rhs)
	case ValuePrivate:
		ret = lhs.Add(rhs)
	default:
		return NotReady(ErrorUnexpectedValue(lhs, nil, proc.PC))
	}
	if err := proc.Stack.Push(ret); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	proc.PC++
	return Ready()
}

func (proc *Process) execInstRand(inst InstRand) Return {
	if inst.RhoCh == nil || inst.SigmaCh == nil {
		ρCh := make(chan shamir.Share, 1)
		σCh := make(chan shamir.Share, 1)
		inst.RhoCh = ρCh
		inst.SigmaCh = σCh
		proc.Code[proc.PC] = inst
		return NotReady(GenerateRn(ρCh, σCh))
	}

	if !inst.RhoReady {
		select {
		case ρ := <-inst.RhoCh:
			inst.RhoReady = true
			inst.Rho = ρ
			proc.Code[proc.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	if !inst.SigmaReady {
		select {
		case σ := <-inst.SigmaCh:
			inst.SigmaReady = true
			inst.Sigma = σ
			proc.Code[proc.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	proc.Push(ValuePrivateRn{
		Rho:   inst.Rho,
		Sigma: inst.Sigma,
	})

	proc.PC++
	return Ready()
}

func (proc *Process) execInstMul(inst InstMul) Return {
	if inst.RetCh == nil {

		rnValue, err := proc.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
		rn, ok := rnValue.(ValuePrivateRn)
		if !ok {
			return NotReady(ErrorUnexpectedValue(rnValue, ValuePrivateRn{}, proc.PC))
		}

		yValue, err := proc.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
		y, ok := yValue.(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedValue(yValue, ValuePrivate{}, proc.PC))
		}

		xValue, err := proc.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
		x, ok := xValue.(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedValue(xValue, ValuePrivate{}, proc.PC))
		}

		retCh := make(chan shamir.Share, 1)
		inst.RetCh = retCh
		proc.Code[proc.PC] = inst
		return NotReady(Multiply(x.Share, y.Share, rn.Rho, rn.Sigma, retCh))
	}

	if !inst.RetReady {
		select {
		case ret := <-inst.RetCh:
			inst.RetReady = true
			inst.Ret = ret
			proc.Code[proc.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	proc.Push(ValuePrivate{
		Share: inst.Ret,
	})

	proc.PC++
	return Ready()
}

func (proc *Process) execInstOpen(inst InstOpen) Return {
	if inst.RetCh == nil {

		value, err := proc.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, proc.PC))
		}
		v, ok := value.(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedValue(value, ValuePrivate{}, proc.PC))
		}

		retCh := make(chan algebra.FpElement, 1)
		inst.RetCh = retCh
		proc.Code[proc.PC] = inst
		return NotReady(Open(v.Share, retCh))
	}

	if !inst.RetReady {
		select {
		case ret := <-inst.RetCh:
			inst.RetReady = true
			inst.Ret = ret
			proc.Code[proc.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	proc.Push(ValuePublic{
		Value: inst.Ret,
	})

	proc.PC++
	return Ready()
}
