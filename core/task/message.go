package task

import (
	"fmt"
	"runtime/debug"
	"time"
)

// A MessageID uniquely identifies a Message, or series of related Messages. The
// first 32 bytes are generally expected to be reserved for ensuring uniqueness
// of the MessageID between different Message series (for example, by using the
// keccak256 hash of some data that the Message series is related to). The last
// 8 bytes are generally expected to be dedicated to ensuring uniqueness between
// Messages in the same series.
type MessageID [40]byte

type Message interface {
	IsMessage()
}

type Messages []Message

func NewMessages(messages ...Message) Message {
	return Messages(messages)
}

func (message Messages) IsMessage() {
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
