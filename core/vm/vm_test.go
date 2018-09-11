package vm_test

import (
	"fmt"
	"log"
	"math/big"
	"math/rand"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/oro-go/core/buffer"
	"github.com/republicprotocol/oro-go/core/stack"
	"github.com/republicprotocol/oro-go/core/vm/process"
	"github.com/republicprotocol/oro-go/core/vm/rng"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/pedersen"
	"github.com/republicprotocol/oro-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/vm"
)

var _ = Describe("Virtual Machine", func() {

	P := big.NewInt(8589934583)
	Q := big.NewInt(4294967291)
	G := algebra.NewFpElement(big.NewInt(592772542), P)
	H := algebra.NewFpElement(big.NewInt(4799487786), P)
	SecretField := algebra.NewField(Q)
	PedersenScheme := pedersen.New(G, H, SecretField)
	BufferLimit := 64

	type TestResult struct {
		result Result
		from   uint64
	}

	// initVMs for a secure multi-party computation network. The VMs will
	// communicate to execute processes.
	initVMs := func(n, k, leader uint, cap int) ([]VM, []buffer.ReaderWriter, []buffer.ReaderWriter) {
		// Initialize the VMs
		ins := make([]buffer.ReaderWriter, n)
		outs := make([]buffer.ReaderWriter, n)
		vms := make([]VM, n)
		for i := uint(0); i < n; i++ {
			ins[i] = buffer.NewReaderWriter(cap)
			outs[i] = buffer.NewReaderWriter(cap)
			vms[i] = New(ins[i], outs[i], uint64(i), uint64(leader), PedersenScheme, n, k, cap)
		}
		return vms, ins, outs
	}

	// runVMs until the done channel is closed.
	runVMs := func(done <-chan struct{}, vms []VM) {
		co.ParForAll(vms, func(i int) {
			vms[i].Run(done)
		})
	}

	routeMessages := func(done <-chan struct{}, ins, outs []buffer.ReaderWriter) <-chan TestResult {
		results := make(chan TestResult, len(outs))

		go func() {
			defer close(results)

			co.ParForAll(outs, func(i int) {
				defer GinkgoRecover()
				var message buffer.Message
				var ok bool

				for {
					select {
					case <-done:
						return
					case message, ok = <-outs[i].Reader():
						if !ok {
							return
						}
					}

					switch message := message.(type) {
					case RemoteProcedureCall:
						switch message := message.Message.(type) {

						case rng.ProposeRn:
							ins[message.To] <- NewRemoteProcedureCall(message)

						case rng.LocalRnShares:
							ins[message.To] <- NewRemoteProcedureCall(message)

						case rng.ProposeGlobalRnShare:
							ins[message.To] <- NewRemoteProcedureCall(message)

						default:
							for _, in := range ins {
								in <- NewRemoteProcedureCall(message)
							}
						}

					case Result:
						select {
						case <-done:
						case results <- TestResult{message, uint64(i)}:
						}

					default:
						log.Fatalf("unexpected message type %T", message)
					}
				}
			})
		}()

		return results
	}

	Context("when running the virtual machines in a fully connected network", func() {

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
				It("should add public numbers", func(doneT Done) {
					defer close(doneT)

					done := make(chan (struct{}))
					vms, ins, outs := initVMs(entry.n, entry.k, 0, entry.bufferCap)
					co.ParBegin(
						func() {
							runVMs(done, vms)
						},
						func() {
							defer GinkgoRecover()
							defer close(done)

							id := [31]byte{0x69}
							a, b := SecretField.Random(), SecretField.Random()
							valueA, valueB := process.NewValuePublic(a), process.NewValuePublic(b)
							expected := process.NewValuePublic(a.Add(b))

							for i := range vms {
								stack := stack.New(100)
								mem := process.Memory{}
								code := process.Code{
									process.InstPush(valueA),
									process.InstPush(valueB),
									process.InstAdd(),
								}
								proc := process.New(id, stack, mem, code)
								init := NewExec(proc)

								ins[i] <- init
							}

							for i := range vms {
								var actual Result
								Eventually(outs[i], 60).Should(Receive(&actual))

								res, ok := actual.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())
								Expect(res.Value.Eq(expected.Value)).To(BeTrue())
							}
						})
				}, 60)

				It("should add private numbers", func(doneT Done) {
					defer close(doneT)

					done := make(chan (struct{}))
					vms, ins, outs := initVMs(entry.n, entry.k, 0, entry.bufferCap)
					co.ParBegin(
						func() {
							runVMs(done, vms)
						},
						func() {
							defer GinkgoRecover()
							defer close(done)

							results := routeMessages(done, ins, outs)

							id := [31]byte{0x69}
							a, b := SecretField.Random(), SecretField.Random()
							polyA := algebra.NewRandomPolynomial(SecretField, entry.k-1, a)
							polyB := algebra.NewRandomPolynomial(SecretField, entry.k-1, b)
							sharesA := shamir.Split(polyA, uint64(entry.n))
							sharesB := shamir.Split(polyB, uint64(entry.n))

							for i := range vms {
								valueA := process.NewValuePrivate(sharesA[i])
								valueB := process.NewValuePrivate(sharesB[i])

								stack := stack.New(100)
								mem := process.Memory{}
								code := process.Code{
									process.InstPush(valueA),
									process.InstPush(valueB),
									process.InstAdd(),
									process.InstOpen(),
								}
								proc := process.New(id, stack, mem, code)
								init := NewExec(proc)

								ins[i] <- init
							}

							seen := map[uint64]struct{}{}
							for _ = range vms {
								var actual TestResult
								Eventually(results, 60).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								if _, exists := seen[actual.from]; exists {
									Fail(fmt.Sprintf("received more than one result from player %v", actual.from))
								} else {
									seen[actual.from] = struct{}{}
								}
								Expect(ok).To(BeTrue())
								Expect(res.Value.Eq(a.Add(b))).To(BeTrue())
							}
						})
				}, 60)

				It("should add public numbers with private numbers", func(doneT Done) {
					defer close(doneT)

					done := make(chan (struct{}))
					vms, ins, outs := initVMs(entry.n, entry.k, 0, entry.bufferCap)
					co.ParBegin(
						func() {
							runVMs(done, vms)
						},
						func() {
							defer GinkgoRecover()
							defer close(done)

							results := routeMessages(done, ins, outs)

							id := [31]byte{0x69}
							pub, priv := SecretField.Random(), SecretField.Random()
							poly := algebra.NewRandomPolynomial(SecretField, entry.k-1, priv)
							shares := shamir.Split(poly, uint64(entry.n))

							for i := range vms {
								valuePub := process.NewValuePublic(pub)
								valuePriv := process.NewValuePrivate(shares[i])

								stack := stack.New(100)
								mem := process.Memory{}
								code := process.Code{
									process.InstPush(valuePub),
									process.InstPush(valuePriv),
									process.InstAdd(),
									process.InstOpen(),
								}
								proc := process.New(id, stack, mem, code)
								init := NewExec(proc)

								ins[i] <- init
							}

							seen := map[uint64]struct{}{}
							for _ = range vms {
								var actual TestResult
								Eventually(results, 60).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								if _, exists := seen[actual.from]; exists {
									Fail(fmt.Sprintf("received more than one result from player %v", actual.from))
								} else {
									seen[actual.from] = struct{}{}
								}
								Expect(ok).To(BeTrue())
								Expect(res.Value.Eq(pub.Add(priv))).To(BeTrue())
							}
						})
				}, 60)

				It("should generate private random numbers", func(doneT Done) {
					defer close(doneT)

					done := make(chan (struct{}))
					vms, ins, outs := initVMs(entry.n, entry.k, 0, entry.bufferCap)
					co.ParBegin(
						func() {
							runVMs(done, vms)
						},
						func() {
							defer GinkgoRecover()
							defer close(done)

							results := routeMessages(done, ins, outs)

							id := [31]byte{0x69}

							for i := range vms {
								stack := stack.New(100)
								mem := process.Memory{}
								code := process.Code{
									process.InstGenerateRn(),
								}
								proc := process.New(id, stack, mem, code)
								init := NewExec(proc)

								ins[i] <- init
							}

							seen := map[uint64]struct{}{}
							rhoShares := make(shamir.Shares, entry.n)
							sigmaShares := make(shamir.Shares, entry.n)
							for i := range vms {
								var actual TestResult
								Eventually(results, 60).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePrivateRn)
								if _, exists := seen[actual.from]; exists {
									Fail(fmt.Sprintf("received more than one result from player %v", actual.from))
								} else {
									seen[actual.from] = struct{}{}
								}
								Expect(ok).To(BeTrue())
								rhoShares[i] = res.Rho
								sigmaShares[i] = res.Sigma
							}

							rhoExpected, _ := shamir.Join(rhoShares)
							for i := uint64(0); i < uint64(entry.n)-uint64(entry.k); i++ {
								rhoActual, _ := shamir.Join(rhoShares[i : i+uint64(entry.k)])
								Expect(rhoActual.Eq(rhoExpected)).To(BeTrue())
							}

							sigmaExpected, _ := shamir.Join(sigmaShares)
							for i := uint64(0); i < uint64(entry.n)-uint64(entry.k/2); i++ {
								sigmaActual, _ := shamir.Join(sigmaShares[i : i+uint64(entry.k/2)])
								Expect(sigmaActual.Eq(sigmaExpected)).To(BeTrue())
							}
						})
				}, 60)

				It("should multiply private numbers", func(doneT Done) {
					defer close(doneT)

					done := make(chan (struct{}))
					vms, ins, outs := initVMs(entry.n, entry.k, 0, entry.bufferCap)
					co.ParBegin(
						func() {
							runVMs(done, vms)
						},
						func() {
							defer GinkgoRecover()
							defer close(done)

							results := routeMessages(done, ins, outs)

							a, b := SecretField.Random(), SecretField.Random()
							polyA := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, a)
							polyB := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, b)
							sharesA := shamir.Split(polyA, uint64(entry.n))
							sharesB := shamir.Split(polyB, uint64(entry.n))

							for i := range vms {
								valueA := process.NewValuePrivate(sharesA[i])
								valueB := process.NewValuePrivate(sharesB[i])

								id := [31]byte{0x69}
								stack := stack.New(100)
								mem := process.Memory{}
								code := process.Code{
									process.InstPush(valueA),
									process.InstPush(valueB),
									process.InstGenerateRn(),
									process.InstMul(),
									process.InstPush(valueA),
									process.InstGenerateRn(), // FIXME: It is possible for a VM, that is not the leader, to finish producing this Rn (thanks to others being further through execution) before the VM actually gets to this instruction. The result is therefore ignored by the VM, and the VM halts here forever.
									process.InstMul(),
									process.InstOpen(),
								}
								proc := process.New(id, stack, mem, code)
								init := NewExec(proc)

								ins[i] <- init
							}

							seen := map[uint64]struct{}{}
							for _ = range vms {
								var actual TestResult
								Eventually(results, 5).Should(Receive(&actual))

								_, _ = actual.result.Value.(process.ValuePublic)
								if _, exists := seen[actual.from]; exists {
									Fail(fmt.Sprintf("received more than one result from player %v", actual.from))
								} else {
									seen[actual.from] = struct{}{}
								}
								// Expect(ok).To(BeTrue())
								// Expect(res.Value.Eq(a.Mul(b))).To(BeTrue())
							}
						})
				}, 5)

				FIt("should compare 2 bit numbers", func(doneT Done) {
					defer close(doneT)

					done := make(chan (struct{}))
					vms, ins, outs := initVMs(entry.n, entry.k, 0, entry.bufferCap)
					co.ParBegin(
						func() {
							runVMs(done, vms)
						},
						func() {
							defer GinkgoRecover()
							defer close(done)

							results := routeMessages(done, ins, outs)

							id := [31]byte{0x69}

							a0i, a1i := big.NewInt(rand.Int63n(2)), big.NewInt(rand.Int63n(2))
							b0i, b1i := big.NewInt(rand.Int63n(2)), big.NewInt(rand.Int63n(2))
							ai := big.NewInt(0).Add(big.NewInt(0).Mul(big.NewInt(2), a1i), a0i)
							bi := big.NewInt(0).Add(big.NewInt(0).Mul(big.NewInt(2), b1i), b0i)

							a0, a1 := SecretField.NewInField(a0i), SecretField.NewInField(a1i)
							b0, b1 := SecretField.NewInField(b0i), SecretField.NewInField(b1i)
							// a := SecretField.NewInField(big.NewInt(2)).Mul(a1).Add(a0)
							// b := SecretField.NewInField(big.NewInt(2)).Mul(b1).Add(b0)

							polyA0 := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, a0)
							polyA1 := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, a1)
							polyB0 := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, b0)
							polyB1 := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, b1)
							sharesA0 := shamir.Split(polyA0, uint64(entry.n))
							sharesA1 := shamir.Split(polyA1, uint64(entry.n))
							sharesB0 := shamir.Split(polyB0, uint64(entry.n))
							sharesB1 := shamir.Split(polyB1, uint64(entry.n))

							for i := range vms {
								valueA0 := process.NewValuePrivate(sharesA0[i])
								valueA1 := process.NewValuePrivate(sharesA1[i])
								valueB0 := process.NewValuePrivate(sharesB0[i])
								valueB1 := process.NewValuePrivate(sharesB1[i])

								stack := stack.New(100)
								mem := process.NewMemory(100)
								code := process.Code{
									// b0 && !a0 stored at 0
									process.InstPush(valueA0),
									process.Not(SecretField),
									process.InstPush(valueB0),
									process.And(),
									process.InstStore(0),

									// b1 && !a1 stored at 1
									process.InstPush(valueA1),
									process.Not(SecretField),
									process.InstPush(valueB1),
									process.And(),
									process.InstStore(1),

									// !b1 && !a1 stored at 2
									process.InstPush(valueB1),
									process.Not(SecretField),
									process.InstPush(valueA1),
									process.Not(SecretField),
									process.And(),
									process.InstStore(2),

									// b1 && a1 stored at 3
									process.InstPush(valueA1),
									process.InstPush(valueB1),
									process.And(),
									process.InstStore(3),

									// addr 2 || addr 3
									process.InstLoad(2),
									process.InstLoad(3),
									process.Or(),

									// prev && addr 0
									process.InstLoad(0),
									process.And(),

									// prev || addr 1
									process.InstLoad(1),
									process.Or(),

									// open result
									process.InstOpen(),
								}
								proc := process.New(id, stack, mem, code)
								init := NewExec(proc)

								ins[i] <- init
							}

							seen := map[uint64]struct{}{}
							for _ = range vms {
								var actual TestResult
								Eventually(results, 60).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								if _, exists := seen[actual.from]; exists {
									Fail(fmt.Sprintf("received more than one result from player %v", actual.from))
								} else {
									seen[actual.from] = struct{}{}
								}
								Expect(ok).To(BeTrue())
								if res.Value.IsOne() {
									Expect(ai.Cmp(bi)).To(Equal(-1))
								} else {
									Expect(ai.Cmp(bi)).ToNot(Equal(-1))
								}
								// log.Printf("a < b: %v\na: %v\nb: %v", res.Value, a, b)
							}
						})
				}, 60)
			})
		}
	})

})
