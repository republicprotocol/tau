package process

import (
	"encoding/base64"
	"log"

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

func (id ID) String() string {
	idBase64 := base64.StdEncoding.EncodeToString(id[:])
	idRunes := []rune(idBase64)
	return string(idRunes[16:])
}

type Process struct {
	ID
	Memory
	Code
	PC
}

func New(id ID, mem Memory, code Code) Process {
	expandMacros(&code)
	return Process{
		ID:     id,
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
		case instCopy:
			ret = proc.execInstCopy(inst)
		case instMove:
			ret = proc.execInstMove(inst)
		case instAdd:
			ret = proc.execInstAdd(inst)
		case instNeg:
			ret = proc.execInstNeg(inst)
		case instSub:
			ret = proc.execInstSub(inst)
		case instExp:
			ret = proc.execInstExp(inst)
		case instGenerateRn:
			ret = proc.execInstGenerateRn(inst)
		case instGenerateRnZero:
			ret = proc.execInstGenerateRnZero(inst)
		case instGenerateRnTuple:
			ret = proc.execInstGenerateRnTuple(inst)
		case instMul:
			ret = proc.execInstMul(inst)
		case instMulPub:
			ret = proc.execInstMulPub(inst)
		case instOpen:
			ret = proc.execInstOpen(inst)
		case instExit:
			ret = proc.execInstExit(inst)
		case instDebug:
			ret = proc.execInstDebug(inst)

		default:
			ret = NotReady(ErrorUnexpectedInst(inst, proc.PC))
		}
	}

	return ret
}

func (proc *Process) execInstCopy(inst instCopy) Return {
	*inst.dst = *inst.src

	proc.PC++
	return Ready()
}

func (proc *Process) execInstMove(inst instMove) Return {
	*inst.dst = inst.val

	proc.PC++
	return Ready()
}

func (proc *Process) execInstAdd(inst instAdd) Return {
	lhs := *inst.lhs
	rhs := *inst.rhs

	ret := Value(nil)
	switch lhs := lhs.(type) {
	case ValuePublic:
		ret = lhs.Add(rhs.(Value))
	case ValuePrivate:
		ret = lhs.Add(rhs.(Value))
	default:
		return NotReady(ErrorUnexpectedTypeConversion(lhs, nil, proc.PC))
	}
	*inst.dst = ret

	proc.PC++
	return Ready()
}

func (proc *Process) execInstNeg(inst instNeg) Return {
	lhs := *inst.lhs

	ret := Value(nil)
	switch lhs := lhs.(type) {
	case ValuePublic:
		ret = lhs.Neg()
	case ValuePrivate:
		ret = lhs.Neg()
	default:
		return NotReady(ErrorUnexpectedTypeConversion(lhs, nil, proc.PC))
	}
	*inst.dst = ret

	proc.PC++
	return Ready()
}

func (proc *Process) execInstSub(inst instSub) Return {
	lhs := *inst.lhs
	rhs := *inst.rhs

	ret := Value(nil)
	switch lhs := lhs.(type) {
	case ValuePublic:
		ret = lhs.Sub(rhs.(Value))
	case ValuePrivate:
		ret = lhs.Sub(rhs.(Value))
	default:
		return NotReady(ErrorUnexpectedTypeConversion(lhs, nil, proc.PC))
	}
	*inst.dst = ret

	proc.PC++
	return Ready()
}

func (proc *Process) execInstExp(inst instExp) Return {
	exponent, err := proc.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	base, err := proc.Stack.Pop()
	if err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	ret := Value(nil)
	switch base := base.(type) {
	case ValuePublic:
		switch exponent := exponent.(type) {
		case ValuePublic:
			ret = base.Exp(exponent)
		default:
			return NotReady(ErrorUnexpectedTypeConversion(exponent, nil, proc.PC))
		}
	default:
		return NotReady(ErrorUnexpectedTypeConversion(base, nil, proc.PC))
	}
	if err := proc.Stack.Push(ret); err != nil {
		return NotReady(ErrorExecution(err, proc.PC))
	}

	proc.PC++
	return Ready()
}

func (proc *Process) execInstGenerateRn(inst instGenerateRn) Return {
	if inst.σCh == nil {
		σCh := make(chan shamir.Share, 1)
		inst.σCh = σCh
		proc.Code[proc.PC] = inst
		return NotReady(GenerateRn(σCh))
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

	*inst.dst = NewValuePrivate(inst.σ)

	proc.PC++
	return Ready()
}

func (proc *Process) execInstGenerateRnZero(inst instGenerateRnZero) Return {
	if inst.σCh == nil {
		σCh := make(chan shamir.Share, 1)
		inst.σCh = σCh
		proc.Code[proc.PC] = inst
		return NotReady(GenerateRnZero(σCh))
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

	*inst.dst = NewValuePrivate(inst.σ)

	proc.PC++
	return Ready()
}

func (proc *Process) execInstGenerateRnTuple(inst instGenerateRnTuple) Return {
	if inst.ρCh == nil || inst.σCh == nil {
		ρCh := make(chan shamir.Share, 1)
		σCh := make(chan shamir.Share, 1)
		inst.ρCh = ρCh
		inst.σCh = σCh
		proc.Code[proc.PC] = inst
		return NotReady(GenerateRnTuple(ρCh, σCh))
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

	*inst.ρDst = NewValuePrivate(inst.ρ)
	*inst.σDst = NewValuePrivate(inst.σ)

	proc.PC++
	return Ready()
}

func (proc *Process) execInstMul(inst instMul) Return {
	if inst.retCh == nil {

		x, ok := (*inst.lhs).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*inst.lhs, ValuePrivate{}, proc.PC))
		}
		y, ok := (*inst.rhs).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*inst.rhs, ValuePrivate{}, proc.PC))
		}

		ρ, ok := (*inst.ρ).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*inst.ρ, ValuePrivate{}, proc.PC))
		}
		σ, ok := (*inst.σ).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*inst.σ, ValuePrivate{}, proc.PC))
		}

		retCh := make(chan shamir.Share, 1)
		inst.retCh = retCh
		proc.Code[proc.PC] = inst
		return NotReady(Multiply(x.Share, y.Share, ρ.Share, σ.Share, retCh))
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

	*inst.dst = NewValuePrivate(inst.ret)

	proc.PC++
	return Ready()
}

func (proc *Process) execInstMulPub(inst instMulPub) Return {
	if inst.retCh == nil {

		x, ok := (*inst.lhs).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*inst.lhs, ValuePrivate{}, proc.PC))
		}
		y, ok := (*inst.rhs).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*inst.rhs, ValuePrivate{}, proc.PC))
		}

		retCh := make(chan algebra.FpElement, 1)
		inst.retCh = retCh
		proc.Code[proc.PC] = inst
		return NotReady(Open(x.Share.Mul(y.Share), retCh))
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

	*inst.dst = NewValuePublic(inst.ret)

	proc.PC++
	return Ready()
}

func (proc *Process) execInstOpen(inst instOpen) Return {
	if inst.retCh == nil {

		v, ok := (*inst.src).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*inst.src, ValuePrivate{}, proc.PC))
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

	*inst.dst = NewValuePublic(inst.ret)

	proc.PC++
	return Ready()
}

func (proc *Process) execInstExit(inst instExit) Return {

	values := make([]Value, len(inst.src))
	for i := range values {
		values[i] = *(inst.src[i])
	}

	proc.PC++
	ret := Terminated()
	ret.intent = Exit(values)
	return ret
}

func (proc *Process) execInstDebug(inst instDebug) Return {
	inst.d()
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
