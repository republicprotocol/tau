package open

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

// A Signal message signals to an Opener that it should open shares with other
// Openers. Before receiving a Signal message for a particular task.MessageID,
// an Opener will still accept Open messages related to the task.MessageID.
// However, an Opener will not produce a Result for a particular task.MessageID
// until the respective Signal message is received.
type Signal struct {
	task.MessageID
	shamir.Share
}

// NewSignal returns a new Signal message.
func NewSignal(id task.MessageID, share shamir.Share) Signal {
	return Signal{id, share}
}

// IsMessage implements the Message interface.
func (message Signal) IsMessage() {
}

func (message Signal) String() string {
	return fmt.Sprintf("open.Signal {\n\tid: %v\n\tshare: %v\n}", message.MessageID, message.Share)
}

// An Open message is used by an Opener to accept and store shares so that the
// respective secret can be opened. An Open message is related to other Open
// messages, and to a Signal message, by its task.MessageID.
type Open struct {
	task.MessageID
	shamir.Share
}

// NewOpen returns a new Open message.
func NewOpen(id task.MessageID, share shamir.Share) Open {
	return Open{id, share}
}

// IsMessage implements the Message interface.
func (message Open) IsMessage() {
}

func (message Open) String() string {
	return fmt.Sprintf("open.Open {\n\tid: %v\n\tshare: %v\n}", message.MessageID, message.Share)
}

// A Result message is produced by an Opener after it has received (a) a Signal
// message, and (b) a sufficient threshold of Open messages with the same task.MessageID.
// The order in which it receives the Signal message and the Open messages does
// not affect the production of a Result. A Result message is related to a
// Signal message by its task.MessageID.
type Result struct {
	task.MessageID

	Value algebra.FpElement
}

// NewResult returns a new Result message.
func NewResult(id task.MessageID, value algebra.FpElement) Result {
	return Result{
		id, value,
	}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}

func (message Result) String() string {
	return fmt.Sprintf("open.Result {\n\tid: %v\n\tvalue: %v\n}", message.MessageID, message.Value)
}
