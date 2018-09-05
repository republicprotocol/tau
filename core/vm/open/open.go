package open

import (
	"log"

	"github.com/republicprotocol/shamir-go"
	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/node"
)

// An Opener receives `shamir.Shares` with different indices and, once it has
// enough, opens the secret into a public value.
type Opener interface {
	Run(done <-chan (struct{}), reader buffer.Reader, writer buffer.Writer)
}

type opener struct {
	n, k uint

	buffer   buffer.Buffer
	openings map[Nonce](map[node.Addr]Open)
	shares   shamir.Shares
}

func New(n, k uint, cap int) Opener {
	return &opener{
		n: n, k: k,

		buffer:   buffer.New(cap),
		openings: map[Nonce](map[node.Addr]Open){},
		shares:   make(shamir.Shares, n),
	}
}

func (opener *opener) Run(done <-chan (struct{}), reader buffer.Reader, writer buffer.Writer) {
	defer log.Printf("[info] (open) terminating")

	for {
		select {
		case <-done:
			return

		case message, ok := <-reader:
			if !ok {
				return
			}
			opener.recvMessage(message)

		case message := <-opener.buffer.Peek():
			if !opener.buffer.Pop() {
				log.Printf("[error] (open) buffer underflow")
			}
			select {
			case <-done:
			case writer <- message:
			}
		}
	}
}

func (opener *opener) sendMessage(message buffer.Message) {
	if !opener.buffer.Push(message) {
		log.Printf("[error] (open) buffer overflow")
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
		opener.openings[message.Nonce] = map[node.Addr]Open{}
	}
	opener.openings[message.Nonce][message.From] = message

	if uint(len(opener.openings[message.Nonce])) < opener.k {
		return
	}

	n := 0
	for _, opening := range opener.openings[message.Nonce] {
		opener.shares[n] = opening.Value
		n++
	}
	value := shamir.Join(opener.shares[:n])

	// FIXME:
	// * Use proper field addition.
	result := shamir.Share{
		Index: message.Value.Index,
		Value: value,
	}
	opener.sendMessage(NewResult(message.Nonce, result))
}
