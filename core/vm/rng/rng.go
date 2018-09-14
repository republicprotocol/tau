package rng

import (
	"fmt"
	"math/big"

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

	signals  map[task.MessageID]SignalGenerateRn
	rnShares map[task.MessageID]map[uint64]RnShares
	results  map[task.MessageID]Result
}

func New(scheme pedersen.Pedersen, index, n, k, t uint64, cap int) task.Task {
	return task.New(task.NewIO(cap), newRnger(scheme, index, n, k, t))
}

func newRnger(scheme pedersen.Pedersen, index, n, k, t uint64) *rnger {
	return &rnger{
		scheme: scheme,
		index:  index,

		n: n, k: k, t: t,

		signals:  map[task.MessageID]SignalGenerateRn{},
		rnShares: map[task.MessageID]map[uint64]RnShares{},
		results:  map[task.MessageID]Result{},
	}
}

func (rnger *rnger) Reduce(message task.Message) task.Message {
	switch message := message.(type) {

	case SignalGenerateRn:
		return rnger.signalGenerateRn(message)

	case RnShares:
		return rnger.tryBuildRnShareProposals(message)

	case ProposeRnShare:
		return rnger.acceptRnShare(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (rnger *rnger) signalGenerateRn(message SignalGenerateRn) task.Message {
	// Short circuit when results have already been computed
	rnger.signals[message.MessageID] = message
	if result, ok := rnger.results[message.MessageID]; ok {
		return result
	}

	rn := rnger.scheme.SecretField().Random()

	// Generate k threshold shares for the random number
	ρShares := vss.Share(&rnger.scheme, rn, rnger.n, rnger.k)
	ρSharesMap := make(map[uint64]vss.VShare, rnger.n)
	for _, ρShare := range ρShares {
		share := ρShare.Share()
		ρSharesMap[share.Index()] = ρShare
	}

	// Generate k/2 threshold shares for the same random number
	σShares := vss.Share(&rnger.scheme, rn, rnger.n, rnger.k/2)
	σSharesMap := make(map[uint64]vss.VShare, rnger.n)
	for _, σShare := range σShares {
		share := σShare.Share()
		σSharesMap[share.Index()] = σShare
	}

	return NewRnShares(message.MessageID, ρSharesMap, σSharesMap, rnger.index)
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
	if _, ok := rnger.signals[message.MessageID]; !ok {
		return nil
	}

	messages := rnger.buildRnShareProposals(message.MessageID)
	messages[rnger.index-1] = rnger.acceptRnShare(messages[rnger.index-1].(ProposeRnShare))

	return task.NewMessageBatch(messages...)
}

func (rnger *rnger) acceptRnShare(message ProposeRnShare) task.Message {

	result := NewResult(message.MessageID, message.Rho.Share(), message.Sigma.Share())
	rnger.results[message.MessageID] = result
	delete(rnger.signals, message.MessageID)
	delete(rnger.rnShares, message.MessageID)

	return result
}

func (rnger *rnger) buildRnShareProposals(messageID task.MessageID) []task.Message {

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

	for j := uint64(1); j <= rnger.n; j++ {

		ρShare := shamir.New(j, zero)
		ρt := shamir.New(j, zero)
		ρ := vss.New(ρCommitments, ρShare, ρt)

		σShare := shamir.New(j, rnger.scheme.SecretField().NewInField(big.NewInt(0)))
		σt := shamir.New(j, rnger.scheme.SecretField().NewInField(big.NewInt(0)))
		σ := vss.New(σCommitments, σShare, σt)

		for _, rnShares := range rnger.rnShares[messageID] {
			// TODO: Remember, we need an additively homomorphic encryption
			// scheme for these shares to ensure that this technique works.

			ρFromShares := rnShares.Rho[j]
			ρ = ρ.Add(&ρFromShares)

			σFromShares := rnShares.Sigma[j]
			σ = σ.Add(&σFromShares)
		}

		rnShareProposals[j-1] = NewProposeRnShare(messageID, ρ, σ)
	}

	return rnShareProposals
}
