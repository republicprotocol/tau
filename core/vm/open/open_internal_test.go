package open

import (
	"encoding/binary"
	"math/big"
	"math/rand"
	"time"

	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Open", func() {
	P := big.NewInt(8589934583)
	// Q := big.NewInt(4294967291)
	Field := algebra.NewField(P)
	// OtherField := algebra.NewField(Q)
	BufferLimit := 64

	tableNK := []struct {
		n, k uint64
	}{
		{3, 2},
		{6, 4},
		{12, 8},
		{24, 16},
	}

	idFromUint64 := func(n uint64) [32]byte {
		ret := [32]byte{0x0}
		id := make([]byte, 8)
		binary.LittleEndian.PutUint64(id, n)
		for i, b := range id {
			ret[i] = b
		}
		return ret
	}

	for _, entryNK := range tableNK {
		entryNK := entryNK

		Context("when trying to open", func() {
			Context("when there are insufficient messages", func() {
				Specify("output messages should be nil", func() {

					opener := newOpener(entryNK.n, entryNK.k)
					secret := Field.Random()
					poly := algebra.NewRandomPolynomial(Field, uint(entryNK.k)-1, secret)
					shares := shamir.Split(poly, entryNK.n)

					for i := 0; i < 100; i++ {
						msgCount := rand.Int63n(int64(entryNK.k))
						id := idFromUint64(uint64(i))

						for j := int64(0); j < msgCount; j++ {
							msg := opener.reduce(NewOpen(id, shares[j]))
							Expect(msg).To(BeNil())
						}
					}
				})
			})

			Context("when there has not been a signal to open", func() {
				It("output messages should be nil", func() {

					opener := newOpener(entryNK.n, entryNK.k)
					secret := Field.Random()
					poly := algebra.NewRandomPolynomial(Field, uint(entryNK.k)-1, secret)
					shares := shamir.Split(poly, entryNK.n)

					for i := 0; i < 100; i++ {
						msgCount := rand.Int63n(int64(entryNK.n-entryNK.k+1)) + int64(entryNK.k)
						id := idFromUint64(uint64(i))

						for j := int64(0); j < msgCount; j++ {
							msg := opener.reduce(NewOpen(id, shares[j]))
							Expect(msg).To(BeNil())
						}
					}
				})
			})

			Context("when there has already been an opening for the message id", func() {
				It("output messages should be nil", func() {

					opener := New(entryNK.n, entryNK.k, entryCap.cap).(*opener)
					secret := Field.Random()
					poly := algebra.NewRandomPolynomial(Field, uint(entryNK.k)-1, secret)
					shares := shamir.Split(poly, entryNK.n)

					for i := 0; i < 100; i++ {
						defer GinkgoRecover()

						msgCount := rand.Int63n(int64(entryNK.n + 1))
						id := idFromUint64(uint64(i))
						dummyResult := NewResult(id, Field.Random())
						opener.results[id] = dummyResult

						for j := int64(0); j < msgCount; j++ {
							msg := NewOpen(id, shares[j])

							opener.tryOpen(msg)
						}
					}

					ioDidFlush := ioFlush(opener.io)
					time.Sleep(10 * time.Millisecond)

					Expect(ioDidFlush).ToNot(BeClosed())
				})
			})

			// Context("when there has been an open signal and no result has yet been computed and there are k messages", func() {
			// 	Context("when not all shares are in the same field", func() {
			// 		It("should write an error to the output", func() {

			// 			ioTest := task.NewIO(entryCap.cap)

			// 			opener := New(entryNK.n, entryNK.k, entryCap.cap).(*opener)
			// 			poly := algebra.NewRandomPolynomial(Field, uint(entryNK.k)-1, Field.NewInField(big.NewInt(0)))
			// 			shares := shamir.Split(poly, entryNK.n)
			// 			badShare := shamir.New(1, OtherField.Random())

			// 			for i := 0; i < 100; i++ {
			// 				defer GinkgoRecover()

			// 				id := idFromUint64(uint64(i))
			// 				dummySignal := NewSignal(id, badShare)
			// 				opener.signals[id] = dummySignal

			// 				msg := NewOpen(id, badShare)
			// 				opener.tryOpen(msg)

			// 				for j := uint64(1); j < entryNK.k; j++ {
			// 					msg := NewOpen(id, shares[j])

			// 					opener.tryOpen(msg)
			// 				}

			// 				opener.io.Flush(nil)
			// 				message, okFlush := ioTest.Flush(nil, opener.Channel())
			// 				Expect(okFlush).To(BeTrue())

			// 				_, okType := message.(task.Error)
			// 				Expect(okType).To(BeTrue())
			// 			}
			// 		})
			// 	})

			// 	Context("when the shares are correct", func() {
			// 		It("should write the secret to the output", func() {

			// 			ioTest := task.NewIO(entryCap.cap)

			// 			opener := New(entryNK.n, entryNK.k, entryCap.cap).(*opener)
			// 			secret := Field.Random()
			// 			poly := algebra.NewRandomPolynomial(Field, uint(entryNK.k)-1, secret)
			// 			shares := shamir.Split(poly, entryNK.n)

			// 			for i := 0; i < 100; i++ {
			// 				defer GinkgoRecover()

			// 				id := idFromUint64(uint64(i))
			// 				dummySignal := NewSignal(id, shares[0])
			// 				opener.signals[id] = dummySignal

			// 				for j := uint64(0); j < entryNK.k; j++ {
			// 					msg := NewOpen(id, shares[j])

			// 					opener.tryOpen(msg)
			// 				}

			// 				opener.io.Flush(nil)
			// 				message, okFlush := ioTest.Flush(nil, opener.Channel())
			// 				Expect(okFlush).To(BeTrue())

			// 				res, okType := message.(Result)
			// 				Expect(okType).To(BeTrue())
			// 				Expect(res.MessageID).To(BeEquivalentTo(id))
			// 				Expect(res.Value.Eq(secret)).To(BeTrue())
			// 			}
			// 		})
			// 	})
			// })
		})
	}

})
