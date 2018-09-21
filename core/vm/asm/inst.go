package asm

import (
	"fmt"
)

// An Inst is executed by a process in the virtual machine. Synchronous
// instructions are executed immediately by a process and do not require rounds
// of communication. Asynchronous instructions require rounds of communication
// before execution can complete.
type Inst interface {

	// Eval an instruction and return a Result. Synchronous instructions will
	// always return a Result that is complete. Asynchronuos instructions can
	// return a pending Result and will need to be evaluated again some time in
	// the future.
	Eval(State) Result
}

type instDebug struct {
	f func()
}

func InstDebug(f func()) Inst {
	return instDebug{f}
}

func (inst instDebug) Eval(State) Result {
	inst.f()
	return Ready()
}

type instCopy struct {
	dst Addr
	src Addr

	dstStep int
	srcStep int
	n       int
}

func InstCopy(dst, src Addr, dstStep, srcStep, n int) Inst {
	return instCopy{dst, src, dstStep, srcStep, n}
}

func (inst instCopy) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		inst.dst.Store(i*inst.dstStep, inst.src.Load(i*inst.srcStep))
	}
	return Ready()
}

type instMove struct {
	dst    Addr
	values []Value
}

func InstMove(dst Addr, values ...Value) Inst {
	return instMove{dst, values}
}

func (inst instMove) Eval(State) Result {
	for i := range inst.values {
		inst.dst.Store(i, inst.values[i])
	}
	return Ready()
}

type instAdd struct {
	dst Addr
	lhs Addr
	rhs Addr

	dstStep int
	lhsStep int
	rhsStep int
	n       int
}

func InstAdd(dst, lhs, rhs Addr, dstStep, lhsStep, rhsStep, n int) Inst {
	return instAdd{dst, lhs, rhs, dstStep, lhsStep, rhsStep, n}
}

func (inst instAdd) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i * inst.lhsStep)
		rhs := inst.rhs.Load(i * inst.rhsStep)
		switch lhs := lhs.(type) {
		case ValuePublic:
			inst.dst.Store(i*inst.dstStep, lhs.Add(rhs))
		case ValuePrivate:
			inst.dst.Store(i*inst.dstStep, lhs.Add(rhs))
		default:
			panic(fmt.Sprintf("unexpected value type %T", lhs))
		}
	}
	return Ready()
}

type instSub struct {
	dst Addr
	lhs Addr
	rhs Addr

	dstStep int
	lhsStep int
	rhsStep int
	n       int
}

func InstSub(dst, lhs, rhs Addr, dstStep, lhsStep, rhsStep, n int) Inst {
	return instSub{dst, lhs, rhs, dstStep, lhsStep, rhsStep, n}
}

func (inst instSub) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i * inst.lhsStep)
		rhs := inst.rhs.Load(i * inst.rhsStep)
		switch lhs := lhs.(type) {
		case ValuePublic:
			inst.dst.Store(i*inst.dstStep, lhs.Sub(rhs))
		case ValuePrivate:
			inst.dst.Store(i*inst.dstStep, lhs.Sub(rhs))
		default:
			panic(fmt.Sprintf("unexpected value type %T", lhs))
		}
	}
	return Ready()
}

type instNeg struct {
	dst Addr
	src Addr

	dstStep int
	srcStep int
	n       int
}

func InstNeg(dst, src Addr, dstStep, srcStep, n int) Inst {
	return instNeg{dst, src, dstStep, srcStep, n}
}

func (inst instNeg) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		src := inst.src.Load(i * inst.srcStep)
		switch src := src.(type) {
		case ValuePublic:
			inst.dst.Store(i*inst.dstStep, src.Neg())
		case ValuePrivate:
			inst.dst.Store(i*inst.dstStep, src.Neg())
		default:
			panic(fmt.Sprintf("unexpected value type %T", src))
		}
	}
	return Ready()
}

type instExp struct {
	dst Addr
	lhs Addr
	rhs Addr

	dstStep int
	lhsStep int
	rhsStep int
	n       int
}

func InstExp(dst, lhs, rhs Addr, dstStep, lhsStep, rhsStep, n int) Inst {
	return instExp{dst, lhs, rhs, dstStep, lhsStep, rhsStep, n}
}

func (inst instExp) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i * inst.lhsStep)
		rhs := inst.rhs.Load(i * inst.rhsStep)
		switch lhs := lhs.(type) {
		case ValuePublic:
			switch rhs := rhs.(type) {
			case ValuePublic:
				inst.dst.Store(i*inst.dstStep, lhs.Exp(rhs))
			default:
				panic(fmt.Sprintf("unexpected value type %T", rhs))
			}
		default:
			panic(fmt.Sprintf("unexpected value type %T", lhs))
		}
	}
	return Ready()
}

type instInv struct {
	dst Addr
	src Addr

	dstStep int
	srcStep int
	n       int
}

func InstInv(dst, src Addr, dstStep, srcStep, n int) Inst {
	return instInv{dst, src, dstStep, srcStep, n}
}

func (inst instInv) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		src := inst.src.Load(i * inst.srcStep)
		switch src := src.(type) {
		case ValuePublic:
			inst.dst.Store(i*inst.dstStep, src.Inv())
		default:
			panic(fmt.Sprintf("unexpected value type %T", src))
		}
	}
	return Ready()
}

type instMod struct {
	dst Addr
	lhs Addr
	rhs Addr

	dstStep int
	lhsStep int
	rhsStep int
	n       int
}

func InstMod(dst, lhs, rhs Addr, dstStep, lhsStep, rhsStep, n int) Inst {
	return instMod{dst, lhs, rhs, dstStep, lhsStep, rhsStep, n}
}

func (inst instMod) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i * inst.lhsStep)
		rhs := inst.rhs.Load(i * inst.rhsStep)
		switch lhs := lhs.(type) {
		case ValuePublic:
			switch rhs := rhs.(type) {
			case ValuePublic:
				inst.dst.Store(i*inst.dstStep, lhs.Mod(rhs))
			default:
				panic(fmt.Sprintf("unexpected value type %T", rhs))
			}
		default:
			panic(fmt.Sprintf("unexpected value type %T", lhs))
		}
	}
	return Ready()
}

type instGenerateRn struct {
	dst Addr

	dstStep int
	n       int
}

func InstGenerateRn(dst Addr, dstStep, n int) Inst {
	return instGenerateRn{
		dst: dst,

		dstStep: dstStep,
		n:       n,
	}
}

func (inst instGenerateRn) Eval(state State) Result {
	if state == nil {
		return NotReady(NewInstGenerateRnState(inst.n))
	}
	switch state := state.(type) {
	case *InstGenerateRnState:
		for i := 0; i < inst.n; i++ {
			inst.dst.Store(i*inst.dstStep, NewValuePrivate(state.Sigmas[i].Share()))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

type instGenerateRnZero struct {
	dst Addr

	dstStep int
	n       int
}

func InstGenerateRnZero(dst Addr, dstStep, n int) Inst {
	return instGenerateRnZero{
		dst: dst,

		dstStep: dstStep,
		n:       n,
	}
}

func (inst instGenerateRnZero) Eval(state State) Result {
	if state == nil {
		return NotReady(NewInstGenerateRnZeroState(inst.n))
	}
	switch state := state.(type) {
	case *InstGenerateRnZeroState:
		for i := 0; i < inst.n; i++ {
			inst.dst.Store(i*inst.dstStep, NewValuePrivate(state.Sigmas[i].Share()))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

type instGenerateRnTuple struct {
	ρDst Addr
	σDst Addr

	ρDstStep int
	σDstStep int
	n        int
}

func InstGenerateRnTuple(ρDst, σDst Addr, ρDstStep, σDstStep, n int) Inst {
	return instGenerateRnTuple{
		ρDst: ρDst,
		σDst: σDst,

		ρDstStep: ρDstStep,
		σDstStep: σDstStep,
		n:        n,
	}
}

func (inst instGenerateRnTuple) Eval(state State) Result {
	if state == nil {
		return NotReady(NewInstGenerateRnTupleState(inst.n))
	}
	switch state := state.(type) {
	case *InstGenerateRnTupleState:
		for i := 0; i < inst.n; i++ {
			inst.ρDst.Store(i*inst.ρDstStep, NewValuePrivate(state.Rhos[i].Share()))
			inst.σDst.Store(i*inst.σDstStep, NewValuePrivate(state.Sigmas[i].Share()))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

type instMul struct {
	dst  Addr
	lhs  Addr
	rhs  Addr
	ρSrc Addr
	σSrc Addr

	dstStep  int
	lhsStep  int
	rhsStep  int
	ρSrcStep int
	σSrcStep int
	n        int
}

func InstMul(dst, lhs, rhs, ρSrc, σSrc Addr, dstStep, lhsStep, rhsStep, ρSrcStep, σSrcStep, n int) Inst {
	return instMul{
		dst:  dst,
		lhs:  lhs,
		rhs:  rhs,
		ρSrc: ρSrc,
		σSrc: σSrc,

		dstStep:  dstStep,
		lhsStep:  lhsStep,
		rhsStep:  rhsStep,
		ρSrcStep: ρSrcStep,
		σSrcStep: σSrcStep,
		n:        n,
	}
}

func (inst instMul) Eval(state State) Result {
	switch lhs := inst.lhs.Load(0).(type) {
	case ValuePublic:
		switch rhs := inst.rhs.Load(0).(type) {
		case ValuePublic:
			return inst.evalPubPub(state)
		case ValuePrivate:
			return inst.evalPubPriv(state)
		default:
			panic(fmt.Sprintf("unexpected type %v", rhs))
		}
	case ValuePrivate:
		switch rhs := inst.rhs.Load(0).(type) {
		case ValuePublic:
			return inst.evalPrivPub(state)
		case ValuePrivate:
			return inst.evalPrivPriv(state)
		default:
			panic(fmt.Sprintf("unexpected type %v", rhs))
		}
	default:
		panic(fmt.Sprintf("unexpected type %v", lhs))
	}
}

func (inst instMul) evalPubPub(state State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i * inst.lhsStep)
		rhs := inst.rhs.Load(i * inst.rhsStep)
		inst.dst.Store(i*inst.dstStep, rhs.(ValuePublic).Mul(lhs.(ValuePublic)))
	}
	return Ready()
}

func (inst instMul) evalPubPriv(state State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i * inst.lhsStep)
		rhs := inst.rhs.Load(i * inst.rhsStep)
		inst.dst.Store(i*inst.dstStep, rhs.(ValuePrivate).Mul(lhs.(ValuePublic)))
	}
	return Ready()
}

func (inst instMul) evalPrivPub(state State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i * inst.lhsStep)
		rhs := inst.rhs.Load(i * inst.rhsStep)
		inst.dst.Store(i*inst.dstStep, lhs.(ValuePrivate).Mul(rhs.(ValuePublic)))
	}
	return Ready()
}

func (inst instMul) evalPrivPriv(state State) Result {
	if state == nil {
		mulState := NewInstMulState(inst.n)
		for i := 0; i < inst.n; i++ {
			x, ok := inst.lhs.Load(i * inst.lhsStep).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.lhs.Load(i*inst.lhsStep)))
			}
			y, ok := inst.rhs.Load(i * inst.rhsStep).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.rhs.Load(i*inst.rhsStep)))
			}
			ρ, ok := inst.ρSrc.Load(i * inst.ρSrcStep).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.ρSrc.Load(i*inst.ρSrcStep)))
			}
			σ, ok := inst.σSrc.Load(i * inst.σSrcStep).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.σSrc.Load(i*inst.σSrcStep)))
			}
			mulState.Xs[i] = x.Share
			mulState.Ys[i] = y.Share
			mulState.Rhos[i] = ρ.Share
			mulState.Sigmas[i] = σ.Share
		}
		return NotReady(mulState)
	}
	switch state := state.(type) {
	case *InstMulState:
		for i := 0; i < inst.n; i++ {
			inst.dst.Store(i*inst.dstStep, NewValuePrivate(state.Results[i]))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

type instOpen struct {
	dst Addr
	src Addr

	dstStep int
	srcStep int
	n       int
}

func InstOpen(dst, src Addr, dstStep, srcStep, n int) Inst {
	return instOpen{
		dst: dst,
		src: src,

		dstStep: dstStep,
		srcStep: srcStep,
		n:       n,
	}
}

func (inst instOpen) Eval(state State) Result {
	if state == nil {
		openState := NewInstOpenState(inst.n)
		for i := 0; i < inst.n; i++ {
			value, ok := inst.src.Load(i * inst.srcStep).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.src.Load(i*inst.srcStep)))
			}
			openState.Shares[i] = value.Share
		}
		return NotReady(openState)
	}
	switch state := state.(type) {
	case *InstOpenState:
		for i := 0; i < inst.n; i++ {
			inst.dst.Store(i*inst.dstStep, NewValuePublic(state.Results[i]))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

type instExit struct {
	src Addr

	srcStep int
	n       int
}

func InstExit(src Addr, srcStep, n int) Inst {
	return instExit{src, srcStep, n}
}

func (inst instExit) Eval(State) Result {
	state := NewInstExitState(inst.n)
	for i := 0; i < inst.n; i++ {
		state.Values[i] = inst.src.Load(i * inst.srcStep)
	}
	return Exit(state)
}
