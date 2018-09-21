package process

import (
	"github.com/republicprotocol/oro-go/core/vss"
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
	dst  Addr
	src  Addr
	step int
	n    int
}

// InstCopy Values from the source Addr to the destination Addr.
func InstCopy(dst, src Addr, step int, n int) Inst {
	return instCopy{dst, src, step, n}
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

type instExp struct {
	dst Addr
	lhs Addr
	rhs Addr
}

// InstExp will pop two public values form the stack, raise one to the power of
// the other, and then push the result to the stack. This Inst is synchronous.
func InstExp(dst, lhs, rhs Addr) Inst {
	return instExp{dst, lhs, rhs}
}

// IsInst implements the Inst interface.
func (inst instExp) IsInst() {
}

type instInv struct {
	dst Addr
	lhs Addr
}

func InstInv(dst, lhs Addr) Inst {
	return instInv{dst, lhs}
}

// IsInst implements the Inst interface.
func (inst instInv) IsInst() {
}

type instMod struct {
	dst Addr
	lhs Addr
	rhs Addr
}

func InstMod(dst, lhs, rhs Addr) Inst {
	return instMod{dst, lhs, rhs}
}

// IsInst implements the Inst interface.
func (inst instMod) IsInst() {
}

type instGenerateRn struct {
	dst   Addr
	batch int

	σsReady bool
	σsCh    <-chan []vss.VShare
	σs      []vss.VShare
}

// InstGenerateRn and move the private Value to a destination Addr. This
// instruction is asynchronous.
func InstGenerateRn(dst Addr, batch int) Inst {
	return instGenerateRn{
		dst:   dst,
		batch: batch,

		σsReady: false,
		σsCh:    nil,
		σs:      nil,
	}
}

// IsInst implements the Inst interface.
func (inst instGenerateRn) IsInst() {
}

type instGenerateRnZero struct {
	dst   Addr
	batch int

	σsReady bool
	σsCh    <-chan []vss.VShare
	σs      []vss.VShare
}

// InstGenerateRnZero and move the private Value to a destination Addr. This
// instruction is asynchronous.
func InstGenerateRnZero(dst Addr, batch int) Inst {
	return instGenerateRnZero{
		dst:   dst,
		batch: batch,

		σsReady: false,
		σsCh:    nil,
		σs:      nil,
	}
}

// IsInst implements the Inst interface.
func (inst instGenerateRnZero) IsInst() {
}

type instGenerateRnTuple struct {
	dst   Addr
	batch int

	ρsReady bool
	ρsCh    <-chan []vss.VShare
	ρs      []vss.VShare
	σsReady bool
	σsCh    <-chan []vss.VShare
	σs      []vss.VShare
}

// InstGenerateRnTuple and move the private Values to two destination Addrs.
// This instruction is asynchronous.
func InstGenerateRnTuple(dst Addr, batch int) Inst {
	return instGenerateRnTuple{
		dst:   dst,
		batch: batch,

		ρsReady: false,
		ρsCh:    nil,
		ρs:      nil,
		σsReady: false,
		σsCh:    nil,
		σs:      nil,
	}
}

// IsInst implements the Inst interface.
func (inst instGenerateRnTuple) IsInst() {
}

type instMulPub struct {
	dst Addr
	lhs Addr
	rhs Addr
}

func InstMulPub(dst, lhs, rhs Addr) Inst {
	return instMulPub{
		dst: dst,
		lhs: lhs,
		rhs: rhs,
	}
}

// IsInst implements the Inst interface.
func (inst instMulPub) IsInst() {
}

type instMul struct {
	dst   Addr
	lhs   Addr
	rhs   Addr
	ρσs   Addr
	batch int

	retReady bool
	retCh    <-chan []shamir.Share
	ret      []shamir.Share
}

// InstMul a left-hand private Value with a right-hand private Value and move
// the result to a destination Addr. Executing a multiplication also requires a
// random number tuple. This instruction is asynchronous.
func InstMul(dst, lhs, rhs, ρσs Addr, batch int) Inst {
	return instMul{
		dst:   dst,
		lhs:   lhs,
		rhs:   rhs,
		ρσs:   ρσs,
		batch: batch,

		retReady: false,
		retCh:    nil,
		ret:      nil,
	}
}

// IsInst implements the Inst interface.
func (inst instMul) IsInst() {
}

type instMulOpen struct {
	dst      Addr
	lhs      Addr
	rhs      Addr
	retReady bool
	retCh    <-chan algebra.FpElement
	ret      algebra.FpElement
}

// InstMulOpen a left-hand private Value with a right-hand private Value and
// open the result into a public Value. The public Value is moved to a
// destination Addr. This instruction is asynchronous.
func InstMulOpen(dst, lhs, rhs Addr) Inst {
	return instMulOpen{
		dst:      dst,
		lhs:      lhs,
		rhs:      rhs,
		retReady: false,
		retCh:    nil,
		ret:      algebra.FpElement{},
	}
}

// IsInst implements the Inst interface.
func (inst instMulOpen) IsInst() {
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
