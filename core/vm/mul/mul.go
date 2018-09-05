package mul

import (
	"log"

	"github.com/republicprotocol/shamir-go"
	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/vm/task"
)

type multiplier struct {
	io         task.IO
	ioExternal task.IO

	n, k     uint
	pendings map[Nonce]Multiply
	openings map[Nonce](map[uint64]Open)
	shares   shamir.Shares
}

func New(r, w buffer.ReaderWriter, n, k uint, cap int) task.Task {
	return &multiplier{
		io:         task.NewIO(buffer.New(cap), r.Reader(), w.Writer()),
		ioExternal: task.NewIO(buffer.New(cap), w.Reader(), r.Writer()),

		n: n, k: k,
		pendings: map[Nonce]Multiply{},
		openings: map[Nonce](map[uint64]Open){},
		shares:   make(shamir.Shares, n),
	}
}

func (multiplier *multiplier) IO() task.IO {
	return multiplier.ioExternal
}

func (multiplier *multiplier) Run(done <-chan struct{}) {
	defer log.Printf("[info] (mul) terminating")

	for {
		ok := task.Select(
			done,
			multiplier.recvMessage,
			multiplier.io,
		)
		if !ok {
			return
		}
	}
}

func (multiplier *multiplier) recvMessage(message buffer.Message) {
	switch message := message.(type) {

	case Multiply:
		multiplier.multiply(message)

	case Open:
		multiplier.open(message)

	default:
		log.Printf("[error] unexpected message type %T", message)
	}
}

func (multiplier *multiplier) multiply(message Multiply) {
	// FIXME: Use proper field multiplication / addition.
	share := shamir.Share{
		Index: message.x.Index,
		Value: message.x.Value*message.y.Value + message.ρ.Value,
	}

	multiplier.pendings[message.Nonce] = message
	multiplier.io.Send(NewOpen(message.Nonce, share))
	multiplier.recvMessage(NewOpen(message.Nonce, share))
}

// TODO:
// * Do we delete the pendings/openings from memory once we have received
//   enough to open the secret?
// * Do we produce duplicate results when we receive more than `k` openings?
func (multiplier *multiplier) open(message Open) {
	if _, ok := multiplier.openings[message.Nonce]; !ok {
		multiplier.openings[message.Nonce] = map[uint64]Open{}
	}
	multiplier.openings[message.Nonce][message.Index] = message

	if uint(len(multiplier.openings[message.Nonce])) < multiplier.k {
		return
	}

	n := 0
	for _, opening := range multiplier.openings[message.Nonce] {
		multiplier.shares[n] = opening.Share
		n++
	}
	value := shamir.Join(multiplier.shares[:n])

	// FIXME: Use proper field addition.
	result := shamir.Share{
		Index: multiplier.pendings[message.Nonce].σ.Index,
		Value: value - multiplier.pendings[message.Nonce].σ.Value,
	}
	multiplier.io.Send(NewResult(message.Nonce, result))
}
