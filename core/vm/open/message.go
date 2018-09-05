package open

import (
	"math/big"

	shamir "github.com/republicprotocol/shamir-go"
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

	Value *big.Int
}

func NewResult(nonce Nonce, value *big.Int) Result {
	return Result{
		nonce, value,
	}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}
