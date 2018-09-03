package open

import shamir "github.com/republicprotocol/shamir-go"

type Nonce [32]byte

type Addr uint64

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
