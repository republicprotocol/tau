package process

import (
	"github.com/republicprotocol/oro-go/core/vss/algebra"

	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type PC uint64

type Code []Inst

type Inst interface {
	IsInst()
}

type instMacro struct {
	code Code
}

// InstMacro stores Code that is expanded by the Process before execution.
func InstMacro(code Code) Inst {
	return instMacro{code}
}

// IsInst implements the Inst interface.
func (inst instMacro) IsInst() {
}

type instCopy struct {
	dst Addr
	src Addr
}

// InstCopy a Value from a source Addr to a destination Addr.
func InstCopy(dst, src Addr) Inst {
	return instCopy{dst, src}
}

// IsInst implements the Inst interface.
func (inst instCopy) IsInst() {
}

type instMove struct {
	dst Addr
	val Value
}

// InstMove a Value to a destination Addr.
func InstMove(dst Addr, val Value) Inst {
	return instMove{dst, val}
}

// IsInst implements the Inst interface.
func (inst instMove) IsInst() {
}

type instAdd struct {
	dst Addr
	lhs Addr
	rhs Addr
}

// InstAdd a left-hand Value to a right-hand Value and move the result to a
// destination Addr.
func InstAdd(dst, lhs, rhs Addr) Inst {
	return instAdd{dst, lhs, rhs}
}

// IsInst implements the Inst interface.
func (inst instAdd) IsInst() {
}

type instNeg struct {
	dst Addr
	lhs Addr
}

// InstNeg a Value and move the result to a destination Addr.
func InstNeg(dst, lhs Addr) Inst {
	return instNeg{dst, lhs}
}

// IsInst implements the Inst interface.
func (inst instNeg) IsInst() {
}

type instSub struct {
	dst Addr
	lhs Addr
	rhs Addr
}

// InstSub a right-hand Value from a left-hand Value and move the result to a
// destination Addr.
func InstSub(dst, lhs, rhs Addr) Inst {
	return instSub{dst, lhs, rhs}
}

// IsInst implements the Inst interface.
func (inst instSub) IsInst() {
}

type instGenerateRn struct {
	dst    Addr
	σReady bool
	σCh    <-chan shamir.Share
	σ      shamir.Share
}

// InstGenerateRn and move the private Value to a destination Addr. This
// instruction is asynchronous.
func InstGenerateRn(dst Addr) Inst {
	return instGenerateRn{
		dst:    dst,
		σReady: false,
		σCh:    nil,
		σ:      shamir.Share{},
	}
}

// IsInst implements the Inst interface.
func (inst instGenerateRn) IsInst() {
}

type instGenerateRnZero struct {
	dst    Addr
	σReady bool
	σCh    <-chan shamir.Share
	σ      shamir.Share
}

// InstGenerateRnZero and move the private Value to a destination Addr. This
// instruction is asynchronous.
func InstGenerateRnZero(dst Addr) Inst {
	return instGenerateRnZero{
		dst:    dst,
		σReady: false,
		σCh:    nil,
		σ:      shamir.Share{},
	}
}

// IsInst implements the Inst interface.
func (inst instGenerateRnZero) IsInst() {
}

type instGenerateRnTuple struct {
	ρDst   Addr
	ρReady bool
	ρCh    <-chan shamir.Share
	ρ      shamir.Share

	σDst   Addr
	σReady bool
	σCh    <-chan shamir.Share
	σ      shamir.Share
}

// InstGenerateRnTuple and move the private Values to two destination Addrs.
// This instruction is asynchronous.
func InstGenerateRnTuple(ρDst, σDst Addr) Inst {
	return instGenerateRnTuple{
		ρDst:   ρDst,
		ρReady: false,
		ρCh:    nil,
		ρ:      shamir.Share{},

		σDst:   σDst,
		σReady: false,
		σCh:    nil,
		σ:      shamir.Share{}}
}

// IsInst implements the Inst interface.
func (inst instGenerateRnTuple) IsInst() {
}

type instMul struct {
	dst      Addr
	lhs      Addr
	rhs      Addr
	ρ        Addr
	σ        Addr
	retReady bool
	retCh    <-chan shamir.Share
	ret      shamir.Share
}

// InstMul a left-hand private Value with a right-hand private Value and move
// the result to a destination Addr. Executing a multiplication also requires a
// random number tuple. This instruction is asynchronous.
func InstMul(dst, lhs, rhs, ρ, σ Addr) Inst {
	return instMul{
		dst:      dst,
		lhs:      lhs,
		rhs:      rhs,
		ρ:        ρ,
		σ:        σ,
		retReady: false,
		retCh:    nil,
		ret:      shamir.Share{},
	}
}

// IsInst implements the Inst interface.
func (inst instMul) IsInst() {
}

type instMulPub struct {
	dst      Addr
	lhs      Addr
	rhs      Addr
	retReady bool
	retCh    <-chan algebra.FpElement
	ret      algebra.FpElement
}

// InstMulPub a left-hand private Value with a right-hand private Value and open
// the result into a public Value. The public Value is moved to a destination
// Addr. This instruction is asynchronous.
func InstMulPub(dst, lhs, rhs Addr) Inst {
	return instMul{
		dst:      dst,
		lhs:      lhs,
		rhs:      rhs,
		retReady: false,
		retCh:    nil,
		ret:      shamir.Share{},
	}
}

// IsInst implements the Inst interface.
func (inst instMulPub) IsInst() {
}

type instOpen struct {
	dst      Addr
	src      Addr
	retReady bool
	retCh    <-chan algebra.FpElement
	ret      algebra.FpElement
}

// InstOpen a private Value and move the resulting public Value to a destination
// Addr.
func InstOpen(dst, src Addr) Inst {
	return instOpen{
		dst:      dst,
		src:      src,
		retReady: false,
		retCh:    nil,
		ret:      algebra.FpElement{},
	}
}

// IsInst implements the Inst interface.
func (inst instOpen) IsInst() {
}

type instExit struct {
	src []Addr
}

// InstExit the Process and return the results at the source Addrs.
func InstExit(src ...Addr) Inst {
	return instExit{src}
}

// IsInst implements the Inst interface.
func (inst instExit) IsInst() {
}

type instDebug struct {
	d func()
}

func InstDebug(d func()) Inst {
	return instDebug{d}
}

func (inst instDebug) IsInst() {

}
