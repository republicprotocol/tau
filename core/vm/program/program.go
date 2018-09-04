package program

import (
	"math/big"

	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

type Return struct {
	intent Intent
	ready  bool
}

func Ready() Return {
	return Return{
		intent: nil,
		ready:  true,
	}
}

func NotReady(intent Intent) Return {
	return Return{
		intent: intent,
		ready:  false,
	}
}

func (ret Return) Intent() Intent {
	return ret.intent
}

func (ret Return) IsReady() bool {
	return ret.ready
}

type ID [32]byte

type Program struct {
	ID
	Stack
	Memory
	Code
	PC
}

func New(id ID, stack Stack, mem Memory, code Code) Program {
	return Program{
		ID:     id,
		Stack:  stack,
		Memory: mem,
		Code:   code,
		PC:     0,
	}
}

func (prog *Program) Exec() Return {
	if prog.PC >= PC(len(prog.Code)) {
		return NotReady(ErrorCodeOverflow(prog.PC))
	}

	switch inst := prog.Code[prog.PC].(type) {

	case InstPush:
		return prog.execInstPush(inst)

	case InstAdd:
		return prog.execInstAdd(inst)

	case InstRand:
		return prog.execInstRand(inst)

	case InstMul:
		return prog.execInstMul(inst)

	case InstOpen:
		return prog.execInstOpen(inst)

	default:
		return NotReady(ErrorUnexpectedInst(inst, prog.PC))
	}
}

func (prog *Program) execInstPush(inst InstPush) Return {
	if err := prog.Stack.Push(inst.Value); err != nil {
		return NotReady(ErrorExecution(err, prog.PC))
	}

	prog.PC++
	return Ready()
}

func (prog *Program) execInstAdd(inst InstAdd) Return {
	rhs, err := prog.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, prog.PC))
	}
	lhs, err := prog.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, prog.PC))
	}

	ret := Value(nil)
	switch lhs := lhs.(type) {
	case ValuePublic:
		ret = lhs.Add(rhs)
	case ValuePrivate:
		ret = lhs.Add(rhs)
	default:
		return NotReady(ErrorUnexpectedValue(lhs, nil, prog.PC))
	}
	if err := prog.Stack.Push(ret); err != nil {
		return NotReady(ErrorExecution(err, prog.PC))
	}

	prog.PC++
	return Ready()
}

func (prog *Program) execInstRand(inst InstRand) Return {
	if inst.RhoCh == nil || inst.SigmaCh == nil {
		ρCh := make(chan shamir.Share, 1)
		σCh := make(chan shamir.Share, 1)
		inst.RhoCh = ρCh
		inst.SigmaCh = σCh
		prog.Code[prog.PC] = inst
		return NotReady(GenRn(ρCh, σCh))
	}

	if !inst.RhoReady {
		select {
		case ρ := <-inst.RhoCh:
			inst.RhoReady = true
			inst.Rho = ρ
			prog.Code[prog.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	if !inst.SigmaReady {
		select {
		case σ := <-inst.SigmaCh:
			inst.SigmaReady = true
			inst.Sigma = σ
			prog.Code[prog.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	prog.Push(ValuePrivateRn{
		Rho:   inst.Rho,
		Sigma: inst.Sigma,
	})

	prog.PC++
	return Ready()
}

func (prog *Program) execInstMul(inst InstMul) Return {
	if inst.RetCh == nil {

		rnValue, err := prog.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, prog.PC))
		}
		rn, ok := rnValue.(ValuePrivateRn)
		if !ok {
			return NotReady(ErrorUnexpectedValue(rnValue, ValuePrivateRn{}, prog.PC))
		}

		yValue, err := prog.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, prog.PC))
		}
		y, ok := yValue.(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedValue(yValue, ValuePrivate{}, prog.PC))
		}

		xValue, err := prog.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, prog.PC))
		}
		x, ok := xValue.(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedValue(xValue, ValuePrivate{}, prog.PC))
		}

		retCh := make(chan shamir.Share, 1)
		inst.RetCh = retCh
		prog.Code[prog.PC] = inst
		return NotReady(Multiply(x.Share, y.Share, rn.Rho, rn.Sigma, retCh))
	}

	if !inst.RetReady {
		select {
		case ret := <-inst.RetCh:
			inst.RetReady = true
			inst.Ret = ret
			prog.Code[prog.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	prog.Push(ValuePrivate{
		Share: inst.Ret,
	})

	prog.PC++
	return Ready()
}

func (prog *Program) execInstOpen(inst InstOpen) Return {
	if inst.RetCh == nil {

		value, err := prog.Stack.Pop()
		if err != nil {
			return NotReady(ErrorExecution(err, prog.PC))
		}
		v, ok := value.(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedValue(value, ValuePrivate{}, prog.PC))
		}

		retCh := make(chan *big.Int, 1)
		inst.RetCh = retCh
		prog.Code[prog.PC] = inst
		return NotReady(Open(v.Share, retCh))
	}

	if !inst.RetReady {
		select {
		case ret := <-inst.RetCh:
			inst.RetReady = true
			inst.Ret = ret
			prog.Code[prog.PC] = inst
		default:
			return NotReady(nil)
		}
	}

	prog.Push(ValuePublic{
		Int: inst.Ret,
	})

	prog.PC++
	return Ready()
}
