package process

import (
	"fmt"
	"math/big"

	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

type Intent interface {
	IsIntent()
}

type IntentToGenRn struct {
	Rho   chan<- shamir.Share
	Sigma chan<- shamir.Share
}

func GenRn(ρ, σ chan<- shamir.Share) IntentToGenRn {
	return IntentToGenRn{
		Rho:   ρ,
		Sigma: σ,
	}
}

func (intent IntentToGenRn) IsIntent() {
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

	Ret chan<- *big.Int
}

func Open(value shamir.Share, ret chan<- *big.Int) IntentToOpen {
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
