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

// A Message is an interface that can be sent between Tasks.
type Message interface {

	// MessageID returns the `MessageID` of the Message.
	MessageID() MessageID
}

// A MessageBatch is a Message containing multiple Messages. During reduction, a
// MessageBatch will be flattened into individual Messages and the Reducer will
// be invoked multiple times. No order if invocation is guaranteed.
type MessageBatch []Message

// NewMessageBatch returns a MessageBatch that contains a slice of Messages.
func NewMessageBatch(messages []Message) Message {
	return MessageBatch(messages)
}

// MessageID implements the Message interface for MessageBatch.
func (message MessageBatch) MessageID() MessageID {
	return [40]byte{}
}

// An Error is a Message wrapper type for sending errors between Tasks. It
// automatically catches the stack trace to help with debugging the origin of
// the error.
type Error struct {
	Error error
	ID    MessageID
}

// NewError returns an Error. The stack trace is captured at the moment this
// function is called.
func NewError(err error, messageID MessageID) Message {
	return Error{fmt.Errorf("err = %v; stack = %v", err, string(debug.Stack())), messageID}
}

// MessageID implements the Message interface for Error.
func (message Error) MessageID() MessageID {
	return message.ID
}

// RandomMessage is an empty message that can be used for testing internally and
// externally (projects using tau framework).
type RandomMessage struct {
	ID MessageID
}

// MessageID implements the Message interface for Random Message.
func (message RandomMessage) MessageID() MessageID {
	return message.ID
}

// A Tick is a Message that is used to signal the passing of time. Tasks should
// rely on Ticks to keep track of time, instead of tracking it internally.
type Tick struct {
	Time time.Time
	ID   MessageID
}

// NewTick returns a Tick for a moment in time.
func NewTick(time time.Time, messageID MessageID) Message {
	return Tick{time, messageID}
}

// MessageID implements the Message interface for Tick.
func (message Tick) MessageID() MessageID {
	return message.ID
}
