package open

import (
	"fmt"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type opener struct {
	io task.IO

	n, k        uint64
	sharesCache shamir.Shares

	signals map[task.MessageID]Signal
	opens   map[task.MessageID]map[uint64]Open
	results map[task.MessageID]Result
}

func New(n, k uint64, cap int) task.Task {
	return &opener{
		io: task.NewIO(cap),

		n: n, k: k,
		sharesCache: make(shamir.Shares, n),

		signals: map[task.MessageID]Signal{},
		opens:   map[task.MessageID]map[uint64]Open{},
		results: map[task.MessageID]Result{},
	}
}

func (opener *opener) Channel() task.Channel {
	return opener.io.Channel()
}

func (opener *opener) Run(done <-chan struct{}) {
	for {
		message, ok := opener.io.Flush(done)
		if !ok {
			return
		}
		if message != nil {
			opener.recv(message)
		}
	}
}

func (opener *opener) recv(message task.Message) {
	switch message := message.(type) {

	case Signal:
		opener.signal(message)

	case Open:
		opener.tryOpen(message)

	default:
		panic(fmt.Sprintf("unexpected message type %T", message))
	}
}

func (opener *opener) signal(message Signal) {
	if result, ok := opener.results[message.MessageID]; ok {
		opener.io.Write(result)
		return
	}
	opener.signals[message.MessageID] = message

	opener.tryOpen(NewOpen(message.MessageID, message.Share))

	opener.io.Write(NewOpen(message.MessageID, message.Share))
}

func (opener *opener) tryOpen(message Open) {
	if _, ok := opener.opens[message.MessageID]; !ok {
		opener.opens[message.MessageID] = map[uint64]Open{}
	}
	opener.opens[message.MessageID][message.Index()] = message

	// Do not continue if there is an insufficient number of shares
	if uint64(len(opener.opens[message.MessageID])) < opener.k {
		return
	}
	// Do not continue if we have not received a signal to open
	if _, ok := opener.signals[message.MessageID]; !ok {
		return
	}
	// Do not continue if we have already completed the opening
	if _, ok := opener.results[message.MessageID]; ok {
		return
	}

	n := 0
	for _, opening := range opener.opens[message.MessageID] {
		opener.sharesCache[n] = opening.Share
		n++
	}
	value, err := shamir.Join(opener.sharesCache[:n])
	if err != nil {
		opener.io.Write(task.NewError(err))
		return
	}
	result := NewResult(message.MessageID, value)

	opener.results[message.MessageID] = result
	delete(opener.signals, message.MessageID)
	delete(opener.opens, message.MessageID)

	opener.io.Write(result)
}
