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
		rnger.handleProposeRn(message)

	case ProposeGlobalRnShare:
		rnger.handleProposeGlobalRnShare(message)

	case VoteGlobalRnShare:
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

// func (rnger *rnger) handleVote(message Vote) {
// 	if message.To != rnger.addr {
// 		// This message is not meant for us
// 		return
// 	}

// 	sort.Slice(message.Players, func(i, j int) bool {
// 		return message.Players[i] < message.Players[j]
// 	})
// 	// log.Printf("[debug] player %v received vote = (%v) %v", rnger.addr, len(message.Players), message.Players)

// 	// Initialise the map for this nonce if it does not already exist
// 	if _, ok := rnger.votes[message.Nonce]; !ok {
// 		rnger.votes[message.Nonce] = VoteTable{
// 			StartedAt: time.Now(),
// 			Table:     map[Address]Vote{},
// 		}
// 	}
// 	rnger.votes[message.Nonce].Table[message.From] = message

// 	// Once we have acquired a Vote from each player in the network we
// 	// can produce a GlobalRnShare
// 	if int64(len(rnger.votes[message.Nonce].Table)) == rnger.n {
// 		rnger.buildGlobalRnShare(message.Nonce)
// 	}
// }

// func (rnger *rnger) handleCheckDeadline(message CheckDeadline) {
// 	now := time.Now()
// 	for nonce := range rnger.localRnShares {
// 		if rnger.localRnShares[nonce].StartedAt.Add(rnger.timeout).Before(now) {
// 			rnger.vote(nonce)
// 		}
// 	}
// 	for nonce := range rnger.votes {
// 		if rnger.votes[nonce].StartedAt.Add(rnger.timeout * 20).Before(now) {
// 			rnger.buildGlobalRnShare(nonce)
// 		}
// 	}
// }

// func (rnger *rnger) vote(nonce Nonce) {
// 	// Prevent broadcasting a vote more than once
// 	if rnger.hasVoted[nonce] {
// 		return
// 	}
// 	rnger.hasVoted[nonce] = true

// 	vote := Vote{
// 		Nonce:   nonce,
// 		From:    rnger.addr,
// 		Players: make([]Address, 0, rnger.n),
// 	}
// 	for addr := range rnger.localRnShares[nonce].Table {
// 		vote.Players = append(vote.Players, addr)
// 	}
// 	for j := int64(0); j < rnger.n; j++ {
// 		if Address(j) == rnger.addr {
// 			continue
// 		}
// 		vote.To = Address(j)
// 		rnger.sendMessage(vote)
// 	}

// 	vote.To = rnger.addr
// 	rnger.handleVote(vote)

// 	// Time events from when we are ready
// 	table := rnger.votes[nonce]
// 	table.StartedAt = time.Now()
// 	rnger.votes[nonce] = table
// }

// func (rnger *rnger) buildGlobalRnShare(nonce Nonce) {
// 	// Prevent building a GlobalRnShare more than once
// 	if rnger.hasBuiltGlobalRnShare[nonce] {
// 		return
// 	}
// 	rnger.hasBuiltGlobalRnShare[nonce] = true

// 	votes := make([]Vote, 0, rnger.n)
// 	for _, vote := range rnger.votes[nonce].Table {
// 		votes = append(votes, vote)
// 	}

// 	players, err := PickPlayers(votes, rnger.k)
// 	if err != nil {
// 		log.Printf("[error] player %v: %v", rnger.addr, err)
// 		rnger.sendMessage(GenerateRnErr{
// 			Nonce: nonce,
// 			error: err,
// 		})
// 		return
// 	}

// 	globalRnShare := GlobalRnShare{
// 		Nonce: nonce,
// 		Share: shamir.Share{
// 			Index: uint64(rnger.addr) + 1,
// 			Value: 0,
// 		},
// 		Players: players,
// 	}
// 	for _, player := range players {
// 		localRnShare, ok := rnger.localRnShares[nonce].Table[player]
// 		if !ok {
// 			log.Printf("[error] player %v: not invited to the party", rnger.addr)
// 			rnger.sendMessage(GenerateRnErr{
// 				Nonce: nonce,
// 				error: errors.New("not invited to the party"),
// 			})
// 			return
// 		}
// 		globalRnShare.Share = globalRnShare.Share.Add(&localRnShare.Share)
// 	}
// 	// log.Printf("[debug] player %v building = (%v) %v", rnger.addr, len(players), players)

// 	log.Printf("[debug] player %v: global random number", rnger.addr)
// 	rnger.sendMessage(globalRnShare)
// }

// // FIXME: Used only for debugging.
// var debugMu = new(sync.Mutex)

// // PickPlayers finds a subset of the players that are contained in all of the
// // votes. This subset of players will determine the shares that are added
// // together to contruct the global random number. PickPlayers will return an
// // error if no subset of size at least k exists.
// func PickPlayers(votes []Vote, k int64) ([]Address, error) {
// 	playerList, err := potentialPlayers(votes, k)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Convert lists of players into constant-time lookup sets
// 	rngerAddr := Address(696969)
// 	voteSets := make([]map[Address](struct{}), 0, len(votes))
// 	for _, vote := range votes {
// 		rngerAddr = vote.To
// 		set := map[Address](struct{}){}
// 		for _, addr := range vote.Players {
// 			set[addr] = struct{}{}
// 		}
// 		voteSets = append(voteSets, set)
// 	}

// 	debugMu.Lock()
// 	log.Printf("[debug] player %v has voting set =\n", rngerAddr)
// 	for _, vote := range votes {
// 		log.Printf("[debug]\t%v", vote.Players)
// 	}
// 	debugMu.Unlock()

// 	// Check all subsets of size at least k for one that is in at least k votes
// 	max := len(playerList)
// 	currentPlayerList := make([]Address, 0, max)

// 	for i := max; int64(i) >= k/2; i-- {
// 		combin := NewCombinator(max, i)
// 		for {
// 			// Extract the subset based on the bit mask
// 			currentPlayerList = currentPlayerList[0:0]
// 			for i, m := range combin.mapping {
// 				if m {
// 					currentPlayerList = append(currentPlayerList, playerList[max-i-1])
// 				}
// 			}

// 			subsetHits := int64(0)
// 			for _, voteSet := range voteSets {
// 				if containsAddressSubset(currentPlayerList, voteSet) {
// 					subsetHits++
// 				}
// 			}

// 			if subsetHits >= k {
// 				return currentPlayerList, nil
// 			}

// 			if !combin.next() {
// 				break
// 			}
// 		}

// 	}

// 	return nil, errors.New("insufficient players to form a majority")
// }

// func potentialPlayers(votes []Vote, k int64) ([]Address, error) {
// 	playerCounts := map[Address]int64{}

// 	// Count the number of times a player is in a vote
// 	rngerAddr := Address(696969)
// 	for _, vote := range votes {
// 		rngerAddr = vote.To
// 		for _, addr := range vote.Players {
// 			playerCounts[addr]++
// 		}
// 	}

// 	// Remove players that are not in enough votes
// 	for key, value := range playerCounts {
// 		if value < k {
// 			delete(playerCounts, key)
// 		}
// 	}

// 	max := len(playerCounts)
// 	if int64(max) < k {

// 		debugMu.Lock()
// 		log.Printf("[error] player %v has voting set =\n", rngerAddr)
// 		for _, vote := range votes {
// 			log.Printf("[error]\t%v", vote.Players)
// 		}
// 		debugMu.Unlock()

// 		// Not enough players to proceed
// 		return nil, errors.New("insufficient players to form a majority")
// 	}

// 	// Extract the potential players from the map
// 	playerList := make([]Address, 0, max)
// 	for addr := range playerCounts {
// 		playerList = append(playerList, addr)
// 	}

// 	// Sort the list so that picking is deterministic
// 	sort.Slice(playerList, func(i, j int) bool {
// 		return playerList[i] < playerList[j]
// 	})

// 	return playerList, nil
// }

// func bitCount(n int) (count int) {
// 	for n > 0 {
// 		if n%2 == 1 {
// 			count++
// 		}
// 		n /= 2
// 	}
// 	return
// }

// func containsAddressSubset(subset []Address, set map[Address](struct{})) bool {
// 	for _, addr := range subset {
// 		if _, ok := set[addr]; !ok {
// 			return false
// 		}
// 	}
// 	return true
// }

// type Combinator struct {
// 	mapping []bool
// 	n       int
// 	x       int
// 	y       int
// }

// func NewCombinator(n, k int) Combinator {
// 	s, t := n-k, k
// 	mapping := make([]bool, s+t)
// 	for i := 0; i < t; i++ {
// 		mapping[i] = true
// 	}
// 	return Combinator{mapping, s + t, t, t}
// }

// func (c *Combinator) next() bool {
// 	if c.x >= c.n {
// 		return false
// 	}

// 	c.mapping[c.x-1], c.mapping[c.y-1] = false, true
// 	c.x++
// 	c.y++

// 	if !c.mapping[c.x-1] {
// 		c.mapping[c.x-1], c.mapping[0] = true, false

// 		if c.y > 2 {
// 			c.x = 2
// 		}

// 		c.y = 1
// 	}

// 	return true
// }

// func (c *Combinator) swap(i, j int) {
// 	c.mapping[i], c.mapping[j] = c.mapping[j], c.mapping[i]
// }
