package rng

import (
	"fmt"
	"math/big"

	"github.com/republicprotocol/co-go"

	"github.com/republicprotocol/oro-go/core/vss/algebra"

	"github.com/republicprotocol/oro-go/core/vss/pedersen"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type rnger struct {
	scheme pedersen.Pedersen
	index  uint64

	n, k uint64

	genRns      map[task.MessageID]GenerateRn
	genRnTuples map[task.MessageID]GenerateRnTuple
	genRnZeros  map[task.MessageID]GenerateRnZero
	rnShares    map[task.MessageID]map[uint64]RnShares
	results     map[task.MessageID]Result
}

func New(scheme pedersen.Pedersen, index, n, k uint64, cap int) task.Task {
	return task.New(task.NewIO(cap), newRnger(scheme, index, n, k))
}

func newRnger(scheme pedersen.Pedersen, index, n, k uint64) *rnger {
	return &rnger{
		scheme: scheme,
		index:  index,

		n: n, k: k,

		genRns:      map[task.MessageID]GenerateRn{},
		genRnTuples: map[task.MessageID]GenerateRnTuple{},
		genRnZeros:  map[task.MessageID]GenerateRnZero{},
		rnShares:    map[task.MessageID]map[uint64]RnShares{},
		results:     map[task.MessageID]Result{},
	}
}

func (rnger *rnger) Reduce(message task.Message) task.Message {
	switch message := message.(type) {

	case GenerateRn:
		return rnger.generateRn(message)

	case GenerateRnTuple:
		return rnger.generateRnTuple(message)

	case GenerateRnZero:
		return rnger.generateRnZero(message)

	case RnShares:
		return rnger.tryBuildRnShareProposals(message)

	case ProposeRnShare:
		return rnger.acceptRnShare(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (rnger *rnger) generateRn(message GenerateRn) task.Message {
	// Short circuit when results have already been computed
	rnger.genRns[message.MessageID] = message
	if result, ok := rnger.results[message.MessageID]; ok {
		return result
	}

	σSharesMaps := make([]map[uint64]vss.VShare, message.batch)

	co.ForAll(message.batch, func(b int) {
		σSharesMaps[b] = make(map[uint64]vss.VShare, rnger.n)

		// Generate k/2 threshold shares for the random number
		rn := rnger.scheme.SecretField().Random()
		σShares := vss.Share(&rnger.scheme, rn, rnger.n, (rnger.k+1)/2)
		for _, σShare := range σShares {
			share := σShare.Share()
			σSharesMaps[b][share.Index()] = σShare
		}
	})

	return NewRnShares(message.MessageID, rnger.index, nil, σSharesMaps)
}

func (rnger *rnger) generateRnZero(message GenerateRnZero) task.Message {
	// Short circuit when results have already been computed
	rnger.genRnZeros[message.MessageID] = message
	if result, ok := rnger.results[message.MessageID]; ok {
		return result
	}

	zero := rnger.scheme.SecretField().NewInField(big.NewInt(0))
	σSharesMaps := make([]map[uint64]vss.VShare, message.batch)

	co.ForAll(message.batch, func(b int) {
		σSharesMaps[b] = make(map[uint64]vss.VShare, rnger.n)

		// Generate k/2 threshold shares for the random number
		σShares := vss.Share(&rnger.scheme, zero, rnger.n, (rnger.k+1)/2)
		for _, σShare := range σShares {
			share := σShare.Share()
			σSharesMaps[b][share.Index()] = σShare
		}
	})

	return NewRnShares(message.MessageID, rnger.index, nil, σSharesMaps)
}

func (rnger *rnger) generateRnTuple(message GenerateRnTuple) task.Message {
	// Short circuit when results have already been computed
	rnger.genRnTuples[message.MessageID] = message
	if result, ok := rnger.results[message.MessageID]; ok {
		return result
	}

	ρSharesMaps := make([]map[uint64]vss.VShare, message.batch)
	σSharesMaps := make([]map[uint64]vss.VShare, message.batch)

	co.ForAll(message.batch, func(b int) {
		ρSharesMaps[b] = make(map[uint64]vss.VShare, rnger.n)
		σSharesMaps[b] = make(map[uint64]vss.VShare, rnger.n)

		rn := rnger.scheme.SecretField().Random()
		co.ParBegin(
			func() {
				// Generate k threshold shares for a random number
				ρShares := vss.Share(&rnger.scheme, rn, rnger.n, rnger.k)
				for _, ρShare := range ρShares {
					share := ρShare.Share()
					ρSharesMaps[b][share.Index()] = ρShare
				}
			},
			func() {
				// Generate k/2 threshold shares for the same random number
				σShares := vss.Share(&rnger.scheme, rn, rnger.n, (rnger.k+1)/2)
				for _, σShare := range σShares {
					share := σShare.Share()
					σSharesMaps[b][share.Index()] = σShare
				}
			})
	})

	return NewRnShares(message.MessageID, rnger.index, ρSharesMaps, σSharesMaps)
}

func (rnger *rnger) tryBuildRnShareProposals(message RnShares) task.Message {
	// Do not continue if we have already completed the opening
	if _, ok := rnger.results[message.MessageID]; ok {
		return nil
	}

	// Store the received message
	if _, ok := rnger.rnShares[message.MessageID]; !ok {
		rnger.rnShares[message.MessageID] = map[uint64]RnShares{}
	}
	rnger.rnShares[message.MessageID][message.From] = message

	// Do not continue if there is an insufficient number of messages
	if uint64(len(rnger.rnShares[message.MessageID])) < rnger.k/2 {
		return nil
	}
	// Do not continue if we have not received a signal to generate a random
	// number
	if _, ok := rnger.genRns[message.MessageID]; !ok {
		if _, ok := rnger.genRnTuples[message.MessageID]; !ok {
			if _, ok := rnger.genRnZeros[message.MessageID]; !ok {
				return nil
			}
		}
	}

	messages := rnger.buildRnShareProposals(message)
	messages[rnger.index-1] = rnger.acceptRnShare(messages[rnger.index-1].(ProposeRnShare))

	return task.NewMessageBatch(messages)
}

func (rnger *rnger) acceptRnShare(message ProposeRnShare) task.Message {
	if rnger.index != message.To {
		return nil
	}

	result := NewResult(message.MessageID, message.Rhos, message.Sigmas)
	rnger.results[message.MessageID] = result
	delete(rnger.genRns, message.MessageID)
	delete(rnger.genRnTuples, message.MessageID)
	delete(rnger.genRnZeros, message.MessageID)
	delete(rnger.rnShares, message.MessageID)

	return result
}

// TODO: Remember, we need an additively homomorphic encryption
// scheme for these shares to ensure that this technique works.
func (rnger *rnger) buildRnShareProposals(message RnShares) []task.Message {

	zero := rnger.scheme.SecretField().NewInField(big.NewInt(0))
	one := rnger.scheme.Commit(zero, zero)
	ρCommitments := make(algebra.FpElements, rnger.k)
	for i := 0; i < len(ρCommitments); i++ {
		ρCommitments[i] = one
	}
	σCommitments := make(algebra.FpElements, (rnger.k+1)/2)
	for i := 0; i < len(σCommitments); i++ {
		σCommitments[i] = one
	}

	batch := len(message.Sigmas)
	rnShareProposals := make([]task.Message, rnger.n)

	for j := uint64(0); j < rnger.n; j++ {
		jIndex := j + 1

		var ρs, σs []vss.VShare
		if message.Rhos != nil {
			ρs = make([]vss.VShare, batch)
		}
		if message.Sigmas != nil {
			σs = make([]vss.VShare, batch)
		}

		co.ForAll(len(message.Sigmas), func(b int) {

			ρShare := shamir.New(jIndex, zero)
			ρt := shamir.New(jIndex, zero)
			ρ := vss.New(ρCommitments, ρShare, ρt)

			σShare := shamir.New(jIndex, zero)
			σt := shamir.New(jIndex, zero)
			σ := vss.New(σCommitments, σShare, σt)

			for _, rnShares := range rnger.rnShares[message.MessageID] {
				if rnShares.Rhos != nil {
					ρFromShares := rnShares.Rhos[b][jIndex]
					ρ = ρ.Add(&ρFromShares)
				}
				if rnShares.Sigmas != nil {
					σFromShares := rnShares.Sigmas[b][jIndex]
					σ = σ.Add(&σFromShares)
				}
			}

			if message.Rhos != nil {
				ρs[b] = ρ
			}
			if message.Sigmas != nil {
				σs[b] = σ
			}
		})

		rnShareProposals[j] = NewProposeRnShare(message.MessageID, jIndex, ρs, σs)
	}

	return rnShareProposals
}

type GenerateRn struct {
	task.MessageID
	batch int
}

// NewGenerateRn creates a new GenerateRn message.
func NewGenerateRn(id task.MessageID, batch int) GenerateRn {
	return GenerateRn{id, batch}
}

// IsMessage implements the Message interface for GenerateRn.
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

// IsMessage implements the Message interface for GenerateRnZero.
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

// IsMessage implements the Message interface for GenerateRnTuple.
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

// IsMessage implements the Message interface for RnShares.
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

// IsMessage implements the Message interface for ProposeRnShare.
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

// IsMessage implements the Message interface for Result.
func (message Result) IsMessage() {
}
