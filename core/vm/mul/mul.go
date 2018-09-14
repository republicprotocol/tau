package mul

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type multiplier struct {
	n, k        uint64
	sharesCache shamir.Shares

	signals map[task.MessageID]SignalMul
	opens   map[task.MessageID]map[uint64]OpenMul
	results map[task.MessageID]Result
}

func New(n, k uint64, cap int) task.Task {
	return task.New(task.NewIO(cap), newMultiplier(n, k, cap))
}

func newMultiplier(n, k uint64, cap int) *multiplier {
	return &multiplier{
		n: n, k: k,
		sharesCache: make(shamir.Shares, n),

		signals: map[task.MessageID]SignalMul{},
		opens:   map[task.MessageID]map[uint64]OpenMul{},
		results: map[task.MessageID]Result{},
	}
}

func (multiplier *multiplier) Reduce(message task.Message) task.Message {
	switch message := message.(type) {

	case SignalMul:
		return multiplier.signalMul(message)

	case OpenMul:
		return multiplier.tryOpenMul(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (multiplier *multiplier) signalMul(message SignalMul) task.Message {
	if result, ok := multiplier.results[message.MessageID]; ok {
		return result
	}
	multiplier.signals[message.MessageID] = message

	share := message.x.Mul(message.y)
	mul := NewOpenMul(message.MessageID, share.Add(message.ρ))
	result := multiplier.tryOpenMul(mul)

	return task.NewMessageBatch(mul, result)
}

func (multiplier *multiplier) tryOpenMul(message OpenMul) task.Message {
	if _, ok := multiplier.opens[message.MessageID]; !ok {
		multiplier.opens[message.MessageID] = map[uint64]OpenMul{}
	}
	multiplier.opens[message.MessageID][message.Index()] = message

	// Do not continue if there is an insufficient number of shares
	if uint64(len(multiplier.opens[message.MessageID])) < multiplier.k {
		return nil
	}
	// Do not continue if we have not received a signal to open
	if _, ok := multiplier.signals[message.MessageID]; !ok {
		return nil
	}
	// Do not continue if we have already completed the multiplication
	if _, ok := multiplier.results[message.MessageID]; ok {
		return nil
	}

	n := 0
	for _, opening := range multiplier.opens[message.MessageID] {
		multiplier.sharesCache[n] = opening.Share
		n++
	}
	value, err := shamir.Join(multiplier.sharesCache[:n])
	if err != nil {
		return task.NewError(err)
	}
	σ := multiplier.signals[message.MessageID].σ
	share := shamir.New(σ.Index(), value)
	share = share.Sub(σ)
	result := NewResult(message.MessageID, share)

	multiplier.results[message.MessageID] = result
	delete(multiplier.signals, message.MessageID)
	delete(multiplier.opens, message.MessageID)

	return result
}
