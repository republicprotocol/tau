package open

import (
	"log"

	"github.com/republicprotocol/oro-go/core/buffer"
	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type opener struct {
	io         task.IO
	ioExternal task.IO

	n, k        uint
	sharesCache shamir.Shares

	opens       map[Nonce]Open
	broadcasts  map[Nonce]map[uint64]BroadcastShare
	completions map[Nonce]struct{}
}

func New(r, w buffer.ReaderWriter, n, k uint, cap int) task.Task {
	return &opener{
		io:         task.NewIO(buffer.New(cap), r.Reader(), w.Writer()),
		ioExternal: task.NewIO(buffer.New(cap), w.Reader(), r.Writer()),

		n: n, k: k,
		sharesCache: make(shamir.Shares, n),

		opens:       map[Nonce]Open{},
		broadcasts:  map[Nonce]map[uint64]BroadcastShare{},
		completions: map[Nonce]struct{}{},
	}
}

func (opener *opener) IO() task.IO {
	return opener.ioExternal
}

func (opener *opener) Run(done <-chan struct{}) {
	// defer log.Printf("[info] (open) terminating")

	for {
		ok := task.Select(
			done,
			opener.recvMessage,
			opener.io,
		)
		if !ok {
			return
		}
	}
}

func (opener *opener) recvMessage(message buffer.Message) {
	switch message := message.(type) {

	case Open:
		opener.open(message)

	case BroadcastShare:
		opener.recvBroadcastShare(message)

	default:
		log.Printf("[error] unexpected message type %T", message)
	}
}

// TODO:
// * Do we delete the openings from memory once we have received enough to open
//   the secret?
// * Do we produce duplicate results when we receive more than `k` openings?
func (opener *opener) open(message Open) {
	if _, ok := opener.opens[message.Nonce]; ok {
		return
	}
	opener.opens[message.Nonce] = message
	opener.recvBroadcastShare(NewBroadcastShare(message.Nonce, message.Share))
	opener.io.Send(NewBroadcastShare(message.Nonce, message.Share))
}

func (opener *opener) recvBroadcastShare(message BroadcastShare) {
	if _, ok := opener.broadcasts[message.Nonce]; !ok {
		opener.broadcasts[message.Nonce] = map[uint64]BroadcastShare{}
	}
	opener.broadcasts[message.Nonce][message.Index()] = message

	// Do not continue if there is an insufficient number of shares
	if uint(len(opener.broadcasts[message.Nonce])) < opener.k {
		return
	}
	// Do not continue if we have not received a signal to open
	if _, ok := opener.opens[message.Nonce]; !ok {
		return
	}
	// Do not continue if we have already completed the opening
	if _, ok := opener.completions[message.Nonce]; ok {
		return
	}

	n := 0
	for _, broadcastShare := range opener.broadcasts[message.Nonce] {
		opener.sharesCache[n] = broadcastShare.Share
		n++
	}
	result, err := shamir.Join(opener.sharesCache[:n])
	if err != nil {
		panic("unimplemented")
	}

	delete(opener.opens, message.Nonce)
	delete(opener.broadcasts, message.Nonce)
	opener.completions[message.Nonce] = struct{}{}

	opener.io.Send(NewResult(message.Nonce, result))
}
