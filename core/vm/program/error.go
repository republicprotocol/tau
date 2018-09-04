package program

import (
	"errors"
	"fmt"
)

var (
	ErrCodeOverflow = errors.New("code overflow")
)

type ExecutionError struct {
	error
}

func NewExecutionError(err error, pc PC) error {
	return ExecutionError{
		fmt.Errorf("execution error at instruction %v = %v", pc, err),
	}
}

type UnexpectedInstError struct {
	error
}

func NewUnexpectedInstError(inst Inst, pc PC) error {
	return NewExecutionError(
		UnexpectedInstError{
			fmt.Errorf("unexpected instruction type %T", inst),
		},
		pc,
	)
}
