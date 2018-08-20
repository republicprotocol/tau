package vm

import (
	"math/rand"
	"time"

	"github.com/republicprotocol/shamir-go"
)

// A Nonce is used to uniquely identify the generation of a secure random
// number. All players in the secure multi-party computation network must use
// the same nonce to identify the same generation.
type Nonce [32]byte

// An Address identifies a unique player within the secure multi-party
// computation network.
type Address uint64

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
	Run(done <-chan (struct{}), input <-chan RngInputMessage, output chan<- RngOutputMessage)
}

// An RngInputMessage can be passed to the Rnger as an input. It will be
// processed by the Rnger and an error will be output if the message is an
// unexpected type. No types external to this package should implement this
// interface.
type RngInputMessage interface {

	// IsRngInputMessage is a marker used to restrict RngInputMessages to types
	// that have been explicitly marked. It is never called.
	IsRngInputMessage()
}

// An RngOutputMessage can be passed from the Rnger as an output. Depending on
// the type of output message, the user must route the message to the
// appropriate Rnger in the network. See the documentation specific to each
// message for information on how to handle it. No types external to this
// package should implement this interface.
type RngOutputMessage interface {

	// IsRngOutputMessage is a marker used to restrict RngOutputMessages to
	// types that have been explicitly marked. It is never called.
	IsRngOutputMessage()
}

// A GenerateRn message signals to the Rnger that is should begin a secure
// random number generation. The secure random number that will be generated is
// identified by a nonce. The nonce must be unique and must be agreed on by all
// Rngers in the network. After receiving this message, an Rnger will produce a
// LocalRnShare for all Rngers in the network. The user must route these
// LocalRnShare messages to their respective Rngers.
type GenerateRn struct {
	Nonce
}

// IsRngInputMessage implements the RngInputMessage interface.
func (message GenerateRn) IsRngInputMessage() {
}

// A GenerateRnErr message is produced by an Rnger when an error is encountered
// during the secure random number generation algorithm. It is up to the user to
// handle this error in a way that is appropriate for the specific application.
type GenerateRnErr struct {
	Nonce
	error
}

// IsRngOutputMessage implements the RngOutputMessage interface.
func (message GenerateRnErr) IsRngOutputMessage() {
}

// A LocalRnShare message is produced by an Rnger after receiving a GenerateRn
// message. A LocalRnShare message will be produced for each Rnger in the
// network and it is up to the user to route this message to the appropriate
// Rnger. A LocalRnShare message can also be passed to an Rnger as input,
// representing the LocalRnShare messages sent to it by other Rngers in the
// network.
type LocalRnShare struct {
	Nonce
	shamir.Share

	To   Address
	From Address
}

// IsRngInputMessage implements the RngInputMessage interface.
func (message LocalRnShare) IsRngInputMessage() {
}

// IsRngOutputMessage implements the RngOutputMessage interface.
func (message LocalRnShare) IsRngOutputMessage() {
}

// A VoteToCommit message is produced by an Rnger after receiving a sufficient
// number of LocalRnShares messages, or after a secure random number generation
// has exceeded its deadline. A VoteToCommit message will be produced for each
// Rnger in the network and it is up to the user to route this message to the
// appropriate Rnger.
type VoteToCommit struct {
	Nonce

	To      Address
	From    Address
	Players []Address
}

// IsRngInputMessage implements the RngInputMessage interface.
func (message VoteToCommit) IsRngInputMessage() {
}

// IsRngOutputMessage implements the RngOutputMessage interface.
func (message VoteToCommit) IsRngOutputMessage() {
}

// A GlobalRnShare message is produced by an Rnger at the end of a successful
// secure random number generation. It is the Shamir's secret share of the
// secure random number that has been generated.
type GlobalRnShare struct {
	Nonce
	shamir.Share
}

// IsRngOutputMessage implements the RngOutputMessage interface.
func (message GlobalRnShare) IsRngOutputMessage() {
}

// A CheckDeadline message signals to the Rnger that it should clean up all
// pending random number generations that have exceeded their deadline. It is up
// to the user to determine the frequency of this message. Higher frequencies
// will result in more accurate clean up times, but slower performance.
type CheckDeadline struct {
	time.Time
}

// IsRngInputMessage implements the RngInputMessage interface.
func (message CheckDeadline) IsRngInputMessage() {
}

type LocalRnSharesTable struct {
	StartedAt time.Time
	Table     map[Address]LocalRnShare
}

type VoteTable struct {
	StartedAt time.Time
	Table     map[Address]VoteToCommit
}

type rnger struct {
	addr         Address
	n, k         int64
	outputBuffer []RngOutputMessage
	timeout      time.Duration

	localRnShares map[Nonce]LocalRnSharesTable
	votes         map[Nonce]LocalRnSharesTable
}

// NewRnger returns an Rnger that is identified as the i-th player in a network
// with n players and k threshold. The Rnger will allocate a buffer for its
// output messages and this buffer will grow indefinitely if the messages output
// from the Rnger are not consumed.
func NewRnger(addr Address, n, k int64, timeout time.Duration, bufferCap int) Rnger {
	return &rnger{
		addr:          addr,
		n:             n,
		k:             k,
		outputBuffer:  make([]RngOutputMessage, 0, bufferCap),
		localRnShares: map[Nonce]LocalRnSharesTable{},
		timeout:       timeout,
	}
}

// Run implements the Rnger interface. Calls to Rnger.Run are blocking and
// should be run in a background goroutine. It is recommended that the input and
// output channels are buffered, however it is not required.
func (rnger *rnger) Run(done <-chan (struct{}), input <-chan RngInputMessage, output chan<- RngOutputMessage) {
	for {
		var outputMessage RngOutputMessage
		var outputMaybe chan<- RngOutputMessage
		if len(rnger.outputBuffer) > 0 {
			outputMessage = rnger.outputBuffer[0]
			outputMaybe = output
		}

		select {
		case <-done:
			return

		case message, ok := <-input:
			if !ok {
				return
			}
			rnger.handleInputMessage(message)

		case outputMaybe <- outputMessage:
			rnger.outputBuffer = rnger.outputBuffer[1:]
		}
	}
}

func (rnger *rnger) handleInputMessage(message RngInputMessage) {
	switch message := message.(type) {

	case GenerateRn:
		rnger.handleGenerateRn(message)

	case LocalRnShare:
		rnger.handleLocalRnShare(message)

	case VoteToCommit:
		rnger.handleVoteToCommit(message)

	case CheckDeadline:
		rnger.handleCheckDeadline(message)
	}
}

func (rnger *rnger) handleGenerateRn(message GenerateRn) {

	// Generate a local random number and split it into shares for each player
	// in the network
	rn := rand.Uint64() % shamir.Prime
	rnShares, err := shamir.Split(rnger.n, rnger.k, rn)
	if err != nil {
		rnger.outputBuffer = append(rnger.outputBuffer, GenerateRnErr{
			Nonce: message.Nonce,
			error: err,
		})
		return
	}

	// Send each share to the appropriate player by outputting a LocalRnShare
	// message
	for j := uint64(0); j < uint64(len(rnShares)); j++ {
		localRnShare := LocalRnShare{
			Nonce: message.Nonce,
			To:    Address(j),
			From:  rnger.addr,
			Share: rnShares[j],
		}
		if Address(j) == rnger.addr {
			rnger.handleLocalRnShare(localRnShare)
			continue
		}
		rnger.outputBuffer = append(rnger.outputBuffer, localRnShare)
	}
}

func (rnger *rnger) handleLocalRnShare(message LocalRnShare) {
	if message.To != rnger.addr || message.Share.Index != uint64(rnger.addr)+1 {
		// This message is not meant for us
		return
	}

	// Initialise the map for this nonce if it does not already exist
	if _, ok := rnger.localRnShares[message.Nonce]; !ok {
		rnger.localRnShares[message.Nonce] = LocalRnSharesTable{
			StartedAt: time.Now(),
			Table:     map[Address]LocalRnShare{},
		}
	}
	rnger.localRnShares[message.Nonce].Table[message.From] = message

	// Once we have acquired a LocalRnShare from each player in the network we
	// can add them to produce our GlobalRnShare
	if int64(len(rnger.localRnShares[message.Nonce].Table)) == rnger.n {
		rnger.voteForNonce(message.Nonce)

	}
}

func (rnger *rnger) handleVoteToCommit(message VoteToCommit) {
	panic("unimplemented")
}

func (rnger *rnger) handleCheckDeadline(message CheckDeadline) {
	now := time.Now()
	for nonce := range rnger.localRnShares {
		if rnger.localRnShares[nonce].StartedAt.Add(rnger.timeout).Before(now) {
			if int64(len(rnger.localRnShares[nonce].Table)) >= rnger.k {
				rnger.voteForNonce(nonce)
			}
			delete(rnger.localRnShares, nonce)
		}
	}
}

func (rnger *rnger) voteForNonce(nonce Nonce) {
	if _, ok := rnger.localRnShares[nonce]; !ok {
		// We have already voted, or the deadline was exceeded before enough
		// shares were received to vote
		return
	}

	vote := VoteToCommit{
		Nonce:   nonce,
		From:    rnger.addr,
		Players: make([]Address, 0, rnger.k),
	}
	for addr := range rnger.localRnShares[nonce].Table {
		vote.Players = append(vote.Players, addr)
	}
	for _, player := range vote.Players {
		vote.To = player
		rnger.outputBuffer = append(rnger.outputBuffer, vote)
	}
}

func (rnger *rnger) buildGlobalRnShare(nonce Nonce) {
	globalRnShare := GlobalRnShare{
		Nonce: nonce,
		Share: shamir.Share{
			Index: uint64(rnger.addr) + 1,
			Value: 0,
		},
	}
	for _, localRngShare := range rnger.localRnShares[nonce].Table {
		globalRnShare.Share = globalRnShare.Share.Add(&localRngShare.Share)
	}
	rnger.outputBuffer = append(rnger.outputBuffer, globalRnShare)
	delete(rnger.localRnShares, nonce)
}
