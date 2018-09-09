package open

import (
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type Nonce [32]byte

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
