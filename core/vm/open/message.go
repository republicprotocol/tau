package open

import (
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type Nonce [32]byte

// An Open message signals to an Opener that it should open shares with other
// Openers. Before receiving an Open message for a particular Nonce, an Opener
// will still accept BroadcastShare messages related to the Nonce. However, an
// Opener will not produce a Result for a particular Nonce until the respective
// Open message is received.
type Open struct {
	Nonce
	shamir.Share
}

func NewOpen(nonce Nonce, share shamir.Share) Open {
	return Open{nonce, share}
}

// IsMessage implements the Message interface.
func (message Open) IsMessage() {
}

// A BroadcastShare message is used by an Opener to accept and store shares so
// that the respective secret can be opened. A BroadcastShare message is related
// to other BroadcastShare messages, and to an Open message, by its Nonce.
type BroadcastShare struct {
	Nonce
	shamir.Share
}

func NewBroadcastShare(nonce Nonce, share shamir.Share) BroadcastShare {
	return BroadcastShare{nonce, share}
}

// IsMessage implements the Message interface.
func (message BroadcastShare) IsMessage() {
}

// A Result message is produced by an Opener after it has received (a) an Open
// message, and (b) a sufficient threshold of BroadcastShare messages with the
// same Nonce. The order in which it receives the Open message and the
// BroadcastShare messages does not affect the production of a Result. A Result
// message is related to an Open message by its Nonce.
type Result struct {
	Nonce

	Value algebra.FpElement
}

func NewResult(nonce Nonce, value algebra.FpElement) Result {
	return Result{
		nonce, value,
	}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}
