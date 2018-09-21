package process

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/vm/asm"
)

type IntentID [40]byte

type Intent interface {
	IID() IntentID
}

func TransitionState(state asm.State) Intent {
	panic("unimplemented")
}

type IntentToError struct {
	error
}

func ErrorExecution(err error, pc PC, inst Inst) IntentToError {
	return IntentToError{
		fmt.Errorf("execution error at instruction %T(%v) = %v", inst, pc, err),
	}
}

func ErrorUnexpectedInst(inst Inst, pc PC) IntentToError {
	return ErrorExecution(
		fmt.Errorf("unexpected instruction type %T", inst),
		pc,
		inst,
	)
}

func ErrorInvalidMemoryAddr(addr Addr, pc PC, inst Inst) IntentToError {
	return ErrorExecution(
		fmt.Errorf("invalid memory address %v", addr),
		pc,
		inst,
	)
}

func ErrorCodeOverflow(pc PC, inst Inst) IntentToError {
	return ErrorExecution(
		fmt.Errorf("code overflow"),
		pc,
		inst,
	)
}

func ErrorUnexpectedTypeConversion(got, expected interface{}, pc PC, inst Inst) IntentToError {
	if expected == nil {
		return ErrorExecution(
			fmt.Errorf("unexpected type conversion of %T", got),
			pc,
			inst,
		)
	}
	return ErrorExecution(
		fmt.Errorf("unexpected type conversion of %T into %T", got, expected),
		pc,
		inst,
	)
}

func (intent IntentToError) IntentID() IntentID {
	return IntentID{}
}
