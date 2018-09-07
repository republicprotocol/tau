package rng

import (
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/vm/task"
	"github.com/republicprotocol/smpc-go/core/vss"
	"github.com/republicprotocol/smpc-go/core/vss/pedersen"
	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

// A Nonce is used to uniquely identify the generation of a secure random
// number. All players in the secure multi-party computation network must use
// the same nonce to identify the same generation.
type Nonce [32]byte

// An Address identifies a unique player within the secure multi-party
// computation network.
type Address uint64

// State enumerates the possible states that the player can be in for a
// particular computation (nonce).
type State uint

const (
	// StateNil is the initial default state for a computation (nonce).
	StateNil State = iota

	// StateWaitingForLocalRnShares is the state that the computation leader is
	// in once they have sent out ProposeRn and are waiting for the compute
	// nodes to respond with their shares of their local random number.
	StateWaitingForLocalRnShares

	// StateWaitingForGlobalRnShares is the state that the compute nodes are in
	// once they have sent out the shares of their local random number and are
	// waiting to receive the shares for the global random number form the
	// computation leader.
	StateWaitingForGlobalRnShares

	// StateFinished is the state that a player is in once they have completed
	// the computation for a given nonce. This is achieved once they have
	// received their global random number shares and have added the together to
	// produce their single global random number share.
	StateFinished
)

// A VoteTable stores all votes received for a nonce and the timestamp at which
// the first vote was received.
type VoteTable struct {
	StartedAt time.Time
	Table     map[Address]VoteGlobalRnShare
}

type rnger struct {
	io         task.IO
	ioExternal task.IO

	timeout      time.Duration
	addr, leader Address
	n, k, t      uint
	ped          pedersen.Pedersen

	states        map[Nonce]State
	leaders       map[Nonce]Address
	localRnShares map[Nonce](map[Address]LocalRnShares)
}

// New returns an Rnger that is identified as the i-th player in a network with
// n players and k threshold. The Rnger will allocate a buffer for its output
// messages and this buffer will grow indefinitely if the messages output from
// the Rnger are not consumed.
func New(r, w buffer.ReaderWriter, timeout time.Duration, addr, leader Address, n, k, t uint, ped pedersen.Pedersen, cap int) task.Task {
	return &rnger{
		io:         task.NewIO(buffer.New(cap), r.Reader(), w.Writer()),
		ioExternal: task.NewIO(buffer.New(cap), w.Reader(), r.Writer()),

		timeout: timeout,
		addr:    addr,
		leader:  leader,
		n:       n,
		k:       k,
		t:       t,
		ped:     ped,

		states:        map[Nonce]State{},
		leaders:       map[Nonce]Address{},
		localRnShares: map[Nonce](map[Address]LocalRnShares){},
	}
}

func (rnger *rnger) IO() task.IO {
	return rnger.ioExternal
}

// Run implements the Rnger interface. Calls to Rnger.Run are blocking and
// should be run in a background goroutine. It is recommended that the input and
// output channels are buffered, however it is not required.
func (rnger *rnger) Run(done <-chan struct{}) {
	defer log.Printf("[info] (rng) terminating")

	for {
		ok := task.Select(
			done,
			rnger.recvMessage,
			rnger.io,
		)
		if !ok {
			return
		}
	}
}

func (rnger *rnger) isLeader(nonce Nonce) bool {
	return rnger.leaders[nonce] == rnger.addr
}

func (rnger *rnger) recvMessage(message buffer.Message) {
	switch message := message.(type) {
	case Nominate:
		// log.Printf("[debug] player %v received message of type %T", rnger.addr, message)
		rnger.handleNominate(message)

	case GenerateRn:
		// log.Printf("[debug] player %v received message of type %T", rnger.addr, message)
		rnger.handleGenerateRn(message)

	case LocalRnShares:
		// log.Printf("[debug] player %v received message of type %T", rnger.addr, message)
		rnger.handleLocalRnShares(message)

	case ProposeRn:
		// log.Printf("[debug] player %v received message of type %T", rnger.addr, message)
		rnger.handleProposeRn(message)

	case ProposeGlobalRnShare:
		// log.Printf("[debug] player %v received message of type %T", rnger.addr, message)
		rnger.handleProposeGlobalRnShare(message)

	case VoteGlobalRnShare:
		// log.Printf("[debug] player %v received message of type %T", rnger.addr, message)
		rnger.handleVoteGlobalRnShare(message)

		// case Vote:
		// 	// log.Printf("[debug] player %v received message of type %T", rnger.addr, message)
		// 	rnger.handleVote(message)

		// case CheckDeadline:
		// 	rnger.handleCheckDeadline(message)
	}
}

func (rnger *rnger) handleNominate(message Nominate) {
	// TODO: Logic for whether or not to accept a nomination.
	rnger.leader = message.Leader
}

func (rnger *rnger) handleGenerateRn(message GenerateRn) {
	// Verify the current state of the Rnger
	if rnger.leader != rnger.addr {
		rnger.io.Send(NewErr(message.Nonce, errors.New("cannot accept GenerateRn: must be the leader")))
		return
	}
	if rnger.states[message.Nonce] != StateNil {
		rnger.io.Send(NewErr(message.Nonce, errors.New("cannot accept GenerateRn: not in initial state")))
		return
	}

	// Set rnger to leader for this nonce
	rnger.leaders[message.Nonce] = rnger.addr

	// Send a ProposeRn message to every other Rnger in the network
	for j := uint(0); j < rnger.n; j++ {
		rnger.io.Send(NewProposeRn(message.Nonce, Address(j), rnger.addr))
	}

	// Transition to a new state
	rnger.states[message.Nonce] = StateWaitingForLocalRnShares
}

func (rnger *rnger) handleLocalRnShares(message LocalRnShares) {
	// Verify the current state of the Rnger
	if !rnger.isLeader(message.Nonce) {
		rnger.io.Send(NewErr(message.Nonce, errors.New("cannot accept LocalRnShares: must be the leader")))
		return
	}
	if rnger.states[message.Nonce] != StateWaitingForLocalRnShares {
		// rnger.io.Send(NewErr(message.Nonce, errors.New("cannot accept LocalRnShares: not waiting for LocalRnShares")))
		return
	}

	// TODO: Verify that the shares are well formed.
	if _, ok := rnger.localRnShares[message.Nonce]; !ok {
		rnger.localRnShares[message.Nonce] = map[Address]LocalRnShares{}
	}
	rnger.localRnShares[message.Nonce][message.From] = message

	// Check whether or not this Rnger has received enough LocalRnShares to
	// securely build a GlobalRnShare for every Rnger in the network
	if len(rnger.localRnShares[message.Nonce]) <= int(rnger.t) {
		return
	}

	globalRnShares := rnger.buildProposeGlobalRnShares(message.Nonce)
	for j := uint(0); j < rnger.n; j++ {
		rnger.io.Send(globalRnShares[j])
	}

	// Progress state
	rnger.states[message.Nonce] = StateFinished
}

func (rnger *rnger) handleProposeRn(message ProposeRn) {
	// TODO: Logic for whether or not to accept a leader for a nonce.
	if _, ok := rnger.leaders[message.Nonce]; !ok {
		rnger.leaders[message.Nonce] = message.From
	}
	// TODO: Protection (if wanted) against multiple proposals from the same
	// leader for the same nonce
	if message.From != rnger.leaders[message.Nonce] {
		rnger.io.Send(NewErr(message.Nonce, errors.New("cannot accept ProposeRn: already have a leader")))
		return
	}

	// Verify that the internal state is correct
	if rnger.states[message.Nonce] != StateNil && !rnger.isLeader(message.Nonce) {
		rnger.io.Send(NewErr(message.Nonce, errors.New("cannot accept ProposeRn: not in initial state")))
		return
	}

	rn := rnger.ped.SecretField().Random()
	rhoRnShares := vss.Share(&rnger.ped, rn, uint64(rnger.n), uint64(rnger.k))
	sigmaRnShares := vss.Share(&rnger.ped, rn, uint64(rnger.n), uint64(rnger.k/2))

	rhoShares := ShareMap{}
	sigmaShares := ShareMap{}
	for i := range rhoRnShares {
		// log.Printf("[debug] <replica %v>: sending share %v", rnger.addr, share.SShare)
		rhoShares[Address(i)] = rhoRnShares[i]
		sigmaShares[Address(i)] = sigmaRnShares[i]
	}

	localRnShares := LocalRnShares{
		Nonce:       message.Nonce,
		To:          rnger.leader,
		From:        rnger.addr,
		RhoShares:   rhoShares,
		SigmaShares: sigmaShares,
	}
	rnger.io.Send(localRnShares)

	// Progress state if rnger is not the leader for this nonce
	if !rnger.isLeader(message.Nonce) {
		rnger.states[message.Nonce] = StateWaitingForGlobalRnShares
	}
}

func (rnger *rnger) handleProposeGlobalRnShare(message ProposeGlobalRnShare) {

	globalRnShare := GlobalRnShare{
		Nonce:      message.Nonce,
		RhoShare:   shamir.New(uint64(rnger.addr)+1, rnger.ped.SecretField().NewInField(big.NewInt(0))),
		SigmaShare: shamir.New(uint64(rnger.addr)+1, rnger.ped.SecretField().NewInField(big.NewInt(0))),
		From:       rnger.addr,
	}
	for addr, share := range message.RhoShares {
		// log.Printf("[debug] <replica %v>: received share %v", rnger.addr, share.SShare)

		// Check that the share is correct
		if !vss.Verify(&rnger.ped, share) {
			// log.Println("[debug] replica received a malformed share")
			rnger.io.Send(NewErr(message.Nonce, errors.New("cannot accept incorrect share")))
			return
		}

		rhoShare := message.RhoShares[addr]
		sigmaShare := message.RhoShares[addr]

		globalRnShare.RhoShare = globalRnShare.RhoShare.Add(rhoShare.Share())
		globalRnShare.SigmaShare = globalRnShare.SigmaShare.Add(sigmaShare.Share())
	}
	// log.Printf("[debug] <replica %v>: final share %v", rnger.addr, globalRnShare.Share)

	rnger.io.Send(globalRnShare)

	// Progress state
	rnger.states[message.Nonce] = StateFinished
}

func (rnger *rnger) handleVoteGlobalRnShare(message VoteGlobalRnShare) {
}

func (rnger *rnger) buildProposeGlobalRnShares(nonce Nonce) []ProposeGlobalRnShare {
	globalRnShares := make([]ProposeGlobalRnShare, rnger.n)

	for j := uint(0); j < rnger.n; j++ {
		rhoShares := ShareMap{}
		sigmaShares := ShareMap{}

		for addr, shareMapFromAddr := range rnger.localRnShares[nonce] {
			rhoShares[addr] = shareMapFromAddr.RhoShares[Address(j)]
			sigmaShares[addr] = shareMapFromAddr.SigmaShares[Address(j)]
		}

		globalRnShares[j] = NewProposeGlobalRnShare(nonce, Address(j), rnger.addr, rhoShares, sigmaShares)
	}

	return globalRnShares
}
