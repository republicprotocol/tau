package program

import (
	"fmt"

	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

type ID [32]byte

type Addr uint64

type Memory map[Addr]Value

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
		panic("unimplemented")
	}
	if err := prog.Stack.Push(ret); err != nil {
		return NotReady(ErrorExecution(err, prog.PC))
	}

	prog.PC++
	return Ready()
}

func (prog *Program) execInstRand(inst InstRand) Return {
	if inst.RhoCh == nil || inst.SigmaCh == nil {
		inst.RhoCh = make(chan shamir.Share, 1)
		inst.SigmaCh = make(chan shamir.Share, 1)
		prog.Code[prog.PC] = inst
		return NotReady(GenRn(inst.RhoCh, inst.SigmaCh))
	}

	if !inst.RhoReady {
		select {
		case rho := <-inst.RhoCh:
			inst.RhoReady = true
			inst.Rho = rho
		default:
			return NotReady(nil)
		}
	}

	if !inst.SigmaReady {
		select {
		case sigma := <-inst.SigmaCh:
			inst.SigmaReady = true
			inst.Sigma = sigma
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

type Intent interface {
	IsIntent()
}

type IntentToGenRn struct {
	Rho   chan<- shamir.Share
	Sigma chan<- shamir.Share
}

func GenRn(rho chan<- shamir.Share, sigma chan<- shamir.Share) IntentToGenRn {
	return IntentToGenRn{
		Rho:   rho,
		Sigma: sigma,
	}
}

func (intent IntentToGenRn) IsIntent() {
}

type IntentToError struct {
	error
}

func ErrorExecution(err error, pc PC) IntentToError {
	return IntentToError{
		fmt.Errorf("execution error at instruction %v = %v", pc, err),
	}
}

func ErrorUnexpectedInst(inst Inst, pc PC) IntentToError {
	return ErrorExecution(
		fmt.Errorf("unexpected instruction type %T", inst),
		pc,
	)
}

func ErrorCodeOverflow(pc PC) IntentToError {
	return ErrorExecution(
		fmt.Errorf("code overflow"),
		pc,
	)
}

func (intent IntentToError) IsIntent() {
}
