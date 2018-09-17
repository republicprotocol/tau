package rng

import (
	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss"
)

type GenerateRn struct {
	task.MessageID
}

// NewGenerateRn creates a new GenerateRn message.
func NewGenerateRn(id task.MessageID) GenerateRn {
	return GenerateRn{id}
}

// IsMessage implements the Message interface.
func (message GenerateRn) IsMessage() {
}

type GenerateRnTuple struct {
	task.MessageID
}

// NewGenerateRnTuple creates a new GenerateRnTuple message.
func NewGenerateRnTuple(id task.MessageID) GenerateRnTuple {
	return GenerateRnTuple{id}
}

// IsMessage implements the Message interface.
func (message GenerateRnTuple) IsMessage() {
}

type GenerateRnZero struct {
	task.MessageID
}

// NewGenerateRnZero creates a new GenerateRnZero message.
func NewGenerateRnZero(id task.MessageID) GenerateRnZero {
	return GenerateRnZero{id}
}

// IsMessage implements the Message interface.
func (message GenerateRnZero) IsMessage() {
}

type RnShares struct {
	task.MessageID

	Index uint64
	Rho   map[uint64]vss.VShare
	Sigma map[uint64]vss.VShare
}

// NewRnShares returns a new RnShares message.
func NewRnShares(id task.MessageID, index uint64, ρ, σ map[uint64]vss.VShare) RnShares {
	return RnShares{id, index, ρ, σ}
}

// IsMessage implements the Message interface.
func (message RnShares) IsMessage() {
}

type ProposeRnShare struct {
	task.MessageID

	Rho   *vss.VShare
	Sigma *vss.VShare
}

// NewProposeRnShare returns a new ProposeRnShare message.
func NewProposeRnShare(id task.MessageID, ρ, σ *vss.VShare) ProposeRnShare {
	return ProposeRnShare{id, ρ, σ}
}

// IsMessage implements the Message interface.
func (message ProposeRnShare) IsMessage() {
}

type Result struct {
	task.MessageID

	Rho   *vss.VShare
	Sigma *vss.VShare
}

// NewResult returns a new Result message.
func NewResult(id task.MessageID, ρ, σ *vss.VShare) Result {
	return Result{id, ρ, σ}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}
