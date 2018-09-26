package mul

import (
	"fmt"

	"github.com/republicprotocol/co-go"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type multiplier struct {
	index uint64

	n, k uint64

	muls      map[task.MessageID]Mul
	mulShares map[task.MessageID]map[uint64]BroadcastMulShares
	results   map[task.MessageID]Result
}

func New(index, n, k uint64, cap int) task.Task {
	return task.New(task.NewIO(cap), newMultiplier(index, n, k, cap))
}

func newMultiplier(index, n, k uint64, cap int) *multiplier {
	return &multiplier{
		index: index,

		n: n, k: k,

		muls:      map[task.MessageID]Mul{},
		mulShares: map[task.MessageID]map[uint64]BroadcastMulShares{},
		results:   map[task.MessageID]Result{},
	}
}

func (multiplier *multiplier) Reduce(message task.Message) task.Message {
	switch message := message.(type) {

	case Mul:
		return multiplier.mul(message)

	case BroadcastMulShares:
		return multiplier.tryOpenMul(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (multiplier *multiplier) mul(message Mul) task.Message {
	if result, ok := multiplier.results[message.MessageID]; ok {
		return result
	}
	multiplier.muls[message.MessageID] = message

	batch := len(message.xs)
	shares := make([]shamir.Share, batch)

	co.ForAll(batch, func(b int) {
		share := message.xs[b].Mul(message.ys[b])
		shares[b] = share.Add(message.ρs[b])
	})

	mulShares := NewBroadcastMulShares(message.MessageID, multiplier.index, shares)

	return task.NewMessageBatch([]task.Message{
		multiplier.tryOpenMul(mulShares),
		mulShares,
	})
}

func (multiplier *multiplier) tryOpenMul(message BroadcastMulShares) task.Message {
	if _, ok := multiplier.mulShares[message.MessageID]; !ok {
		multiplier.mulShares[message.MessageID] = map[uint64]BroadcastMulShares{}
	}
	multiplier.mulShares[message.MessageID][message.From] = message

	// Do not continue if there is an insufficient number of shares
	if uint64(len(multiplier.mulShares[message.MessageID])) < multiplier.k {
		return nil
	}
	// Do not continue if we have not received a signal to open
	if _, ok := multiplier.muls[message.MessageID]; !ok {
		return nil
	}
	// Do not continue if we have already completed the multiplication
	if _, ok := multiplier.results[message.MessageID]; ok {
		return nil
	}

	batch := len(message.Shares)
	shares := make([]shamir.Share, batch)

	co.ForAll(batch, func(b int) {
		sharesCache := make([]shamir.Share, multiplier.n)

		n := 0
		for _, opening := range multiplier.mulShares[message.MessageID] {
			sharesCache[n] = opening.Shares[b]
			n++
		}
		value, err := shamir.Join(sharesCache[:n])
		if err != nil {
			panic(err)
		}
		σ := multiplier.muls[message.MessageID].σs[b]
		share := shamir.New(σ.Index(), value)
		shares[b] = share.Sub(σ)
	})

	result := NewResult(message.MessageID, shares)

	multiplier.results[message.MessageID] = result
	delete(multiplier.muls, message.MessageID)
	delete(multiplier.mulShares, message.MessageID)

	return result
}

// A Mul message signals to a Multiplier that it should open intermediate
// multiplication shares with other Multipliers. Before receiving a Mul message
// for a particular task.MessageID, a Multiplier will still accept
// BroadcastMulShares messages related to the task.MessageID. However, a
// Multiplier will not produce a Result for a particular task.MessageID until
// the respective Mul message is received.
type Mul struct {
	task.MessageID

	xs, ys []shamir.Share
	ρs, σs []shamir.Share
}

// NewMul returns a new Mul message.
func NewMul(id task.MessageID, xs, ys, ρs, σs []shamir.Share) Mul {
	return Mul{
		id, xs, ys, ρs, σs,
	}
}

// IsMessage implements the Message interface for Mul.
func (message Mul) IsMessage() {
}

// A BroadcastMulShares message is used by a Multiplier to accept and store
// intermediate multiplication shares so that the respective multiplication can
// be completed. A BroadcastMulShares message is related to other
// BroadcastMulShares messages, and to a Mul message, by its task.MessageID.
type BroadcastMulShares struct {
	task.MessageID

	From   uint64
	Shares []shamir.Share
}

// NewBroadcastMulShares returns a new BroadcastMulShares message.
func NewBroadcastMulShares(id task.MessageID, from uint64, shares []shamir.Share) BroadcastMulShares {
	return BroadcastMulShares{
		id, from, shares,
	}
}

// IsMessage implements the Message interface for BroadcastMulShares.
func (message BroadcastMulShares) IsMessage() {
}

// A Result message is produced by a Multiplier after it has received (a) a Mul
// message, and (b) a sufficient threshold of BroadcastMulShares messages with
// the same task.MessageID. The order in which it receives the Mul message and
// the BroadcastMulShares messages does not affect the production of a Result. A
// Result message is related to a Mul message by its task.MessageID.
type Result struct {
	task.MessageID

	Shares []shamir.Share
}

// NewResult returns a new Result message.
func NewResult(id task.MessageID, shares []shamir.Share) Result {
	return Result{id, shares}
}

// IsMessage implements the Message interface for Result.
func (message Result) IsMessage() {
}
