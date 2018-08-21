package vm

import (
	"errors"
	"log"
	"math/rand"
	"sort"
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
	timeout      time.Duration
	addr         Address
	n, k         int64
	outputBuffer []RngOutputMessage

	localRnShares map[Nonce]LocalRnSharesTable
	votes         map[Nonce]VoteTable

	hasVoted              map[Nonce]bool
	hasBuiltGlobalRnShare map[Nonce]bool
}

// NewRnger returns an Rnger that is identified as the i-th player in a network
// with n players and k threshold. The Rnger will allocate a buffer for its
// output messages and this buffer will grow indefinitely if the messages output
// from the Rnger are not consumed.
func NewRnger(timeout time.Duration, addr Address, n, k int64, bufferCap int) Rnger {
	return &rnger{
		timeout:      timeout,
		addr:         addr,
		n:            n,
		k:            k,
		outputBuffer: make([]RngOutputMessage, 0, bufferCap),

		localRnShares: map[Nonce]LocalRnSharesTable{},
		votes:         map[Nonce]VoteTable{},

		hasVoted:              map[Nonce]bool{},
		hasBuiltGlobalRnShare: map[Nonce]bool{},
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
			log.Println("SELF SENDING =>", rnger.addr)
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
	// can produce a VoteToCommit
	if int64(len(rnger.localRnShares[message.Nonce].Table)) == rnger.n {
		rnger.voteForNonce(message.Nonce)
	}
}

func (rnger *rnger) handleVoteToCommit(message VoteToCommit) {
	if message.To != rnger.addr {
		// This message is not meant for us
		return
	}

	sort.Slice(message.Players, func(i, j int) bool {
		return message.Players[i] < message.Players[j]
	})

	// Initialise the map for this nonce if it does not already exist
	if _, ok := rnger.votes[message.Nonce]; !ok {
		rnger.votes[message.Nonce] = VoteTable{
			StartedAt: time.Now(),
			Table:     map[Address]VoteToCommit{},
		}
	}
	rnger.votes[message.Nonce].Table[message.From] = message

	// Once we have acquired a VoteToCommit from each player in the network we
	// can produce a GlobalRnShare
	if int64(len(rnger.votes[message.Nonce].Table)) == rnger.n {
		rnger.buildGlobalRnShare(message.Nonce)
	}
}

func (rnger *rnger) handleCheckDeadline(message CheckDeadline) {
	// now := time.Now()
	// for nonce := range rnger.localRnShares {
	// 	if rnger.localRnShares[nonce].StartedAt.Add(rnger.timeout).Before(now) {
	// 		if int64(len(rnger.localRnShares[nonce].Table)) >= rnger.k {
	// 			rnger.voteForNonce(nonce)
	// 		}
	// 	}
	// }
	// for nonce := range rnger.votes {
	// 	if rnger.votes[nonce].StartedAt.Add(rnger.timeout).Before(now) {
	// 		if int64(len(rnger.votes[nonce].Table)) >= rnger.k {
	// 			rnger.buildGlobalRnShare(nonce)
	// 		}
	// 	}
	// }
}

func (rnger *rnger) voteForNonce(nonce Nonce) {
	// Prevent broadcasting a vote more than once
	if rnger.hasVoted[nonce] {
		return
	}
	rnger.hasVoted[nonce] = true

	vote := VoteToCommit{
		Nonce:   nonce,
		From:    rnger.addr,
		Players: make([]Address, 0, rnger.k),
	}
	for addr := range rnger.localRnShares[nonce].Table {
		vote.Players = append(vote.Players, addr)
	}
	for _, player := range vote.Players {
		if player == rnger.addr {
			log.Println("TRUE FOR PLAYER =>", rnger.addr)
			vote.To = rnger.addr
			rnger.handleVoteToCommit(vote)
			continue
		}
		vote.To = player
		rnger.outputBuffer = append(rnger.outputBuffer, vote)
	}

}

func (rnger *rnger) buildGlobalRnShare(nonce Nonce) {
	// Prevent building a GlobalRnShare more than once
	// if rnger.hasBuiltGlobalRnShare[nonce] {
	// 	return
	// }
	// rnger.hasBuiltGlobalRnShare[nonce] = true

	votes := make([]VoteToCommit, rnger.n)
	for _, vote := range rnger.votes[nonce].Table {
		votes = append(votes, vote)
	}

	players, err := PickPlayers(votes, rnger.k)
	if err != nil {
		rnger.outputBuffer = append(rnger.outputBuffer, GenerateRnErr{
			Nonce: nonce,
			error: err,
		})
		return
	}

	globalRnShare := GlobalRnShare{
		Nonce: nonce,
		Share: shamir.Share{
			Index: uint64(rnger.addr) + 1,
			Value: 0,
		},
	}

	for _, player := range players {
		localRnShare := rnger.localRnShares[nonce].Table[player]
		globalRnShare.Share = globalRnShare.Share.Add(&localRnShare.Share)
	}
	rnger.outputBuffer = append(rnger.outputBuffer, globalRnShare)
}

func PickPlayers(votes []VoteToCommit, k int64) ([]Address, error) {
	playerList, err := potentialPlayers(votes, k)
	if err != nil {
		return nil, err
	}

	// Convert lists of players into constant-time lookup sets
	voteSets := make([]map[Address](struct{}), 0, len(votes))
	for _, vote := range votes {
		set := map[Address](struct{}){}
		for _, addr := range vote.Players {
			set[addr] = struct{}{}
		}
		voteSets = append(voteSets, set)
	}

	// Check all subsets of size at least k for one that is in at least k votes
	max := len(playerList)
	currentPlayerList := make([]Address, 0, max)
	mask := 1<<uint(max) - 1
	for ; mask >= 1<<uint(k)-1; mask-- {
		if int64(bitCount(mask)) < k {
			continue
		}

		// Extract the subset based on the bit mask
		currentPlayerList = currentPlayerList[0:0]
		for i, m := 0, mask; m > 0; i, m = i+1, m/2 {
			if m%2 == 1 {
				currentPlayerList = append(currentPlayerList, playerList[max-i-1])
			}
		}

		subsetHits := int64(0)
		for _, voteSet := range voteSets {
			if containsAddressSubset(currentPlayerList, voteSet) {
				subsetHits++
			}
		}

		if subsetHits >= k {
			return currentPlayerList, nil
		}
	}

	return nil, errors.New("insufficient players to form a majority")
}

func potentialPlayers(votes []VoteToCommit, k int64) ([]Address, error) {
	playerCounts := map[Address]int64{}

	// Count the number of times a player is in a vote
	for _, vote := range votes {
		for _, addr := range vote.Players {
			playerCounts[addr]++
		}
	}

	// Remove players that are not in enough votes
	for key, value := range playerCounts {
		if value < k {
			delete(playerCounts, key)
		}
	}

	max := len(playerCounts)
	if int64(max) < k {
		// Not enough players to proceed
		return nil, errors.New("insufficient players to form a majority")
	}

	// Extract the potential players from the map
	playerList := make([]Address, 0, max)
	for addr := range playerCounts {
		playerList = append(playerList, addr)
	}

	// Sort the list so that picking is deterministic
	sort.Slice(playerList, func(i, j int) bool {
		return playerList[i] < playerList[j]
	})

	return playerList, nil
}

func bitCount(n int) (count int) {
	for n > 0 {
		if n%2 == 1 {
			count++
		}
		n /= 2
	}
	return
}

func containsAddressSubset(subset []Address, set map[Address](struct{})) bool {
	for _, addr := range subset {
		if _, ok := set[addr]; !ok {
			return false
		}
	}
	return true
}
