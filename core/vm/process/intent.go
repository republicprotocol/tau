package process

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/vss"
	"github.com/republicprotocol/oro-go/core/vss/algebra"

	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type IntentID [40]byte

type Intent interface {
	IntentID() IntentID
}

type IntentToGenerateRn struct {
	ID    IntentID
	Batch int

	Sigmas chan<- []vss.VShare
}

func GenerateRn(id IntentID, batch int, σs chan<- []vss.VShare) IntentToGenerateRn {
	return IntentToGenerateRn{
		ID:    id,
		Batch: batch,

		Sigmas: σs,
	}
}

func (intent IntentToGenerateRn) IntentID() IntentID {
	return intent.ID
}

type IntentToGenerateRnZero struct {
	ID    IntentID
	Batch int

	Sigmas chan<- []vss.VShare
}

func GenerateRnZero(id IntentID, batch int, σs chan<- []vss.VShare) IntentToGenerateRnZero {
	return IntentToGenerateRnZero{
		ID:    id,
		Batch: batch,

		Sigmas: σs,
	}
}

func (intent IntentToGenerateRnZero) IntentID() IntentID {
	return intent.ID
}

type IntentToGenerateRnTuple struct {
	ID    IntentID
	Batch int

	Rhos   chan<- []vss.VShare
	Sigmas chan<- []vss.VShare
}

func GenerateRnTuple(id IntentID, batch int, ρs, σs chan<- []vss.VShare) IntentToGenerateRnTuple {
	return IntentToGenerateRnTuple{
		ID:    id,
		Batch: batch,

		Rhos:   ρs,
		Sigmas: σs,
	}
}

func (intent IntentToGenerateRnTuple) IntentID() IntentID {
	return intent.ID
}

type IntentToMultiply struct {
	ID         IntentID
	X, Y       shamir.Share
	Rho, Sigma shamir.Share

	Ret chan<- shamir.Share
}

func Multiply(id IntentID, x, y, ρ, σ shamir.Share, ret chan<- shamir.Share) IntentToMultiply {
	return IntentToMultiply{
		ID:    id,
		X:     x,
		Y:     y,
		Rho:   ρ,
		Sigma: σ,

		Ret: ret,
	}
}

func (intent IntentToMultiply) IntentID() IntentID {
	return intent.ID
}

type IntentToOpen struct {
	ID    IntentID
	Value shamir.Share

	Ret chan<- algebra.FpElement
}

func Open(id IntentID, value shamir.Share, ret chan<- algebra.FpElement) IntentToOpen {
	return IntentToOpen{
		ID:    id,
		Value: value,

		Ret: ret,
	}
}

func (intent IntentToOpen) IntentID() IntentID {
	return intent.ID
}

type IntentToExit struct {
	ID     IntentID
	Values []Value
}

func Exit(id IntentID, values []Value) IntentToExit {
	return IntentToExit{
		ID:     id,
		Values: values,
	}
}

func (intent IntentToExit) IntentID() IntentID {
	return intent.ID
}

type IntentToAwait struct {
	ID      IntentID
	Intents []Intent
}

func Await(id IntentID, intents []Intent) IntentToAwait {
	return IntentToAwait{
		ID:      id,
		Intents: intents,
	}
}

func (intent IntentToAwait) IntentID() IntentID {
	return intent.ID
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

func (intent IntentToError) IntentID() IntentID {
	return IntentID{}
}
