package task

import (
	"fmt"
	"runtime/debug"
	"time"
)

type MessageID [32]byte

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

type Tick struct {
	time.Time
}

func NewTick(time time.Time) Message {
	return Tick{time}
}

func (message Tick) IsMessage() {
}
