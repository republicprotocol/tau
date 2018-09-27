package rng_test

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/taskutils"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/pedersen"
	"github.com/republicprotocol/oro-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/vm/rng"
)

var _ = Describe("Random number generators", func() {

	p := big.NewInt(8589934583)
	q := big.NewInt(4294967291)
	g := algebra.NewFpElement(big.NewInt(592772542), p)
	h := algebra.NewFpElement(big.NewInt(4799487786), p)
	fp := algebra.NewField(q)
	scheme := pedersen.New(g, h, fp)

	init := func(n, k uint64, cap int) task.Tasks {
		ts := make(task.Tasks, n)
		for i := range ts {
			ts[i] = New(scheme, uint64(i)+1, n, k, cap)
		}
		return ts
	}

	run := func(done <-chan struct{}, results chan<- Result, ts task.Tasks, simulatedFailureRate float64, simulatedFailureLimit int) {

		rnSharesFailures := 0
		proposeRnSharesFailures := 0

		task.New(task.NewIO(1024), task.NewReducer(func(message task.Message) task.Message {

			switch message := message.(type) {

			case RnShares:
				modifiedSimulatedFailureLimit := 1
				if rnSharesFailures >= simulatedFailureLimit {
					modifiedSimulatedFailureLimit = 0
				}
				rnSharesFailures += taskutils.RouteMessage(done, message, task.Tasks{ts[0]}, simulatedFailureRate, modifiedSimulatedFailureLimit)

			case ProposeRnShare:
				modifiedSimulatedFailureLimit := 1
				if proposeRnSharesFailures >= simulatedFailureLimit {
					modifiedSimulatedFailureLimit = 0
				}
				proposeRnSharesFailures += taskutils.RouteMessage(done, message, task.Tasks{ts[message.To-1]}, simulatedFailureRate, modifiedSimulatedFailureLimit)

			case Result:
				select {
				case <-done:
				case results <- message:
				}

			default:
			}
			return nil

		}), ts...).Run(done)
	}

	verifyShares := func(shares shamir.Shares, k uint64) (algebra.FpElement, error) {
		secret, err := shamir.Join(shares)
		if err != nil {
			return secret, err
		}
		for i := 0; i < len(shares); i++ {
			rand.Shuffle(len(shares), func(i, j int) {
				shares[i], shares[j] = shares[j], shares[i]
			})
			kSecret, err := shamir.Join(shares[:k])
			if err != nil {
				return secret, err
			}
			if !secret.Eq(kSecret) {
				return secret, errors.New("malformed shares")
			}
		}
		return secret, nil
	}

	sendGenerateRnMessage := func(msgid task.MessageID, tasks []task.Task) {
		co.ParForAll(tasks, func(i int) {
			tasks[i].IO().InputWriter() <- NewGenerateRn(msgid, 1)
		})
	}

	sendGenerateRnZeroMessage := func(msgid task.MessageID, tasks []task.Task) {
		co.ParForAll(tasks, func(i int) {
			tasks[i].IO().InputWriter() <- NewGenerateRnZero(msgid, 1)
		})
	}

	sendGenerateRnTupleMessage := func(msgid task.MessageID, tasks []task.Task) {
		co.ParForAll(tasks, func(i int) {
			tasks[i].IO().InputWriter() <- NewGenerateRnTuple(msgid, 1)
		})
	}

	runGenerateRn := func(n, k uint64, cap int, failureRate float64, failureRateLimit int, isZero bool) (result algebra.FpElement, err error) {

		rngers := init(n, k, cap)
		done := make(chan struct{})
		results := make(chan Result)

		msgid := taskutils.RandomMessageID()
		if isZero {
			sendGenerateRnZeroMessage(msgid, rngers)
		} else {
			sendGenerateRnMessage(msgid, rngers)
		}
		sharesResult := map[uint64]shamir.Share{}

		co.ParBegin(
			func() {
				run(done, results, rngers, failureRate, failureRateLimit)
			},
			func() {
				defer close(done)
				for result := range results {
					share := result.Sigmas[0].Share()
					sharesResult[share.Index()] = share
					if len(sharesResult) == int(n)-failureRateLimit {
						break
					}
				}
			})

		shares := shamir.Shares{}
		for _, share := range sharesResult {
			shares = append(shares, share)
		}

		result, err = verifyShares(shares, (k+1)/2)
		return
	}

	runGenerateRnTuple := func(n, k uint64, cap int, failureRate float64, failureRateLimit int) (ρResult, σResult algebra.FpElement, err error) {

		rngers := init(n, k, cap)
		done := make(chan struct{})
		results := make(chan Result)

		sendGenerateRnTupleMessage(taskutils.RandomMessageID(), rngers)
		ρSharesResult := map[uint64]shamir.Share{}
		σSharesResult := map[uint64]shamir.Share{}

		co.ParBegin(
			func() {
				run(done, results, rngers, failureRate, failureRateLimit)
			},
			func() {
				defer close(done)
				for result := range results {
					ρShare := result.Rhos[0].Share()
					σShare := result.Sigmas[0].Share()
					ρSharesResult[ρShare.Index()] = ρShare
					σSharesResult[σShare.Index()] = σShare
					if len(ρSharesResult) == int(n)-failureRateLimit {
						break
					}
				}
			})

		ρShares := shamir.Shares{}
		σShares := shamir.Shares{}
		for i := range ρSharesResult {
			ρShares = append(ρShares, ρSharesResult[i])
			σShares = append(σShares, σSharesResult[i])
		}
		ρResult, err = verifyShares(ρShares, k)
		if err != nil {
			return
		}
		σResult, err = verifyShares(σShares, (k+1)/2)
		return
	}

	BeforeEach(func() {
		rand.Seed(time.Now().Unix())
	})

	Context("when closing the done channel", func() {

		tableNK := []struct {
			n, k uint64
		}{
			{1, 1},
			{3, 2},
			{6, 4},
			{12, 8},
			{24, 16},
		}
		tableCap := []struct {
			cap int
		}{
			{1},
			{2},
			{4},
			{8},
			{16},
		}

		for _, entryNK := range tableNK {
			entryNK := entryNK

			for _, entryCap := range tableCap {
				entryCap := entryCap

				Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entryNK.n, entryNK.k, entryCap.cap), func() {
					It("should shutdown and clean up", func(doneT Done) {
						defer close(doneT)

						rngers := init(entryNK.n, entryNK.k, entryCap.cap)
						done := make(chan struct{})
						results := make(chan Result)

						co.ParBegin(
							func() {
								run(done, results, rngers, 0.0, 0)
							},
							func() {
								close(done)
							})
					})
				})
			}
		}
	})

	Context("when generating a random number in a fully connected network", func() {

		tableNK := []struct {
			n, k uint64
		}{
			{1, 1},
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
					Context("when generation a random number", func() {
						It("should use secure multiparty computations to generate a random number", func(doneT Done) {
							defer close(doneT)

							_, err := runGenerateRn(entryNK.n, entryNK.k, entryCap.cap, 0.0, 0, false)
							Expect(err).To(BeNil())
						})
					})

					Context("when generation a random zero", func() {
						It("should use secure multiparty computations to generate a random zero", func(doneT Done) {
							defer close(doneT)

							zero, err := runGenerateRn(entryNK.n, entryNK.k, entryCap.cap, 0.0, 0, true)
							Expect(err).To(BeNil())
							Expect(zero.IsZero()).To(BeTrue())
						})
					})

					Context("when generation a random tuple", func() {
						It("should use secure multiparty computations to generate a random tuple", func(doneT Done) {
							defer close(doneT)

							ρResult, σResult, err := runGenerateRnTuple(entryNK.n, entryNK.k, entryCap.cap, 0.0, 0)
							Expect(err).To(BeNil())
							Expect(ρResult.Eq(σResult)).To(BeTrue())
						})
					})
				})
			}
		}
	})

	Context("when generating a random number in a partially connected network", func() {

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
			failureRate float64
		}{
			{0.01}, {0.05}, {0.10}, {0.15}, {0.20}, {0.25}, {0.30}, {0.35}, {0.40}, {0.45}, {0.50},
		}

		for _, entryNK := range tableNK {
			entryNK := entryNK

			for _, entryCap := range tableCap {
				entryCap := entryCap

				for _, entryFailureRate := range tableFailureRate {
					entryFailureRate := entryFailureRate

					Context(fmt.Sprintf("when the failure rate of the network is %v%%", entryFailureRate.failureRate), func() {
						Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entryNK.n, entryNK.k, entryCap.cap), func() {

							Context("when generation a random number", func() {
								It("should use secure multiparty computations to generate a random number", func(doneT Done) {
									defer close(doneT)

									_, err := runGenerateRn(entryNK.n, entryNK.k, entryCap.cap, entryFailureRate.failureRate, int(entryNK.n-entryNK.k), false)
									Expect(err).To(BeNil())
								})

								It("should result in consistent shares", func(doneT Done) {
									defer close(doneT)

									_, err := runGenerateRn(entryNK.n, entryNK.k, entryCap.cap, entryFailureRate.failureRate, int(entryNK.n-entryNK.k)/2, false)
									Expect(err).To(BeNil())
								})
							})

							Context("when generation a random zero", func() {
								It("should use secure multiparty computations to generate a random zero", func(doneT Done) {
									defer close(doneT)

									zero, err := runGenerateRn(entryNK.n, entryNK.k, entryCap.cap, entryFailureRate.failureRate, int(entryNK.n-entryNK.k), true)
									Expect(err).To(BeNil())
									Expect(zero.IsZero()).To(BeTrue())
								})

								It("should result in consistent shares", func(doneT Done) {
									defer close(doneT)

									zero, err := runGenerateRn(entryNK.n, entryNK.k, entryCap.cap, entryFailureRate.failureRate, int(entryNK.n-entryNK.k)/2, true)
									Expect(err).To(BeNil())
									Expect(zero.IsZero()).To(BeTrue())
								})
							})

							Context("when generation a random tuple", func() {
								It("should use secure multiparty computations to generate a random tuple", func(doneT Done) {
									defer close(doneT)
									defer GinkgoRecover()

									ρResult, σResult, err := runGenerateRnTuple(entryNK.n, entryNK.k, entryCap.cap, entryFailureRate.failureRate, int(entryNK.n-entryNK.k))
									Expect(err).To(BeNil())
									Expect(ρResult.Eq(σResult)).To(BeTrue())
								})

								It("should result in consistent shares", func(doneT Done) {
									defer close(doneT)

									ρResult, σResult, err := runGenerateRnTuple(entryNK.n, entryNK.k, entryCap.cap, entryFailureRate.failureRate, int(entryNK.n-entryNK.k)/4)
									Expect(err).To(BeNil())
									Expect(ρResult.Eq(σResult)).To(BeTrue())
								})
							})
						})
					})
				}
			}
		}
	})

	Context("when creating messages", func() {
		It("should implement the message interface for all messages", func() {
			GenerateRn{}.IsMessage()
			GenerateRnZero{}.IsMessage()
			GenerateRnTuple{}.IsMessage()
			RnShares{}.IsMessage()
			ProposeRnShare{}.IsMessage()
			Result{}.IsMessage()
		})
	})
})
