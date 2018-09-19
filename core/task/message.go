package task

import (
	"encoding/base64"
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

func (id MessageID) String() string {
	idBase64 := base64.StdEncoding.EncodeToString(id[:])
	idRunes := []rune(idBase64)
	return string(idRunes[24:])
}

type Message interface {
	IsMessage()
}

type MessageBatch []Message

func NewMessageBatch(messages []Message) Message {
	return MessageBatch(messages)
}

func (message MessageBatch) IsMessage() {
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
