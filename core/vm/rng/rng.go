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

	n, k, t uint64

	genRns      map[task.MessageID]GenerateRn
	genRnTuples map[task.MessageID]GenerateRnTuple
	genRnZeros  map[task.MessageID]GenerateRnZero
	rnShares    map[task.MessageID]map[uint64]RnShares
	results     map[task.MessageID]Result
}

func New(scheme pedersen.Pedersen, index, n, k, t uint64, cap int) task.Task {
	return task.New(task.NewIO(cap), newRnger(scheme, index, n, k, t))
}

func newRnger(scheme pedersen.Pedersen, index, n, k, t uint64) *rnger {
	return &rnger{
		scheme: scheme,
		index:  index,

		n: n, k: k, t: t,

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

	rn := rnger.scheme.SecretField().Random()
	σSharesMap := make(map[uint64]vss.VShare, rnger.n)

	// TODO: Remove duplication.
	// Generate k/2 threshold shares for the random number
	σShares := vss.Share(&rnger.scheme, rn, rnger.n, rnger.k/2)
	for _, σShare := range σShares {
		share := σShare.Share()
		σSharesMap[share.Index()] = σShare
	}

	return NewRnShares(message.MessageID, rnger.index, nil, σSharesMap)
}

func (rnger *rnger) generateRnZero(message GenerateRnZero) task.Message {
	// Short circuit when results have already been computed
	rnger.genRnZeros[message.MessageID] = message
	if result, ok := rnger.results[message.MessageID]; ok {
		return result
	}

	zero := rnger.scheme.SecretField().NewInField(big.NewInt(0))
	σSharesMap := make(map[uint64]vss.VShare, rnger.n)

	// TODO: Remove duplication.
	// Generate k/2 threshold shares for the random number
	σShares := vss.Share(&rnger.scheme, zero, rnger.n, rnger.k/2)
	for _, σShare := range σShares {
		share := σShare.Share()
		σSharesMap[share.Index()] = σShare
	}

	return NewRnShares(message.MessageID, rnger.index, nil, σSharesMap)
}

func (rnger *rnger) generateRnTuple(message GenerateRnTuple) task.Message {
	// Short circuit when results have already been computed
	rnger.genRnTuples[message.MessageID] = message
	if result, ok := rnger.results[message.MessageID]; ok {
		return result
	}

	rn := rnger.scheme.SecretField().Random()
	ρSharesMap := make(map[uint64]vss.VShare, rnger.n)
	σSharesMap := make(map[uint64]vss.VShare, rnger.n)
	co.ParBegin(
		func() {
			// TODO: Remove duplication.
			// Generate k threshold shares for the random number
			ρShares := vss.Share(&rnger.scheme, rn, rnger.n, rnger.k)
			for _, ρShare := range ρShares {
				share := ρShare.Share()
				ρSharesMap[share.Index()] = ρShare
			}
		},
		func() {
			// TODO: Remove duplication.
			// Generate k/2 threshold shares for the same random number
			σShares := vss.Share(&rnger.scheme, rn, rnger.n, rnger.k/2)
			for _, σShare := range σShares {
				share := σShare.Share()
				σSharesMap[share.Index()] = σShare
			}
		})

	return NewRnShares(message.MessageID, rnger.index, ρSharesMap, σSharesMap)
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
	rnger.rnShares[message.MessageID][message.Index] = message

	// Do not continue if there is an insufficient number of messages
	if uint64(len(rnger.rnShares[message.MessageID])) < rnger.t {
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

	return task.NewMessageBatch(messages...)
}

func (rnger *rnger) acceptRnShare(message ProposeRnShare) task.Message {

	result := NewResult(message.MessageID, message.Rho, message.Sigma)
	rnger.results[message.MessageID] = result
	delete(rnger.genRns, message.MessageID)
	delete(rnger.genRnTuples, message.MessageID)
	delete(rnger.genRnZeros, message.MessageID)
	delete(rnger.rnShares, message.MessageID)

	return result
}

func (rnger *rnger) buildRnShareProposals(message RnShares) []task.Message {

	zero := rnger.scheme.SecretField().NewInField(big.NewInt(0))

	// TODO: Create more efficient way to get the one element (in the larger
	// field, not the secret field).
	one := rnger.scheme.Commit(zero, zero)

	ρCommitments := make(algebra.FpElements, rnger.k)
	for i := 0; i < len(ρCommitments); i++ {
		ρCommitments[i] = one
	}
	σCommitments := make(algebra.FpElements, rnger.k/2)
	for i := 0; i < len(σCommitments); i++ {
		σCommitments[i] = one
	}

	rnShareProposals := make([]task.Message, rnger.n)

	co.ForAll(int(rnger.n), func(j int) {
		index := uint64(j) + 1

		ρShare := shamir.New(index, zero)
		ρt := shamir.New(index, zero)
		ρ := vss.New(ρCommitments, ρShare, ρt)

		σShare := shamir.New(index, zero)
		σt := shamir.New(index, zero)
		σ := vss.New(σCommitments, σShare, σt)

		for _, rnShares := range rnger.rnShares[message.MessageID] {
			// TODO: Remember, we need an additively homomorphic encryption
			// scheme for these shares to ensure that this technique works.

			if rnShares.Rho != nil {
				ρFromShares := rnShares.Rho[index]
				ρ = ρ.Add(&ρFromShares)
			}
			if rnShares.Sigma != nil {
				σFromShares := rnShares.Sigma[index]
				σ = σ.Add(&σFromShares)
			}
		}

		var ρPtr *vss.VShare
		var σPtr *vss.VShare
		if message.Rho != nil {
			ρPtr = &ρ
		}
		if message.Sigma != nil {
			σPtr = &σ
		}

		rnShareProposals[j] = NewProposeRnShare(message.MessageID, ρPtr, σPtr)
	})

	return rnShareProposals
}
