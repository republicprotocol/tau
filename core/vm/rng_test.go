package vm_test

import (
	"log"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/shamir-go"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vm"
)

var _ = Describe("Random number generator", func() {

	const (
		BufferCap = 0
	)

	Context("when generating random numbers", func() {

		It("should shutdown when the done channel is closed", func(doneT Done) {
			defer close(doneT)

			rnger := NewRnger(0, 1, 1, BufferCap)

			done := make(chan (struct{}))
			co.ParBegin(
				func() {
					rnger.Run(done, nil, nil)
				},
				func() {
					close(done)
				})
		})

		Context("when using a single machine", func() {
			It("should output a global random number immediately", func(doneT Done) {
				defer close(doneT)

				rnger := NewRnger(0, 1, 1, BufferCap)
				input := make(chan RngMessage)
				output := make(chan RngResult)

				done := make(chan (struct{}))
				co.ParBegin(
					func() {
						rnger.Run(done, input, output)
					},
					func() {
						input <- Gen{Nonce: []byte{1}}
						result, ok := <-output
						Expect(ok).To(BeTrue())
						globalRngShare, ok := result.(GlobalRngShare)
						Expect(ok).To(BeTrue())
						Expect(globalRngShare.Nonce).To(Equal([]byte{1}))
						close(done)
					})
			})
		})

		Context("when using multiple machines", func() {
			It("should output a global random number after receiving all local shares", func(doneT Done) {
				defer close(doneT)

				numRngers := int64(32)
				n := numRngers
				k := n / 2
				rngers := make([]Rnger, numRngers)
				inputs := make([]chan RngMessage, numRngers)
				outputs := make([]chan RngResult, numRngers)
				for i := range rngers {
					rngers[i] = NewRnger(int64(i), n, k, BufferCap)
					inputs[i] = make(chan RngMessage)
					outputs[i] = make(chan RngResult)
				}

				done := make(chan (struct{}))
				co.ParBegin(
					func() {
						co.ParForAll(rngers, func(i int) {
							rngers[i].Run(done, inputs[i], outputs[i])
						})
					},
					func() {
						defer GinkgoRecover()

						co.ParForAll(inputs, func(i int) {
							inputs[i] <- Gen{Nonce: []byte{1}}
						})

						globalRngShares := make([]GlobalRngShare, numRngers)
						co.ParForAll(outputs, func(i int) {
							defer GinkgoRecover()

							numLocalRngShares := int64(0)

							for j := int64(0); j < n; j++ {
								result, ok := <-outputs[i]
								Expect(ok).To(BeTrue())

								switch result.(type) {
								case LocalRngShare:
									localRngShare, ok := result.(LocalRngShare)
									Expect(ok).To(BeTrue())
									Expect(localRngShare.Nonce).To(Equal([]byte{1}))
									Expect(localRngShare.From).To(Equal(int64(i)))
									inputs[localRngShare.J] <- localRngShare
									numLocalRngShares++

								case GlobalRngShare:
									globalRngShare, ok := result.(GlobalRngShare)
									Expect(ok).To(BeTrue())
									Expect(globalRngShare.Nonce).To(Equal([]byte{1}))
									globalRngShares[i] = globalRngShare
								}
							}

							Expect(numLocalRngShares).To(Equal(numRngers - 1))
						})

						var prevSecret *uint64
						for i := int64(0); i < n-k; i++ {
							subset := globalRngShares[i : i+k]
							shares := shamir.Shares{}
							for _, elem := range subset {
								shares = append(shares, elem.Share)
							}

							log.Printf("%v =>\n\tshares: %v", i, shares)

							secret := shamir.Join(shares)
							if prevSecret == nil {
								prevSecret = &secret
							} else {
								Expect(secret).To(Equal(*prevSecret))
							}
						}

						close(done)
					})
			})
		})

	})

})
