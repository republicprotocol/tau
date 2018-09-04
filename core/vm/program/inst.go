package program

import "github.com/republicprotocol/smpc-go/core/vss/shamir"

type PC uint64

type Code []Inst

type Inst interface {
	IsInst()
}

type InstPush struct {
	Value
}

func (inst InstPush) IsInst() {
}

type InstAdd struct {
}

func (inst InstAdd) IsInst() {
}

type InstRand struct {
	SigmaReady bool
	SigmaCh    chan shamir.Share
	Sigma      shamir.Share

	RhoReady bool
	RhoCh    chan shamir.Share
	Rho      shamir.Share
}

func (inst InstRand) IsInst() {
}
