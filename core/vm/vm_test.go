package vm_test

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"math/rand"

	"github.com/republicprotocol/oro-go/core/stack"
	"github.com/republicprotocol/oro-go/core/task"
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

	Zero := SecretField.NewInField(big.NewInt(0))
	One := SecretField.NewInField(big.NewInt(1))

	type TestResult struct {
		result Result
		// from   uint64
	}

	// initVMs for a secure multi-party computation network. The VMs will
	// communicate to execute processes.
	initVMs := func(n, k uint64, cap int) []task.Task {
		// Initialize the VMs
		vms := make([]task.Task, n)
		for i := 0; i < len(vms); i++ {
			vms[i] = New(PedersenScheme, uint64(i)+1, n, k, cap)
		}
		return vms
	}

	runVMs := func(done <-chan struct{}, tasks []task.Task) <-chan TestResult {
		results := make(chan TestResult, len(tasks))

		go task.New(task.NewIO(128), task.NewReducer(func(message task.Message) task.Message {

			switch message := message.(type) {
			case RemoteProcedureCall:
				switch message := message.Message.(type) {

				case rng.RnShares:
					tasks[0].Send(NewRemoteProcedureCall(message))

				case rng.ProposeRnShare:
					share := message.Rho.Share()
					tasks[share.Index()-1].Send(NewRemoteProcedureCall(message))

				default:
					for _, in := range tasks {
						in.Send(NewRemoteProcedureCall(message))
					}
				}

			case Result:
				select {
				case <-done:
				case results <- TestResult{message}:
				}

			default:
				log.Fatalf("unexpected message type %T", message)
			}

			return nil

		}), tasks...).Run(done)

		return results
	}

	// randomBit := func() algebra.FpElement {
	// 	return SecretField.NewInField(big.NewInt(rand.Int63n(2)))
	// }

	idFromUint64 := func(n uint64) [32]byte {
		ret := [32]byte{0x0}
		binary.LittleEndian.PutUint64(ret[24:], n)
		return ret
	}

	Context("when running the virtual machines in a fully connected network", func() {

		table := []struct {
			n, k      uint64
			bufferCap int
		}{
			{3, 2, BufferLimit}, {3, 2, BufferLimit * 2}, {3, 2, BufferLimit * 3}, {3, 2, BufferLimit * 4},
			{6, 4, BufferLimit}, {6, 4, BufferLimit * 2}, {6, 4, BufferLimit * 3}, {6, 4, BufferLimit * 4},
			{12, 8, BufferLimit}, {12, 8, BufferLimit * 2}, {12, 8, BufferLimit * 3}, {12, 8, BufferLimit * 4},
			// {24, 16, BufferLimit}, {24, 16, BufferLimit * 2}, {24, 16, BufferLimit * 3}, {24, 16, BufferLimit * 4},
		}

		for _, entry := range table {
			entry := entry

			Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entry.n, entry.k, entry.bufferCap), func() {
				It("should add public numbers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)

					defer close(done)

					id := [32]byte{0x69}
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

						vms[i].Send(NewExec(proc))
					}
					results := runVMs(done, vms)

					for _ = range vms {
						var actual TestResult
						Eventually(results, 60).Should(Receive(&actual))

						res, ok := actual.result.Value.(process.ValuePublic)
						Expect(ok).To(BeTrue())
						Expect(res.Value.Eq(expected.Value)).To(BeTrue())
					}
				}, 60)

				It("should add private numbers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)

					defer close(done)

					id := [32]byte{0x69}
					a, b := SecretField.Random(), SecretField.Random()
					polyA := algebra.NewRandomPolynomial(SecretField, uint(entry.k-1), a)
					polyB := algebra.NewRandomPolynomial(SecretField, uint(entry.k-1), b)
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

						vms[i].Send(NewExec(proc))
					}
					results := runVMs(done, vms)

					for _ = range vms {
						var actual TestResult
						Eventually(results, 60).Should(Receive(&actual))

						res, ok := actual.result.Value.(process.ValuePublic)
						Expect(ok).To(BeTrue())
						Expect(res.Value.Eq(a.Add(b))).To(BeTrue())
					}
				}, 60)

				It("should add public numbers with private numbers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)

					defer close(done)

					id := [32]byte{0x69}
					pub, priv := SecretField.Random(), SecretField.Random()
					poly := algebra.NewRandomPolynomial(SecretField, uint(entry.k-1), priv)
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

						vms[i].Send(NewExec(proc))
					}

					results := runVMs(done, vms)

					for _ = range vms {
						var actual TestResult
						Eventually(results, 60).Should(Receive(&actual))

						res, ok := actual.result.Value.(process.ValuePublic)
						Expect(ok).To(BeTrue())
						Expect(res.Value.Eq(pub.Add(priv))).To(BeTrue())
					}
				}, 60)

				It("should generate private random numbers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)

					defer close(done)

					id := [32]byte{0x69}

					for i := range vms {
						stack := stack.New(100)
						mem := process.Memory{}
						code := process.Code{
							process.InstGenerateRn(),
						}
						proc := process.New(id, stack, mem, code)

						vms[i].Send(NewExec(proc))
					}
					results := runVMs(done, vms)

					rhoShares := make(shamir.Shares, entry.n)
					sigmaShares := make(shamir.Shares, entry.n)
					for i := range vms {
						var actual TestResult
						Eventually(results, 60).Should(Receive(&actual))

						res, ok := actual.result.Value.(process.ValuePrivateRn)
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
				}, 60)

				It("should multiply private numbers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)

					defer close(done)

					a, b := SecretField.Random(), SecretField.Random()
					polyA := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), a)
					polyB := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), b)
					sharesA := shamir.Split(polyA, uint64(entry.n))
					sharesB := shamir.Split(polyB, uint64(entry.n))

					for i := range vms {
						valueA := process.NewValuePrivate(sharesA[i])
						valueB := process.NewValuePrivate(sharesB[i])

						id := [32]byte{0x69}
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

						vms[i].Send(NewExec(proc))
					}
					results := runVMs(done, vms)

					for _ = range vms {
						var actual TestResult
						Eventually(results, 5).Should(Receive(&actual))
						log.Printf("[debug] received value!")

						res, ok := actual.result.Value.(process.ValuePublic)
						Expect(ok).To(BeTrue())
						Expect(res.Value.Eq(a.Mul(a.Mul(b)))).To(BeTrue())
					}
				}, 5)

				It("should compare 2 bit numbers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)

					defer close(done)

					id := [32]byte{0x69}

					a0i, a1i := big.NewInt(rand.Int63n(2)), big.NewInt(rand.Int63n(2))
					b0i, b1i := big.NewInt(rand.Int63n(2)), big.NewInt(rand.Int63n(2))
					ai := big.NewInt(0).Add(big.NewInt(0).Mul(big.NewInt(2), a1i), a0i)
					bi := big.NewInt(0).Add(big.NewInt(0).Mul(big.NewInt(2), b1i), b0i)

					a0, a1 := SecretField.NewInField(a0i), SecretField.NewInField(a1i)
					b0, b1 := SecretField.NewInField(b0i), SecretField.NewInField(b1i)
					// a := SecretField.NewInField(big.NewInt(2)).Mul(a1).Add(a0)
					// b := SecretField.NewInField(big.NewInt(2)).Mul(b1).Add(b0)

					polyA0 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), a0)
					polyA1 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), a1)
					polyB0 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), b0)
					polyB1 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), b1)
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

						vms[i].Send(NewExec(proc))
					}
					results := runVMs(done, vms)

					for _ = range vms {
						var actual TestResult
						Eventually(results, 5).Should(Receive(&actual))

						res, ok := actual.result.Value.(process.ValuePublic)
						Expect(ok).To(BeTrue())
						if res.Value.IsOne() {
							Expect(ai.Cmp(bi)).To(Equal(-1))
						} else {
							Expect(ai.Cmp(bi)).ToNot(Equal(-1))
						}
						// log.Printf("a < b: %v\na: %v\nb: %v", res.Value, a, b)
					}
				}, 5)

				Context("when using macros", func() {
					FIt("should compute a not gate", func(doneT Done) {
						defer close(doneT)
						defer GinkgoRecover()

						done := make(chan (struct{}))
						vms := initVMs(entry.n, entry.k, entry.bufferCap)
						var results <-chan TestResult = nil

						defer close(done)

						logicTable := []struct {
							x, out algebra.FpElement
						}{
							{Zero, One},
							{One, Zero},
						}

						for i, assignment := range logicTable {
							poly := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.x)
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

								vms[j].Send(NewExec(proc))
							}

							if results == nil {
								results = runVMs(done, vms)
							}

							for _ = range vms {
								var actual TestResult
								Eventually(results, 1).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())

								Expect(res.Value.Eq(assignment.out)).To(BeTrue())
							}
						}
					})

					It("should compute an or gate", func(doneT Done) {
						defer close(doneT)
						defer GinkgoRecover()

						done := make(chan (struct{}))
						vms := initVMs(entry.n, entry.k, entry.bufferCap)

						defer close(done)

						logicTable := []struct {
							x, y, out algebra.FpElement
						}{
							{Zero, Zero, Zero},
							{Zero, One, One},
							{One, Zero, One},
							{One, One, One},
						}

						for i, assignment := range logicTable {
							polyX := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.x)
							polyY := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.y)
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

								vms[j].Send(NewExec(proc))
							}
						}
						results := runVMs(done, vms)

						for _, assignment := range logicTable {
							for _ = range vms {
								var actual TestResult
								Eventually(results, 1).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())

								Expect(res.Value.Eq(assignment.out)).To(BeTrue())
							}
						}
					})

					It("should compute an xor gate", func(doneT Done) {
						defer close(doneT)
						defer GinkgoRecover()

						done := make(chan (struct{}))
						vms := initVMs(entry.n, entry.k, entry.bufferCap)

						defer close(done)

						logicTable := []struct {
							x, y, out algebra.FpElement
						}{
							{Zero, Zero, Zero},
							{Zero, One, One},
							{One, Zero, One},
							{One, One, Zero},
						}

						for i, assignment := range logicTable {
							polyX := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.x)
							polyY := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.y)
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

								vms[j].Send(NewExec(proc))
							}
						}
						results := runVMs(done, vms)

						for _, assignment := range logicTable {
							for _ = range vms {
								var actual TestResult
								Eventually(results, 1).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())

								Expect(res.Value.Eq(assignment.out)).To(BeTrue())
							}
						}
					})

					It("should compute an and gate", func(doneT Done) {
						defer close(doneT)
						defer GinkgoRecover()

						done := make(chan (struct{}))
						vms := initVMs(entry.n, entry.k, entry.bufferCap)

						defer close(done)

						logicTable := []struct {
							x, y, out algebra.FpElement
						}{
							{Zero, Zero, Zero},
							{Zero, One, Zero},
							{One, Zero, Zero},
							{One, One, One},
						}

						for i, assignment := range logicTable {
							polyX := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.x)
							polyY := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.y)
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

								vms[j].Send(NewExec(proc))
							}
						}
						results := runVMs(done, vms)

						for _, assignment := range logicTable {
							for _ = range vms {
								var actual TestResult
								Eventually(results, 1).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())

								Expect(res.Value.Eq(assignment.out)).To(BeTrue())
							}
						}
					})

					It("should correctly swap elements on the stack", func(doneT Done) {
						defer close(doneT)
						defer GinkgoRecover()

						done := make(chan (struct{}))
						vms := initVMs(entry.n, entry.k, entry.bufferCap)

						defer close(done)

						x := SecretField.Random()
						y := SecretField.Random()

						polyX := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), x)
						polyY := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), y)
						sharesX := shamir.Split(polyX, uint64(entry.n))
						sharesY := shamir.Split(polyY, uint64(entry.n))

						// MacroSwap should swap the elements on the stack
						for j := range vms {
							valueX := process.NewValuePrivate(sharesX[j])
							valueY := process.NewValuePrivate(sharesY[j])

							id := idFromUint64(1)
							stack := stack.New(100)
							mem := process.NewMemory(10)
							code := process.Code{
								process.InstPush(valueX),
								process.InstPush(valueY),
								process.MacroSwap(),
								process.InstOpen(),
							}
							proc := process.New(id, stack, mem, code)

							vms[j].Send(NewExec(proc))
						}
						results := runVMs(done, vms)

						for _ = range vms {
							var actual TestResult
							Eventually(results, 1).Should(Receive(&actual))

							res, ok := actual.result.Value.(process.ValuePublic)
							Expect(ok).To(BeTrue())

							Expect(res.Value.Eq(x)).To(BeTrue())
						}

						// Two applications of MacroSwap should leave the stack unchanged
						for j := range vms {
							valueX := process.NewValuePrivate(sharesX[j])
							valueY := process.NewValuePrivate(sharesY[j])

							id := idFromUint64(1)
							stack := stack.New(100)
							mem := process.NewMemory(10)
							code := process.Code{
								process.InstPush(valueX),
								process.InstPush(valueY),
								process.MacroSwap(),
								process.MacroSwap(),
								process.InstOpen(),
							}
							proc := process.New(id, stack, mem, code)

							vms[j].Send(NewExec(proc))
						}

						for _ = range vms {
							var actual TestResult
							Eventually(results, 1).Should(Receive(&actual))

							res, ok := actual.result.Value.(process.ValuePublic)
							Expect(ok).To(BeTrue())

							Expect(res.Value.Eq(y)).To(BeTrue())
						}
					})

					It("should correctly compute the propagator and generator", func(doneT Done) {
						defer close(doneT)
						defer GinkgoRecover()

						done := make(chan (struct{}))
						vms := initVMs(entry.n, entry.k, entry.bufferCap)

						defer close(done)

						logicTable := []struct {
							x, y, p, g algebra.FpElement
						}{
							{Zero, Zero, Zero, Zero},
							{Zero, One, One, Zero},
							{One, Zero, One, Zero},
							{One, One, Zero, One},
						}

						for i, assignment := range logicTable {
							polyX := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.x)
							polyY := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.y)
							sharesX := shamir.Split(polyX, uint64(entry.n))
							sharesY := shamir.Split(polyY, uint64(entry.n))

							// Check that computing the generator is correct
							for j := range vms {
								valueX := process.NewValuePrivate(sharesX[j])
								valueY := process.NewValuePrivate(sharesY[j])

								id := idFromUint64(uint64(i))
								stack := stack.New(100)
								mem := process.NewMemory(10)
								code := process.Code{
									process.InstPush(valueX),
									process.InstPush(valueY),
									process.MacroPropGen(),
									process.InstOpen(),
								}
								proc := process.New(id, stack, mem, code)

								vms[j].Send(NewExec(proc))
							}
						}
						results := runVMs(done, vms)

						for i, assignment := range logicTable {
							for _ = range vms {
								var actual TestResult
								Eventually(results, 5).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())

								Expect(res.Value.Eq(assignment.g)).To(BeTrue())
							}

							polyX := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.x)
							polyY := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.y)
							sharesX := shamir.Split(polyX, uint64(entry.n))
							sharesY := shamir.Split(polyY, uint64(entry.n))

							// Check that computing the porpagator is correct
							for j := range vms {
								valueX := process.NewValuePrivate(sharesX[j])
								valueY := process.NewValuePrivate(sharesY[j])

								id := idFromUint64(uint64(i + 4))
								stack := stack.New(100)
								mem := process.NewMemory(10)
								code := process.Code{
									process.InstPush(valueX),
									process.InstPush(valueY),
									process.MacroPropGen(),
									process.MacroSwap(),
									process.InstOpen(),
								}
								proc := process.New(id, stack, mem, code)

								vms[j].Send(NewExec(proc))
							}

							for _ = range vms {
								var actual TestResult
								Eventually(results, 5).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())

								Expect(res.Value.Eq(assignment.p)).To(BeTrue())
							}
						}
					}, 5)

					It("should correctly compute the CLA operation", func(doneT Done) {
						defer close(doneT)
						defer GinkgoRecover()

						done := make(chan (struct{}))
						vms := initVMs(entry.n, entry.k, entry.bufferCap)

						defer close(done)

						logicTable := []struct {
							p1, g1, p2, g2, pp, gg algebra.FpElement
						}{
							{Zero, Zero, Zero, Zero, Zero, Zero},
							{Zero, Zero, Zero, One, Zero, One},
							{Zero, Zero, One, Zero, Zero, Zero},
							{Zero, Zero, One, One, Zero, One},
							{Zero, One, Zero, Zero, Zero, Zero},
							{Zero, One, Zero, One, Zero, One},
							{Zero, One, One, Zero, Zero, One},
							{Zero, One, One, One, Zero, One},
							{One, Zero, Zero, Zero, Zero, Zero},
							{One, Zero, Zero, One, Zero, One},
							{One, Zero, One, Zero, One, Zero},
							{One, Zero, One, One, One, One},
							{One, One, Zero, Zero, Zero, Zero},
							{One, One, Zero, One, Zero, One},
							{One, One, One, Zero, One, One},
							{One, One, One, One, One, One},
						}

						for i, assignment := range logicTable {
							polyP1 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.p1)
							polyG1 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.g1)
							polyP2 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.p2)
							polyG2 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.g2)
							sharesP1 := shamir.Split(polyP1, uint64(entry.n))
							sharesG1 := shamir.Split(polyG1, uint64(entry.n))
							sharesP2 := shamir.Split(polyP2, uint64(entry.n))
							sharesG2 := shamir.Split(polyG2, uint64(entry.n))

							// Check that computing the generator is correct
							for j := range vms {
								valueP1 := process.NewValuePrivate(sharesP1[j])
								valueG1 := process.NewValuePrivate(sharesG1[j])
								valueP2 := process.NewValuePrivate(sharesP2[j])
								valueG2 := process.NewValuePrivate(sharesG2[j])

								id := idFromUint64(uint64(i))
								stack := stack.New(100)
								mem := process.NewMemory(10)
								code := process.Code{
									process.InstPush(valueP1),
									process.InstPush(valueG1),
									process.InstPush(valueP2),
									process.InstPush(valueG2),
									process.MacroOpCLA(),
									process.InstOpen(),
								}
								proc := process.New(id, stack, mem, code)

								vms[j].Send(NewExec(proc))
							}
						}
						results := runVMs(done, vms)

						for i, assignment := range logicTable {
							for _ = range vms {
								var actual TestResult
								Eventually(results, 10).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())

								Expect(res.Value.Eq(assignment.gg)).To(BeTrue())
							}

							polyP1 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.p1)
							polyG1 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.g1)
							polyP2 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.p2)
							polyG2 := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), assignment.g2)
							sharesP1 := shamir.Split(polyP1, uint64(entry.n))
							sharesG1 := shamir.Split(polyG1, uint64(entry.n))
							sharesP2 := shamir.Split(polyP2, uint64(entry.n))
							sharesG2 := shamir.Split(polyG2, uint64(entry.n))

							// Check that computing the porpagator is correct
							for j := range vms {
								valueP1 := process.NewValuePrivate(sharesP1[j])
								valueG1 := process.NewValuePrivate(sharesG1[j])
								valueP2 := process.NewValuePrivate(sharesP2[j])
								valueG2 := process.NewValuePrivate(sharesG2[j])

								id := idFromUint64(uint64(i + 16))
								stack := stack.New(100)
								mem := process.NewMemory(10)
								code := process.Code{
									process.InstPush(valueP1),
									process.InstPush(valueG1),
									process.InstPush(valueP2),
									process.InstPush(valueG2),
									process.MacroOpCLA(),
									process.MacroSwap(),
									process.InstOpen(),
								}
								proc := process.New(id, stack, mem, code)

								vms[j].Send(NewExec(proc))
							}

							for _ = range vms {
								var actual TestResult
								Eventually(results, 10).Should(Receive(&actual))

								res, ok := actual.result.Value.(process.ValuePublic)
								Expect(ok).To(BeTrue())

								Expect(res.Value.Eq(assignment.pp)).To(BeTrue())
							}
						}
					}, 10)
				})

				It("should compare 64 bit numbers with the CLA adder", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)

					defer close(done)

					id := [32]byte{0x69}

					a := big.NewInt(0).SetUint64(rand.Uint64())
					b := big.NewInt(0).SetUint64(rand.Uint64()) // Set(a)
					notB := ^b.Uint64()
					notB += 1

					aTemp := big.NewInt(0).Set(a)
					bTemp := big.NewInt(0).SetUint64(notB)

					aBits := make([]algebra.FpElement, 64)
					bBits := make([]algebra.FpElement, 64)
					for i := range aBits {
						ar := big.NewInt(0).Mod(aTemp, big.NewInt(2))
						br := big.NewInt(0).Mod(bTemp, big.NewInt(2))
						aBits[i] = SecretField.NewInField(ar)
						bBits[i] = SecretField.NewInField(br)
						aTemp.Div(aTemp, big.NewInt(2))
						bTemp.Div(bTemp, big.NewInt(2))
					}

					aVals := make([][]process.ValuePrivate, entry.n)
					bVals := make([][]process.ValuePrivate, entry.n)
					for i := range aVals {
						aVals[i] = make([]process.ValuePrivate, 64)
						bVals[i] = make([]process.ValuePrivate, 64)
					}

					for i := 0; i < 64; i++ {
						polyA := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), aBits[i])
						polyB := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), bBits[i])
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
						for j := 0; j < 64; j++ {
							mem[process.Addr(10+2*j)] = aVals[i][j]
							mem[process.Addr(11+2*j)] = bVals[i][j]
						}
						code := process.Code{
							process.MacroBitwiseCOut(SecretField, process.Addr(10), 64),
							process.InstOpen(),
						}
						proc := process.New(id, stack, mem, code)

						vms[i].Send(NewExec(proc))
					}
					results := runVMs(done, vms)

					for _ = range vms {
						var actual TestResult
						Eventually(results, 5).Should(Receive(&actual))
						res, ok := actual.result.Value.(process.ValuePublic)
						Expect(ok).To(BeTrue())
						if a.Cmp(b) == -1 {
							Expect(res.Value.Eq(SecretField.NewInField(big.NewInt(0)))).To(BeTrue())
						} else {
							Expect(res.Value.Eq(SecretField.NewInField(big.NewInt(1)))).To(BeTrue())
						}
					}
				}, 5)
			})
		}
	})

})
