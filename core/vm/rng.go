package vm

import (
	"math/rand"

	"github.com/republicprotocol/shamir-go"
)

// Rnger generates secure random numbers using a secure multi-party
// computation. The random number generated is not known by any Rnger in the
// network, unless some threshold of Rngers are malicious.
type Rnger interface {

	// Run the Rnger. It will read messages from the input channel and write
	// messages to the output channel. Close the done channel to stop the Rnger.
	Run(done <-chan (struct{}), input <-chan RngInputMessage, output chan<- RngOutputMessage)
}

// An RngInputMessage can be passed to the Rnger as an input. It will process
// the message and output an error when it encounters an unexpected type. No
// types external to this package should implement this interface.
type RngInputMessage interface {

	// IsRngInputMessage is a marker used to restrict RngInputMessages to types
	// that have been explicitly marked. It is never called.
	IsRngInputMessage()
}

// An RngOutputMessage can be passed from the Rnger as an output. The user must
// check the message type and handle the message appropriately. No types
// external to this package should implement this interface.
type RngOutputMessage interface {

	// IsRngOutputMessage is a marker used to restrict RngOutputMessages to
	// types that have been explicitly marked. It is never called.
	IsRngOutputMessage()
}

// The GenerateRn message signals the Rnger to begin a secure random number
// generation. The secure random number is identified by a nonce, that must be
// agreed upon by all Rngers running the secure random number generation
// algorithm.
type GenerateRn struct {
	Nonce []byte
}

// IsRngInputMessage implements the RngInputMessage interface.
func (msg GenerateRn) IsRngInputMessage() {
}

// A GenerateRnErr is output by an Rnger when an error is encountered during
// the secure random number generation algorithm. No specific handling is
// required by the user.
type GenerateRnErr struct {
	Nonce []byte
	Err   error
}

// IsRngOutputMessage implements the RngOutputMessage interface.
func (msg GenerateRnErr) IsRngOutputMessage() {
}

// A LocalRnShare is output by an Rnger for each other Rnger in the network. It
// is also accepted as input from other Rngers. The user must route this
// message to the appropriate Rnger when it is output.
type LocalRnShare struct {
	Nonce []byte
	To    uint64
	From  uint64
	Share shamir.Share
}

// IsRngInputMessage implements the RngInputMessage interface.
func (msg LocalRnShare) IsRngInputMessage() {
}

// IsRngOutputMessage implements the RngOutputMessage interface.
func (msg LocalRnShare) IsRngOutputMessage() {
}

// A GlobalRnShare is output by an Rnger at the end of the secure random number
// generation algorithm. It represents its Shamir's share of the secure random
// number identified across the network by the nonce.
type GlobalRnShare struct {
	Nonce []byte
	Share shamir.Share
}

// IsRngOutputMessage implements the RngOutputMessage interface.
func (msg GlobalRnShare) IsRngOutputMessage() {
}

type rnger struct {
	i             uint64
	n, k          int64
	outputBuffer  []RngOutputMessage
	localRnShares map[string](map[uint64]LocalRnShare)
}

// NewRnger returns an Rnger that is identified as the i-th player in a network
// with n players and k threshold. The user can specify a buffer capacity for
// messages output by the Rnger.
func NewRnger(i uint64, n, k int64, bufferCap int) Rnger {
	return &rnger{
		i: i,
		n: n, k: k,
		outputBuffer:  make([]RngOutputMessage, 0, bufferCap),
		localRnShares: map[string](map[uint64]LocalRnShare){},
	}
}

// Run implements the Rnger interface. It is blocking and should be run in a
// background goroutine by the user. It is recommended that the input and
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
			Err:   err,
		})
		return
	}

	// Send each share to the appropriate player by outputting a LocalRnShare
	// message
	for j := uint64(0); j < uint64(len(rnShares)); j++ {
		localRnShare := LocalRnShare{
			Nonce: message.Nonce,
			To:    j,
			From:  rnger.i,
			Share: rnShares[j],
		}
		if j == rnger.i {
			rnger.handleLocalRnShare(localRnShare)
			continue
		}
		rnger.outputBuffer = append(rnger.outputBuffer, localRnShare)
	}
}

func (rnger *rnger) handleLocalRnShare(message LocalRnShare) {
	if message.To != rnger.i || message.Share.Index != rnger.i+1 {
		// This message is not meant for us
		return
	}

	// Initialise the map for this nonce if it does not already exist
	nonce := string(message.Nonce)
	if _, ok := rnger.localRnShares[nonce]; !ok {
		rnger.localRnShares[nonce] = map[uint64]LocalRnShare{}
	}
	rnger.localRnShares[nonce][message.From] = message

	// Once we have acquired a LocalRnShare from each player in the network we
	// can add them to produce our GlobalRnShare
	if int64(len(rnger.localRnShares[nonce])) == rnger.n {
		share := shamir.Share{
			Index: uint64(rnger.i + 1),
			Value: 0,
		}
		for _, localRngShare := range rnger.localRnShares[nonce] {
			share = share.Add(&localRngShare.Share)
		}
		globalRnShare := GlobalRnShare{
			Nonce: message.Nonce,
			Share: share,
		}
		rnger.outputBuffer = append(rnger.outputBuffer, globalRnShare)
		delete(rnger.localRnShares, nonce)
	}
}
