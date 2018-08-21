package vm_test

import (
	"crypto/rand"
	"errors"
	"fmt"
	mathRand "math/rand"
	"sync"
	"time"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/shamir-go"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vm"
)

var _ = Describe("Random number generators", func() {

	// initPlayers for a secure multi-party computation network. These players
	// will communicate to run the secure random number generation algorithm.
	initPlayers := func(n, k int64, bufferCap int) ([]Rnger, [](chan RngInputMessage), [](chan RngOutputMessage)) {
		// Initialis the players
		rngers := make([]Rnger, n)
		inputs := make([]chan RngInputMessage, n)
		outputs := make([]chan RngOutputMessage, n)
		for i := int64(0); i < n; i++ {
			rngers[i] = NewRnger(time.Second, Address(i), n, k, bufferCap)
			inputs[i] = make(chan RngInputMessage, bufferCap)
			outputs[i] = make(chan RngOutputMessage, bufferCap)
		}
		return rngers, inputs, outputs
	}

	// runPlayers unless the done channel is closed. The number of players,
	// input channels, and output channels must match. The Address of a player
	// must match the position of their channels.
	runPlayers := func(done <-chan (struct{}), rngers []Rnger, inputs [](chan RngInputMessage), outputs [](chan RngOutputMessage)) {
		co.ParForAll(rngers, func(i int) {
			rngers[i].Run(done, inputs[i], outputs[i])
		})
	}

	// runTicker will send a CheckDeadline message, once per tick duration, to
	// all input channels. The ticker will stop after the done channel is
	// closed.
	runTicker := func(done <-chan (struct{}), inputs [](chan RngInputMessage), duration time.Duration) {
		ticker := time.NewTicker(duration)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				now := time.Now()
				co.ParForAll(inputs, func(i int) {
					select {
					case <-done:
						return
					case inputs[i] <- CheckDeadline{Time: now}:
					}
				})
			}
		}
	}

	// genRn will send a GenerateRn message to all input channels. This will
	// initiate the generation of a secure random number for all players
	// associated with one of the input channels.
	genRn := func(done <-chan (struct{}), inputs [](chan RngInputMessage), nonce Nonce) {
		co.ParForAll(inputs, func(i int) {
			select {
			case <-done:
				return
			case inputs[i] <- GenerateRn{Nonce: nonce}:
			}
		})
	}

	// verifyShares reconstruct to a consistent global random number. Different
	// k sized subsets of shares are used to reconstruct a secret and an error
	// is returned if the secrets are not equal.
	verifyShares := func(shares shamir.Shares, n, k int64) error {
		secret := shamir.Join(shares)
		for i := int64(0); i < n-k; i++ {
			kSecret := shamir.Join(shares[i : i+k])
			if secret != kSecret {
				return errors.New("malformed secret sharing")
			}
		}
		return nil
	}

	// routingResults from routing messages between players.
	type routingResults struct {
		LocalRnShareMessages  map[Address]([]LocalRnShare)
		VoteMessages          map[Address]([]VoteToCommit)
		GlobalRnShareMessages map[Address]([]GlobalRnShare)
		GenerateRnErrMessages map[Address]([]GenerateRnErr)
	}

	routeMessages := func(done <-chan (struct{}), inputs [](chan RngInputMessage), outputs [](chan RngOutputMessage), messagesPerPlayer int, failureRate int) routingResults {
		// Initialise results
		resultsMu := new(sync.Mutex)
		results := routingResults{
			LocalRnShareMessages:  map[Address]([]LocalRnShare){},
			VoteMessages:          map[Address]([]VoteToCommit){},
			GlobalRnShareMessages: map[Address]([]GlobalRnShare){},
			GenerateRnErrMessages: map[Address]([]GenerateRnErr){},
		}
		for i := range inputs {
			addr := Address(i)
			results.LocalRnShareMessages[addr] = make([]LocalRnShare, 0)
			results.VoteMessages[addr] = make([]VoteToCommit, 0)
			results.GlobalRnShareMessages[addr] = make([]GlobalRnShare, 0)
			results.GenerateRnErrMessages[addr] = make([]GenerateRnErr, 0)
		}

		co.ParForAll(outputs, func(i int) {
			addr := Address(i)

			// Expect to route a specific number of messages per player
			var message RngOutputMessage
			var ok bool
			for n := 0; n < messagesPerPlayer; n++ {
				select {
				case <-done:
					return
				case message, ok = <-outputs[i]:
					if !ok {
						return
					}
				}
				if mathRand.Intn(100) < failureRate {
					// Simluate an unstable network connection and randomly drop
					// messages
					continue
				}

				func() {
					resultsMu.Lock()
					defer resultsMu.Unlock()

					switch message := message.(type) {
					case LocalRnShare:
						// Route LocalRnShare messages to their respective player
						select {
						case <-done:
							return
						case inputs[message.To] <- message:
							results.LocalRnShareMessages[addr] = append(results.LocalRnShareMessages[addr], message)
						}

					case VoteToCommit:
						// Route VoteToCommit messages to their respective player
						select {
						case <-done:
							return
						case inputs[message.To] <- message:
							results.VoteMessages[addr] = append(results.VoteMessages[addr], message)
						}

					case GlobalRnShare:
						// GlobalRnShare messages do not need to be routed
						results.GlobalRnShareMessages[addr] = append(results.GlobalRnShareMessages[addr], message)

					case GenerateRnErr:
						// GenerateRnErr messages do not need to be routed
						results.GenerateRnErrMessages[addr] = append(results.GenerateRnErrMessages[addr], message)
					}
				}()
			}
		})
		return results
	}

	Context("when closing the done channel", func() {
		It("should clean up and shutdown", func(doneT Done) {
			defer close(doneT)

			rnger := NewRnger(time.Second, 0, 1, 1, 1)
			input := make(chan RngInputMessage)
			output := make(chan RngOutputMessage)

			done := make(chan (struct{}))
			co.ParBegin(
				func() {
					rnger.Run(done, input, output)
				},
				func() {
					close(done)
				})
		})
	})

	Context("when closing the input channel", func() {
		It("should clean up and shutdown", func(doneT Done) {
			defer close(doneT)

			rnger := NewRnger(time.Second, 0, 1, 1, 1)
			input := make(chan RngInputMessage)
			output := make(chan RngOutputMessage)

			done := make(chan (struct{}))
			co.ParBegin(
				func() {
					rnger.Run(done, input, output)
				},
				func() {
					close(input)
				})
		})
	})

	Context("when running the secure random number generation algorithm", func() {

		table := []struct {
			n, k                   int64
			bufferCap, failureRate int
		}{
			// Failure rate = 0%
			{3, 2, 0, 0}, {3, 2, 1, 0}, {3, 2, 2, 0}, {3, 2, 4, 0},
			{6, 4, 0, 0}, {6, 4, 1, 0}, {6, 4, 2, 0}, {6, 4, 4, 0},
			{12, 8, 0, 0}, {12, 8, 1, 0}, {12, 8, 2, 0}, {12, 8, 4, 0},
			{24, 16, 0, 0}, {24, 16, 1, 0}, {24, 16, 2, 0}, {24, 16, 4, 0},
			// {48, 32, 0, 0}, {48, 32, 1, 0}, {48, 32, 2, 0}, {48, 32, 4, 0},

			// Failure rate = 10%
			{3, 2, 0, 10}, {3, 2, 1, 10}, {3, 2, 2, 10}, {3, 2, 4, 10},
			{6, 4, 0, 10}, {6, 4, 1, 10}, {6, 4, 2, 10}, {6, 4, 4, 10},
			{12, 8, 0, 10}, {12, 8, 1, 10}, {12, 8, 2, 10}, {12, 8, 4, 10},
			{24, 16, 0, 10}, {24, 16, 1, 10}, {24, 16, 2, 10}, {24, 16, 4, 10},
			// {48, 32, 0, 10}, {48, 32, 1, 10}, {48, 32, 2, 10}, {48, 32, 4, 10},

			// // Failure rate = 20%
			// {3, 2, 0, 20}, {3, 2, 1, 20}, {3, 2, 2, 20}, {3, 2, 4, 20},
			// {6, 4, 0, 20}, {6, 4, 1, 20}, {6, 4, 2, 20}, {6, 4, 4, 20},
			// {12, 8, 0, 20}, {12, 8, 1, 20}, {12, 8, 2, 20}, {12, 8, 4, 20},
			// {24, 16, 0, 20}, {24, 16, 1, 20}, {24, 16, 2, 20}, {24, 16, 4, 20},
			// {48, 32, 0, 20}, {48, 32, 1, 20}, {48, 32, 2, 20}, {48, 32, 4, 20},

			// // Failure rate = 30%
			// {3, 2, 0, 30}, {3, 2, 1, 30}, {3, 2, 2, 30}, {3, 2, 4, 30},
			// {6, 4, 0, 30}, {6, 4, 1, 30}, {6, 4, 2, 30}, {6, 4, 4, 30},
			// {12, 8, 0, 30}, {12, 8, 1, 30}, {12, 8, 2, 30}, {12, 8, 4, 30},
			// {24, 16, 0, 30}, {24, 16, 1, 30}, {24, 16, 2, 30}, {24, 16, 4, 30},
			// {48, 32, 0, 30}, {48, 32, 1, 30}, {48, 32, 2, 30}, {48, 32, 4, 30},
		}

		for _, entry := range table {
			entry := entry

			Context(fmt.Sprintf("when n = %v and k = %v and each player has a buffer capacity of %v with a failure rate of %v%%", entry.n, entry.k, entry.bufferCap, entry.failureRate), func() {
				It("should produce consistent global random number shares", func(doneT Done) {
					defer close(doneT)

					rngers, inputs, outputs := initPlayers(entry.n, entry.k, entry.bufferCap)

					// Nonce that will be used to identify the secure random
					// number
					nonce := Nonce{}
					n, err := rand.Read(nonce[:])
					Expect(n).To(Equal(len(nonce)))
					Expect(err).To(BeNil())

					done := make(chan (struct{}))
					co.ParBegin(
						func() {
							// Run the players until the done channel is closed
							runPlayers(done, rngers, inputs, outputs)
						},
						func() {
							// Run a globally timer for all players
							runTicker(done, inputs, time.Millisecond)
						},
						func() {
							// Instruct all players to generate a random number
							genRn(done, inputs, nonce)
						},
						func() {
							defer GinkgoRecover()
							defer close(done)

							// Route messages between players until the expected
							// number of messages has been routed; n-1
							// LocalRnShare messages, n-1 VoteToCommite
							// messages, and n GlobalRnShare messages
							messagesPerPlayerPerBroadcast := int(entry.n - 1)
							messagesPerPlayer := 2*messagesPerPlayerPerBroadcast + 1
							failureRate := entry.failureRate
							results := routeMessages(done, inputs, outputs, messagesPerPlayer, failureRate)

							globalRnShares := make(shamir.Shares, len(rngers))
							for i := range rngers {
								addr := Address(i)

								minMessagesPerBroadcast := int(float64(messagesPerPlayerPerBroadcast) * float64(failureRate/2) / float64(100))

								// Expect the correct number of messages
								Expect(len(results.LocalRnShareMessages[addr])).To(BeNumerically(">", minMessagesPerBroadcast))
								Expect(len(results.VoteMessages[addr])).To(BeNumerically(">", minMessagesPerBroadcast))
								Expect(results.GlobalRnShareMessages[addr]).To(HaveLen(1))
								Expect(results.GenerateRnErrMessages[addr]).To(HaveLen(0))

								// Expect the correct form of messages
								for _, message := range results.LocalRnShareMessages[addr] {
									Expect(message.Nonce).To(Equal(nonce))
									Expect(message.From).To(Equal(addr))
								}
								for _, message := range results.VoteMessages[addr] {
									Expect(message.Nonce).To(Equal(nonce))
									Expect(message.From).To(Equal(addr))
									Expect(len(message.Players)).To(BeNumerically(">=", int(entry.k)))
								}
								for _, message := range results.VoteMessages[addr] {
									Expect(message.Nonce).To(Equal(nonce))
								}

								globalRnShares[i] = results.GlobalRnShareMessages[addr][0].Share
							}

							// Reconstruct the secret using different subsets
							// of shares and expect that all reconstructed
							// secrets are equal
							err := verifyShares(globalRnShares, entry.n, entry.k)
							Expect(err).To(BeNil())
						})
				}, 30 /* 4 second timeout */)
			})
		}
	})
})
