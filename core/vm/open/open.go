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

	signals map[task.MessageID]Signal
	opens   map[task.MessageID]map[uint64]Open
	results map[task.MessageID]Result
}

func New(index, n, k uint64, cap int) task.Task {
	return task.New(task.NewIO(cap), newOpener(index, n, k))
}

func newOpener(index, n, k uint64) *opener {
	return &opener{
		index: index,

		n: n, k: k,

		signals: map[task.MessageID]Signal{},
		opens:   map[task.MessageID]map[uint64]Open{},
		results: map[task.MessageID]Result{},
	}
}

func (opener *opener) Reduce(message task.Message) task.Message {
	switch message := message.(type) {

	case Signal:
		return opener.signal(message)

	case Open:
		return opener.tryOpen(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (opener *opener) signal(message Signal) task.Message {
	if result, ok := opener.results[message.MessageID]; ok {
		return result
	}
	opener.signals[message.MessageID] = message

	open := NewOpen(message.MessageID, opener.index, message.Shares)

	return task.NewMessageBatch([]task.Message{
		opener.tryOpen(open),
		open,
	})
}

func (opener *opener) tryOpen(message Open) task.Message {
	if _, ok := opener.opens[message.MessageID]; !ok {
		opener.opens[message.MessageID] = map[uint64]Open{}
	}
	opener.opens[message.MessageID][message.From] = message

	// Do not continue if there is an insufficient number of shares
	if uint64(len(opener.opens[message.MessageID])) < opener.k {
		return nil
	}
	// Do not continue if we have not received a signal to open
	if _, ok := opener.signals[message.MessageID]; !ok {
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
		for _, opening := range opener.opens[message.MessageID] {
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
	delete(opener.signals, message.MessageID)
	delete(opener.opens, message.MessageID)

	return result
}
