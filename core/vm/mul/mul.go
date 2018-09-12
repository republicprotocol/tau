package mul

import (
	"log"

	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

type multiplier struct {
	io task.IO

	n, k        uint
	sharesCache shamir.Shares

	muls        map[Nonce]Mul
	broadcasts  map[Nonce]map[uint64]BroadcastIntermediateShare
	completions map[Nonce]shamir.Share
}

func New(n, k uint, cap int) task.Task {
	return &multiplier{
		io: task.NewIO(cap),

		n: n, k: k,
		sharesCache: make(shamir.Shares, n),

		muls:        map[Nonce]Mul{},
		broadcasts:  map[Nonce]map[uint64]BroadcastIntermediateShare{},
		completions: map[Nonce]shamir.Share{},
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
		multiplier.recvMessage(message)
	}
}

func (multiplier *multiplier) recvMessage(message task.Message) {
	switch message := message.(type) {

	case Mul:
		multiplier.multiply(message)

	case BroadcastIntermediateShare:
		multiplier.recvBroadcastIntermediateShare(message)

	default:
		log.Printf("[error] unexpected message type %T", message)
	}
}

func (multiplier *multiplier) multiply(message Mul) {
	if share, ok := multiplier.completions[message.Nonce]; ok {
		multiplier.io.Write(NewResult(message.Nonce, share))
	}
	share := message.x.Mul(message.y)
	share = share.Add(message.ρ)

	multiplier.muls[message.Nonce] = message
	multiplier.recvMessage(NewBroadcastIntermediateShare(message.Nonce, share))
	multiplier.io.Write(NewBroadcastIntermediateShare(message.Nonce, share))
}

// TODO:
// * Do we delete the pendings/openings from memory once we have received
//   enough to open the secret?
// * Do we produce duplicate results when we receive more than `k` openings?
func (multiplier *multiplier) recvBroadcastIntermediateShare(message BroadcastIntermediateShare) {
	if _, ok := multiplier.broadcasts[message.Nonce]; !ok {
		multiplier.broadcasts[message.Nonce] = map[uint64]BroadcastIntermediateShare{}
	}
	multiplier.broadcasts[message.Nonce][message.Index()] = message

	// Do not continue if there is an insufficient number of shares
	if uint(len(multiplier.broadcasts[message.Nonce])) < multiplier.k {
		return
	}
	// Do not continue if we have not received a signal to open
	if _, ok := multiplier.muls[message.Nonce]; !ok {
		return
	}
	// Do not continue if we have already completed the opening
	if _, ok := multiplier.completions[message.Nonce]; ok {
		return
	}

	n := 0
	for _, broadcastIntermediateShare := range multiplier.broadcasts[message.Nonce] {
		multiplier.sharesCache[n] = broadcastIntermediateShare.Share
		n++
	}
	value, err := shamir.Join(multiplier.sharesCache[:n])
	if err != nil {
		log.Printf("[error] (mul) join error = %v", multiplier, err)
		return
	}

	σ := multiplier.muls[message.Nonce].σ
	result := shamir.New(σ.Index(), value)
	result = result.Sub(σ)

	delete(multiplier.muls, message.Nonce)
	delete(multiplier.broadcasts, message.Nonce)
	multiplier.completions[message.Nonce] = result

	multiplier.io.Write(NewResult(message.Nonce, result))
}
