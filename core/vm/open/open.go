package open

import (
	"log"

	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/vm/task"
	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

type opener struct {
	io         task.IO
	ioExternal task.IO

	n, k     uint
	openings map[Nonce](map[uint64]Open)
	shares   shamir.Shares
}

func New(r, w buffer.ReaderWriter, n, k uint, cap int) task.Task {
	return &opener{
		io:         task.NewIO(buffer.New(cap), r.Reader(), w.Writer()),
		ioExternal: task.NewIO(buffer.New(cap), w.Reader(), r.Writer()),

		n: n, k: k,
		openings: map[Nonce](map[uint64]Open){},
		shares:   make(shamir.Shares, n),
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

	default:
		log.Printf("[error] unexpected message type %T", message)
	}
}

// TODO:
// * Do we delete the openings from memory once we have received enough to open
//   the secret?
// * Do we produce duplicate results when we receive more than `k` openings?
func (opener *opener) open(message Open) {
	if _, ok := opener.openings[message.Nonce]; !ok {
		opener.openings[message.Nonce] = map[uint64]Open{}
	}
	opener.openings[message.Nonce][message.Index()] = message

	if uint(len(opener.openings[message.Nonce])) < opener.k {
		return
	}

	n := 0
	for _, opening := range opener.openings[message.Nonce] {
		opener.shares[n] = opening.Share
		n++
	}
	result, err := shamir.Join(opener.shares[:n])
	if err != nil {
		panic("unimplemented")
	}

	opener.io.Send(NewResult(message.Nonce, result))
}
