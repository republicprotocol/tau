package open

import (
	"fmt"

	"github.com/republicprotocol/co-go"

	"github.com/republicprotocol/oro-go/core/vss/algebra"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type opener struct {
	index uint64

	n, k uint64

	opens   map[task.MessageID]Open
	shares  map[task.MessageID]map[uint64]BroadcastShares
	results map[task.MessageID]Result
}

func New(index, n, k uint64, cap int) task.Task {
	return task.New(task.NewIO(cap), newOpener(index, n, k))
}

func newOpener(index, n, k uint64) *opener {
	return &opener{
		index: index,

		n: n, k: k,

		opens:   map[task.MessageID]Open{},
		shares:  map[task.MessageID]map[uint64]BroadcastShares{},
		results: map[task.MessageID]Result{},
	}
}

func (opener *opener) Reduce(message task.Message) task.Message {
	switch message := message.(type) {

	case Open:
		return opener.open(message)

	case BroadcastShares:
		return opener.tryOpen(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (opener *opener) open(message Open) task.Message {
	if result, ok := opener.results[message.MessageID]; ok {
		return result
	}
	opener.opens[message.MessageID] = message

	shares := NewBroadcastShares(message.MessageID, opener.index, message.Shares)

	return task.NewMessageBatch([]task.Message{
		opener.tryOpen(shares),
		shares,
	})
}

func (opener *opener) tryOpen(message BroadcastShares) task.Message {
	if _, ok := opener.shares[message.MessageID]; !ok {
		opener.shares[message.MessageID] = map[uint64]BroadcastShares{}
	}
	opener.shares[message.MessageID][message.From] = message

	// Do not continue if there is an insufficient number of shares
	if uint64(len(opener.shares[message.MessageID])) < opener.k {
		return nil
	}
	// Do not continue if we have not received a signal to open
	if _, ok := opener.opens[message.MessageID]; !ok {
		return nil
	}
	// Do not continue if we have already completed the opening
	if _, ok := opener.results[message.MessageID]; ok {
		return nil
	}

	batch := len(message.Shares)
	values := make([]algebra.FpElement, batch)

	co.ForAll(batch, func(b int) {
		sharesCache := make([]shamir.Share, opener.n)

		n := 0
		for _, opening := range opener.shares[message.MessageID] {
			sharesCache[n] = opening.Shares[b]
			n++
		}
		value, err := shamir.Join(sharesCache[:n])
		if err != nil {
			panic(err)
		}
		values[b] = value
	})
	result := NewResult(message.MessageID, values)

	opener.results[message.MessageID] = result
	delete(opener.opens, message.MessageID)
	delete(opener.shares, message.MessageID)

	return result
}

// An Open message signals to an Opener that it should open shares with other
// Openers. Before receiving an Open message for a particular task.MessageID,
// an Opener will still accept BroadcastShares messages related to the task.MessageID.
// However, an Opener will not produce a Result for a particular task.MessageID
// until the respective Open message is received.
type Open struct {
	task.MessageID

	Shares []shamir.Share
}

// NewOpen returns a new Open message.
func NewOpen(id task.MessageID, shares []shamir.Share) Open {
	return Open{id, shares}
}

// IsMessage implements the Message interface for Open.
func (message Open) IsMessage() {
}

// An BroadcastShares message is used by an Opener to accept and store shares so that the
// respective secret can be opened. An BroadcastShares message is related to other BroadcastShares
// messages, and to an Open message, by its task.MessageID.
type BroadcastShares struct {
	task.MessageID

	From   uint64
	Shares []shamir.Share
}

// NewBroadcastShares returns a new BroadcastShares message.
func NewBroadcastShares(id task.MessageID, from uint64, shares []shamir.Share) BroadcastShares {
	return BroadcastShares{id, from, shares}
}

// IsMessage implements the Message interface for BroadcastShares.
func (message BroadcastShares) IsMessage() {
}

// A Result message is produced by an Opener after it has received (a) an Open
// message, and (b) a sufficient threshold of BroadcastShares messages with the same task.MessageID.
// The order in which it receives the Open message and the BroadcastShares messages does
// not affect the production of a Result. A Result message is related to a
// Open message by its task.MessageID.
type Result struct {
	task.MessageID

	Values []algebra.FpElement
}

// NewResult returns a new Result message.
func NewResult(id task.MessageID, values []algebra.FpElement) Result {
	return Result{
		id, values,
	}
}

// IsMessage implements the Message interface for Result.
func (message Result) IsMessage() {
}
