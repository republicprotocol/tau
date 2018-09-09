package mul

import (
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type Nonce [32]byte

// A Mul message signals to a Multiplier that it should open intermediate
// multiplication shares with other Multipliers. Before receiving a Mul message
// for a particular Nonce, a Multiplier will still accept
// BroadcastIntermediateShare messages related to the Nonce. However, a
// Multiplier will not produce a Result for a particular Nonce until the
// respective Mul message is received.
type Mul struct {
	Nonce

	x, y shamir.Share
	ρ, σ shamir.Share
}

func NewMul(nonce Nonce, x, y, ρ, σ shamir.Share) Mul {
	return Mul{
		nonce, x, y, ρ, σ,
	}
}

// IsMessage implements the Message interface.
func (message Mul) IsMessage() {
}

// A BroadcastIntermediateShare message is used by a Multiplier to accept and
// store intermediate multiplication shares so that the respective
// multiplication can be completed. A BroadcastIntermediateShare message is
// related to other BroadcastIntermediateShare messages, and to a Mul message,
// by its Nonce.
type BroadcastIntermediateShare struct {
	Nonce
	shamir.Share
}

func NewBroadcastIntermediateShare(nonce Nonce, share shamir.Share) BroadcastIntermediateShare {
	return BroadcastIntermediateShare{
		nonce, share,
	}
}

// IsMessage implements the Message interface.
func (message BroadcastIntermediateShare) IsMessage() {
}

// A Result message is produced by a Multiplier after it has received (a) a Mul
// message, and (b) a sufficient threshold of BroadcastIntermediateShare
// messages with the same Nonce. The order in which it receives the Mul message
// and the BroadcastIntermediateShare messages does not affect the production of
// a Result. A Result message is related to a Mul message by its Nonce.
type Result struct {
	Nonce
	shamir.Share
}

func NewResult(nonce Nonce, share shamir.Share) Result {
	return Result{
		nonce, share,
	}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}
