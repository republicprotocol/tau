package rng_test

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	mathRand "math/rand"
	"time"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/vss/algebra"
	"github.com/republicprotocol/smpc-go/core/vss/pedersen"
	"github.com/republicprotocol/smpc-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vm/rng"
	"github.com/republicprotocol/smpc-go/core/vm/task"
)

var _ = Describe("Random number generators", func() {

	P := big.NewInt(8589934583)
	Q := big.NewInt(4294967291)
	G := algebra.NewFpElement(big.NewInt(592772542), P)
	H := algebra.NewFpElement(big.NewInt(4799487786), P)
	SecretField := algebra.NewField(Q)
	PedersenScheme := pedersen.New(G, H, SecretField)
	BufferLimit := 64

	// initPlayers for a secure multi-party computation network. These players
	// will communicate to run the secure random number generation algorithm.
	initPlayers := func(timeout time.Duration, n, k, t uint, bufferCap int) ([]task.Task, []buffer.ReaderWriter, []buffer.ReaderWriter) {
		// Initialis the players
		rngers := make([]task.Task, n)
		inputs := make([]buffer.ReaderWriter, n)
		outputs := make([]buffer.ReaderWriter, n)
		for i := uint(0); i < n; i++ {
			inputs[i] = buffer.NewReaderWriter(bufferCap)
			outputs[i] = buffer.NewReaderWriter(bufferCap)
			rngers[i] = New(inputs[i], outputs[i], timeout, Address(i), Address(0), n, k, t, PedersenScheme, bufferCap)
		}
		return rngers, inputs, outputs
	}

	// runPlayers unless the done channel is closed. The number of players,
	// input channels, and output channels must match. The Address of a player
	// must match the position of their channels.
	runPlayers := func(done <-chan struct{}, rngers []task.Task) {
		co.ParForAll(rngers, func(i int) {
			rngers[i].Run(done)
		})
	}

	// runTicker will send a CheckDeadline message, once per tick duration, to
	// all input channels. The ticker will stop after the done channel is
	// closed.
	runTicker := func(done <-chan struct{}, inputs []buffer.ReaderWriter, duration time.Duration) {
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
					case inputs[i].Writer() <- CheckDeadline{Time: now}:
					}
				})
			}
		}
	}

	// genLeader will send a Nominate message to all input channels for the
	// given leader. This will initiate the generation of a secure random number
	// for all players associated with one of the input channels.
	genLeader := func(done <-chan struct{}, leader Address, inputs []buffer.ReaderWriter, nonce Nonce) {
		co.ParForAll(inputs, func(i int) {
			select {
			case <-done:
				return
			case inputs[i].Writer() <- Nominate{Leader: leader}:
				if leader == Address(i) {
					// The leader starts the random number generation
					inputs[i].Writer() <- GenerateRn{Nonce: nonce}
				}
			}
		})
	}

	// verifyShares reconstruct to a consistent global random number. Different
	// k sized subsets of shares are used to reconstruct a secret and an error
	// is returned if the secrets are not equal.
	verifyShares := func(shares shamir.Shares, n, k int64) error {
		secret := shamir.Join(&SecretField, shares)
		for i := int64(0); i < n-k; i++ {
			kSecret := shamir.Join(&SecretField, shares[i:i+k])
			if secret.Cmp(kSecret) != 0 {
				return errors.New("malformed secret sharing")
			}
		}
		return nil
	}

	routeMessage := func(done <-chan (struct{}), input buffer.ReaderWriter, message buffer.Message, failureRate int) {
		if mathRand.Intn(100) < failureRate {
			// Simluate an unstable network connection and randomly drop
			// messages
			return
		}
		// Route LocalRnShare messages to their respective player
		select {
		case <-done:
			return
		case input.Writer() <- message:
		}
	}

	routeMessages := func(done <-chan struct{}, inputs []buffer.ReaderWriter, outputs []buffer.ReaderWriter, failureRate int) (<-chan GlobalRnShare, <-chan Err) {
		results := make(chan GlobalRnShare, len(outputs))
		errs := make(chan Err, len(outputs))

		go func() {
			defer close(results)
			defer close(errs)

			co.ParForAll(outputs, func(i int) {
				var message buffer.Message
				var ok bool

				for {
					select {
					case <-done:
						return
					case message, ok = <-outputs[i].Reader():
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

			input := buffer.NewReaderWriter(BufferLimit)
			output := buffer.NewReaderWriter(BufferLimit)
			rnger := New(input, output, time.Second, 0, 0, 1, 1, 1, PedersenScheme, 1)

			done := make(chan (struct{}))
			co.ParBegin(
				func() {
					rnger.Run(done)
				},
				func() {
					close(done)
				})
		})
	})

	Context("when closing the input channel", func() {
		It("should clean up and shutdown", func(doneT Done) {
			defer close(doneT)

			input := buffer.NewReaderWriter(BufferLimit)
			output := buffer.NewReaderWriter(BufferLimit)
			rnger := New(input, output, time.Second, 0, 0, 1, 1, 1, PedersenScheme, 1)

			done := make(chan (struct{}))
			co.ParBegin(
				func() {
					rnger.Run(done)
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
			{3, 2, BufferLimit}, {3, 2, BufferLimit * 2}, {3, 2, BufferLimit * 3}, {3, 2, BufferLimit * 4},
			{6, 4, BufferLimit}, {6, 4, BufferLimit * 2}, {6, 4, BufferLimit * 3}, {6, 4, BufferLimit * 4},
			{12, 8, BufferLimit}, {12, 8, BufferLimit * 2}, {12, 8, BufferLimit * 3}, {12, 8, BufferLimit * 4},
			{24, 16, BufferLimit}, {24, 16, BufferLimit * 2}, {24, 16, BufferLimit * 3}, {24, 16, BufferLimit * 4},
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
							runPlayers(done, rngers)
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

	Context("when running the secure random number generation algorithm in a partially connected network", func() {

		table := []struct {
			n, k, t                uint
			bufferCap, failureRate int
		}{
			// Failure rate = 1%
			{12, 8, 4, BufferLimit, 1}, {12, 8, 4, BufferLimit * 2, 1}, {12, 8, 4, BufferLimit * 3, 1}, {12, 8, 4, BufferLimit * 4, 1},
			{24, 16, 8, BufferLimit, 1}, {24, 16, 8, BufferLimit * 2, 1}, {24, 16, 8, BufferLimit * 3, 1}, {24, 16, 8, BufferLimit * 4, 1},
			{48, 32, 16, BufferLimit, 1}, {48, 32, 16, BufferLimit * 2, 1}, {48, 32, 16, BufferLimit * 3, 1}, {48, 32, 16, BufferLimit * 4, 1},

			// Failure rate = 5%
			{12, 8, 4, BufferLimit, 5}, {12, 8, 4, BufferLimit * 2, 5}, {12, 8, 4, BufferLimit * 3, 5}, {12, 8, 4, BufferLimit * 4, 5},
			{24, 16, 8, BufferLimit, 5}, {24, 16, 8, BufferLimit * 2, 5}, {24, 16, 8, BufferLimit * 3, 5}, {24, 16, 8, BufferLimit * 4, 5},
			{48, 32, 16, BufferLimit, 5}, {48, 32, 16, BufferLimit * 2, 5}, {48, 32, 16, BufferLimit * 3, 5}, {48, 32, 16, BufferLimit * 4, 5},

			// Failure rate = 10%
			{12, 8, 4, BufferLimit, 10}, {12, 8, 4, BufferLimit * 2, 10}, {12, 8, 4, BufferLimit * 3, 10}, {12, 8, 4, BufferLimit * 4, 10},
			{24, 16, 8, BufferLimit, 10}, {24, 16, 8, BufferLimit * 2, 10}, {24, 16, 8, BufferLimit * 3, 10}, {24, 16, 8, BufferLimit * 4, 10},
			{48, 32, 16, BufferLimit, 10}, {48, 32, 16, BufferLimit * 2, 10}, {48, 32, 16, BufferLimit * 3, 10}, {48, 32, 16, BufferLimit * 4, 10},

			// Failure rate = 15%
			{12, 8, 4, BufferLimit, 15}, {12, 8, 4, BufferLimit * 2, 15}, {12, 8, 4, BufferLimit * 3, 15}, {12, 8, 4, BufferLimit * 4, 15},
			{24, 16, 8, BufferLimit, 15}, {24, 16, 8, BufferLimit * 2, 15}, {24, 16, 8, BufferLimit * 3, 15}, {24, 16, 8, BufferLimit * 4, 15},
			{48, 32, 16, BufferLimit, 15}, {48, 32, 16, BufferLimit * 2, 15}, {48, 32, 16, BufferLimit * 3, 15}, {48, 32, 16, BufferLimit * 4, 15},

			// Failure rate = 20%
			{12, 8, 4, BufferLimit, 20}, {12, 8, 4, BufferLimit * 2, 20}, {12, 8, 4, BufferLimit * 3, 20}, {12, 8, 4, BufferLimit * 4, 20},
			{24, 16, 8, BufferLimit, 20}, {24, 16, 8, BufferLimit * 2, 20}, {24, 16, 8, BufferLimit * 3, 20}, {24, 16, 8, BufferLimit * 4, 20},
			{48, 32, 16, BufferLimit, 20}, {48, 32, 16, BufferLimit * 2, 20}, {48, 32, 16, BufferLimit * 3, 20}, {48, 32, 16, BufferLimit * 4, 20},
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
								runPlayers(done, rngers)
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
