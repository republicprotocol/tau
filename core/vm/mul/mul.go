package mul

import (
	"log"

	shamir "github.com/republicprotocol/shamir-go"
	"github.com/republicprotocol/smpc-go/core/vm/buffer"
)

type Multiplier interface {
	Run(done <-chan (struct{}), sender <-chan buffer.Message, receiver chan<- buffer.Message)
}

type multiplier struct {
	n, k, t uint

	addr   Addr
	addrs  []Addr
	buffer buffer.Buffer

	multipliers map[Nonce]Multiply
	openings    map[Nonce](map[Addr]Open)
	cache       shamir.Shares
}

func NewMultiplier(n, k, t uint, cap int) Multiplier {
	return &multiplier{
		n: n, k: k, t: t,

		buffer: buffer.New(cap),
		cache:  make(shamir.Shares, n),
	}
}

func (multer *multiplier) Run(done <-chan (struct{}), sender <-chan buffer.Message, receiver chan<- buffer.Message) {
	defer log.Printf("[info] (mul) terminating")

	for {
		select {
		case <-done:
			return

		case message, ok := <-sender:
			if !ok {
				return
			}
			multer.recvMessage(message)

		case message := <-multer.buffer.Peek():
			if !multer.buffer.Pop() {
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

func (multer *multiplier) sendMessage(message buffer.Message) {
	if !multer.buffer.Push(message) {
		log.Printf("[error] (mul) buffer overflow")
	}
}

func (multer *multiplier) recvMessage(message buffer.Message) {
	switch message := message.(type) {

	case Multiply:
		multer.multiply(message)

	case Open:
		multer.open(message)

	default:
		log.Printf("[error] unexpected message type %T", message)
	}
}

func (multer *multiplier) multiply(message Multiply) {
	// FIXME: Use proper field multiplication / addition.
	open := shamir.Share{
		Index: message.x.Index,
		Value: message.x.Value*message.y.Value + message.ρ.Value,
	}

	for _, addr := range multer.addrs {
		multer.sendMessage(NewOpenMessage(message.Nonce, addr, multer.addr, open))
	}
}

func (multer *multiplier) open(message Open) {
	if _, ok := multer.openings[message.Nonce]; !ok {
		multer.openings[message.Nonce] = map[Addr]Open{}
	}
	multer.openings[message.Nonce][message.From] = message

	if uint(len(multer.openings[message.Nonce])) < multer.k {
		return
	}

	i := 0
	for _, opening := range multer.openings[message.Nonce] {
		multer.cache[i] = opening.Value
		i++
	}
	value := shamir.Join(multer.cache[:i])

	// FIXME: Use proper field addition.
	result := shamir.Share{
		Index: multer.multipliers[message.Nonce].σ.Index,
		Value: value - multer.multipliers[message.Nonce].σ.Value,
	}
	multer.sendMessage(NewResultMessage(message.Nonce, result))
}
