package mul

import (
	"log"

	"github.com/republicprotocol/shamir-go"
	"github.com/republicprotocol/smpc-go/core/buffer"
)

type Multiplier interface {
	Run(done <-chan (struct{}), reader buffer.Reader, writer buffer.Writer)
}

type multiplier struct {
	n, k uint

	buffer   buffer.Buffer
	pendings map[Nonce]Multiply
	openings map[Nonce](map[uint64]Open)
	shares   shamir.Shares
}

func New(n, k uint, cap int) Multiplier {
	return &multiplier{
		n: n, k: k,

		buffer:   buffer.New(cap),
		pendings: map[Nonce]Multiply{},
		openings: map[Nonce](map[uint64]Open){},
		shares:   make(shamir.Shares, n),
	}
}

func (multiplier *multiplier) Run(done <-chan (struct{}), reader buffer.Reader, writer buffer.Writer) {
	defer log.Printf("[info] (mul) terminating")

	for {
		select {
		case <-done:
			return

		case message, ok := <-reader:
			if !ok {
				return
			}
			multiplier.recvMessage(message)

		case message := <-multiplier.buffer.Peek():
			if !multiplier.buffer.Pop() {
				log.Printf("[error] (mul) buffer underflow")
			}
			select {
			case <-done:
			case writer <- message:
			}
		}
	}
}

func (multiplier *multiplier) sendMessage(message buffer.Message) {
	if !multiplier.buffer.Push(message) {
		log.Printf("[error] (mul) buffer overflow")
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
	multiplier.sendMessage(NewOpen(message.Nonce, share))
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
	multiplier.sendMessage(NewResult(message.Nonce, result))
}
