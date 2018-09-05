package open

import (
	shamir "github.com/republicprotocol/shamir-go"
	"github.com/republicprotocol/smpc-go/core/node"
)

type Nonce [32]byte

type Open struct {
	Nonce

	From  node.Addr
	Value shamir.Share
}

func NewOpen(nonce Nonce, from node.Addr, value shamir.Share) Open {
	return Open{
		nonce, from, value,
	}
}

// IsMessage implements the Message interface.
func (message Open) IsMessage() {
}

type Result struct {
	Nonce

	Value shamir.Share
}

func NewResult(nonce Nonce, value shamir.Share) Result {
	return Result{
		nonce, value,
	}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}
