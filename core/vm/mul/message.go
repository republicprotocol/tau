package mul

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

// A Mul message signals to a Multiplier that it should open intermediate
// multiplication shares with other Multipliers. Before receiving a Mul
// message for a particular task.MessageID, a Multiplier will still accept OpenMul
// messages related to the task.MessageID. However, a Multiplier will not produce a
// Result for a particular task.MessageID until the respective Mul message is
// received.
type Mul struct {
	task.MessageID

	xs, ys []shamir.Share
	ρs, σs []shamir.Share
}

// NewMul returns a new Mul message.
func NewMul(id task.MessageID, xs, ys, ρs, σs []shamir.Share) Mul {
	return Mul{
		id, xs, ys, ρs, σs,
	}
}

// IsMessage implements the Message interface.
func (message Mul) IsMessage() {
}

func (message Mul) String() string {
	return fmt.Sprintf("mul.Mul {\n\tid: %v\n\tx: %v\n\ty: %v\n\tρ: %v\n\tσ: %v\n}", message.MessageID, message.xs, message.ys, message.ρs, message.σs)
}

// An OpenMul message is used by a Multiplier to accept and store intermediate
// multiplication shares so that the respective multiplication can be completed.
// An OpenMul message is related to other OpenMul messages, and to a Mul
// message, by its task.MessageID.
type OpenMul struct {
	task.MessageID

	From   uint64
	Shares []shamir.Share
}

// NewOpenMul returns a new OpenMul message.
func NewOpenMul(id task.MessageID, from uint64, shares []shamir.Share) OpenMul {
	return OpenMul{
		id, from, shares,
	}
}

// IsMessage implements the Message interface.
func (message OpenMul) IsMessage() {
}

func (message OpenMul) String() string {
	return fmt.Sprintf("mul.OpenMul {\n\tid: %v\n\tshare: %v\n}", message.MessageID, message.Shares)
}

// A Result message is produced by a Multiplier after it has received (a) a
// Mul message, and (b) a sufficient threshold of OpenMul messages with
// the same task.MessageID. The order in which it receives the Mul message and the
// OpenMul messages does not affect the production of a Result. A Result message
// is related to a Mul message by its task.MessageID.
type Result struct {
	task.MessageID

	Shares []shamir.Share
}

// NewResult returns a new Result message.
func NewResult(id task.MessageID, shares []shamir.Share) Result {
	return Result{id, shares}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}

func (message Result) String() string {
	return fmt.Sprintf("mul.Result {\n\tid: %v\n\tshare: %v\n}", message.MessageID, message.Shares)
}
