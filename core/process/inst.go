package process

import (
	"github.com/republicprotocol/smpc-go/core/vss/algebra"

	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

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
	RhoReady bool
	RhoCh    <-chan shamir.Share
	Rho      shamir.Share

	SigmaReady bool
	SigmaCh    <-chan shamir.Share
	Sigma      shamir.Share
}

func (inst InstRand) IsInst() {
}

type InstMul struct {
	RetReady bool
	RetCh    <-chan shamir.Share
	Ret      shamir.Share
}

func (inst InstMul) IsInst() {
}

type InstOpen struct {
	RetReady bool
	RetCh    <-chan algebra.FpElement
	Ret      algebra.FpElement
}

func (inst InstOpen) IsInst() {
}
