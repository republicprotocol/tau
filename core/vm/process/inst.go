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

type instPush struct {
	value Value
}

// InstPush will push a Value to the Stack. This Inst is synchronous.
func InstPush(value Value) Inst {
	return instPush{value}
}

// IsInst implements the Inst interface.
func (inst instPush) IsInst() {
}

type instCopy struct {
	depth uint64
}

// InstCopy will push a copy of the top element of the stack to the stack. This
// Inst is synchronous.
func InstCopy(depth uint64) Inst {
	return instCopy{depth}
}

// IsInst implements the Inst interface.
func (inst instCopy) IsInst() {
}

type instStore struct {
	addr Addr
}

// InstStore will pop a Value from the Stack and store it in Memory. This Inst
// is synchronous.
func InstStore(addr Addr) Inst {
	return instStore{addr}
}

// IsInst implements the Inst interface.
func (inst instStore) IsInst() {
}

type instLoad struct {
	addr Addr
}

// InstLoad will load a Value from Memory and push it to the Stack. This Inst is
// synchronous.
func InstLoad(addr Addr) Inst {
	return instLoad{addr}
}

// IsInst implements the Inst interface.
func (inst instLoad) IsInst() {
}

type instLoadStack struct {
	offset uint64
}

// InstLoadStack will load a Value from the stack and push it to the top of the
// Stack. This Inst is synchronous.
func InstLoadStack(offset uint64) Inst {
	return instLoadStack{offset}
}

// IsInst implements the Inst interface.
func (inst instLoadStack) IsInst() {
}

type instAdd struct {
}

// InstAdd will pop two Values from the Stack, add them, and then push the
// result to the Stack. This Inst is synchronous.
func InstAdd() Inst {
	return instAdd{}
}

// IsInst implements the Inst interface.
func (inst instAdd) IsInst() {
}

type instNeg struct {
}

// InstNeg will negate an element on the stack. This Inst is synchronous.
func InstNeg() Inst {
	return instNeg{}
}

// IsInst implements the Inst interface.
func (inst instNeg) IsInst() {
}

type instSub struct {
}

// InstSub will pop two Values from the Stack, subtract them, and then push the
// result to the Stack. This Inst is synchronous.
func InstSub() Inst {
	return instSub{}
}

// IsInst implements the Inst interface.
func (inst instSub) IsInst() {
}

type instExp struct {
}

// InstExp will pop two public values form the stack, raise one to the power of
// the other, and then push the result to the stack. This Inst is synchronous.
func InstExp() Inst {
	return instExp{}
}

// IsInst implements the Inst interface.
func (inst instExp) IsInst() {
}

type instGenerateRn struct {
	σReady bool
	σCh    <-chan shamir.Share
	σ      shamir.Share
}

// InstGenerateRn will generate a secure random number and push it to the Stack.
// This Inst is asynchronous.
func InstGenerateRn() Inst {
	return instGenerateRn{
		σReady: false,
		σCh:    nil,
		σ:      shamir.Share{},
	}
}

// IsInst implements the Inst interface.
func (inst instGenerateRn) IsInst() {
}

type instGenerateRnZero struct {
	σReady bool
	σCh    <-chan shamir.Share
	σ      shamir.Share
}

// InstGenerateRnZero will generate a secure random zero and push it to the
// Stack. This Inst is asynchronous.
func InstGenerateRnZero() Inst {
	return instGenerateRnZero{
		σReady: false,
		σCh:    nil,
		σ:      shamir.Share{},
	}
}

// IsInst implements the Inst interface.
func (inst instGenerateRnZero) IsInst() {
}

type instGenerateRnTuple struct {
	ρReady bool
	ρCh    <-chan shamir.Share
	ρ      shamir.Share

	σReady bool
	σCh    <-chan shamir.Share
	σ      shamir.Share
}

// InstGenerateRnTuple will generate a secure random number tuple and push the
// tuple to the Stack. This Inst is asynchronous.
func InstGenerateRnTuple() Inst {
	return instGenerateRnTuple{
		ρReady: false,
		ρCh:    nil,
		ρ:      shamir.Share{},

		σReady: false,
		σCh:    nil,
		σ:      shamir.Share{}}
}

// IsInst implements the Inst interface.
func (inst instGenerateRnTuple) IsInst() {
}

type instMul struct {
	retReady bool
	retCh    <-chan shamir.Share
	ret      shamir.Share
}

// InstMul will pop a private random number tuple from the Stack, and then pop
// two Values from the Stack. It will use the private random number tuple to
// multiply the two Values, and then push the result to the Stack. This Inst is
// asynchronous.
func InstMul() Inst {
	return instMul{
		retReady: false,
		retCh:    nil,
		ret:      shamir.Share{},
	}
}

// IsInst implements the Inst interface.
func (inst instMul) IsInst() {
}

type instOpen struct {
	retReady bool
	retCh    <-chan algebra.FpElement
	ret      algebra.FpElement
}

// InstOpen will pop a private Value from the Stack, and open it into a public
// Value. This Inst is asynchronous.
func InstOpen() Inst {
	return instOpen{
		retReady: false,
		retCh:    nil,
		ret:      algebra.FpElement{},
	}
}

// IsInst implements the Inst interface.
func (inst instOpen) IsInst() {
}

type instMacro struct {
	code Code
}

// InstMacro will insert code into the list of instructions. This Inst is
// synchronous.
func InstMacro(code Code) Inst {
	return instMacro{code}
}

// IsInst implements the Inst interface.
func (inst instMacro) IsInst() {
}
