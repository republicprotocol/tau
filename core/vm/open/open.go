package open

import (
	"log"

	shamir "github.com/republicprotocol/shamir-go"
	"github.com/republicprotocol/smpc-go/core/vm/buffer"
)

type Opener interface {
	Run(done <-chan (struct{}), sender <-chan buffer.Message, receiver chan<- buffer.Message)
}

type opener struct {
	n, k, t uint

	addr  Addr
	addrs []Addr

	buffer   buffer.Buffer
	openings map[Nonce](map[Addr]Open)
	cache    shamir.Shares
}

func NewOpener(addr Addr, addrs []Addr, n, k, t uint, cap int) Opener {
	return &opener{
		n: n, k: k, t: t,

		addr:  addr,
		addrs: addrs,

		buffer:   buffer.New(cap),
		openings: map[Nonce](map[Addr]Open){},
		cache:    make(shamir.Shares, n),
	}
}

func (opener *opener) Run(done <-chan (struct{}), sender <-chan buffer.Message, receiver chan<- buffer.Message) {
	defer log.Printf("[info] (mul) terminating")

	for {
		select {
		case <-done:
			return

		case message, ok := <-sender:
			if !ok {
				return
			}
			opener.recvMessage(message)

		case message := <-opener.buffer.Peek():
			if !opener.buffer.Pop() {
				log.Printf("[error] (mul) buffer underflow")
			}

			select {
			case <-done:
				return
			case receiver <- message:
			}
		}
	}
}

func (opener *opener) sendMessage(message buffer.Message) {
	if !opener.buffer.Push(message) {
		log.Printf("[error] (mul) buffer overflow")
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

func (opener *opener) open(message Open) {
	if _, ok := opener.openings[message.Nonce]; !ok {
		opener.openings[message.Nonce] = map[Addr]Open{}
	}
	opener.openings[message.Nonce][message.From] = message

	if uint(len(opener.openings[message.Nonce])) < opener.k {
		return
	}

	i := 0
	for _, opening := range opener.openings[message.Nonce] {
		opener.cache[i] = opening.Value
		i++
	}
	value := shamir.Join(opener.cache[:i])

	// FIXME: Use proper field addition.
	result := shamir.Share{
		Index: message.Value.Index,
		Value: value,
	}
	opener.sendMessage(NewResultMessage(message.Nonce, result))
}
