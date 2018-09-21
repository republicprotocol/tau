package asm

import (
	"github.com/republicprotocol/oro-go/core/vss"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type Result struct {
	Ready bool
	State State
}

func Ready() Result {
	return Result{Ready: true, State: nil}
}

func NotReady(state State) Result {
	return Result{Ready: false, State: state}
}

func Exit(state State) Result {
	return Result{Ready: true, State: state}
}

type State interface {
	IsState()
}

type InstGenerateRnState struct {
	Num    int
	Sigmas vss.VShares
}

func NewInstGenerateRnState(n int) *InstGenerateRnState {
	return &InstGenerateRnState{
		Num:    n,
		Sigmas: make(vss.VShares, n),
	}
}

func (state *InstGenerateRnState) IsState() {}

type InstGenerateRnZeroState struct {
	Num    int
	Sigmas vss.VShares
}

func NewInstGenerateRnZeroState(n int) *InstGenerateRnZeroState {
	return &InstGenerateRnZeroState{
		Num:    n,
		Sigmas: make(vss.VShares, n),
	}
}

func (state *InstGenerateRnZeroState) IsState() {}

type InstGenerateRnTupleState struct {
	Num    int
	Rhos   vss.VShares
	Sigmas vss.VShares
}

func NewInstGenerateRnTupleState(n int) *InstGenerateRnTupleState {
	return &InstGenerateRnTupleState{
		Num:    n,
		Rhos:   make(vss.VShares, n),
		Sigmas: make(vss.VShares, n),
	}
}

func (state *InstGenerateRnTupleState) IsState() {}

type InstMulState struct {
	Num     int
	Xs      shamir.Shares
	Ys      shamir.Shares
	Rhos    shamir.Shares
	Sigmas  shamir.Shares
	Results shamir.Shares
}

func NewInstMulState(n int) *InstMulState {
	return &InstMulState{
		Num:     n,
		Xs:      make(shamir.Shares, n),
		Ys:      make(shamir.Shares, n),
		Rhos:    make(shamir.Shares, n),
		Sigmas:  make(shamir.Shares, n),
		Results: make(shamir.Shares, n),
	}
}

func (state *InstMulState) IsState() {}

type InstOpenState struct {
	Num     int
	Shares  shamir.Shares
	Results algebra.FpElements
}

func NewInstOpenState(n int) *InstOpenState {
	return &InstOpenState{
		Num:     n,
		Shares:  make(shamir.Shares, n),
		Results: make(algebra.FpElements, n),
	}
}

func (state *InstOpenState) IsState() {}

type InstExitState struct {
	Num    int
	Values []Value
}

func NewInstExitState(n int) *InstExitState {
	return &InstExitState{
		Num:    n,
		Values: make([]Value, n),
	}
}

func (state *InstExitState) IsState() {}
