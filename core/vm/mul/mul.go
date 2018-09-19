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

	muls    map[task.MessageID]Mul
	opens   map[task.MessageID]map[uint64]OpenMul
	results map[task.MessageID]Result
}

func New(index, n, k uint64, cap int) task.Task {
	return task.New(task.NewIO(cap), newMultiplier(index, n, k, cap))
}

func newMultiplier(index, n, k uint64, cap int) *multiplier {
	return &multiplier{
		index: index,

		n: n, k: k,

		muls:    map[task.MessageID]Mul{},
		opens:   map[task.MessageID]map[uint64]OpenMul{},
		results: map[task.MessageID]Result{},
	}
}

func (multiplier *multiplier) Reduce(message task.Message) task.Message {
	switch message := message.(type) {

	case Mul:
		return multiplier.mul(message)

	case OpenMul:
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

	mul := NewOpenMul(message.MessageID, multiplier.index, shares)

	return task.NewMessageBatch([]task.Message{
		multiplier.tryOpenMul(mul),
		mul,
	})
}

func (multiplier *multiplier) tryOpenMul(message OpenMul) task.Message {
	if _, ok := multiplier.opens[message.MessageID]; !ok {
		multiplier.opens[message.MessageID] = map[uint64]OpenMul{}
	}
	multiplier.opens[message.MessageID][message.From] = message

	// Do not continue if there is an insufficient number of shares
	if uint64(len(multiplier.opens[message.MessageID])) < multiplier.k {
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
		for _, opening := range multiplier.opens[message.MessageID] {
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
	delete(multiplier.opens, message.MessageID)

	return result
}
