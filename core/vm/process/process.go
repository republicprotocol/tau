package process

import (
	"encoding/base64"
	"fmt"
	"unsafe"

	"github.com/republicprotocol/oro-go/core/vm/asm"
	"github.com/republicprotocol/oro-go/core/vss/algebra"

	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type PC uint64

type Code []Inst

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

	PC
	Insts  []asm.Inst
	States []asm.State
}

func New(id ID, insts []asm.Inst) Process {
	expandMacros(&code)
	return Process{
		ID: id,

		PC:     0,
		Inst:   insts,
		States: make([]asm.State, len(insts)),
	}
}

func (proc *Process) Exec() Return {
	for {
		if proc.PC == PC(len(proc.Insts)) {
			return NotReady(ErrorCodeOverflow(proc.PC, nil))
		}
		if result := proc.Insts[proc.PC].Eval(proc.States[proc.PC]); !result.Ok {
			proc.States[proc.PC] = result.State
			return NotReady(IntentToTransitionState(result.State))
		}
		proc.PC++
	}
}

func (proc *Process) execInstMul(inst instMul) Return {
	switch (*inst.lhs).(type) {
	case ValuePublic:
		switch (*inst.rhs).(type) {
		case ValuePublic:
			return proc.execInstMulPubPub(inst)
		case ValuePrivate:
			return proc.execInstMulPubPriv(inst)
		default:
			panic(fmt.Sprintf("unexpected value type %T", *inst.lhs))
		}
	case ValuePrivate:
		switch (*inst.rhs).(type) {
		case ValuePublic:
			return proc.execInstMulPrivPub(inst)
		case ValuePrivate:
			return proc.execInstMulPrivPriv(inst)
		default:
			panic(fmt.Sprintf("unexpected value type %T", *inst.lhs))
		}
	default:
		panic(fmt.Sprintf("unexpected value type %T", *inst.lhs))
	}
}

func (proc *Process) execInstMulPubPub(inst instMul) Return {
	size := unsafe.Sizeof(Value(nil))

	lhs := unsafe.Pointer(inst.lhs)
	rhs := unsafe.Pointer(inst.rhs)
	dst := unsafe.Pointer(inst.dst)

	for b := 0; b < inst.batch; b++ {
		xPtr := (*Value)(unsafe.Pointer(uintptr(lhs) + uintptr(b)*size))
		yPtr := (*Value)(unsafe.Pointer(uintptr(rhs) + uintptr(b)*size))

		x, ok := (*xPtr).(ValuePublic)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*xPtr, ValuePublic{}, proc.PC, inst))
		}
		y, ok := (*yPtr).(ValuePublic)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*yPtr, ValuePublic{}, proc.PC, inst))
		}

		*(*Value)(unsafe.Pointer(uintptr(dst) + uintptr(b)*size)) = x.Mul(y)
	}

	return Ready()
}

func (proc *Process) execInstMulPubPriv(inst instMul) Return {
	size := unsafe.Sizeof(Value(nil))

	lhs := unsafe.Pointer(inst.lhs)
	rhs := unsafe.Pointer(inst.rhs)
	dst := unsafe.Pointer(inst.dst)

	for b := 0; b < inst.batch; b++ {
		xPtr := (*Value)(unsafe.Pointer(uintptr(lhs) + uintptr(b)*size))
		yPtr := (*Value)(unsafe.Pointer(uintptr(rhs) + uintptr(b)*size))

		x, ok := (*xPtr).(ValuePublic)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*xPtr, ValuePublic{}, proc.PC, inst))
		}
		y, ok := (*yPtr).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*yPtr, ValuePrivate{}, proc.PC, inst))
		}

		*(*Value)(unsafe.Pointer(uintptr(dst) + uintptr(b)*size)) = y.Mul(x)
	}

	return Ready()
}

func (proc *Process) execInstMulPrivPub(inst instMul) Return {
	size := unsafe.Sizeof(Value(nil))

	lhs := unsafe.Pointer(inst.lhs)
	rhs := unsafe.Pointer(inst.rhs)
	dst := unsafe.Pointer(inst.dst)

	for b := 0; b < inst.batch; b++ {
		xPtr := (*Value)(unsafe.Pointer(uintptr(lhs) + uintptr(b)*size))
		yPtr := (*Value)(unsafe.Pointer(uintptr(rhs) + uintptr(b)*size))

		x, ok := (*xPtr).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*xPtr, ValuePublic{}, proc.PC, inst))
		}
		y, ok := (*yPtr).(ValuePublic)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*yPtr, ValuePrivate{}, proc.PC, inst))
		}

		*(*Value)(unsafe.Pointer(uintptr(dst) + uintptr(b)*size)) = x.Mul(y)
	}

	return Ready()
}

func (proc *Process) execInstMulPrivPriv(inst instMul) Return {
	size := unsafe.Sizeof(Value(nil))

	if inst.retCh == nil {

		xs := make([]shamir.Share, inst.batch)
		ys := make([]shamir.Share, inst.batch)
		ρs := make([]shamir.Share, inst.batch)
		σs := make([]shamir.Share, inst.batch)

		lhs := unsafe.Pointer(inst.lhs)
		rhs := unsafe.Pointer(inst.rhs)
		ρσs := unsafe.Pointer(inst.ρσs)

		for b := 0; b < inst.batch; b++ {
			xPtr := (*Value)(unsafe.Pointer(uintptr(lhs) + uintptr(b)*size))
			yPtr := (*Value)(unsafe.Pointer(uintptr(rhs) + uintptr(b)*size))
			ρPtr := (*Value)(unsafe.Pointer(uintptr(ρσs) + uintptr(2*b)*size))
			σPtr := (*Value)(unsafe.Pointer(uintptr(ρσs) + uintptr(2*b+1)*size))

			x, ok := (*xPtr).(ValuePrivate)
			if !ok {
				return NotReady(ErrorUnexpectedTypeConversion(*xPtr, ValuePrivate{}, proc.PC, inst))
			}
			y, ok := (*yPtr).(ValuePrivate)
			if !ok {
				return NotReady(ErrorUnexpectedTypeConversion(*yPtr, ValuePrivate{}, proc.PC, inst))
			}

			ρ, ok := (*ρPtr).(ValuePrivate)
			if !ok {
				return NotReady(ErrorUnexpectedTypeConversion(*ρPtr, ValuePrivate{}, proc.PC, inst))
			}
			σ, ok := (*σPtr).(ValuePrivate)
			if !ok {
				return NotReady(ErrorUnexpectedTypeConversion(*σPtr, ValuePrivate{}, proc.PC, inst))
			}

			xs[b] = x.Share
			ys[b] = y.Share
			ρs[b] = ρ.Share
			σs[b] = σ.Share
		}

		retCh := make(chan []shamir.Share, 1)
		inst.retCh = retCh
		proc.Code[proc.PC] = inst
		return NotReady(Multiply(proc.iid(), xs, ys, ρs, σs, retCh))
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

	dst := unsafe.Pointer(inst.dst)
	for b := 0; b < inst.batch; b++ {
		*(*Value)(unsafe.Pointer(uintptr(dst) + uintptr(b)*size)) = NewValuePrivate(inst.ret[b])
	}

	return Ready()
}

func (proc *Process) execInstMulOpen(inst instMulOpen) Return {
	if inst.retCh == nil {

		x, ok := (*inst.lhs).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*inst.lhs, ValuePrivate{}, proc.PC, inst))
		}
		y, ok := (*inst.rhs).(ValuePrivate)
		if !ok {
			return NotReady(ErrorUnexpectedTypeConversion(*inst.rhs, ValuePrivate{}, proc.PC, inst))
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
			return NotReady(ErrorUnexpectedTypeConversion(*inst.src, ValuePrivate{}, proc.PC, inst))
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
