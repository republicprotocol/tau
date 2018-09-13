package mul

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type multiplier struct {
	io task.IO

	n, k        uint64
	sharesCache shamir.Shares

	signals map[task.MessageID]SignalMul
	opens   map[task.MessageID]map[uint64]OpenMul
	results map[task.MessageID]Result
}

func New(n, k uint64, cap int) task.Task {
	return &multiplier{
		io: task.NewIO(cap),

		n: n, k: k,
		sharesCache: make(shamir.Shares, n),

		signals: map[task.MessageID]SignalMul{},
		opens:   map[task.MessageID]map[uint64]OpenMul{},
		results: map[task.MessageID]Result{},
	}
}

func (multiplier *multiplier) Channel() task.Channel {
	return multiplier.io.Channel()
}

func (multiplier *multiplier) Run(done <-chan struct{}) {
	// defer log.Printf("[info] (mul) terminating")

	for {
		message, ok := multiplier.io.Flush(done)
		if !ok {
			return
		}
		multiplier.recv(message)
	}
}

func (multiplier *multiplier) recv(message task.Message) {
	switch message := message.(type) {

	case SignalMul:
		multiplier.signalMul(message)

	case OpenMul:
		multiplier.tryOpenMul(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (multiplier *multiplier) signalMul(message SignalMul) {
	if result, ok := multiplier.results[message.MessageID]; ok {
		multiplier.io.Write(result)
		return
	}
	multiplier.signals[message.MessageID] = message

	share := message.x.Mul(message.y)
	share = share.Add(message.ρ)
	multiplier.tryOpenMul(NewOpenMul(message.MessageID, share))

	multiplier.io.Write(NewOpenMul(message.MessageID, share))
}

func (multiplier *multiplier) tryOpenMul(message OpenMul) {
	if _, ok := multiplier.opens[message.MessageID]; !ok {
		multiplier.opens[message.MessageID] = map[uint64]OpenMul{}
	}
	multiplier.opens[message.MessageID][message.Index()] = message

	// Do not continue if there is an insufficient number of shares
	if uint64(len(multiplier.opens[message.MessageID])) < multiplier.k {
		return
	}
	// Do not continue if we have not received a signal to open
	if _, ok := multiplier.signals[message.MessageID]; !ok {
		return
	}
	// Do not continue if we have already completed the opening
	if _, ok := multiplier.results[message.MessageID]; ok {
		return
	}

	n := 0
	for _, opening := range multiplier.opens[message.MessageID] {
		multiplier.sharesCache[n] = opening.Share
		n++
	}
	value, err := shamir.Join(multiplier.sharesCache[:n])
	if err != nil {
		multiplier.io.Write(task.NewError(err))
		return
	}
	σ := multiplier.signals[message.MessageID].σ
	share := shamir.New(σ.Index(), value)
	share = share.Sub(σ)
	result := NewResult(message.MessageID, share)

	multiplier.results[message.MessageID] = result
	delete(multiplier.signals, message.MessageID)
	delete(multiplier.opens, message.MessageID)

	multiplier.io.Write(result)
}
