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
	io task.IO

	scheme pedersen.Pedersen
	index  uint64

	n, k, t uint64

	signals  map[task.MessageID]SignalGenerateRn
	rnShares map[task.MessageID]map[uint64]RnShares
	results  map[task.MessageID]Result
}

func New(scheme pedersen.Pedersen, index, n, k, t uint64, cap int) task.Task {
	return &rnger{
		io: task.NewIO(cap),

		scheme: scheme,
		index:  index,

		n: n, k: k, t: t,

		signals:  map[task.MessageID]SignalGenerateRn{},
		rnShares: map[task.MessageID]map[uint64]RnShares{},
		results:  map[task.MessageID]Result{},
	}
}

func (rnger *rnger) Channel() task.Channel {
	return rnger.io.Channel()
}

func (rnger *rnger) Run(done <-chan struct{}) {
	for {
		message, ok := rnger.io.Flush(done)
		if !ok {
			return
		}
		if message != nil {
			rnger.recv(message)
		}
	}
}

func (rnger *rnger) recv(message task.Message) {
	switch message := message.(type) {

	case SignalGenerateRn:
		rnger.signalGenerateRn(message)

	case RnShares:
		rnger.tryBuildRnShares(message)

	case ProposeRnShare:
		rnger.acceptRnShare(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (rnger *rnger) signalGenerateRn(message SignalGenerateRn) {
	// Short circuit when results have already been computed
	rnger.signals[message.MessageID] = message
	if result, ok := rnger.results[message.MessageID]; ok {
		rnger.io.Write(result)
		return
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

	rnger.io.Write(NewRnShares(message.MessageID, ρSharesMap, σSharesMap, rnger.index))
}

func (rnger *rnger) tryBuildRnShares(message RnShares) {
	// Do not continue if we have already completed the opening
	if _, ok := rnger.results[message.MessageID]; ok {
		return
	}

	// Store the received message
	if _, ok := rnger.rnShares[message.MessageID]; !ok {
		rnger.rnShares[message.MessageID] = map[uint64]RnShares{}
	}
	rnger.rnShares[message.MessageID][message.Index] = message

	// Do not continue if there is an insufficient number of messages
	if uint64(len(rnger.rnShares[message.MessageID])) < rnger.t {
		return
	}
	// Do not continue if we have not received a signal to generate a random
	// number
	if _, ok := rnger.signals[message.MessageID]; !ok {
		return
	}

	rnShareProposals := make([]ProposeRnShare, rnger.n)
	for j := uint64(1); j <= rnger.n; j++ {
		// FIXME: Implement the VSS commitments correctly.

		ρShare := shamir.New(j, rnger.scheme.SecretField().NewInField(big.NewInt(0)))
		ρt := shamir.New(j, rnger.scheme.SecretField().NewInField(big.NewInt(0)))
		ρ := vss.New([]algebra.FpElement{}, ρShare, ρt)

		σShare := shamir.New(j, rnger.scheme.SecretField().NewInField(big.NewInt(0)))
		σt := shamir.New(j, rnger.scheme.SecretField().NewInField(big.NewInt(0)))
		σ := vss.New([]algebra.FpElement{}, σShare, σt)

		for _, rnShares := range rnger.rnShares[message.MessageID] {
			// TODO: Remember, we need an additively homomorphic encryption
			// scheme for these shares to ensure that this technique works.

			ρFromShares := rnShares.Rho[j-1]
			ρ = ρ.Add(&ρFromShares)

			σFromShares := rnShares.Sigma[j-1]
			σ = σ.Add(&σFromShares)
		}

		rnShareProposals[j-1] = NewProposeRnShare(message.MessageID, ρ, σ)
	}

	rnger.acceptRnShare(rnShareProposals[rnger.index-1])

	for i, rnShareProposal := range rnShareProposals {
		if uint64(i) == rnger.index-1 {
			continue
		}
		rnger.io.Write(rnShareProposal)
	}
}

func (rnger *rnger) acceptRnShare(message ProposeRnShare) {

	result := NewResult(message.MessageID, message.Rho.Share(), message.Sigma.Share())
	rnger.results[message.MessageID] = result
	delete(rnger.signals, message.MessageID)
	delete(rnger.rnShares, message.MessageID)

	rnger.io.Write(result)
}
