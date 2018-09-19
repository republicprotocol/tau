package rng

import (
	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss"
)

type GenerateRn struct {
	task.MessageID
	batch int
}

// NewGenerateRn creates a new GenerateRn message.
func NewGenerateRn(id task.MessageID, batch int) GenerateRn {
	return GenerateRn{id, batch}
}

// IsMessage implements the Message interface.
func (message GenerateRn) IsMessage() {
}

type GenerateRnZero struct {
	task.MessageID
	batch int
}

// NewGenerateRnZero creates a new GenerateRnZero message.
func NewGenerateRnZero(id task.MessageID, batch int) GenerateRnZero {
	return GenerateRnZero{id, batch}
}

// IsMessage implements the Message interface.
func (message GenerateRnZero) IsMessage() {
}

type GenerateRnTuple struct {
	task.MessageID
	batch int
}

// NewGenerateRnTuple creates a new GenerateRnTuple message.
func NewGenerateRnTuple(id task.MessageID, batch int) GenerateRnTuple {
	return GenerateRnTuple{id, batch}
}

// IsMessage implements the Message interface.
func (message GenerateRnTuple) IsMessage() {
}

type RnShares struct {
	task.MessageID

	From   uint64
	Rhos   []map[uint64]vss.VShare
	Sigmas []map[uint64]vss.VShare
}

// NewRnShares returns a new RnShares message.
func NewRnShares(id task.MessageID, from uint64, ρs, σs []map[uint64]vss.VShare) RnShares {
	return RnShares{id, from, ρs, σs}
}

// IsMessage implements the Message interface.
func (message RnShares) IsMessage() {
}

type ProposeRnShare struct {
	task.MessageID

	To     uint64
	Rhos   []vss.VShare
	Sigmas []vss.VShare
}

// NewProposeRnShare returns a new ProposeRnShare message.
func NewProposeRnShare(id task.MessageID, to uint64, ρs, σs []vss.VShare) ProposeRnShare {
	return ProposeRnShare{id, to, ρs, σs}
}

// IsMessage implements the Message interface.
func (message ProposeRnShare) IsMessage() {
}

type Result struct {
	task.MessageID

	Rhos   []vss.VShare
	Sigmas []vss.VShare
}

// NewResult returns a new Result message.
func NewResult(id task.MessageID, ρs, σs []vss.VShare) Result {
	return Result{id, ρs, σs}
}

// IsMessage implements the Message interface.
func (message Result) IsMessage() {
}
