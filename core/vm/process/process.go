package process

import (
	"encoding/base64"

	"github.com/republicprotocol/oro-go/core/vss/algebra"

	"github.com/republicprotocol/oro-go/core/vss/shamir"
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
	for {
		if proc.PC == PC(len(proc.Code)) {
			return NotReady(ErrorCodeOverflow(proc.PC))
		}
		ret := proc.execInst(proc.Code[proc.PC])
		if !ret.IsReady() {
			return ret
		}
		proc.PC++
	}
}

func (proc *Process) execInst(inst Inst) Return {
	switch inst := inst.(type) {
	case instCopy:
		return proc.execInstCopy(inst)
	case instMove:
		return proc.execInstMove(inst)
	case instAdd:
		return proc.execInstAdd(inst)
	case instNeg:
		return proc.execInstNeg(inst)
	case instSub:
		return proc.execInstSub(inst)
	case instGenerateRn:
		return proc.execInstGenerateRn(inst)
	case instGenerateRnZero:
		return proc.execInstGenerateRnZero(inst)
	case instGenerateRnTuple:
		return proc.execInstGenerateRnTuple(inst)
	case instMul:
		return proc.execInstMul(inst)
	case instMulPub:
		return proc.execInstMulPub(inst)
	case instOpen:
		return proc.execInstOpen(inst)
	case instAsync:
		return proc.execInstAsync(inst)
	case instAwait:
		return proc.execInstAwait(inst)
	case instExit:
		return proc.execInstExit(inst)
	case instDebug:
		return proc.execInstDebug(inst)
	default:
		return NotReady(ErrorUnexpectedInst(inst, proc.PC))
	}
}

func (proc *Process) execInstCopy(inst instCopy) Return {
	*inst.dst = *inst.src

	return Ready()
}

func (proc *Process) execInstMove(inst instMove) Return {
	*inst.dst = inst.val

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

	return Ready()
}

func (proc *Process) execInstGenerateRn(inst instGenerateRn) Return {
	if inst.σCh == nil {
		σCh := make(chan shamir.Share, 1)
		inst.σCh = σCh
		proc.Code[proc.PC] = inst
		return NotReady(GenerateRn(proc.iid(), σCh))
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

	return Ready()
}

func (proc *Process) execInstGenerateRnZero(inst instGenerateRnZero) Return {
	if inst.σCh == nil {
		σCh := make(chan shamir.Share, 1)
		inst.σCh = σCh
		proc.Code[proc.PC] = inst
		return NotReady(GenerateRnZero(proc.iid(), σCh))
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

	return Ready()
}

func (proc *Process) execInstGenerateRnTuple(inst instGenerateRnTuple) Return {
	if inst.ρCh == nil || inst.σCh == nil {
		ρCh := make(chan shamir.Share, 1)
		σCh := make(chan shamir.Share, 1)
		inst.ρCh = ρCh
		inst.σCh = σCh
		proc.Code[proc.PC] = inst
		return NotReady(GenerateRnTuple(proc.iid(), ρCh, σCh))
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
		return NotReady(Multiply(proc.iid(), x.Share, y.Share, ρ.Share, σ.Share, retCh))
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

	*inst.dst = NewValuePrivate(inst.ret)

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
		return NotReady(Open(proc.iid(), x.Share.Mul(y.Share), retCh))
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
		return NotReady(Open(proc.iid(), v.Share, retCh))
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

	return Ready()
}

func (proc *Process) execInstAsync(inst instAsync) Return {

	// Store the current PC and move to the first instruction in the async block
	pc := proc.PC
	proc.PC++

	asyncRet := Ready()
	asyncRetIntents := []Intent{}

	// Execute instructions inside the async block until the required number of
	// await instructions have been seen
	for awaits := 1; awaits > 0; proc.PC++ {

		if proc.PC == PC(len(proc.Code)) {
			return NotReady(ErrorCodeOverflow(proc.PC))
		}

		// Execute an instruction and store all intent
		inst := proc.Code[proc.PC]

		// Increment the number of await instructions that need to be seen
		// before the async block can end
		if _, ok := inst.(instAsync); ok {
			awaits++
			continue
		}
		// Decrement the number of await instructions that need to be seen
		// before the async block can end
		if _, ok := inst.(instAwait); ok {
			awaits--
			continue
		}

		ret := proc.execInst(inst)
		if !ret.IsReady() {
			asyncRet = NotReady(nil)
			asyncRetIntents = append(asyncRetIntents, ret.Intent())
		}
	}

	if !asyncRet.IsReady() {
		// At least one instruction in the async block is not ready, so we need
		// to reset the PC to the beginning of the async block and try again
		// later
		proc.PC = pc
		asyncRet = NotReady(Await(proc.iid(), asyncRetIntents))
	} else {
		// Decrement the PC to make sure it points to the final await
		// instruction for the async block
		proc.PC--
	}

	return asyncRet
}

func (proc *Process) execInstAwait(inst instAwait) Return {
	// Passively ignore the direct execution of await instructions because they
	// are meaningless outside the context of an async block
	return Ready()
}

func (proc *Process) execInstExit(inst instExit) Return {
	values := make([]Value, len(inst.src))
	for i := range values {
		values[i] = *(inst.src[i])
	}
	return NotReady(Exit(proc.iid(), values))
}

func (proc *Process) execInstDebug(inst instDebug) Return {
	inst.d()
	return Ready()
}

func (proc *Process) iid() IntentID {
	id := IntentID{}
	copy(id[:32], proc.ID[:32])
	id[32] = byte(proc.PC)
	id[33] = byte(proc.PC >> 8)
	id[34] = byte(proc.PC >> 16)
	id[35] = byte(proc.PC >> 24)
	id[36] = byte(proc.PC >> 32)
	id[37] = byte(proc.PC >> 40)
	id[38] = byte(proc.PC >> 48)
	id[39] = byte(proc.PC >> 56)
	return id
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
