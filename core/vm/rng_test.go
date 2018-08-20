package vm_test

import (
	"crypto/rand"
	"fmt"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/shamir-go"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vm"
)

var _ = Describe("Random number generators", func() {

	Context("when closing the done channel", func() {
		It("should clean up and shutdown", func(doneT Done) {
			defer close(doneT)

			rnger := NewRnger(0, 1, 1, 1)
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

			rnger := NewRnger(0, 1, 1, 1)
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
			n, k      int64
			bufferCap int
		}{
			{3, 2, 0}, {3, 2, 1}, {3, 2, 2}, {3, 2, 4},
			{6, 4, 0}, {6, 4, 1}, {6, 4, 2}, {6, 4, 4},
			{12, 8, 0}, {12, 8, 1}, {12, 8, 2}, {12, 8, 4},
			{24, 16, 0}, {24, 16, 1}, {24, 16, 2}, {24, 16, 4},
			{48, 32, 0}, {48, 32, 1}, {48, 32, 2}, {48, 32, 4},
		}

		for _, entry := range table {
			entry := entry
			Context(fmt.Sprintf("when n = %v and k = %v and each player has a buffer capacity of %v", entry.n, entry.k, entry.bufferCap), func() {
				It("should produce consistent global random number shares", func(doneT Done) {
					defer close(doneT)

					// Initialis the players
					rngers := make([]Rnger, entry.n)
					inputs := make([]chan RngInputMessage, entry.n)
					outputs := make([]chan RngOutputMessage, entry.n)
					for i := int64(0); i < entry.n; i++ {
						rngers[i] = NewRnger(uint64(i), entry.n, entry.k, entry.bufferCap)
						inputs[i] = make(chan RngInputMessage, entry.bufferCap)
						outputs[i] = make(chan RngOutputMessage, entry.bufferCap)
					}

					// Nonce that will be used to identify the secure random
					// number
					nonce := make([]byte, 32)
					n, err := rand.Read(nonce[:])
					Expect(n).To(Equal(32))
					Expect(err).To(BeNil())

					done := make(chan (struct{}))
					co.ParBegin(
						func() {
							// Run the players until the done channel is closed
							co.ParForAll(rngers, func(i int) {
								rngers[i].Run(done, inputs[i], outputs[i])
							})
						},
						func() {
							// Instruct all players to generate a random number
							co.ParForAll(inputs, func(i int) {
								inputs[i] <- GenerateRn{Nonce: nonce[:]}
							})
						},
						func() {
							defer close(done)

							globalRnShares := make(shamir.Shares, entry.n)
							co.ParForAll(outputs, func(i int) {
								defer GinkgoRecover()

								numLocalRnShares := int64(0)
								numGlobalRnShares := int64(0)

								// Expect exactly n messages from each player;
								// a LocalRnShare for each other player and a
								// GlobalRnShare of the secure random number
								// that is generated
								for n := int64(0); n < entry.n; n++ {
									message, ok := <-outputs[i]
									Expect(ok).To(BeTrue())

									switch message := message.(type) {
									case LocalRnShare:
										// Route the LocalRnShare to the
										// appropriate player
										Expect(message.Nonce).To(Equal(nonce))
										Expect(message.From).To(Equal(uint64(i)))
										inputs[message.To] <- message
										numLocalRnShares++

									case GlobalRnShare:
										Expect(message.Nonce).To(Equal(nonce))
										globalRnShares[i] = message.Share
										numGlobalRnShares++
									}
								}

								// Expect a LocalRnShare from each player for
								// every other player
								Expect(numLocalRnShares).To(Equal(entry.n - 1))
								Expect(numGlobalRnShares).To(Equal(int64(1)))
							})

							// Reconstruct the secret using different subsets
							// of shares and expect that all reconstructed
							// secrets are equal
							secret := shamir.Join(globalRnShares)
							for i := int64(0); i < entry.n-entry.k; i++ {
								kSecret := shamir.Join(globalRnShares[i : i+entry.k])
								Expect(secret).To(Equal(kSecret))
							}
						})
				})
			})
		}

	})

})
