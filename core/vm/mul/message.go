package mul

import (
	shamir "github.com/republicprotocol/shamir-go"
)

type Nonce [32]byte

type Addr uint64

type Multiply struct {
	Nonce

	x, y shamir.Share
	ρ, σ shamir.Share
}

func NewMultiplyMessage(nonce Nonce, x, y, ρ, σ shamir.Share) Multiply {
	return Multiply{
		nonce, x, y, ρ, σ,
	}
}

// IsMessage implements the Message interface.
func (message Multiply) IsMessage() {
}

type Open struct {
	Nonce

	To    Addr
	From  Addr
	Value shamir.Share
}

func NewOpenMessage(nonce Nonce, to, from Addr, value shamir.Share) Open {
	return Open{
		nonce, to, from, value,
	}
}

// IsMessage implements the Message interface.
func (message Open) IsMessage() {
}

type Result struct {
	Nonce

	Value shamir.Share
}

func NewResultMessage(nonce Nonce, value shamir.Share) Result {
	return Result{
		nonce, value,
	}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}
