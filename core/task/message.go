package task

import (
	"fmt"
	"runtime/debug"
)

type Message interface {
	IsMessage()
}

type Error struct {
	error
}

func NewError(err error) Message {
	return Error{fmt.Errorf("err = %v\nstack = %v", err, string(debug.Stack()))}
}

func (message Error) IsMessage() {
}
