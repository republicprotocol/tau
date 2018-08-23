package rng_test

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	mathRand "math/rand"
	"sync"
	"time"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/shamir-go"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vm/rng"
)

var _ = Describe("Random number generators", func() {

	// initPlayers for a secure multi-party computation network. These players
	// will communicate to run the secure random number generation algorithm.
	initPlayers := func(timeout time.Duration, n, k, t uint, bufferCap int) ([]Rnger, [](chan InputMessage), [](chan OutputMessage)) {
		// Initialis the players
		rngers := make([]Rnger, n)
		inputs := make([]chan InputMessage, n)
		outputs := make([]chan OutputMessage, n)
		for i := uint(0); i < n; i++ {
			rngers[i] = NewRnger(timeout, Address(i), Address(0), n, k, t, bufferCap)
			inputs[i] = make(chan InputMessage, bufferCap)
			outputs[i] = make(chan OutputMessage, bufferCap)
		}
		return rngers, inputs, outputs
	}

	// runPlayers unless the done channel is closed. The number of players,
	// input channels, and output channels must match. The Address of a player
	// must match the position of their channels.
	runPlayers := func(done <-chan (struct{}), rngers []Rnger, inputs [](chan InputMessage), outputs [](chan OutputMessage)) {
		co.ParForAll(rngers, func(i int) {
			rngers[i].Run(done, inputs[i], outputs[i])
		})
	}

	// runTicker will send a CheckDeadline message, once per tick duration, to
	// all input channels. The ticker will stop after the done channel is
	// closed.
	runTicker := func(done <-chan (struct{}), inputs [](chan InputMessage), duration time.Duration) {
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

	// genLeader will send a Nominate message to all input channels for the
	// given leader. This will initiate the generation of a secure random number
	// for all players associated with one of the input channels.
	genLeader := func(done <-chan (struct{}), leader Address, inputs [](chan InputMessage), nonce Nonce) {
		co.ParForAll(inputs, func(i int) {
			select {
			case <-done:
				return
			case inputs[i] <- Nominate{Leader: leader}:
				if leader == Address(i) {
					// The leader starts the random number generation
					inputs[i] <- GenerateRn{Nonce: nonce}
				}
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
		ProposeRnMessages            map[Address]([]ProposeRn)
		LocalRnSharesMessages        map[Address]([]LocalRnShares)
		ProposeGlobalRnShareMessages map[Address]([]ProposeGlobalRnShare)
		GlobalRnShareMessages        map[Address]([]GlobalRnShare)
		ErrMessages                  map[Address]([]Err)
	}

	routeMessage := func(done <-chan (struct{}), input chan InputMessage, message InputMessage, failureRate int) {
		if mathRand.Intn(100) < failureRate {
			log.Printf("[debug] simulated network error")
			// Simluate an unstable network connection and randomly drop
			// messages
			return
		}
		// Route LocalRnShare messages to their respective player
		select {
		case <-done:
			return
		case input <- message:
			// log.Printf("sent message of type %T", message)
		}
	}

	routeMessages := func(done <-chan (struct{}), inputs [](chan InputMessage), outputs [](chan OutputMessage), messagesPerPlayer map[Address]int, failureRate int) routingResults {
		// Initialise results
		resultsMu := new(sync.Mutex)
		results := routingResults{
			ProposeRnMessages:            map[Address]([]ProposeRn){},
			LocalRnSharesMessages:        map[Address]([]LocalRnShares){},
			ProposeGlobalRnShareMessages: map[Address]([]ProposeGlobalRnShare){},
			GlobalRnShareMessages:        map[Address]([]GlobalRnShare){},
			ErrMessages:                  map[Address]([]Err){},
		}
		for i := range inputs {
			addr := Address(i)
			results.ProposeRnMessages[addr] = make([]ProposeRn, 0)
			results.LocalRnSharesMessages[addr] = make([]LocalRnShares, 0)
			results.ProposeGlobalRnShareMessages[addr] = make([]ProposeGlobalRnShare, 0)
			results.GlobalRnShareMessages[addr] = make([]GlobalRnShare, 0)
			results.ErrMessages[addr] = make([]Err, 0)
		}

		co.ParForAll(outputs, func(i int) {
			addr := Address(i)

			// Expect to route a specific number of messages per player
			var message OutputMessage
			var ok bool
			for n := 0; n < messagesPerPlayer[addr]; n++ {
				select {
				case <-done:
					return
				case message, ok = <-outputs[i]:
					if !ok {
						return
					}
				}

				func() {
					resultsMu.Lock()
					defer resultsMu.Unlock()

					switch message := message.(type) {
					case ProposeRn:
						results.ProposeRnMessages[addr] = append(results.ProposeRnMessages[addr], message)
						routeMessage(done, inputs[message.To], message, failureRate)

					case LocalRnShares:
						results.LocalRnSharesMessages[addr] = append(results.LocalRnSharesMessages[addr], message)
						routeMessage(done, inputs[message.To], message, failureRate)

					case ProposeGlobalRnShare:
						results.ProposeGlobalRnShareMessages[addr] = append(results.ProposeGlobalRnShareMessages[addr], message)
						routeMessage(done, inputs[message.To], message, failureRate)

					case GlobalRnShare:
						results.GlobalRnShareMessages[addr] = append(results.GlobalRnShareMessages[addr], message)

					case Err:
						results.ErrMessages[addr] = append(results.ErrMessages[addr], message)
					}
				}()
			}
		})
		return results
	}

	Context("when closing the done channel", func() {
		It("should clean up and shutdown", func(doneT Done) {
			defer close(doneT)

			rnger := NewRnger(time.Second, 0, 0, 1, 1, 1, 1)
			input := make(chan InputMessage)
			output := make(chan OutputMessage)

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

			rnger := NewRnger(time.Second, 0, 0, 1, 1, 1, 1)
			input := make(chan InputMessage)
			output := make(chan OutputMessage)

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

	FContext("when running the secure random number generation algorithm in a fully connected network", func() {

		table := []struct {
			n, k      uint
			bufferCap int
		}{
			{3, 2, 0}, {3, 2, 1}, {3, 2, 2}, {3, 2, 4},
			{6, 4, 0}, {6, 4, 1}, {6, 4, 2}, {6, 4, 4},
			{12, 8, 0}, {12, 8, 1}, {12, 8, 2}, {12, 8, 4},
			{24, 16, 0}, {24, 16, 1}, {24, 16, 2}, {24, 16, 4},
		}

		for _, entry := range table {
			entry := entry

			Context(fmt.Sprintf("when n = %v and k = %v and each player has a buffer capacity of %v", entry.n, entry.k, entry.bufferCap), func() {
				It("should produce consistent global random number shares", func(doneT Done) {
					defer close(doneT)

					mathRand.Seed(time.Now().UnixNano())
					rngers, inputs, outputs := initPlayers(100*time.Millisecond, entry.n, entry.k, entry.k/2, entry.bufferCap)

					// Nonce that will be used to identify the secure random
					// number
					nonce := Nonce{}
					n, err := rand.Read(nonce[:])
					Expect(n).To(Equal(len(nonce)))
					Expect(err).To(BeNil())

					leader := Address(mathRand.Uint64() % uint64(entry.n))
					done := make(chan (struct{}))
					co.ParBegin(
						func() {
							// Run the players until the done channel is closed
							runPlayers(done, rngers, inputs, outputs)
						},
						func() {
							// Run a globally timer for all players
							runTicker(done, inputs, 10*time.Millisecond)
						},
						func() {
							defer GinkgoRecover()
							defer close(done)

							// Instruct all players to generate a random number
							genLeader(done, leader, inputs, nonce)

							// Route messages between players until the expected
							// number of messages has been routed; n-1
							// LocalRnShare messages, n-1 Vote messages, and n
							// GlobalRnShare (of GenerateRnErr) messages
							messagesPerPlayer := map[Address]int{}
							failureRate := 0
							for i := range rngers {
								addr := Address(i)
								if addr == leader {
									messagesPerPlayer[addr] = 2*int(entry.n) + 2
								} else {
									messagesPerPlayer[addr] = 2
								}
							}
							results := routeMessages(done, inputs, outputs, messagesPerPlayer, failureRate)

							globalRnShares := make(shamir.Shares, 0, len(rngers))
							errs := make([]Err, 0, len(rngers))
							for i := range rngers {
								addr := Address(i)
								var messageCount int

								// Expect the correct number of messages
								if addr == leader {
									messageCount = int(entry.n)
								} else {
									messageCount = 0
								}
								Expect(results.ProposeRnMessages[addr]).To(HaveLen(messageCount))

								if addr == leader {
									messageCount = 1
								} else {
									messageCount = 1
								}
								log.Printf("possibly failing at address %v", addr)
								Expect(results.LocalRnSharesMessages[addr]).To(HaveLen(messageCount))

								if addr == leader {
									messageCount = int(entry.n)
								} else {
									messageCount = 0
								}
								Expect(results.ProposeGlobalRnShareMessages[addr]).To(HaveLen(messageCount))

								if addr == leader {
									messageCount = 1
								} else {
									messageCount = 1
								}
								Expect(results.GlobalRnShareMessages[addr]).To(HaveLen(messageCount))

								Expect(results.ErrMessages[addr]).To(HaveLen(0))

								// Expect the correct form of messages
								for _, message := range results.ProposeRnMessages[addr] {
									Expect(message.Nonce).To(Equal(nonce))
									Expect(message.From).To(Equal(addr))
								}
								for _, message := range results.LocalRnSharesMessages[addr] {
									Expect(message.Nonce).To(Equal(nonce))
									Expect(message.From).To(Equal(addr))
									Expect(message.Shares).To(HaveLen(int(entry.n)))
								}
								for _, message := range results.ProposeGlobalRnShareMessages[addr] {
									Expect(message.Nonce).To(Equal(nonce))
									Expect(message.From).To(Equal(addr))
								}
								if len(results.GlobalRnShareMessages[addr]) > 0 {
									globalRnShares = append(globalRnShares, results.GlobalRnShareMessages[addr][0].Share)
								}
								if len(results.ErrMessages[addr]) > 0 {
									errs = append(errs, results.ErrMessages[addr][0])
								}
							}
							Expect(globalRnShares).To(HaveLen(int(entry.n)))
							Expect(errs).To(HaveLen(0))

							// Reconstruct the secret using different subsets
							// of shares and expect that all reconstructed
							// secrets are equal
							err := verifyShares(globalRnShares, int64(entry.n), int64(entry.k))
							Expect(err).To(BeNil())
						})
				})
			})
		}
	})

	// Context("when running the secure random number generation algorithm in a partially connected network", func() {

	// 	table := []struct {
	// 		n, k, t                uint
	// 		bufferCap, failureRate int
	// 	}{
	// 		// Failure rate = 1%
	// 		{3, 2, 2, 0, 1}, {3, 2, 2, 1, 1}, {3, 2, 2, 2, 1}, {3, 2, 2, 4, 1},
	// 		{6, 4, 2, 0, 1}, {6, 4, 2, 1, 1}, {6, 4, 2, 2, 1}, {6, 4, 2, 4, 1},
	// 		{12, 8, 2, 0, 1}, {12, 8, 2, 1, 1}, {12, 8, 2, 2, 1}, {12, 8, 2, 4, 1},
	// 		{24, 16, 2, 0, 1}, {24, 16, 2, 1, 1}, {24, 16, 2, 2, 1}, {24, 16, 2, 4, 1},

	// 		// Failure rate = 5%
	// 		{3, 2, 2, 0, 5}, {3, 2, 2, 1, 5}, {3, 2, 2, 2, 5}, {3, 2, 2, 4, 5},
	// 		{6, 4, 2, 0, 5}, {6, 4, 2, 1, 5}, {6, 4, 2, 2, 5}, {6, 4, 2, 4, 5},
	// 		{12, 8, 2, 0, 5}, {12, 8, 2, 1, 5}, {12, 8, 2, 2, 5}, {12, 8, 2, 4, 5},
	// 		{24, 16, 2, 0, 5}, {24, 16, 2, 1, 5}, {24, 16, 2, 2, 5}, {24, 16, 2, 4, 5},

	// 		// Failure rate = 10%
	// 		{3, 2, 2, 0, 10}, {3, 2, 2, 1, 10}, {3, 2, 2, 2, 10}, {3, 2, 2, 4, 10},
	// 		{6, 4, 2, 0, 10}, {6, 4, 2, 1, 10}, {6, 4, 2, 2, 10}, {6, 4, 2, 4, 10},
	// 		{12, 8, 2, 0, 10}, {12, 8, 2, 1, 10}, {12, 8, 2, 2, 10}, {12, 8, 2, 4, 10},

	// 		// Failure rate = 20%
	// 		{3, 2, 2, 0, 20}, {3, 2, 2, 1, 20}, {3, 2, 2, 2, 20}, {3, 2, 2, 4, 20},
	// 		{6, 4, 2, 0, 20}, {6, 4, 2, 1, 20}, {6, 4, 2, 2, 20}, {6, 4, 2, 4, 20},
	// 		{12, 8, 2, 0, 20}, {12, 8, 2, 1, 20}, {12, 8, 2, 2, 20}, {12, 8, 2, 4, 20},

	// 		// Failure rate = 30%
	// 		{3, 2, 2, 0, 30}, {3, 2, 2, 1, 30}, {3, 2, 2, 2, 30}, {3, 2, 2, 4, 30},
	// 		{6, 4, 2, 0, 30}, {6, 4, 2, 1, 30}, {6, 4, 2, 2, 30}, {6, 4, 2, 4, 30},
	// 		{12, 8, 2, 0, 30}, {12, 8, 2, 1, 30}, {12, 8, 2, 2, 30}, {12, 8, 2, 4, 30},
	// 	}

	// 	for _, entry := range table {
	// 		entry := entry

	// 		Context(fmt.Sprintf("when n = %v and k = %v and each player has a buffer capacity of %v with a failure rate of %v%%", entry.n, entry.k, entry.bufferCap, entry.failureRate), func() {
	// 			It("should produce consistent global random number shares", func(doneT Done) {
	// 				defer close(doneT)

	// 				mathRand.Seed(time.Now().UnixNano())
	// 				rngers, inputs, outputs := initPlayers(100*time.Millisecond*time.Duration(entry.n), entry.n, entry.k, entry.bufferCap)

	// 				// Nonce that will be used to identify the secure random
	// 				// number
	// 				nonce := Nonce{}
	// 				n, err := rand.Read(nonce[:])
	// 				Expect(n).To(Equal(len(nonce)))
	// 				Expect(err).To(BeNil())

	// 				done := make(chan (struct{}))
	// 				co.ParBegin(
	// 					func() {
	// 						// Run the players until the done channel is closed
	// 						runPlayers(done, rngers, inputs, outputs)
	// 					},
	// 					func() {
	// 						// Run a globally timer for all players
	// 						runTicker(done, inputs, 10*time.Millisecond*time.Duration(entry.n))
	// 					},
	// 					func() {
	// 						// Instruct all players to generate a random number
	// 						genRn(done, inputs, nonce)
	// 					},
	// 					func() {
	// 						defer GinkgoRecover()
	// 						defer close(done)

	// 						// Route messages between players until the expected
	// 						// number of messages has been routed; n-1
	// 						// LocalRnShare messages, n-1 Vote messages,
	// 						// and n GlobalRnShare (of GenerateRnErr) messages
	// 						messagesPerPlayerPerBroadcast := int(entry.n - 1)
	// 						messagesPerPlayer := 2*messagesPerPlayerPerBroadcast + 1
	// 						failureRate := entry.failureRate
	// 						results := routeMessages(done, inputs, outputs, messagesPerPlayer, failureRate)

	// 						globalRnShares := map[string]shamir.Shares{}
	// 						generateRnErrs := make([]GenerateRnErr, 0, len(rngers))
	// 						for i := range rngers {
	// 							addr := Address(i)

	// 							minMessagesPerBroadcast := int(float64(messagesPerPlayerPerBroadcast) * float64(failureRate/2) / float64(100))

	// 							// Expect the correct number of messages
	// 							Expect(len(results.LocalRnShareMessages[addr])).To(BeNumerically(">", minMessagesPerBroadcast))
	// 							Expect(len(results.VoteMessages[addr])).To(BeNumerically(">", minMessagesPerBroadcast))
	// 							Expect(len(results.GlobalRnShareMessages[addr]) + len(results.GenerateRnErrMessages[addr])).To(Equal(1))

	// 							// Expect the correct form of messages
	// 							for _, message := range results.LocalRnShareMessages[addr] {
	// 								Expect(message.Nonce).To(Equal(nonce))
	// 								Expect(message.From).To(Equal(addr))
	// 							}
	// 							for _, message := range results.VoteMessages[addr] {
	// 								Expect(message.Nonce).To(Equal(nonce))
	// 								Expect(message.From).To(Equal(addr))
	// 							}
	// 							for _, message := range results.VoteMessages[addr] {
	// 								Expect(message.Nonce).To(Equal(nonce))
	// 							}
	// 							if len(results.GlobalRnShareMessages[addr]) > 0 {
	// 								key := fmt.Sprintf("%v", results.GlobalRnShareMessages[addr][0].Players)
	// 								if _, ok := globalRnShares[key]; !ok {
	// 									globalRnShares[key] = shamir.Shares{}
	// 								}
	// 								globalRnShares[key] = append(globalRnShares[key], results.GlobalRnShareMessages[addr][0].Share)
	// 							}
	// 							if len(results.GenerateRnErrMessages[addr]) > 0 {
	// 								generateRnErrs = append(generateRnErrs, results.GenerateRnErrMessages[addr][0])
	// 							}
	// 						}
	// 						Expect(len(generateRnErrs)).To(BeNumerically("<=", int(entry.n-entry.k)))

	// 						// Exactly one set of players should receive a
	// 						// sufficient vote
	// 						sufficientKey := ""
	// 						for key := range globalRnShares {
	// 							if int64(len(globalRnShares[key])) >= entry.k {
	// 								Expect(sufficientKey).To(Equal(""))
	// 								sufficientKey = key
	// 							}
	// 						}
	// 						log.Printf("global random shares =\n")
	// 						for key, value := range globalRnShares {
	// 							log.Printf("\t%v => %v\n", key, len(value))
	// 						}
	// 						Expect(sufficientKey).ToNot(Equal(""))

	// 						// Reconstruct the secret using different subsets
	// 						// of shares and expect that all reconstructed
	// 						// secrets are equal
	// 						err := verifyShares(globalRnShares[sufficientKey], int64(len(globalRnShares[sufficientKey])), entry.k)
	// 						Expect(err).To(BeNil())
	// 					})
	// 			}, 120 /* 1 minute timeout */)
	// 		})
	// 	}
	// })
})
