package mul_test

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	mathRand "math/rand"
	"time"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/vm/mul"
)

var _ = Describe("Multipliers", func() {

	fp := algebra.NewField(big.NewInt(8113765242226142771))

	randomMessageID := func() (task.MessageID, error) {
		mathRand.Seed(time.Now().UnixNano())
		messageID := task.MessageID{}
		_, err := rand.Read(messageID[:])
		return messageID, err
	}

	init := func(n, k uint64, cap int) []task.Task {
		tasks := make([]task.Task, n)
		for i := 0; i < len(tasks); i++ {
			tasks[i] = New(uint64(i)+1, n, k, cap)
		}
		return tasks
	}

	run := func(done <-chan struct{}, tasks []task.Task) {
		co.ParForAll(tasks, func(i int) {
			tasks[i].Run(done)
		})
	}

	verifyShares := func(shares shamir.Shares, n, k uint64) (algebra.FpElement, error) {
		secret, err := shamir.Join(shares)
		if err != nil {
			return secret, err
		}
		for i := uint64(0); i < n-k; i++ {
			kSecret, err := shamir.Join(shares[i : i+k])
			if err != nil {
				return secret, err
			}
			if !secret.Eq(kSecret) {
				return secret, errors.New("malformed shares")
			}
		}
		return secret, nil
	}

	multiply := func(messageID task.MessageID, tasks []task.Task, n, k uint64) algebra.FpElement {
		x := fp.Random()
		y := fp.Random()
		r := fp.Random()

		xPoly := algebra.NewRandomPolynomial(fp, uint(k/2)-1, x)
		xShares := shamir.Split(xPoly, n)

		yPoly := algebra.NewRandomPolynomial(fp, uint(k/2)-1, y)
		yShares := shamir.Split(yPoly, n)

		ρPoly := algebra.NewRandomPolynomial(fp, uint(k)-1, r)
		ρShares := shamir.Split(ρPoly, n)

		σPoly := algebra.NewRandomPolynomial(fp, uint(k/2)-1, r)
		σShares := shamir.Split(σPoly, n)

		co.ParForAll(tasks, func(i int) {
			tasks[i].Channel().Send(NewSignalMul(messageID, xShares[i], yShares[i], ρShares[i], σShares[i]))
		})

		return x.Mul(y)
	}

	routeMessage := func(done <-chan struct{}, message task.Message, tasks []task.Task, failureRate int) {
		for _, task := range tasks {
			if mathRand.Intn(100) < failureRate {
				// Simluate an unstable network connection and randomly drop
				// messages
				continue
			}
			task.Channel().Send(message)
		}
	}

	routeMessages := func(done <-chan struct{}, tasks []task.Task, failureRate int) <-chan Result {

		io := task.NewIO(len(tasks))
		results := make(chan Result, len(tasks))

		go func() {
			defer close(results)

			channels := make([]task.Channel, len(tasks))
			for i := range channels {
				channels[i] = tasks[i].Channel()
			}

			for {
				message, ok := io.Flush(done, channels...)
				if !ok {
					return
				}
				if message != nil {
					switch message := message.(type) {
					case Result:
						select {
						case <-done:
						case results <- message:
						}
					case task.Error:
						panic(message)
					default:
						routeMessage(done, message, tasks, failureRate)
					}
				}
			}
		}()

		return results
	}

	Context("when closing the done channel", func() {

		table := []struct {
			n uint64
		}{
			{1}, {2}, {4}, {8}, {16}, {32}, {64},
		}

		for _, entry := range table {
			entry := entry

			Context(fmt.Sprintf("when n = %v", entry.n), func() {
				It("should clean up and shutdown", func(doneT Done) {
					defer close(doneT)

					tasks := init(entry.n, 1, 1)
					done := make(chan struct{})

					co.ParBegin(
						func() {
							run(done, tasks)
						},
						func() {
							close(done)
						})
				})
			})
		}
	})

	Context("when multiplying in a fully connected network", func() {

		tableNK := []struct {
			n, k uint64
		}{
			{3, 2},
			{6, 4},
			{12, 8},
			{24, 16},
		}
		tableCap := []struct {
			cap int
		}{
			{64},
			{128},
			{256},
			{512},
			{1024},
		}

		for _, entryNK := range tableNK {
			entryNK := entryNK

			for _, entryCap := range tableCap {
				entryCap := entryCap

				Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entryNK.n, entryNK.k, entryCap.cap), func() {
					It("should multiply two field elements", func(doneT Done) {
						defer close(doneT)

						mathRand.Seed(time.Now().UnixNano())
						multipliers := init(entryNK.n, entryNK.k, entryCap.cap)

						done := make(chan struct{})
						co.ParBegin(
							func() {
								run(done, multipliers)
							},
							func() {
								defer GinkgoRecover()

								messageID, err := randomMessageID()
								Expect(err).To(BeNil())
								expectedResult := multiply(messageID, multipliers, entryNK.n, entryNK.k)

								results := routeMessages(done, multipliers, 0)
								mulShares := map[uint64]shamir.Share{}
								func() {
									// Close the done channel when we are
									// finished collecting results
									defer close(done)

									// Expect to collect a result from each Rnger
									for result := range results {
										mulShares[result.Share.Index()] = result.Share
										if len(mulShares) == int(entryNK.n) {
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
								result, err := verifyShares(shares, entryNK.n, entryNK.k)
								Expect(err).To(BeNil())
								Expect(result.Eq(expectedResult)).To(BeTrue())
							})
					})
				})
			}
		}
	})

	Context("when multiplying in a partially connected network", func() {

		tableNK := []struct {
			n, k uint64
		}{
			{3, 2},
			{6, 4},
			{12, 8},
			{24, 16},
		}
		tableCap := []struct {
			cap int
		}{
			{64},
			{128},
			{256},
			{512},
			{1024},
		}
		tableFailureRate := []struct {
			failureRate int
		}{
			{1}, {5}, {10}, {15}, {20},
		}

		for _, entryNK := range tableNK {
			entryNK := entryNK

			for _, entryCap := range tableCap {
				entryCap := entryCap

				for _, entryFailureRate := range tableFailureRate {
					entryFailureRate := entryFailureRate

					Context(fmt.Sprintf("when the failure rate of the network is %v%%", entryFailureRate.failureRate), func() {
						Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entryNK.n, entryNK.k, entryCap.cap), func() {
							It("should multiply two private variables", func(doneT Done) {
								defer close(doneT)

								mathRand.Seed(time.Now().UnixNano())
								multipliers := init(entryNK.n, entryNK.k, entryCap.cap)

								done := make(chan struct{})
								co.ParBegin(
									func() {
										run(done, multipliers)
									},
									func() {
										defer GinkgoRecover()

										messageID, err := randomMessageID()
										Expect(err).To(BeNil())
										expectedMul := multiply(messageID, multipliers, entryNK.n, entryNK.k)

										results := routeMessages(done, multipliers, entryFailureRate.failureRate)
										mulShares := map[uint64]shamir.Share{}
										func() {
											// Close the done channel when we are
											// finished collecting results
											defer close(done)

											// Expect to collect a result from each Rnger
											for result := range results {
												mulShares[result.Share.Index()] = result.Share
												if len(mulShares) == int(entryNK.k) {
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
										mul, err := verifyShares(shares, entryNK.k, entryNK.k/2)
										Expect(err).To(BeNil())
										Expect(mul.Eq(expectedMul)).To(BeTrue())
									})
							})
						})
					})
				}
			}
		}
	})
})
