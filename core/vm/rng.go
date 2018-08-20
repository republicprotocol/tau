package vm

import (
	"fmt"
	"math/rand"

	shamir "github.com/republicprotocol/shamir-go"
)

type Rnger interface {
	Run(done <-chan (struct{}), input <-chan RngMessage, output chan<- RngResult)
}

type RngMessage interface {
	IsRngMessage()
}

type Gen struct {
	Nonce []byte
}

func (g Gen) IsRngMessage() {
}

type LocalRngShare struct {
	Nonce []byte
	J     int64
	From  int64
	Share shamir.Share
}

func (share LocalRngShare) IsRngMessage() {
}

func (share LocalRngShare) IsRngResult() {
}

type RngResult interface {
	IsRngResult()
}

type GlobalRngShare struct {
	Nonce []byte
	Share shamir.Share
}

func (share GlobalRngShare) IsRngResult() {
}

type rnger struct {
	i, n, k      int64
	resultBuffer []RngResult
	r            map[string](map[int64]LocalRngShare)
}

func NewRnger(i, n, k int64, bufferCap int) Rnger {
	return &rnger{
		i: i, n: n, k: k,
		resultBuffer: make([]RngResult, 0, bufferCap),
		r:            map[string](map[int64]LocalRngShare){},
	}
}

func (actor *rnger) Run(done <-chan (struct{}), messages <-chan RngMessage, results chan<- RngResult) {
	for {
		var res RngResult
		var resCh chan<- RngResult
		if len(actor.resultBuffer) > 0 {
			res = actor.resultBuffer[0]
			resCh = results
		}

		select {
		case <-done:
			return

		case message, ok := <-messages:
			if !ok {
				return
			}
			actor.handleMessage(message)

		case resCh <- res:
			actor.resultBuffer = actor.resultBuffer[1:]
		}
	}
}

func (actor *rnger) handleMessage(message RngMessage) {
	switch message := message.(type) {
	case Gen:
		actor.handleGen(message)

	case LocalRngShare:
		actor.handleLocalRngShare(message)
	}
}

func (actor *rnger) handleGen(message Gen) {
	r := rand.Uint64() % shamir.Prime
	rShares, err := shamir.Split(actor.n, actor.k, r)
	if err != nil {
		panic(fmt.Errorf("probably want to output this error: %v", err))
	}

	for j := int64(0); j < int64(len(rShares)); j++ {
		localRngShare := LocalRngShare{
			Nonce: message.Nonce,
			J:     j,
			From:  actor.i,
			Share: rShares[j],
		}
		if j == actor.i {
			actor.handleLocalRngShare(localRngShare)
			continue
		}
		actor.resultBuffer = append(actor.resultBuffer, localRngShare)
	}
}

func (actor *rnger) handleLocalRngShare(message LocalRngShare) {
	if message.J != actor.i {
		// This message is not meant for us
		return
	}
	if message.Share.Index != uint64(actor.i+1) {
		// This message is not meant for us
		return
	}

	// Initialise the map for this nonce if it does not already exist
	nonce := string(message.Nonce)
	if _, ok := actor.r[nonce]; !ok {
		actor.r[nonce] = map[int64]LocalRngShare{}
	}

	actor.r[nonce][message.From] = message
	if int64(len(actor.r[nonce])) == actor.n {
		share := shamir.Share{
			Index: uint64(actor.i + 1),
			Value: 0,
		}
		for _, localRngShare := range actor.r[nonce] {
			share = share.Add(&localRngShare.Share)
		}
		globalRngShare := GlobalRngShare{
			Nonce: message.Nonce,
			Share: share,
		}
		actor.resultBuffer = append(actor.resultBuffer, globalRngShare)
		delete(actor.r, nonce)
	}
}
