package rng_test

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	mathRand "math/rand"
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
			// Simluate an unstable network connection and randomly drop
			// messages
			return
		}
		// Route LocalRnShare messages to their respective player
		select {
		case <-done:
			return
		case input <- message:
		}
	}

	routeMessages := func(done <-chan (struct{}), inputs [](chan InputMessage), outputs [](chan OutputMessage), failureRate int) (<-chan GlobalRnShare, <-chan Err) {
		results := make(chan GlobalRnShare, len(outputs))
		errs := make(chan Err, len(outputs))

		go func() {
			defer close(results)
			defer close(errs)

			co.ParForAll(outputs, func(i int) {
				var message OutputMessage
				var ok bool

				for {
					select {
					case <-done:
						return
					case message, ok = <-outputs[i]:
						if !ok {
							return
						}
					}

					modifiedFailureRate := failureRate
					switch message := message.(type) {
					case ProposeRn:
						if message.To == message.From {
							modifiedFailureRate = 0
						}
						routeMessage(done, inputs[message.To], message, modifiedFailureRate)

					case LocalRnShares:
						if message.To == message.From {
							modifiedFailureRate = 0
						}
						routeMessage(done, inputs[message.To], message, modifiedFailureRate)

					case ProposeGlobalRnShare:
						if message.To == message.From {
							modifiedFailureRate = 0
						}
						routeMessage(done, inputs[message.To], message, modifiedFailureRate)

					case GlobalRnShare:
						select {
						case <-done:
						case results <- message:
						}

					case Err:
						select {
						case <-done:
						case errs <- message:
						}
					}
				}
			})
		}()

		return results, errs
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

	Context("when running the secure random number generation algorithm in a fully connected network", func() {

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

			Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entry.n, entry.k, entry.bufferCap), func() {
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

							// Nominate a random leader and instruct the leader
							// to generate a random number
							leader := Address(mathRand.Uint64() % uint64(entry.n))
							genLeader(done, leader, inputs, nonce)

							failureRate := 0
							results, errs := routeMessages(done, inputs, outputs, failureRate)

							globalRnShares := map[Address]GlobalRnShare{}
							co.ParBegin(
								func() {
									// Close the done channel when we are
									// finished collecting results
									defer close(done)

									// Expect to collect a result from each Rnger
									for result := range results {
										globalRnShares[result.From] = result
										if len(globalRnShares) == int(entry.n) {
											return
										}
									}
								},
								func() {
									for err := range errs {
										Expect(err).To(BeNil())
									}
								})

							// Extract the Shamir's secret shares from the results
							shares := shamir.Shares{}
							for _, globalRnShare := range globalRnShares {
								shares = append(shares, globalRnShare.Share)
							}

							// Reconstruct the secret using different subsets
							// of shares and expect that all reconstructed
							// secrets are equal
							err := verifyShares(shares, int64(entry.n), int64(entry.k))
							Expect(err).To(BeNil())
						})
				})
			})
		}
	})

	FContext("when running the secure random number generation algorithm in a partially connected network", func() {

		table := []struct {
			n, k, t                uint
			bufferCap, failureRate int
		}{
			// Failure rate = 1%
			{12, 8, 4, 0, 1}, {12, 8, 4, 1, 1}, {12, 8, 4, 2, 1}, {12, 8, 4, 4, 1},
			{24, 16, 8, 0, 1}, {24, 16, 8, 1, 1}, {24, 16, 8, 2, 1}, {24, 16, 8, 4, 1},
			{48, 32, 16, 0, 1}, {48, 32, 16, 1, 1}, {48, 32, 16, 2, 1}, {48, 32, 16, 4, 1},

			// Failure rate = 5%
			{12, 8, 4, 0, 5}, {12, 8, 4, 1, 5}, {12, 8, 4, 2, 5}, {12, 8, 4, 4, 5},
			{24, 16, 8, 0, 5}, {24, 16, 8, 1, 5}, {24, 16, 8, 2, 5}, {24, 16, 8, 4, 5},
			{48, 32, 16, 0, 5}, {48, 32, 16, 1, 5}, {48, 32, 16, 2, 5}, {48, 32, 16, 4, 5},

			// Failure rate = 10%
			{12, 8, 4, 0, 10}, {12, 8, 4, 1, 10}, {12, 8, 4, 2, 10}, {12, 8, 4, 4, 10},
			{24, 16, 8, 0, 10}, {24, 16, 8, 1, 10}, {24, 16, 8, 2, 10}, {24, 16, 8, 4, 10},
			{48, 32, 16, 0, 10}, {48, 32, 16, 1, 10}, {48, 32, 16, 2, 10}, {48, 32, 16, 4, 10},

			// Failure rate = 15%
			{12, 8, 4, 0, 15}, {12, 8, 4, 1, 15}, {12, 8, 4, 2, 15}, {12, 8, 4, 4, 15},
			{24, 16, 8, 0, 15}, {24, 16, 8, 1, 15}, {24, 16, 8, 2, 15}, {24, 16, 8, 4, 15},
			{48, 32, 16, 0, 15}, {48, 32, 16, 1, 15}, {48, 32, 16, 2, 15}, {48, 32, 16, 4, 15},

			// Failure rate = 20%
			{12, 8, 4, 0, 20}, {12, 8, 4, 1, 20}, {12, 8, 4, 2, 20}, {12, 8, 4, 4, 20},
			{24, 16, 8, 0, 20}, {24, 16, 8, 1, 20}, {24, 16, 8, 2, 20}, {24, 16, 8, 4, 20},
			{48, 32, 16, 0, 20}, {48, 32, 16, 1, 20}, {48, 32, 16, 2, 20}, {48, 32, 16, 4, 20},
		}

		for _, entry := range table {
			entry := entry

			Context(fmt.Sprintf("when messages fail to send %v%% of the time", entry.failureRate), func() {
				Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entry.n, entry.k, entry.bufferCap), func() {
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

								// Nominate a random leader and instruct the leader
								// to generate a random number
								leader := Address(mathRand.Uint64() % uint64(entry.n))
								genLeader(done, leader, inputs, nonce)

								failureRate := entry.failureRate
								results, errs := routeMessages(done, inputs, outputs, failureRate)

								successRate := 1.0 - float64(failureRate)*0.01
								successRate = successRate * successRate * successRate // 3 rounds of messaging can fail
								successRate = successRate * float64(entry.n)
								if uint(successRate) < entry.k {
									successRate = float64(entry.k)
								}

								errMessages := []Err{}
								globalRnSharesMessages := map[Address]GlobalRnShare{}
								co.ParBegin(
									func() {
										// Close the done channel when we are
										// finished collecting results
										defer close(done)

										// Expect to collect a result from each Rnger
										timeout := time.After(500 * time.Millisecond)

									EarlyExit:
										for {
											select {
											case <-timeout:
												break EarlyExit
											case result := <-results:
												globalRnSharesMessages[result.From] = result
												if len(globalRnSharesMessages) >= int(successRate) {
													return
												}
											}
										}
									},
									func() {
										for err := range errs {
											log.Printf("[error] %v", err.Error())
											errMessages = append(errMessages, err)
										}
									})

								// Expect the super majority to hold shares
								Expect(len(errMessages)).To(BeNumerically("<=", 0))
								Expect(len(globalRnSharesMessages)).To(BeNumerically(">=", int(entry.k)))

								// Extract the Shamir's secret shares from the results
								shares := shamir.Shares{}
								for _, globalRnShare := range globalRnSharesMessages {
									shares = append(shares, globalRnShare.Share)
								}

								// Reconstruct the secret using different subsets
								// of shares and expect that all reconstructed
								// secrets are equal
								err := verifyShares(shares, int64(len(shares)), int64(entry.k))
								Expect(err).To(BeNil())
							})
					})
				})
			})
		}
	})
})
