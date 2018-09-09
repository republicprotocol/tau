package mul_test

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	mathRand "math/rand"
	"time"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/oro-go/core/buffer"
	"github.com/republicprotocol/oro-go/core/vm/task"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/vm/mul"
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
	initPlayers := func(n, k uint, bufferCap int) ([]task.Task, []buffer.ReaderWriter, []buffer.ReaderWriter) {
		// Initialis the players
		multipliers := make([]task.Task, n)
		inputs := make([]buffer.ReaderWriter, n)
		outputs := make([]buffer.ReaderWriter, n)
		for i := uint(0); i < n; i++ {
			inputs[i] = buffer.NewReaderWriter(bufferCap)
			outputs[i] = buffer.NewReaderWriter(bufferCap)
			multipliers[i] = New(inputs[i], outputs[i], n, k, BufferLimit)
		}
		return multipliers, inputs, outputs
	}

	// runPlayers unless the done channel is closed. The number of players,
	// input channels, and output channels must match. The Address of a player
	// must match the position of their channels.
	runPlayers := func(done <-chan (struct{}), multipliers []task.Task) {
		co.ParForAll(multipliers, func(i int) {
			multipliers[i].Run(done)
		})
	}

	routeMessage := func(done <-chan (struct{}), inputs []buffer.ReaderWriter, message buffer.Message, failureRate int) {
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
			case input.Writer() <- message:
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

	initiateMultiply := func(n, k uint, inputs []buffer.ReaderWriter) {
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
			inputs[i].Writer() <- NewMultiply(nonce, xShares[i], yShares[i], ρShares[i], σShares[i])
		})

	}

	routeMessages := func(done <-chan (struct{}), inputs []buffer.ReaderWriter, outputs []buffer.ReaderWriter, failureRate int) <-chan Result {
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
					case message, ok = <-outputs[i].Reader():
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
			input := buffer.NewReaderWriter(1)
			output := buffer.NewReaderWriter(1)
			multiplier := New(input, output, 1, 1, 1)

			done := make(chan (struct{}))
			co.ParBegin(
				func() {
					multiplier.Run(done)
				},
				func() {
					close(done)
				})
		})
	})

	Context("when closing the input channel", func() {
		It("should clean up and shutdown", func(doneT Done) {
			defer close(doneT)
			input := buffer.NewReaderWriter(1)
			output := buffer.NewReaderWriter(1)
			multiplier := New(input, output, 1, 1, 1)

			done := make(chan (struct{}))
			co.ParBegin(
				func() {
					multiplier.Run(done)
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
							runPlayers(done, multipliers)
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

	Context("when running the multiplication algorithm in a partially connected network", func() {

		table := []struct {
			n, k                   uint
			bufferCap, failureRate int
		}{
			// Failure rate = 1%
			{12, 8, 4, 1}, {12, 8, 8, 1}, {12, 8, 16, 1}, {12, 8, 32, 1},
			{24, 16, 8, 1}, {24, 16, 16, 1}, {24, 16, 32, 1}, {24, 16, 64, 1},
			{48, 32, 16, 1}, {48, 32, 32, 1}, {48, 32, 64, 1}, {48, 32, 128, 1},

			// Failure rate = 5%
			{12, 8, 4, 5}, {12, 8, 8, 5}, {12, 8, 16, 5}, {12, 8, 32, 5},
			{24, 16, 8, 5}, {24, 16, 16, 5}, {24, 16, 32, 5}, {24, 16, 64, 5},
			{48, 32, 16, 5}, {48, 32, 32, 5}, {48, 32, 64, 5}, {48, 32, 128, 5},

			// Failure rate = 10%
			{12, 8, 4, 10}, {12, 8, 8, 10}, {12, 8, 16, 10}, {12, 8, 32, 10},
			{24, 16, 8, 10}, {24, 16, 16, 10}, {24, 16, 32, 10}, {24, 16, 64, 10},
			{48, 32, 16, 10}, {48, 32, 32, 10}, {48, 32, 64, 10}, {48, 32, 128, 10},

			// Failure rate = 15%
			{12, 8, 4, 15}, {12, 8, 8, 15}, {12, 8, 16, 15}, {12, 8, 32, 15},
			{24, 16, 8, 15}, {24, 16, 16, 15}, {24, 16, 32, 15}, {24, 16, 64, 15},
			{48, 32, 16, 15}, {48, 32, 32, 15}, {48, 32, 64, 15}, {48, 32, 128, 15},

			// Failure rate = 20%
			{12, 8, 4, 20}, {12, 8, 8, 20}, {12, 8, 16, 20}, {12, 8, 32, 20},
			{24, 16, 8, 20}, {24, 16, 16, 20}, {24, 16, 32, 20}, {24, 16, 64, 20},
			{48, 32, 16, 20}, {48, 32, 32, 20}, {48, 32, 64, 20}, {48, 32, 128, 20},
		}

		for _, entry := range table {
			entry := entry

			Context(fmt.Sprintf("when messages fail to send %v%% of the time", entry.failureRate), func() {
				Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entry.n, entry.k, entry.bufferCap), func() {
					It("should multiply two private variables", func(doneT Done) {
						defer close(doneT)

						mathRand.Seed(time.Now().UnixNano())
						multipliers, inputs, outputs := initPlayers(entry.n, entry.k, entry.bufferCap)

						done := make(chan (struct{}))

						co.ParBegin(
							func() {
								// Run the players until the done channel is closed
								runPlayers(done, multipliers)
							},
							func() {
								defer GinkgoRecover()

								// Initiate smpc multiplication
								initiateMultiply(entry.n, entry.k, inputs)

								failureRate := entry.failureRate
								results := routeMessages(done, inputs, outputs, failureRate)

								successRate := 1.0 - float64(failureRate)*0.01
								successRate = successRate * float64(entry.n)
								if uint(successRate) < entry.k {
									successRate = float64(entry.k)
								}

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
			})
		}
	})
})
