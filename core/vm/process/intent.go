package process

import (
	"fmt"

	"github.com/republicprotocol/smpc-go/core/vss/algebra"

	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

type Intent interface {
	IsIntent()
}

type IntentToGenerateRn struct {
	Rho   chan<- shamir.Share
	Sigma chan<- shamir.Share
}

func GenerateRn(ρ, σ chan<- shamir.Share) IntentToGenerateRn {
	return IntentToGenerateRn{
		Rho:   ρ,
		Sigma: σ,
	}
}

func (intent IntentToGenerateRn) IsIntent() {
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

func ErrorUnexpectedValue(got, expected Value, pc PC) IntentToError {
	return ErrorExecution(
		fmt.Errorf("unexpected value type %T expected %T", got, expected),
		pc,
	)
}

func (intent IntentToError) IsIntent() {
}
