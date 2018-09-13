package mul

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

// A SignalMul message signals to a Multiplier that it should open intermediate
// multiplication shares with other Multipliers. Before receiving a SignalMul
// message for a particular task.MessageID, a Multiplier will still accept OpenMul
// messages related to the task.MessageID. However, a Multiplier will not produce a
// Result for a particular task.MessageID until the respective SignalMul message is
// received.
type SignalMul struct {
	task.MessageID

	x, y shamir.Share
	ρ, σ shamir.Share
}

// NewSignalMul returns a new SignalMul message.
func NewSignalMul(id task.MessageID, x, y, ρ, σ shamir.Share) SignalMul {
	return SignalMul{
		id, x, y, ρ, σ,
	}
}

// IsMessage implements the Message interface.
func (message SignalMul) IsMessage() {
}

func (message SignalMul) String() string {
	return fmt.Sprintf("mul.SignalMul {\n\tid: %v\n\tx: %v\n\ty: %v\n\tρ: %v\n\tσ: %v\n}", message.MessageID, message.x, message.y, message.ρ, message.σ)
}

// An OpenMul message is used by a Multiplier to accept and store intermediate
// multiplication shares so that the respective multiplication can be completed.
// An OpenMul message is related to other OpenMul messages, and to a Mul
// message, by its task.MessageID.
type OpenMul struct {
	task.MessageID
	shamir.Share
}

// NewOpenMul returns a new OpenMul message.
func NewOpenMul(id task.MessageID, share shamir.Share) OpenMul {
	return OpenMul{
		id, share,
	}
}

// IsMessage implements the Message interface.
func (message OpenMul) IsMessage() {
}

func (message OpenMul) String() string {
	return fmt.Sprintf("mul.OpenMul {\n\tid: %v\n\tshare: %v\n}", message.MessageID, message.Share)
}

// A Result message is produced by a Multiplier after it has received (a) a
// SignalMul message, and (b) a sufficient threshold of OpenMul messages with
// the same task.MessageID. The order in which it receives the SignalMul message and the
// OpenMul messages does not affect the production of a Result. A Result message
// is related to a SignalMul message by its task.MessageID.
type Result struct {
	task.MessageID
	shamir.Share
}

// NewResult returns a new Result message.
func NewResult(id task.MessageID, share shamir.Share) Result {
	return Result{id, share}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}

func (message Result) String() string {
	return fmt.Sprintf("mul.Result {\n\tid: %v\n\tshare: %v\n}", message.MessageID, message.Share)
}
