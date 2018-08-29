package mul

import shamir "github.com/republicprotocol/shamir-go"

type Nonce [32]byte

type InputMessage interface {

	// IsInputMessage is a marker used to restrict InputMessages to types that
	// have been explicitly marked. It is never called.
	IsInputMessage()
}

type OutputMessage interface {

	// IsOutputMessage is a marker used to restrict OutputMessages to types that
	// have been explicitly marked. It is never called.
	IsOutputMessage()
}

type Nominate struct {
	Leader uint
}

// IsInputMessage implements the InputMessage interface.
func (message Nominate) IsInputMessage() {
}

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

// IsInputMessage implements the InputMessage interface.
func (message Mul) IsInputMessage() {
}

type Open struct {
	Nonce

	Value shamir.Share
}

// IsInputMessage implements the InputMessage interface.
func (message Open) IsInputMessage() {
}
