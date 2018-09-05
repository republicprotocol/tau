package mul_test

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	mathRand "math/rand"
	"time"

	"github.com/republicprotocol/smpc-go/core/vss/algebra"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/smpc-go/core/buffer"
	"github.com/republicprotocol/smpc-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vm/mul"
)

var _ = Describe("Multipliers", func() {

	Field := algebra.NewField(big.NewInt(8113765242226142771))
	X := Field.NewInField(big.NewInt(123456789))
	Y := Field.NewInField(big.NewInt(987654321))
	XY := X.Mul(Y)
	R := Field.NewInField(big.NewInt(123459876))

	BufferLimit := 128

	// initPlayers for a secure multi-party computation network. These players
	// will communicate to run the multiplication algorithm.
	initPlayers := func(n, k uint, bufferCap int) ([]Multiplier, [](chan buffer.Message), [](chan buffer.Message)) {
		// Initialis the players
		multipliers := make([]Multiplier, n)
		inputs := make([]chan buffer.Message, n)
		outputs := make([]chan buffer.Message, n)
		for i := uint(0); i < n; i++ {
			multipliers[i] = New(n, k, BufferLimit)
			inputs[i] = make(chan buffer.Message, bufferCap)
			outputs[i] = make(chan buffer.Message, bufferCap)
		}
		return multipliers, inputs, outputs
	}

	// runPlayers unless the done channel is closed. The number of players,
	// input channels, and output channels must match. The Address of a player
	// must match the position of their channels.
	runPlayers := func(done <-chan (struct{}), multipliers []Multiplier, inputs [](chan buffer.Message), outputs [](chan buffer.Message)) {
		co.ParForAll(multipliers, func(i int) {
			multipliers[i].Run(done, inputs[i], outputs[i])
		})
	}

	routeMessage := func(done <-chan (struct{}), inputs [](chan buffer.Message), message buffer.Message, failureRate int) {
		if mathRand.Intn(100) < failureRate {
			// Simluate an unstable network connection and randomly drop
			// messages
			return
		}
		// Route LocalRnShare messages to their respective player
		for _, input := range inputs {
			select {
			case <-done:
				return
			case input <- message:
			}
		}

	}

	verifyShares := func(shares shamir.Shares, n, k int64) error {
		secret, err := shamir.Join(shares)
		if err != nil {
			return err
		}
		if !XY.FieldEq(secret) {
			return errors.New("Multiplication Failed")
		}
		return nil
	}

	initiateMultiply := func(n, k uint, inputs [](chan buffer.Message)) {
		xPoly := algebra.NewRandomPolynomial(Field, k, X)
		xShares := shamir.Split(xPoly, uint64(n))
		yPoly := algebra.NewRandomPolynomial(Field, k, Y)
		yShares := shamir.Split(yPoly, uint64(n))

		ρPoly := algebra.NewRandomPolynomial(Field, k, R)
		ρShares := shamir.Split(ρPoly, uint64(n))

		σPoly := algebra.NewRandomPolynomial(Field, k/2, R)
		σShares := shamir.Split(σPoly, uint64(n))

		mathRand.Seed(time.Now().UnixNano())
		nonce := Nonce{}
		_, err := rand.Read(nonce[:])
		Expect(err).To(BeNil())

		co.ParForAll(inputs, func(i int) {
			inputs[i] <- NewMultiply(nonce, xShares[i], yShares[i], ρShares[i], σShares[i])
		})

	}

	routeMessages := func(done <-chan (struct{}), inputs [](chan buffer.Message), outputs [](chan buffer.Message), failureRate int) <-chan Result {
		results := make(chan Result, len(outputs))

		go func() {
			defer close(results)

			co.ParForAll(outputs, func(i int) {
				var message buffer.Message
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
					case Open:
						routeMessage(done, inputs, message, modifiedFailureRate)

					case Result:
						select {
						case <-done:
						case results <- message:
						}
					}

				}
			})
		}()

		return results
	}

	Context("when closing the done channel", func() {
		It("should clean up and shutdown", func(doneT Done) {
			defer close(doneT)
			multiplier := New(1, 1, 1)
			input := make(chan buffer.Message)
			output := make(chan buffer.Message)

			done := make(chan (struct{}))
			co.ParBegin(
				func() {
					multiplier.Run(done, input, output)
				},
				func() {
					close(done)
				})
		})
	})

	Context("when closing the input channel", func() {
		It("should clean up and shutdown", func(doneT Done) {
			defer close(doneT)
			multiplier := New(1, 1, 1)
			input := make(chan buffer.Message)
			output := make(chan buffer.Message)

			done := make(chan (struct{}))
			co.ParBegin(
				func() {
					multiplier.Run(done, input, output)
				},
				func() {
					close(input)
				})
		})
	})

	Context("when running the multiplication algorithm in a fully connected network", func() {

		table := []struct {
			n, k      uint
			bufferCap int
		}{
			{3, 2, 1}, {3, 2, 2}, {3, 2, 4}, {3, 2, 8},
			{6, 4, 2}, {6, 4, 4}, {6, 4, 8}, {6, 4, 16},
			{12, 8, 4}, {12, 8, 8}, {12, 8, 16}, {12, 8, 32},
			{24, 16, 8}, {24, 16, 16}, {24, 16, 32}, {24, 16, 64},
		}

		for _, entry := range table {
			entry := entry

			Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entry.n, entry.k, entry.bufferCap), func() {
				It("should multiply two numbers", func(doneT Done) {
					defer close(doneT)

					mathRand.Seed(time.Now().UnixNano())
					multipliers, inputs, outputs := initPlayers(entry.n, entry.k, entry.bufferCap)

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
							runPlayers(done, multipliers, inputs, outputs)
						},

						func() {
							defer GinkgoRecover()

							// Initiate smpc multiplication
							initiateMultiply(entry.n, entry.k, inputs)

							failureRate := 0
							results := routeMessages(done, inputs, outputs, failureRate)

							mulShares := map[uint64]shamir.Share{}

							func() {
								// Close the done channel when we are
								// finished collecting results
								defer close(done)

								// Expect to collect a result from each Rnger
								for result := range results {
									mulShares[result.Index] = result.Share
									if len(mulShares) == int(entry.n) {
										return
									}
								}
							}()

							// Extract the Shamir's secret shares from the results
							shares := shamir.Shares{}
							for _, mulShare := range mulShares {
								shares = append(shares, mulShare)
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

	// Context("when running the secure random number generation algorithm in a partially connected network", func() {

	// 	table := []struct {
	// 		n, k, t                uint
	// 		bufferCap, failureRate int
	// 	}{
	// 		// Failure rate = 1%
	// 		{12, 8, 4, 0, 1}, {12, 8, 4, 1, 1}, {12, 8, 4, 2, 1}, {12, 8, 4, 4, 1},
	// 		{24, 16, 8, 0, 1}, {24, 16, 8, 1, 1}, {24, 16, 8, 2, 1}, {24, 16, 8, 4, 1},
	// 		{48, 32, 16, 0, 1}, {48, 32, 16, 1, 1}, {48, 32, 16, 2, 1}, {48, 32, 16, 4, 1},

	// 		// Failure rate = 5%
	// 		{12, 8, 4, 0, 5}, {12, 8, 4, 1, 5}, {12, 8, 4, 2, 5}, {12, 8, 4, 4, 5},
	// 		{24, 16, 8, 0, 5}, {24, 16, 8, 1, 5}, {24, 16, 8, 2, 5}, {24, 16, 8, 4, 5},
	// 		{48, 32, 16, 0, 5}, {48, 32, 16, 1, 5}, {48, 32, 16, 2, 5}, {48, 32, 16, 4, 5},

	// 		// Failure rate = 10%
	// 		{12, 8, 4, 0, 10}, {12, 8, 4, 1, 10}, {12, 8, 4, 2, 10}, {12, 8, 4, 4, 10},
	// 		{24, 16, 8, 0, 10}, {24, 16, 8, 1, 10}, {24, 16, 8, 2, 10}, {24, 16, 8, 4, 10},
	// 		{48, 32, 16, 0, 10}, {48, 32, 16, 1, 10}, {48, 32, 16, 2, 10}, {48, 32, 16, 4, 10},

	// 		// Failure rate = 15%
	// 		{12, 8, 4, 0, 15}, {12, 8, 4, 1, 15}, {12, 8, 4, 2, 15}, {12, 8, 4, 4, 15},
	// 		{24, 16, 8, 0, 15}, {24, 16, 8, 1, 15}, {24, 16, 8, 2, 15}, {24, 16, 8, 4, 15},
	// 		{48, 32, 16, 0, 15}, {48, 32, 16, 1, 15}, {48, 32, 16, 2, 15}, {48, 32, 16, 4, 15},

	// 		// Failure rate = 20%
	// 		{12, 8, 4, 0, 20}, {12, 8, 4, 1, 20}, {12, 8, 4, 2, 20}, {12, 8, 4, 4, 20},
	// 		{24, 16, 8, 0, 20}, {24, 16, 8, 1, 20}, {24, 16, 8, 2, 20}, {24, 16, 8, 4, 20},
	// 		{48, 32, 16, 0, 20}, {48, 32, 16, 1, 20}, {48, 32, 16, 2, 20}, {48, 32, 16, 4, 20},
	// 	}

	// 	for _, entry := range table {
	// 		entry := entry

	// 		Context(fmt.Sprintf("when messages fail to send %v%% of the time", entry.failureRate), func() {
	// 			Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entry.n, entry.k, entry.bufferCap), func() {
	// 				It("should produce consistent global random number shares", func(doneT Done) {
	// 					defer close(doneT)

	// 					mathRand.Seed(time.Now().UnixNano())
	// 					rngers, inputs, outputs := initPlayers(100*time.Millisecond, entry.n, entry.k, entry.k/2, entry.bufferCap)

	// 					// Nonce that will be used to identify the secure random
	// 					// number
	// 					nonce := Nonce{}
	// 					n, err := rand.Read(nonce[:])
	// 					Expect(n).To(Equal(len(nonce)))
	// 					Expect(err).To(BeNil())

	// 					done := make(chan (struct{}))

	// 					co.ParBegin(
	// 						func() {
	// 							// Run the players until the done channel is closed
	// 							runPlayers(done, rngers, inputs, outputs)
	// 						},
	// 						func() {
	// 							defer GinkgoRecover()

	// 							// Nominate a random leader and instruct the leader
	// 							// to generate a random number
	// 							leader := Address(mathRand.Uint64() % uint64(entry.n))
	// 							genLeader(done, leader, inputs, nonce)

	// 							failureRate := entry.failureRate
	// 							results, errs := routeMessages(done, inputs, outputs, failureRate)

	// 							successRate := 1.0 - float64(failureRate)*0.01
	// 							successRate = successRate * successRate * successRate // 3 rounds of messaging can fail
	// 							successRate = successRate * float64(entry.n)
	// 							if uint(successRate) < entry.k {
	// 								successRate = float64(entry.k)
	// 							}

	// 							errMessages := []Err{}
	// 							globalRnSharesMessages := map[Address]GlobalRnShare{}
	// 							co.ParBegin(
	// 								func() {
	// 									// Close the done channel when we are
	// 									// finished collecting results
	// 									defer close(done)

	// 									// Expect to collect a result from each Rnger
	// 									timeout := time.After(500 * time.Millisecond)

	// 								EarlyExit:
	// 									for {
	// 										select {
	// 										case <-timeout:
	// 											break EarlyExit
	// 										case result := <-results:
	// 											globalRnSharesMessages[result.From] = result
	// 											if len(globalRnSharesMessages) >= int(successRate) {
	// 												return
	// 											}
	// 										}
	// 									}
	// 								},
	// 								func() {
	// 									for err := range errs {
	// 										log.Printf("[error] %v", err.Error())
	// 										errMessages = append(errMessages, err)
	// 									}
	// 								})

	// 							// Expect the super majority to hold shares
	// 							Expect(len(errMessages)).To(BeNumerically("<=", 0))
	// 							Expect(len(globalRnSharesMessages)).To(BeNumerically(">=", int(entry.k)))

	// 							// Extract the Shamir's secret shares from the results
	// 							shares := shamir.Shares{}
	// 							for _, globalRnShare := range globalRnSharesMessages {
	// 								shares = append(shares, globalRnShare.Share)
	// 							}

	// 							// Reconstruct the secret using different subsets
	// 							// of shares and expect that all reconstructed
	// 							// secrets are equal
	// 							err := verifyShares(shares, int64(len(shares)), int64(entry.k))
	// 							Expect(err).To(BeNil())
	// 						})
	// 				})
	// 			})
	// 		})
	// 	}
	// })
})
