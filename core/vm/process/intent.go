package process

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/vss/algebra"

	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type Intent interface {
	IsIntent()
}

type IntentToGenerateRn struct {
	Sigma chan<- shamir.Share
}

func GenerateRn(σ chan<- shamir.Share) IntentToGenerateRn {
	return IntentToGenerateRn{
		Sigma: σ,
	}
}

func (intent IntentToGenerateRn) IsIntent() {
}

type IntentToGenerateRnTuple struct {
	Rho   chan<- shamir.Share
	Sigma chan<- shamir.Share
}

func GenerateRnTuple(ρ, σ chan<- shamir.Share) IntentToGenerateRnTuple {
	return IntentToGenerateRnTuple{
		Rho:   ρ,
		Sigma: σ,
	}
}

func (intent IntentToGenerateRnTuple) IsIntent() {
}

type IntentToGenerateRnZero struct {
	Sigma chan<- shamir.Share
}

func GenerateRnZero(σ chan<- shamir.Share) IntentToGenerateRnZero {
	return IntentToGenerateRnZero{
		Sigma: σ,
	}
}

func (intent IntentToGenerateRnZero) IsIntent() {
}

type IntentToMultiply struct {
	X, Y       shamir.Share
	Rho, Sigma shamir.Share

	Ret chan<- shamir.Share
}

func Multiply(x, y, ρ, σ shamir.Share, ret chan<- shamir.Share) IntentToMultiply {
	return IntentToMultiply{
		X:     x,
		Y:     y,
		Rho:   ρ,
		Sigma: σ,
		Ret:   ret,
	}
}

func (intent IntentToMultiply) IsIntent() {
}

type IntentToOpen struct {
	Value shamir.Share

	Ret chan<- algebra.FpElement
}

func Open(value shamir.Share, ret chan<- algebra.FpElement) IntentToOpen {
	return IntentToOpen{
		Value: value,
		Ret:   ret,
	}
}

func (intent IntentToOpen) IsIntent() {
}

type IntentToExit struct {
	Values []Value
}

func Exit(values []Value) IntentToExit {
	return IntentToExit{values}
}

func (intent IntentToExit) IsIntent() {
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

func ErrorInvalidMemoryAddr(addr Addr, pc PC) IntentToError {
	return ErrorExecution(
		fmt.Errorf("invalid memory address %v", addr),
		pc,
	)
}

func ErrorCodeOverflow(pc PC) IntentToError {
	return ErrorExecution(
		fmt.Errorf("code overflow"),
		pc,
	)
}

func ErrorUnexpectedTypeConversion(got, expected interface{}, pc PC) IntentToError {
	if expected == nil {
		return ErrorExecution(
			fmt.Errorf("unexpected type conversion of %T", got),
			pc,
		)
	}
	return ErrorExecution(
		fmt.Errorf("unexpected type conversion of %T into %T", got, expected),
		pc,
	)
}

func (intent IntentToError) IsIntent() {
}
