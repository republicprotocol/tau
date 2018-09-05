package rng

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/republicprotocol/smpc-go/core/buffer"
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

// Rnger generates secure random numbers running a secure multi-party
// computation with other Rngers in its network. After generating a secure
// random number, each Rnger in the network will have a Shamir's secret share of
// a global random number. This global random number cannot be opened unless
// some threshold of malicious Rngers collude.
type Rnger interface {

	// Run the Rnger. It will read messages from the input channel and write
	// messages to the output channel. Depending on the type of output message,
	// the user must route the message to the appropriate Rnger in the network.
	// Closing the done channel will stop the Rnger.
	Run(done <-chan (struct{}), input <-chan buffer.Message, output chan<- buffer.Message)
}

// A VoteTable stores all votes received for a nonce and the timestamp at which
// the first vote was received.
type VoteTable struct {
	StartedAt time.Time
	Table     map[Address]VoteGlobalRnShare
}

type rnger struct {
	timeout      time.Duration
	addr, leader Address
	n, k, t      uint
	ped          pedersen.Pedersen

	sendBuffer    []buffer.Message
	sendBufferCap int

	states        map[Nonce]State
	leaders       map[Nonce]Address
	localRnShares map[Nonce](map[Address]ShareMap)
	addresses     []uint64
}

// NewRnger returns an Rnger that is identified as the i-th player in a network
// with n players and k threshold. The Rnger will allocate a buffer for its
// output messages and this buffer will grow indefinitely if the messages output
// from the Rnger are not consumed.
func NewRnger(timeout time.Duration, addr, leader Address, n, k, t uint, ped pedersen.Pedersen, bufferCap int) Rnger {
	addresses := make([]uint64, n)
	for i := range addresses {
		addresses[i] = uint64(i + 1)
	}

	return &rnger{
		timeout: timeout,
		addr:    addr,
		leader:  leader,
		n:       n,
		k:       k,
		t:       t,
		ped:     ped,

		sendBuffer:    make([]buffer.Message, 0, bufferCap),
		sendBufferCap: bufferCap,

		states:        map[Nonce]State{},
		leaders:       map[Nonce]Address{},
		localRnShares: map[Nonce](map[Address]ShareMap){},
		addresses:     addresses,
	}
}

// Run implements the Rnger interface. Calls to Rnger.Run are blocking and
// should be run in a background goroutine. It is recommended that the input and
// output channels are buffered, however it is not required.
func (rnger *rnger) Run(done <-chan (struct{}), input <-chan buffer.Message, output chan<- buffer.Message) {
	for {
		var outputMessage buffer.Message
		var outputMaybe chan<- buffer.Message
		if len(rnger.sendBuffer) > 0 {
			outputMessage = rnger.sendBuffer[0]
			outputMaybe = output
		}

		select {
		case <-done:
			return

		case message, ok := <-input:
			if !ok {
				return
			}
			rnger.recvMessage(message)

		case outputMaybe <- outputMessage:
			rnger.sendBuffer = rnger.sendBuffer[1:]
		}
	}
}

func (rnger *rnger) isLeader(nonce Nonce) bool {
	return rnger.leaders[nonce] == rnger.addr
}

func (rnger *rnger) sendMessage(message buffer.Message) {
	rnger.sendBuffer = append(rnger.sendBuffer, message)
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
		rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot accept GenerateRn: must be the leader")))
		return
	}
	if rnger.states[message.Nonce] != StateNil {
		rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot accept GenerateRn: not in initial state")))
		return
	}

	// Set rnger to leader for this nonce
	rnger.leaders[message.Nonce] = rnger.addr

	// Send a ProposeRn message to every other Rnger in the network
	for j := uint(0); j < rnger.n; j++ {
		rnger.sendMessage(NewProposeRn(message.Nonce, Address(j), rnger.addr))
	}

	// Transition to a new state
	rnger.states[message.Nonce] = StateWaitingForLocalRnShares
}

func (rnger *rnger) handleLocalRnShares(message LocalRnShares) {
	// Verify the current state of the Rnger
	if !rnger.isLeader(message.Nonce) {
		rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot accept LocalRnShares: must be the leader")))
		return
	}
	if rnger.states[message.Nonce] != StateWaitingForLocalRnShares {
		// rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot accept LocalRnShares: not waiting for LocalRnShares")))
		return
	}

	// TODO: Verify that the shares are well formed.
	if _, ok := rnger.localRnShares[message.Nonce]; !ok {
		rnger.localRnShares[message.Nonce] = map[Address]ShareMap{}
	}
	rnger.localRnShares[message.Nonce][message.From] = message.Shares

	// Check whether or not this Rnger has received enough LocalRnShares to
	// securely build a GlobalRnShare for every Rnger in the network
	if len(rnger.localRnShares[message.Nonce]) <= int(rnger.t) {
		return
	}

	globalRnShares := rnger.buildProposeGlobalRnShares(message.Nonce)
	for j := uint(0); j < rnger.n; j++ {
		rnger.sendMessage(globalRnShares[j])
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
		rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot accept ProposeRn: already have a leader")))
		return
	}

	// Verify that the internal state is correct
	if rnger.states[message.Nonce] != StateNil && !rnger.isLeader(message.Nonce) {
		rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot accept ProposeRn: not in initial state")))
		return
	}

	rn, err := rand.Int(rand.Reader, rnger.ped.SubgroupOrder())
	if err != nil {
		panic(err)
	}
	rnShares := vss.Share(&rnger.ped, rn, rnger.k, rnger.addresses)

	shares := ShareMap{}
	for i, share := range rnShares {
		// log.Printf("[debug] <replica %v>: sending share %v", rnger.addr, share.SShare)
		shares[Address(i)] = share
	}

	localRnShares := LocalRnShares{
		Nonce:  message.Nonce,
		To:     rnger.leader,
		From:   rnger.addr,
		Shares: shares,
	}
	rnger.sendMessage(localRnShares)

	// Progress state if rnger is not the leader for this nonce
	if !rnger.isLeader(message.Nonce) {
		rnger.states[message.Nonce] = StateWaitingForGlobalRnShares
	}
}

func (rnger *rnger) handleProposeGlobalRnShare(message ProposeGlobalRnShare) {

	globalRnShare := GlobalRnShare{
		Nonce: message.Nonce,
		Share: shamir.Share{
			Index: uint64(rnger.addr) + 1,
			Value: big.NewInt(0),
		},
		From: rnger.addr,
	}
	for _, share := range message.Shares {
		// log.Printf("[debug] <replica %v>: received share %v", rnger.addr, share.SShare)

		// Check that the share is correct
		if !vss.Verify(&rnger.ped, share) {
			// log.Println("[debug] replica received a malformed share")
			rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot accept incorrect share")))
			return
		}

		if globalRnShare.Share.Index != share.SShare.Index {
			panic("share indices are not the same")
		}
		globalRnShare.Share.Value.Add(globalRnShare.Share.Value, share.SShare.Value)
		globalRnShare.Share.Value.Mod(globalRnShare.Share.Value, rnger.ped.SubgroupOrder())
	}
	// log.Printf("[debug] <replica %v>: final share %v", rnger.addr, globalRnShare.Share)

	rnger.sendMessage(globalRnShare)

	// Progress state
	rnger.states[message.Nonce] = StateFinished
}

func (rnger *rnger) handleVoteGlobalRnShare(message VoteGlobalRnShare) {
}

func (rnger *rnger) buildProposeGlobalRnShares(nonce Nonce) []ProposeGlobalRnShare {
	globalRnShares := make([]ProposeGlobalRnShare, rnger.n)

	for j := uint(0); j < rnger.n; j++ {
		shares := ShareMap{}

		for addr, shareMapFromAddr := range rnger.localRnShares[nonce] {
			shares[addr] = shareMapFromAddr[Address(j)]
		}

		globalRnShares[j] = NewProposeGlobalRnShare(nonce, Address(j), rnger.addr, shares)
	}

	return globalRnShares
}
