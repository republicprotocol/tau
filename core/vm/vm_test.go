package vm_test

import (
	"encoding/binary"
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

	P := big.NewInt(0).SetBytes([]byte{59, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 218, 189})
	Q := big.NewInt(0).SetBytes([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 97})
	G := algebra.NewFpElement(big.NewInt(0).SetBytes([]byte{46, 217, 224, 127, 231, 86, 139, 236, 205, 5, 30, 157, 110, 39, 97, 111, 203}), P)
	H := algebra.NewFpElement(big.NewInt(0).SetBytes([]byte{1, 165, 222, 96, 116, 29, 93, 210, 13, 91, 221, 196, 71, 130, 67, 247, 62}), P)
	SecretField := algebra.NewField(Q)
	PedersenScheme := pedersen.New(G, H, SecretField)
	BufferLimit := 64

	Zero := SecretField.NewInField(big.NewInt(0))
	One := SecretField.NewInField(big.NewInt(1))

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

	// randomBit := func() algebra.FpElement {
	// 	return SecretField.NewInField(big.NewInt(rand.Int63n(2)))
	// }

	idFromUint64 := func(n uint64) [31]byte {
		ret := [31]byte{0x0}
		id := make([]byte, 8)
		binary.LittleEndian.PutUint64(id, n)
		for i, b := range id {
			ret[i] = b
		}
		return ret
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
									process.InstGenerateRn(),
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

				It("should compare 2 bit numbers", func(doneT Done) {
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
									process.MacroNot(SecretField),
									process.InstPush(valueB0),
									process.MacroAnd(),
									process.InstStore(0),

									// b1 && !a1 stored at 1
									process.InstPush(valueA1),
									process.MacroNot(SecretField),
									process.InstPush(valueB1),
									process.MacroAnd(),
									process.InstStore(1),

									// !b1 && !a1 stored at 2
									process.InstPush(valueB1),
									process.MacroNot(SecretField),
									process.InstPush(valueA1),
									process.MacroNot(SecretField),
									process.MacroAnd(),
									process.InstStore(2),

									// b1 && a1 stored at 3
									process.InstPush(valueA1),
									process.InstPush(valueB1),
									process.MacroAnd(),
									process.InstStore(3),

									// addr 2 || addr 3
									process.InstLoad(2),
									process.InstLoad(3),
									process.MacroOr(),

									// prev && addr 0
									process.InstLoad(0),
									process.MacroAnd(),

									// prev || addr 1
									process.InstLoad(1),
									process.MacroOr(),

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
								Eventually(results, 5).Should(Receive(&actual))

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
				}, 5)

				Context("when using macros", func() {
					It("should compute a not gate", func(doneT Done) {
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

								logicTable := []struct {
									x, out algebra.FpElement
								}{
									{Zero, One},
									{One, Zero},
								}

								for i, assignment := range logicTable {
									poly := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, assignment.x)
									shares := shamir.Split(poly, uint64(entry.n))

									for j := range vms {
										value := process.NewValuePrivate(shares[j])

										id := idFromUint64(uint64(i))
										stack := stack.New(100)
										mem := process.NewMemory(10)
										code := process.Code{
											process.InstPush(value),
											process.MacroNot(SecretField),
											process.InstOpen(),
										}
										proc := process.New(id, stack, mem, code)
										init := NewExec(proc)

										ins[j] <- init
									}

									seen := map[uint64]struct{}{}
									for _ = range vms {
										var actual TestResult
										Eventually(results, 1).Should(Receive(&actual))

										res, ok := actual.result.Value.(process.ValuePublic)
										Expect(ok).To(BeTrue())

										if _, exists := seen[actual.from]; exists {
											Fail(fmt.Sprintf("received more than one result from player %v", actual.from))
										} else {
											seen[actual.from] = struct{}{}
										}

										Expect(res.Value.Eq(assignment.out)).To(BeTrue())
									}
								}
							})
					})

					It("should compute an or gate", func(doneT Done) {
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

								logicTable := []struct {
									x, y, out algebra.FpElement
								}{
									{Zero, Zero, Zero},
									{Zero, One, One},
									{One, Zero, One},
									{One, One, One},
								}

								for i, assignment := range logicTable {
									polyX := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, assignment.x)
									polyY := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, assignment.y)
									sharesX := shamir.Split(polyX, uint64(entry.n))
									sharesY := shamir.Split(polyY, uint64(entry.n))

									for j := range vms {
										valueX := process.NewValuePrivate(sharesX[j])
										valueY := process.NewValuePrivate(sharesY[j])

										id := idFromUint64(uint64(i))
										stack := stack.New(100)
										mem := process.NewMemory(10)
										code := process.Code{
											process.InstPush(valueX),
											process.InstPush(valueY),
											process.MacroOr(),
											process.InstOpen(),
										}
										proc := process.New(id, stack, mem, code)
										init := NewExec(proc)

										ins[j] <- init
									}

									seen := map[uint64]struct{}{}
									for _ = range vms {
										var actual TestResult
										Eventually(results, 1).Should(Receive(&actual))

										res, ok := actual.result.Value.(process.ValuePublic)
										Expect(ok).To(BeTrue())

										if _, exists := seen[actual.from]; exists {
											Fail(fmt.Sprintf("received more than one result from player %v", actual.from))
										} else {
											seen[actual.from] = struct{}{}
										}

										Expect(res.Value.Eq(assignment.out)).To(BeTrue())
									}
								}
							})
					})

					It("should compute an xor gate", func(doneT Done) {
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

								logicTable := []struct {
									x, y, out algebra.FpElement
								}{
									{Zero, Zero, Zero},
									{Zero, One, One},
									{One, Zero, One},
									{One, One, Zero},
								}

								for i, assignment := range logicTable {
									polyX := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, assignment.x)
									polyY := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, assignment.y)
									sharesX := shamir.Split(polyX, uint64(entry.n))
									sharesY := shamir.Split(polyY, uint64(entry.n))

									for j := range vms {
										valueX := process.NewValuePrivate(sharesX[j])
										valueY := process.NewValuePrivate(sharesY[j])

										id := idFromUint64(uint64(i))
										stack := stack.New(100)
										mem := process.NewMemory(10)
										code := process.Code{
											process.InstPush(valueX),
											process.InstPush(valueY),
											process.MacroXor(),
											process.InstOpen(),
										}
										proc := process.New(id, stack, mem, code)
										init := NewExec(proc)

										ins[j] <- init
									}

									seen := map[uint64]struct{}{}
									for _ = range vms {
										var actual TestResult
										Eventually(results, 1).Should(Receive(&actual))

										res, ok := actual.result.Value.(process.ValuePublic)
										Expect(ok).To(BeTrue())

										if _, exists := seen[actual.from]; exists {
											Fail(fmt.Sprintf("received more than one result from player %v", actual.from))
										} else {
											seen[actual.from] = struct{}{}
										}

										Expect(res.Value.Eq(assignment.out)).To(BeTrue())
									}
								}
							})
					})

					FIt("should compute an and gate", func(doneT Done) {
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

								logicTable := []struct {
									x, y, out algebra.FpElement
								}{
									{Zero, Zero, Zero},
									{Zero, One, Zero},
									{One, Zero, Zero},
									{One, One, One},
								}

								for i, assignment := range logicTable {
									polyX := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, assignment.x)
									polyY := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, assignment.y)
									sharesX := shamir.Split(polyX, uint64(entry.n))
									sharesY := shamir.Split(polyY, uint64(entry.n))

									for j := range vms {
										valueX := process.NewValuePrivate(sharesX[j])
										valueY := process.NewValuePrivate(sharesY[j])

										id := idFromUint64(uint64(i))
										stack := stack.New(100)
										mem := process.NewMemory(10)
										code := process.Code{
											process.InstPush(valueX),
											process.InstPush(valueY),
											process.MacroAnd(),
											process.InstOpen(),
										}
										proc := process.New(id, stack, mem, code)
										init := NewExec(proc)

										ins[j] <- init
									}

									seen := map[uint64]struct{}{}
									for _ = range vms {
										var actual TestResult
										Eventually(results, 1).Should(Receive(&actual))

										res, ok := actual.result.Value.(process.ValuePublic)
										Expect(ok).To(BeTrue())

										if _, exists := seen[actual.from]; exists {
											Fail(fmt.Sprintf("received more than one result from player %v", actual.from))
										} else {
											seen[actual.from] = struct{}{}
										}

										Expect(res.Value.Eq(assignment.out)).To(BeTrue())
									}
								}
							})
					})
				})

				It("should compare 64 bit numbers with the CLA adder", func(doneT Done) {
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

							a := big.NewInt(0)
							b := big.NewInt(0)

							aBits := make([]algebra.FpElement, 64)
							bBits := make([]algebra.FpElement, 64)
							for i := range aBits {
								ar := big.NewInt(rand.Int63n(2))
								br := big.NewInt(rand.Int63n(2))
								aBits[i] = SecretField.NewInField(ar)
								bBits[i] = SecretField.NewInField(br)
								a.Add(a, big.NewInt(0).Mul(ar, big.NewInt(0).Exp(big.NewInt(2), big.NewInt(int64(i)), Q)))
								b.Add(b, big.NewInt(0).Mul(br, big.NewInt(0).Exp(big.NewInt(2), big.NewInt(int64(i)), Q)))
							}

							aVals := make([][]process.ValuePrivate, entry.n)
							bVals := make([][]process.ValuePrivate, entry.n)
							for i := range aVals {
								aVals[i] = make([]process.ValuePrivate, 64)
								bVals[i] = make([]process.ValuePrivate, 64)
							}

							for i := 0; i < 64; i++ {
								polyA := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, aBits[i])
								polyB := algebra.NewRandomPolynomial(SecretField, entry.k/2-1, bBits[i])
								sharesA := shamir.Split(polyA, uint64(entry.n))
								sharesB := shamir.Split(polyB, uint64(entry.n))

								for j, share := range sharesA {
									aVals[j][i] = process.NewValuePrivate(share)
								}
								for j, share := range sharesB {
									bVals[j][i] = process.NewValuePrivate(share)
								}
							}

							for i := range vms {
								stack := stack.New(100)
								mem := process.NewMemory(300)
								code := make(process.Code, 0)
								for j := 0; j < 64; j++ {
									c := process.Code{
										process.InstPush(aVals[i][j]),
										process.InstStore(process.Addr(10 + 2*j)),
										process.InstPush(bVals[i][j]),
										process.InstStore(process.Addr(11 + 2*j)),
									}
									code = append(code, c...)
								}
								code = append(code, process.MacroCmp64(SecretField, process.Addr(10)), process.InstOpen())
								proc := process.New(id, stack, mem, code)
								init := NewExec(proc)

								ins[i] <- init
							}

							for _ = range vms {
								var actual TestResult
								Eventually(results, 5).Should(Receive(&actual))
								res, ok := actual.result.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())
								if a.Cmp(b) == -1 {
									Expect(res.Value.Eq(SecretField.NewInField(big.NewInt(1)))).To(BeTrue())
								} else {
									Expect(res.Value.Eq(SecretField.NewInField(big.NewInt(0)))).To(BeTrue())
								}
							}
						})
				}, 5)
			})
		}
	})

})
