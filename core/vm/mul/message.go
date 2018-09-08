package mul

import (
	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

type Nonce [32]byte

type Multiply struct {
	Nonce

	x, y shamir.Share
	ρ, σ shamir.Share
}

func NewMultiply(nonce Nonce, x, y, ρ, σ shamir.Share) Multiply {
	return Multiply{
		nonce, x, y, ρ, σ,
	}
}

// IsMessage implements the Message interface.
func (message Multiply) IsMessage() {
}

type Open struct {
	Nonce
	shamir.Share
}

func NewOpen(nonce Nonce, share shamir.Share) Open {
	return Open{
		nonce, share,
	}
}

// IsMessage implements the Message interface.
func (message Open) IsMessage() {
}

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
