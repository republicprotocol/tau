package rng

import (
	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type SignalGenerateRn struct {
	task.MessageID
}

// NewSignalGenerateRn creates a new SignalGenerateRn message.
func NewSignalGenerateRn(id task.MessageID) SignalGenerateRn {
	return SignalGenerateRn{id}
}

// IsMessage implements the Message interface.
func (message SignalGenerateRn) IsMessage() {
}

type RnShares struct {
	task.MessageID

	Rho   map[uint64]vss.VShare
	Sigma map[uint64]vss.VShare
	Index uint64
}

// NewRnShares returns a new RnShares message.
func NewRnShares(id task.MessageID, ρ, σ map[uint64]vss.VShare, index uint64) RnShares {
	return RnShares{id, ρ, σ, index}
}

// IsMessage implements the Message interface.
func (message RnShares) IsMessage() {
}

type ProposeRnShare struct {
	task.MessageID

	Rho   vss.VShare
	Sigma vss.VShare
}

// NewProposeRnShare returns a new ProposeRnShare message.
func NewProposeRnShare(id task.MessageID, ρ, σ vss.VShare) ProposeRnShare {
	return ProposeRnShare{id, ρ, σ}
}

// IsMessage implements the Message interface.
func (message ProposeRnShare) IsMessage() {
}

type Result struct {
	task.MessageID

	Rho   shamir.Share
	Sigma shamir.Share
}

// NewResult returns a new Result message.
func NewResult(id task.MessageID, ρ, σ shamir.Share) Result {
	return Result{id, ρ, σ}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}
