package rng

import (
	"errors"
	"math/rand"
	"time"

	shamir "github.com/republicprotocol/shamir-go"
)

// A Nonce is used to uniquely identify the generation of a secure random
// number. All players in the secure multi-party computation network must use
// the same nonce to identify the same generation.
type Nonce [32]byte

// An Address identifies a unique player within the secure multi-party
// computation network.
type Address uint64

type State uint

const (
	StateNil State = iota
	StateWaitingForLocalRnShares
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
	Run(done <-chan (struct{}), input <-chan InputMessage, output chan<- OutputMessage)
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

	sendBuffer    []OutputMessage
	sendBufferCap int

	states        map[Nonce]State
	localRnShares map[Nonce](map[Address]ShareMap)
}

// NewRnger returns an Rnger that is identified as the i-th player in a network
// with n players and k threshold. The Rnger will allocate a buffer for its
// output messages and this buffer will grow indefinitely if the messages output
// from the Rnger are not consumed.
func NewRnger(timeout time.Duration, addr, leader Address, n, k, t uint, bufferCap int) Rnger {
	return &rnger{
		timeout: timeout,
		addr:    addr,
		leader:  leader,
		n:       n,
		k:       k,
		t:       t,

		sendBuffer:    make([]OutputMessage, 0, bufferCap),
		sendBufferCap: bufferCap,

		states:        map[Nonce]State{},
		localRnShares: map[Nonce](map[Address]ShareMap){},
	}
}

// Run implements the Rnger interface. Calls to Rnger.Run are blocking and
// should be run in a background goroutine. It is recommended that the input and
// output channels are buffered, however it is not required.
func (rnger *rnger) Run(done <-chan (struct{}), input <-chan InputMessage, output chan<- OutputMessage) {
	for {
		var outputMessage OutputMessage
		var outputMaybe chan<- OutputMessage
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

func (rnger *rnger) isLeader() bool {
	return rnger.leader == rnger.addr
}

func (rnger *rnger) sendMessage(message OutputMessage) {
	rnger.sendBuffer = append(rnger.sendBuffer, message)
}

func (rnger *rnger) recvMessage(message InputMessage) {
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
	rnger.leader = message.Leader
}

func (rnger *rnger) handleGenerateRn(message GenerateRn) {
	// Verify the current state of the Rnger
	if !rnger.isLeader() {
		rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot generate random number: must be the leader")))
		return
	}
	if rnger.states[message.Nonce] != StateNil {
		return
	}

	// Send a ProposeRn message to every other Rnger in the network
	for j := uint(0); j < rnger.n; j++ {
		rnger.sendMessage(NewProposeRn(message.Nonce, Address(j), rnger.addr))
	}

	// Transition to a new state
	rnger.states[message.Nonce] = StateWaitingForLocalRnShares
}

func (rnger *rnger) handleLocalRnShares(message LocalRnShares) {
	// Verify the current state of the Rnger
	if !rnger.isLeader() {
		rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot accept local random number shares: must be the leader")))
		return
	}
	if rnger.states[message.Nonce] != StateWaitingForLocalRnShares {
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

	// Transition to a new state
	rnger.states[message.Nonce] = StateFinished
}

func (rnger *rnger) handleProposeRn(message ProposeRn) {
	// Check that the message came from the leader
	if message.From != rnger.leader {
		rnger.sendMessage(NewErr(message.Nonce, errors.New("cannot accept propose request: must come from the leader")))
		return
	}

	rn := rand.Uint64() % shamir.Prime
	rnShares, err := shamir.Split(int64(rnger.n), int64(rnger.k), rn)
	if err != nil {
		rnger.sendMessage(NewErr(message.Nonce, err))
		return
	}

	shares := ShareMap{}
	for i, share := range rnShares {
		shares[Address(i)] = share
	}

	localRnShares := LocalRnShares{
		Nonce:  message.Nonce,
		To:     rnger.leader,
		From:   rnger.addr,
		Shares: shares,
	}
	rnger.sendMessage(localRnShares)
}

func (rnger *rnger) handleProposeGlobalRnShare(message ProposeGlobalRnShare) {

	globalRnShare := GlobalRnShare{
		Nonce: message.Nonce,
		Share: shamir.Share{
			Index: uint64(rnger.addr) + 1,
			Value: 0,
		},
		From: rnger.addr,
	}
	for _, share := range message.Shares {
		globalRnShare.Share = globalRnShare.Share.Add(&share)
	}

	rnger.sendMessage(globalRnShare)
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
