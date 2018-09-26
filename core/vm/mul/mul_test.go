package mul_test

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
	"github.com/republicprotocol/oro-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/vm/mul"
)

var _ = Describe("Multipliers", func() {

	fp := algebra.NewField(big.NewInt(8113765242226142771))

	init := func(n, k uint64, cap int) task.Tasks {
		ts := make(task.Tasks, n)
		for i := range ts {
			ts[i] = New(uint64(i)+1, n, k, cap)
		}
		return ts
	}

	run := func(done <-chan struct{}, results chan<- Result, ts task.Tasks, simulatedFailureRate float64, simulatedFailureLimit int) {
		task.New(task.NewIO(1024), task.NewReducer(func(message task.Message) task.Message {

			switch message := message.(type) {

			case Result:
				select {
				case <-done:
				case results <- message:
				}

			default:
				taskutils.RouteMessage(done, message, ts, simulatedFailureRate, simulatedFailureLimit)
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

	sendMulMessage := func(messageID task.MessageID, tasks []task.Task, n, k uint64) algebra.FpElement {
		x := fp.Random()
		y := fp.Random()
		r := fp.Random()

		xPoly := algebra.NewRandomPolynomial(fp, uint(k+1)/2-1, x)
		xShares := shamir.Split(xPoly, n)

		yPoly := algebra.NewRandomPolynomial(fp, uint(k+1)/2-1, y)
		yShares := shamir.Split(yPoly, n)

		ρPoly := algebra.NewRandomPolynomial(fp, uint(k)-1, r)
		ρShares := shamir.Split(ρPoly, n)

		σPoly := algebra.NewRandomPolynomial(fp, uint(k+1)/2-1, r)
		σShares := shamir.Split(σPoly, n)

		co.ParForAll(tasks, func(i int) {
			// Send the mul twice, with a delay, to trigger the short-circuit result
			tasks[i].IO().InputWriter() <- NewMul(messageID, shamir.Shares{xShares[i]}, shamir.Shares{yShares[i]}, shamir.Shares{ρShares[i]}, shamir.Shares{σShares[i]})
			tasks[i].IO().InputWriter() <- NewMul(messageID, shamir.Shares{xShares[i]}, shamir.Shares{yShares[i]}, shamir.Shares{ρShares[i]}, shamir.Shares{σShares[i]})
		})

		return x.Mul(y)
	}

	runMultiplication := func(n, k uint64, cap int, failureRate float64, failureRateLimit int) (expectedResult, gotResult algebra.FpElement, err error) {

		multipliers := init(n, k, cap)
		done := make(chan struct{})
		results := make(chan Result)

		expectedResult = sendMulMessage(taskutils.RandomMessageID(), multipliers, n, k)
		sharesResult := map[uint64]shamir.Share{}

		co.ParBegin(
			func() {
				run(done, results, multipliers, failureRate, failureRateLimit)
			},
			func() {
				defer close(done)
				for result := range results {
					sharesResult[result.Shares[0].Index()] = result.Shares[0]
					if len(sharesResult) == int(n)-2*failureRateLimit {
						break
					}
				}
			})

		shares := shamir.Shares{}
		for _, share := range sharesResult {
			shares = append(shares, share)
		}

		gotResult, err = verifyShares(shares, (k+1)/2)
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

						multipliers := init(entryNK.n, entryNK.k, entryCap.cap)
						done := make(chan struct{})
						results := make(chan Result)

						co.ParBegin(
							func() {
								run(done, results, multipliers, 0.0, 0)
							},
							func() {
								close(done)
							})
					})
				})
			}
		}
	})

	Context("when multiplying in a fully connected network", func() {

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
					It("should use secure multiparty computations to multiply", func(doneT Done) {
						defer close(doneT)

						expectedResult, gotResult, err := runMultiplication(entryNK.n, entryNK.k, entryCap.cap, 0.0, 0)
						Expect(err).To(BeNil())
						Expect(expectedResult.Eq(gotResult)).To(BeTrue())
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

							It("should use secure multiparty computations to multiply", func(doneT Done) {
								defer close(doneT)

								expectedResult, gotResult, err := runMultiplication(entryNK.n, entryNK.k, entryCap.cap, entryFailureRate.failureRate, int(entryNK.n-entryNK.k))
								Expect(err).To(BeNil())
								Expect(expectedResult.Eq(gotResult)).To(BeTrue())
							})

							It("should result in consistent shares", func(doneT Done) {
								defer close(doneT)

								expectedResult, gotResult, err := runMultiplication(entryNK.n, entryNK.k, entryCap.cap, entryFailureRate.failureRate, (int(entryNK.n-entryNK.k)+1)/2)
								Expect(err).To(BeNil())
								Expect(expectedResult.Eq(gotResult)).To(BeTrue())
							})
						})
					})
				}
			}
		}
	})
})
