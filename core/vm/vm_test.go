package vm_test

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"math/rand"

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
					share := message.Sigma.Share()
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
				log.Fatalf("unexpected message type %T: %v", message, message)
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

	// RandomBool returns a random boolean with equal probability.
	randomBool := func() bool {
		return rand.Float32() < 0.5
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
					results := runVMs(done, vms)

					defer close(done)

					id := [32]byte{0x69}
					a, b := SecretField.Random(), SecretField.Random()
					valueA, valueB := process.NewValuePublic(a), process.NewValuePublic(b)
					expected := process.NewValuePublic(a.Add(b))

					for i := range vms {
						mem := process.NewMemory(2)
						code := process.Code{
							process.InstMove(mem.At(0), valueA),
							process.InstMove(mem.At(1), valueB),
							process.InstAdd(mem.At(0), mem.At(0), mem.At(1)),
							process.InstExit(mem.At(0)),
						}
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					for _ = range vms {
						var actual TestResult
						Eventually(results, 60).Should(Receive(&actual))

						res, ok := actual.result.Values[0].(process.ValuePublic)
						Expect(ok).To(BeTrue())
						Expect(res.Value.Eq(expected.Value)).To(BeTrue())
					}
				}, 60)

				It("should add private numbers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

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

						mem := process.NewMemory(2)
						code := process.Code{
							process.InstMove(mem.At(0), valueA),
							process.InstMove(mem.At(1), valueB),
							process.InstAdd(mem.At(0), mem.At(0), mem.At(1)),
							process.InstOpen(mem.At(0), mem.At(0)),
							process.InstExit(mem.At(0)),
						}
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					for _ = range vms {
						var actual TestResult
						Eventually(results, 60).Should(Receive(&actual))

						res, ok := actual.result.Values[0].(process.ValuePublic)
						Expect(ok).To(BeTrue())
						Expect(res.Value.Eq(a.Add(b))).To(BeTrue())
					}
				}, 60)

				It("should add public numbers with private numbers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

					defer close(done)

					id := [32]byte{0x69}
					pub, priv := SecretField.Random(), SecretField.Random()
					poly := algebra.NewRandomPolynomial(SecretField, uint(entry.k-1), priv)
					shares := shamir.Split(poly, uint64(entry.n))

					for i := range vms {
						valuePub := process.NewValuePublic(pub)
						valuePriv := process.NewValuePrivate(shares[i])

						mem := process.NewMemory(2)
						code := process.Code{
							process.InstMove(mem.At(0), valuePub),
							process.InstMove(mem.At(1), valuePriv),
							process.InstAdd(mem.At(0), mem.At(0), mem.At(1)),
							process.InstOpen(mem.At(0), mem.At(0)),
							process.InstExit(mem.At(0)),
						}
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					for _ = range vms {
						var actual TestResult
						Eventually(results, 60).Should(Receive(&actual))

						res, ok := actual.result.Values[0].(process.ValuePublic)
						Expect(ok).To(BeTrue())
						Expect(res.Value.Eq(pub.Add(priv))).To(BeTrue())
					}
				}, 60)

				It("should generate private random numbers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

					defer close(done)

					id := [32]byte{0x69}

					for i := range vms {
						mem := process.NewMemory(2)
						code := process.Code{
							process.InstGenerateRnTuple(mem.At(0), mem.At(1)),
							process.InstExit(mem.At(0), mem.At(1)),
						}
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					rhoShares := make(shamir.Shares, entry.n)
					sigmaShares := make(shamir.Shares, entry.n)
					for i := range vms {
						var actual TestResult
						Eventually(results, 60).Should(Receive(&actual))

						rho, ok := actual.result.Values[0].(process.ValuePrivate)
						Expect(ok).To(BeTrue())
						rhoShares[i] = rho.Share

						sigma, ok := actual.result.Values[1].(process.ValuePrivate)
						Expect(ok).To(BeTrue())
						sigmaShares[i] = sigma.Share
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
					results := runVMs(done, vms)

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
						mem := process.NewMemory(4)
						code := process.Code{
							process.InstMove(mem.At(0), valueA),
							process.InstMove(mem.At(1), valueB),
							process.InstGenerateRnTuple(mem.At(2), mem.At(3)),
							process.InstMul(mem.At(0), mem.At(0), mem.At(1), mem.At(2), mem.At(3)),
							process.InstOpen(mem.At(0), mem.At(0)),
							process.InstExit(mem.At(0)),
						}
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					for _ = range vms {
						var actual TestResult
						Eventually(results, 5).Should(Receive(&actual))

						res, ok := actual.result.Values[0].(process.ValuePublic)
						Expect(ok).To(BeTrue())
						Expect(res.Value.Eq(a.Mul(b))).To(BeTrue())
					}
				}, 5)

				// 			Context("when using macros", func() {
				It("should compute a not gate", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

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
							mem := process.NewMemory(1)
							code := process.Code{
								process.InstMove(mem.At(0), value),
								process.MacroBitwiseNot(mem.At(0), mem.At(0), SecretField),
								process.InstOpen(mem.At(0), mem.At(0)),
								process.InstExit(mem.At(0)),
							}
							proc := process.New(id, mem, code)

							vms[j].IO().InputWriter() <- NewExec(proc)
						}

						for _ = range vms {
							var actual TestResult
							Eventually(results, 1).Should(Receive(&actual))

							res, ok := actual.result.Values[0].(process.ValuePublic)
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
					results := runVMs(done, vms)

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
							mem := process.NewMemory(2)
							code := process.Code{
								process.InstMove(mem.At(0), valueX),
								process.InstMove(mem.At(1), valueY),
								process.MacroBitwiseOr(mem.At(0), mem.At(0), mem.At(1)),
								process.InstOpen(mem.At(0), mem.At(0)),
								process.InstExit(mem.At(0)),
							}
							proc := process.New(id, mem, code)

							vms[j].IO().InputWriter() <- NewExec(proc)
						}

						for _ = range vms {
							var actual TestResult
							Eventually(results, 1).Should(Receive(&actual))

							res, ok := actual.result.Values[0].(process.ValuePublic)
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
					results := runVMs(done, vms)

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
							mem := process.NewMemory(2)
							code := process.Code{
								process.InstMove(mem.At(0), valueX),
								process.InstMove(mem.At(1), valueY),
								process.MacroBitwiseXor(mem.At(0), mem.At(0), mem.At(1)),
								process.InstOpen(mem.At(0), mem.At(0)),
								process.InstExit(mem.At(0)),
							}
							proc := process.New(id, mem, code)

							vms[j].IO().InputWriter() <- NewExec(proc)
						}

						for _ = range vms {
							var actual TestResult
							Eventually(results, 1).Should(Receive(&actual))

							res, ok := actual.result.Values[0].(process.ValuePublic)
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
					results := runVMs(done, vms)

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
							mem := process.NewMemory(2)
							code := process.Code{
								process.InstMove(mem.At(0), valueX),
								process.InstMove(mem.At(1), valueY),
								process.MacroBitwiseAnd(mem.At(0), mem.At(0), mem.At(1)),
								process.InstOpen(mem.At(0), mem.At(0)),
								process.InstExit(mem.At(0)),
							}
							proc := process.New(id, mem, code)

							vms[j].IO().InputWriter() <- NewExec(proc)
						}

						for _ = range vms {
							var actual TestResult
							Eventually(results, 1).Should(Receive(&actual))

							res, ok := actual.result.Values[0].(process.ValuePublic)
							Expect(ok).To(BeTrue())

							Expect(res.Value.Eq(assignment.out)).To(BeTrue())
						}
					}
				})

				It("should correctly compute the propagator and generator", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

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
							mem := process.NewMemory(2)
							code := process.Code{
								process.InstMove(mem.At(0), valueX),
								process.InstMove(mem.At(1), valueY),
								process.MacroBitwisePropGen(mem.At(0), mem.At(1), mem.At(0), mem.At(1)),
								process.InstOpen(mem.At(0), mem.At(0)),
								process.InstOpen(mem.At(1), mem.At(1)),
								process.InstExit(mem.At(0), mem.At(1)),
							}
							proc := process.New(id, mem, code)

							vms[j].IO().InputWriter() <- NewExec(proc)
						}

						for _ = range vms {
							var actual TestResult
							Eventually(results, 5).Should(Receive(&actual))

							resP, ok := actual.result.Values[0].(process.ValuePublic)
							Expect(ok).To(BeTrue())
							resG, ok := actual.result.Values[1].(process.ValuePublic)
							Expect(ok).To(BeTrue())

							Expect(resP.Value.Eq(assignment.p)).To(BeTrue())
							Expect(resG.Value.Eq(assignment.g)).To(BeTrue())
						}
					}
				}, 5)

				It("should correctly compute the CLA operation", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

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
							mem := process.NewMemory(4)
							code := process.Code{
								process.InstMove(mem.At(0), valueP1),
								process.InstMove(mem.At(1), valueG1),
								process.InstMove(mem.At(2), valueP2),
								process.InstMove(mem.At(3), valueG2),
								process.MacroBitwiseOpCLA(mem.At(0), mem.At(1), mem.At(0), mem.At(1), mem.At(2), mem.At(3)),
								process.InstOpen(mem.At(0), mem.At(0)),
								process.InstOpen(mem.At(1), mem.At(1)),
								process.InstExit(mem.At(0), mem.At(1)),
							}
							proc := process.New(id, mem, code)

							vms[j].IO().InputWriter() <- NewExec(proc)
						}

						for _ = range vms {
							var actual TestResult
							Eventually(results, 10).Should(Receive(&actual))

							resPP, ok := actual.result.Values[0].(process.ValuePublic)
							Expect(ok).To(BeTrue())
							resGG, ok := actual.result.Values[1].(process.ValuePublic)
							Expect(ok).To(BeTrue())

							Expect(resPP.Value.Eq(assignment.pp)).To(BeTrue())
							Expect(resGG.Value.Eq(assignment.gg)).To(BeTrue())
						}

					}
				}, 10)

				It("should compare k bit numbers with the CLA adder", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

					defer close(done)

					id := [32]byte{0x69}

					k := uint64(15)
					a := big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << k))
					b := big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << k))

					aTemp := big.NewInt(0).Set(a)
					bTemp := big.NewInt(0).Set(b)

					aBits := make([]algebra.FpElement, k)
					bBits := make([]algebra.FpElement, k)
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
						aVals[i] = make([]process.ValuePrivate, k)
						bVals[i] = make([]process.ValuePrivate, k)
					}

					for i := uint64(0); i < k; i++ {
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
						mem := process.NewMemory(128)
						for j := uint64(0); j < k; j++ {
							mem[j] = aVals[i][j]
							mem[j+k] = bVals[i][j]
						}
						code := process.Code{
							process.MacroBitwiseLT(mem.At(0), mem.At(0), mem.At(int(k)), SecretField, uint(k)),
							process.InstOpen(mem.At(0), mem.At(0)),
							process.InstExit(mem.At(0)),
						}
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					for _ = range vms {
						var actual TestResult
						Eventually(results, 10).Should(Receive(&actual))
						res, ok := actual.result.Values[0].(process.ValuePublic)
						Expect(ok).To(BeTrue())
						if a.Cmp(b) == -1 {
							Expect(res.Value.Eq(SecretField.NewInField(big.NewInt(1)))).To(BeTrue())
						} else {
							Expect(res.Value.Eq(SecretField.NewInField(big.NewInt(0)))).To(BeTrue())
						}
					}
				}, 10)

				It("should generate random bits", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

					defer close(done)

					id := [32]byte{0x69}
					for i := range vms {
						// Generate 10 random bits
						mem := process.NewMemory(10)
						memLocations := make([]*process.Value, 10)
						code := make(process.Code, 0, 21)
						for j := 0; j < 10; j++ {
							memLocations[j] = mem.At(j)
							code = append(code,
								process.MacroRandBit(mem.At(j), SecretField),
								process.InstOpen(mem.At(j), mem.At(j)),
							)
						}
						code = append(code, process.InstExit(memLocations...))
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					for _ = range vms {
						var actual TestResult
						Eventually(results, 10).Should(Receive(&actual))
						for _, value := range actual.result.Values {
							res, ok := value.(process.ValuePublic)
							Expect(ok).To(BeTrue())

							// Expect the result to be zero or one
							if !res.Value.IsZero() {
								Expect(res.Value.IsOne()).To(BeTrue())
							}
						}
					}
				})

				It("should compute the binary representation of a number", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

					defer close(done)

					a := SecretField.NewInField(big.NewInt(0).SetUint64(uint64(rand.Uint32())))

					id := [32]byte{0x69}
					for i := range vms {
						mem := process.NewMemory(1)
						memLocations := make([]process.Value, 32)
						memLocPtrs := make([]*process.Value, 32)

						for i := range memLocPtrs {
							memLocPtrs[i] = &memLocations[i]
						}

						code := process.Code{
							process.InstMove(mem.At(0), process.NewValuePublic(a)),
							process.MacroBits(&memLocations[0], mem.At(0), 32, SecretField),
							process.InstExit(memLocPtrs...),
						}
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					for _ = range vms {
						var actual TestResult
						Eventually(results, 10).Should(Receive(&actual))

						acc := SecretField.NewInField(big.NewInt(0))
						two := SecretField.NewInField(big.NewInt(2))

						for i := len(actual.result.Values) - 1; i >= 0; i-- {
							res, ok := actual.result.Values[i].(process.ValuePublic)
							Expect(ok).To(BeTrue())
							acc = acc.Mul(two)
							acc = acc.Add(res.Value)

							// Expect the result to be zero or one
							if !res.Value.IsZero() {
								Expect(res.Value.IsOne()).To(BeTrue())
							}
						}

						Expect(acc.Eq(a)).To(BeTrue())
					}
				})

				It("should compute integers modulo powers of two", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

					defer close(done)

					k := uint64(16)
					m := uint64(15)
					a := SecretField.NewInField(big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << (k - 1))))
					poly := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), a)
					shares := shamir.Split(poly, uint64(entry.n))

					negCase := randomBool()
					negCase = false
					if negCase {
						a = a.Neg()
					}

					id := [32]byte{0x69}
					for i := range vms {
						mem := process.NewMemory(1)

						code := process.Code{
							process.InstMove(mem.At(0), process.NewValuePrivate(shares[i])),
							process.MacroMod2m(mem.At(0), mem.At(0), k, m, 10, SecretField),
							process.InstOpen(mem.At(0), mem.At(0)),
							process.InstExit(mem.At(0)),
						}
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					for _ = range vms {
						var actual TestResult
						Eventually(results, 10).Should(Receive(&actual))
						res, ok := actual.result.Values[0].(process.ValuePublic)
						Expect(ok).To(BeTrue())

						twoPow := big.NewInt(0).SetUint64(uint64(1) << m)
						var mod *big.Int
						if negCase {
							mod = big.NewInt(0).Mod(big.NewInt(0).Neg(a.Neg().Value()), twoPow)
						} else {
							mod = big.NewInt(0).Mod(a.Value(), twoPow)
						}

						Expect(mod.Cmp(res.Value.Value())).To(Equal(0))
					}
				}, 5)

				FIt("should compare integers", func(doneT Done) {
					defer close(doneT)
					defer GinkgoRecover()

					done := make(chan (struct{}))
					vms := initVMs(entry.n, entry.k, entry.bufferCap)
					results := runVMs(done, vms)

					defer close(done)

					k := uint64(30)
					a := SecretField.NewInField(big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << (k - 1))))
					b := SecretField.NewInField(big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << (k - 1))))
					polyA := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), a)
					polyB := algebra.NewRandomPolynomial(SecretField, uint(entry.k/2-1), b)
					sharesA := shamir.Split(polyA, uint64(entry.n))
					sharesB := shamir.Split(polyB, uint64(entry.n))

					id := [32]byte{0x69}
					for i := range vms {
						mem := process.NewMemory(2)
						code := process.Code{
							process.InstMove(mem.At(0), process.NewValuePrivate(sharesA[i])),
							process.InstMove(mem.At(1), process.NewValuePrivate(sharesB[i])),
							process.MacroLT(mem.At(0), mem.At(0), mem.At(1), k, 1, SecretField),
							process.InstOpen(mem.At(0), mem.At(0)),
							process.InstExit(mem.At(0)),
						}
						proc := process.New(id, mem, code)

						vms[i].IO().InputWriter() <- NewExec(proc)
					}

					for _ = range vms {
						var actual TestResult
						Eventually(results, 10).Should(Receive(&actual))
						res, ok := actual.result.Values[0].(process.ValuePublic)
						Expect(ok).To(BeTrue())
						if a.Value().Cmp(b.Value()) == -1 {
							Expect(res.Value.Eq(SecretField.NewInField(big.NewInt(1)))).To(BeTrue())
						} else {
							Expect(res.Value.Eq(SecretField.NewInField(big.NewInt(0)))).To(BeTrue())
						}
					}
				}, 10)
			})
		}
	})

})
