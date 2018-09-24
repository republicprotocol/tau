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

	// Expand an Inst into all of its inner instructions. A boolean is returned
	// that indicates whether or not an expansion happened.
	Expand() ([]Inst, bool)
}

type instMacro struct {
	insts []Inst
}

func InstMacro(insts []Inst) Inst {
	return instMacro{insts}
}

func (inst instMacro) Eval(State) Result {
	panic("evaluation of macro")
}

func (inst instMacro) Expand() ([]Inst, bool) {
	return inst.insts, true
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

func (inst instDebug) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instCopy struct {
	dst Memory
	src Memory

	n int
}

func InstCopy(dst, src Memory, n int) Inst {
	return instCopy{dst, src, n}
}

func (inst instCopy) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		inst.dst.Store(i, inst.src.Load(i))
	}
	return Ready()
}

func (inst instCopy) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instMove struct {
	dst Memory

	values []Value
}

func InstMove(dst Memory, values ...Value) Inst {
	return instMove{dst, values}
}

func (inst instMove) Eval(State) Result {
	for i := range inst.values {
		inst.dst.Store(i, inst.values[i])
	}
	return Ready()
}

func (inst instMove) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instAdd struct {
	dst Memory
	lhs Memory
	rhs Memory

	n int
}

func InstAdd(dst, lhs, rhs Memory, n int) Inst {
	return instAdd{dst, lhs, rhs, n}
}

func (inst instAdd) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i)
		rhs := inst.rhs.Load(i)
		switch lhs := lhs.(type) {
		case ValuePublic:
			inst.dst.Store(i, lhs.Add(rhs))
		case ValuePrivate:
			inst.dst.Store(i, lhs.Add(rhs))
		default:
			panic(fmt.Sprintf("unexpected value type %T", lhs))
		}
	}
	return Ready()
}

func (inst instAdd) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instSub struct {
	dst Memory
	lhs Memory
	rhs Memory

	n int
}

func InstSub(dst, lhs, rhs Memory, n int) Inst {
	return instSub{dst, lhs, rhs, n}
}

func (inst instSub) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i)
		rhs := inst.rhs.Load(i)
		switch lhs := lhs.(type) {
		case ValuePublic:
			inst.dst.Store(i, lhs.Sub(rhs))
		case ValuePrivate:
			inst.dst.Store(i, lhs.Sub(rhs))
		default:
			panic(fmt.Sprintf("unexpected value type %T", lhs))
		}
	}
	return Ready()
}

func (inst instSub) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instNeg struct {
	dst Memory
	src Memory

	n int
}

func InstNeg(dst, src Memory, n int) Inst {
	return instNeg{dst, src, n}
}

func (inst instNeg) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		src := inst.src.Load(i)
		switch src := src.(type) {
		case ValuePublic:
			inst.dst.Store(i, src.Neg())
		case ValuePrivate:
			inst.dst.Store(i, src.Neg())
		default:
			panic(fmt.Sprintf("unexpected value type %T", src))
		}
	}
	return Ready()
}

func (inst instNeg) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instExp struct {
	dst Memory
	lhs Memory
	rhs Memory

	n int
}

func InstExp(dst, lhs, rhs Memory, n int) Inst {
	return instExp{dst, lhs, rhs, n}
}

func (inst instExp) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i)
		rhs := inst.rhs.Load(i)
		switch lhs := lhs.(type) {
		case ValuePublic:
			switch rhs := rhs.(type) {
			case ValuePublic:
				inst.dst.Store(i, lhs.Exp(rhs))
			default:
				panic(fmt.Sprintf("unexpected value type %T", rhs))
			}
		default:
			panic(fmt.Sprintf("unexpected value type %T", lhs))
		}
	}
	return Ready()
}

func (inst instExp) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instInv struct {
	dst Memory
	src Memory

	n int
}

func InstInv(dst, src Memory, n int) Inst {
	return instInv{dst, src, n}
}

func (inst instInv) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		src := inst.src.Load(i)
		switch src := src.(type) {
		case ValuePublic:
			inst.dst.Store(i, src.Inv())
		default:
			panic(fmt.Sprintf("unexpected value type %T", src))
		}
	}
	return Ready()
}

func (inst instInv) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instMod struct {
	dst Memory
	lhs Memory
	rhs Memory

	n int
}

func InstMod(dst, lhs, rhs Memory, n int) Inst {
	return instMod{dst, lhs, rhs, n}
}

func (inst instMod) Eval(State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i)
		rhs := inst.rhs.Load(i)
		switch lhs := lhs.(type) {
		case ValuePublic:
			switch rhs := rhs.(type) {
			case ValuePublic:
				inst.dst.Store(i, lhs.Mod(rhs))
			default:
				panic(fmt.Sprintf("unexpected value type %T", rhs))
			}
		default:
			panic(fmt.Sprintf("unexpected value type %T", lhs))
		}
	}
	return Ready()
}

func (inst instMod) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instGenerateRn struct {
	dst Memory

	n int
}

func InstGenerateRn(dst Memory, n int) Inst {
	return instGenerateRn{dst, n}
}

func (inst instGenerateRn) Eval(state State) Result {
	if state == nil {
		return NotReady(NewInstGenerateRnState(inst.n))
	}
	switch state := state.(type) {
	case *InstGenerateRnState:
		for i := 0; i < inst.n; i++ {
			inst.dst.Store(i, NewValuePrivate(state.Sigmas[i].Share()))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

func (inst instGenerateRn) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instGenerateRnZero struct {
	dst Memory

	n int
}

func InstGenerateRnZero(dst Memory, n int) Inst {
	return instGenerateRnZero{dst, n}
}

func (inst instGenerateRnZero) Eval(state State) Result {
	if state == nil {
		return NotReady(NewInstGenerateRnZeroState(inst.n))
	}
	switch state := state.(type) {
	case *InstGenerateRnZeroState:
		for i := 0; i < inst.n; i++ {
			inst.dst.Store(i, NewValuePrivate(state.Sigmas[i].Share()))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

func (inst instGenerateRnZero) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instGenerateRnTuple struct {
	ρDst Memory
	σDst Memory

	n int
}

func InstGenerateRnTuple(ρDst, σDst Memory, n int) Inst {
	return instGenerateRnTuple{ρDst, σDst, n}
}

func (inst instGenerateRnTuple) Eval(state State) Result {
	if state == nil {
		return NotReady(NewInstGenerateRnTupleState(inst.n))
	}
	switch state := state.(type) {
	case *InstGenerateRnTupleState:
		for i := 0; i < inst.n; i++ {
			inst.ρDst.Store(i, NewValuePrivate(state.Rhos[i].Share()))
			inst.σDst.Store(i, NewValuePrivate(state.Sigmas[i].Share()))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

func (inst instGenerateRnTuple) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instMul struct {
	dst Memory
	lhs Memory
	rhs Memory
	ρs  Memory
	σs  Memory

	n int
}

func InstMul(dst, lhs, rhs, ρs, σs Memory, n int) Inst {
	return instMul{
		dst: dst,
		lhs: lhs,
		rhs: rhs,
		ρs:  ρs,
		σs:  σs,

		n: n,
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

func (inst instMul) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

func (inst instMul) evalPubPub(state State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i)
		rhs := inst.rhs.Load(i)
		inst.dst.Store(i, rhs.(ValuePublic).Mul(lhs.(ValuePublic)))
	}
	return Ready()
}

func (inst instMul) evalPubPriv(state State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i)
		rhs := inst.rhs.Load(i)
		inst.dst.Store(i, rhs.(ValuePrivate).Mul(lhs.(ValuePublic)))
	}
	return Ready()
}

func (inst instMul) evalPrivPub(state State) Result {
	for i := 0; i < inst.n; i++ {
		lhs := inst.lhs.Load(i)
		rhs := inst.rhs.Load(i)
		inst.dst.Store(i, lhs.(ValuePrivate).Mul(rhs.(ValuePublic)))
	}
	return Ready()
}

func (inst instMul) evalPrivPriv(state State) Result {
	if state == nil {
		mulState := NewInstMulState(inst.n)
		for i := 0; i < inst.n; i++ {
			x, ok := inst.lhs.Load(i).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.lhs.Load(i)))
			}
			y, ok := inst.rhs.Load(i).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.rhs.Load(i)))
			}
			ρ, ok := inst.ρs.Load(i).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.ρs.Load(i)))
			}
			σ, ok := inst.σs.Load(i).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.σs.Load(i)))
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
			inst.dst.Store(i, NewValuePrivate(state.Results[i]))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

type instOpen struct {
	dst Memory
	src Memory

	n int
}

func InstOpen(dst, src Memory, n int) Inst {
	return instOpen{
		dst: dst,
		src: src,

		n: n,
	}
}

func (inst instOpen) Eval(state State) Result {
	if state == nil {
		openState := NewInstOpenState(inst.n)
		for i := 0; i < inst.n; i++ {
			value, ok := inst.src.Load(i).(ValuePrivate)
			if !ok {
				panic(fmt.Sprintf("unexpected value type %T", inst.src.Load(i)))
			}
			openState.Shares[i] = value.Share
		}
		return NotReady(openState)
	}
	switch state := state.(type) {
	case *InstOpenState:
		for i := 0; i < inst.n; i++ {
			inst.dst.Store(i, NewValuePublic(state.Results[i]))
		}
		return Ready()
	default:
		panic(fmt.Sprintf("unexpected state type %T", state))
	}
}

func (inst instOpen) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}

type instExit struct {
	src Memory

	n int
}

func InstExit(src Memory, n int) Inst {
	return instExit{src, n}
}

func (inst instExit) Eval(State) Result {
	state := NewInstExitState(inst.n)
	for i := 0; i < inst.n; i++ {
		state.Values[i] = inst.src.Load(i)
	}
	return Exit(state)
}

func (inst instExit) Expand() ([]Inst, bool) {
	return []Inst{inst}, false
}
