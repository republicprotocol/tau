package vm_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/oro-go/core/task"
	"github.com/republicprotocol/oro-go/core/taskutils"
	"github.com/republicprotocol/oro-go/core/vm/asm"
	"github.com/republicprotocol/oro-go/core/vm/macro"
	"github.com/republicprotocol/oro-go/core/vm/proc"
	"github.com/republicprotocol/oro-go/core/vm/rng"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/pedersen"
	"github.com/republicprotocol/oro-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/vm"
)

var _ = Describe("Virtual Machine", func() {

	p := big.NewInt(8589934583)
	q := big.NewInt(4294967291)
	g := algebra.NewFpElement(big.NewInt(592772542), p)
	h := algebra.NewFpElement(big.NewInt(4799487786), p)
	fp := algebra.NewField(q)
	scheme := pedersen.New(g, h, fp)

	zero := fp.NewInField(big.NewInt(0))
	one := fp.NewInField(big.NewInt(1))

	init := func(n, k uint64, cap int) task.Tasks {
		ts := make(task.Tasks, n)
		for i := 0; i < len(ts); i++ {
			ts[i] = New(scheme, uint64(i)+1, n, k, cap)
		}
		return ts
	}

	run := func(done <-chan struct{}, results chan<- Result, ts task.Tasks, simulatedFailureRate float64, simulatedFailureLimit int) {

		failures := 0
		rnSharesFailures := 0
		proposeRnSharesFailures := 0

		task.New(task.NewIO(128), task.NewReducer(func(message task.Message) task.Message {

			switch message := message.(type) {
			case RemoteProcedureCall:
				switch message := message.Message.(type) {

				case rng.RnShares:
					modifiedSimulatedFailureLimit := 1
					if rnSharesFailures >= simulatedFailureLimit || failures >= simulatedFailureLimit {
						modifiedSimulatedFailureLimit = 0
					}
					x := taskutils.RouteMessage(done, NewRemoteProcedureCall(message), task.Tasks{ts[0]}, simulatedFailureRate, modifiedSimulatedFailureLimit)
					rnSharesFailures += x
					failures += x

				case rng.ProposeRnShare:
					modifiedSimulatedFailureLimit := 1
					if proposeRnSharesFailures >= simulatedFailureLimit || failures >= simulatedFailureLimit {
						modifiedSimulatedFailureLimit = 0
					}
					x := taskutils.RouteMessage(done, NewRemoteProcedureCall(message), task.Tasks{ts[message.To-1]}, simulatedFailureRate, modifiedSimulatedFailureLimit)
					proposeRnSharesFailures += x
					failures += x

				default:
					modifiedSimulatedFailureLimit := simulatedFailureLimit
					if failures >= simulatedFailureLimit {
						modifiedSimulatedFailureLimit = 0
					}
					x := taskutils.RouteMessage(done, NewRemoteProcedureCall(message), ts, simulatedFailureRate, modifiedSimulatedFailureLimit)
					failures += x
				}

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

	runProcess := func(n, k uint64, cap int, failureRate float64, buildProc func(i int) proc.Proc, verifyResult func(i int, val asm.Value)) {

		vms := init(n, k, cap)
		done := make(chan struct{})
		results := make(chan Result)

		co.ParBegin(
			func() {
				run(done, results, vms, failureRate, int(k/2))
			},
			func() {
				for i := range vms {
					proc := buildProc(i)
					vms[i].IO().InputWriter() <- NewExec(proc)
				}
			},
			func() {
				defer close(done)
				defer GinkgoRecover()

				resultsDone := make(chan struct{})
				resultsNum := uint64(0)
				resultsBuf := make(chan struct {
					i     int
					value asm.Value
				}, int(n))

				go func() {
					defer close(resultsBuf)
					co.ParForAll(vms, func(i int) {
						var result Result
						select {
						case <-resultsDone:
							return
						case result = <-results:
							if x := atomic.AddUint64(&resultsNum, 1); x == k/2 {
								close(resultsDone)
							}
						}
						for i, value := range result.Values {
							resultsBuf <- struct {
								i     int
								value asm.Value
							}{i, value}
						}
					})
				}()

				for result := range resultsBuf {
					verifyResult(result.i, result.value)
				}
				Expect(resultsNum).To(BeNumerically(">=", k/2))
			})
	}

	split := func(value algebra.FpElement, n, k uint64) []asm.ValuePrivate {
		poly := algebra.NewRandomPolynomial(fp, uint(k-1), value)
		shares := shamir.Split(poly, n)
		values := make([]asm.ValuePrivate, n)
		for i := range values {
			values[i] = asm.NewValuePrivate(shares[i])
		}
		return values
	}

	splitToBits := func(value *big.Int, bits int, n, k uint64) [][]asm.ValuePrivate {
		tmp := big.NewInt(0).Set(value)
		tmpBits := make(algebra.FpElements, bits)
		for i := range tmpBits {
			r := big.NewInt(0).Mod(tmp, big.NewInt(2))
			tmpBits[i] = fp.NewInField(r)
			tmp.Div(tmp, big.NewInt(2))
		}
		valuesBits := make([][]asm.ValuePrivate, n)
		for i := range valuesBits {
			valuesBits[i] = make([]asm.ValuePrivate, bits)
		}
		for i := 0; i < bits; i++ {
			values := split(tmpBits[i], n, k)
			for j, value := range values {
				valuesBits[j][i] = value
			}
		}
		return valuesBits
	}

	randomProcID := func() proc.ID {
		pid := proc.ID{}
		n, err := rand.Read(pid[:])
		if n != len(pid) {
			if err != nil {
				panic(fmt.Sprintf("failed to generate %v random bytes = %v", n, err))
			}
			panic(fmt.Sprintf("failed to generate %v random bytes", n))
		}
		if err != nil {
			panic(err)
		}
		return pid
	}

	randomBool := func() bool {
		return rand.Float32() < 0.5
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
			{12, 8},
		}
		tableCap := []struct {
			cap int
		}{
			{32},
			{256},
			{1024},
		}

		for _, entryNK := range tableNK {
			entryNK := entryNK

			for _, entryCap := range tableCap {
				entryCap := entryCap

				Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entryNK.n, entryNK.k, entryCap.cap), func() {
					It("should shutdown and clean up", func(doneT Done) {
						defer close(doneT)

						vms := init(entryNK.n, entryNK.k, entryCap.cap)
						done := make(chan struct{})
						results := make(chan Result)

						co.ParBegin(
							func() {
								run(done, results, vms, 0.0, 0)
							},
							func() {
								close(done)
							})
					})
				})
			}
		}
	})

	Context("when running virtual machines", func() {

		tableNK := []struct {
			n, k uint64
		}{
			{1, 1},
			{3, 2},
			{12, 8},
		}
		tableCap := []struct {
			cap int
		}{
			{32},
			{256},
			{1024},
		}
		tableFailureRate := []struct {
			failureRate float64
		}{
			{0.0}, {0.05}, {0.10}, {0.20}, {0.40},
		}

		for _, entryNK := range tableNK {
			entryNK := entryNK

			for _, entryCap := range tableCap {
				entryCap := entryCap

				for _, entryFailureRate := range tableFailureRate {
					entryFailureRate := entryFailureRate

					Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entryNK.n, entryNK.k, entryCap.cap), func() {
						Context(fmt.Sprintf("when simulated network connectivity = %v%%", 100.0-100*entryFailureRate.failureRate), func() {

							It("should add public numbers", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()
								a, b := asm.NewValuePublic(fp.Random()), asm.NewValuePublic(fp.Random())
								expected := a.Add(b).(asm.ValuePublic)

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(2)
										return proc.New(pid, []asm.Inst{
											asm.InstMove(mem.Offset(0), a),
											asm.InstMove(mem.Offset(1), b),
											asm.InstAdd(mem.Offset(0), mem.Offset(0), mem.Offset(1), 1),
											asm.InstExit(mem.Offset(0), 1),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())
										Expect(res.Value.Eq(expected.Value)).To(BeTrue())
									})
							})

							It("should add private numbers", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()
								a, b := fp.Random(), fp.Random()
								as, bs := split(a, entryNK.n, (entryNK.k+1)/2), split(b, entryNK.n, (entryNK.k+1)/2)
								expected := a.Add(b)

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(2)
										return proc.New(pid, []asm.Inst{
											asm.InstMove(mem.Offset(0), as[i]),
											asm.InstMove(mem.Offset(1), bs[i]),
											asm.InstAdd(mem.Offset(0), mem.Offset(0), mem.Offset(1), 1),
											asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
											asm.InstExit(mem.Offset(0), 1),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())
										Expect(res.Value.Eq(expected)).To(BeTrue())
									})
							})

							It("should add public numbers with private numbers", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()
								a, b := fp.Random(), fp.Random()
								bs := split(b, entryNK.n, (entryNK.k+1)/2)
								expected := a.Add(b)

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(2)
										return proc.New(pid, []asm.Inst{
											asm.InstMove(mem.Offset(0), asm.NewValuePublic(a)),
											asm.InstMove(mem.Offset(1), bs[i]),
											asm.InstAdd(mem.Offset(0), mem.Offset(0), mem.Offset(1), 1),
											asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
											asm.InstExit(mem.Offset(0), 1),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())
										Expect(res.Value.Eq(expected)).To(BeTrue())
									})
							})

							It("should generate private random numbers", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()
								var expected *asm.ValuePublic

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(2)
										return proc.New(pid, []asm.Inst{
											asm.InstGenerateRnTuple(mem.Offset(0), mem.Offset(1), 1),
											asm.InstOpen(mem.Offset(0), mem.Offset(0), 2),
											asm.InstExit(mem.Offset(0), 2),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())
										if expected == nil {
											expected = &res
										}
										Expect(res.Value.Eq(expected.Value)).To(BeTrue())
									})
							})

							It("should generate private random zeros", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(1)
										return proc.New(pid, []asm.Inst{
											asm.InstGenerateRnZero(mem.Offset(0), 1),
											asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
											asm.InstExit(mem.Offset(0), 1),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())
										Expect(res.Value.IsZero()).To(BeTrue())
									})
							})

							It("should multiply private numbers", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()
								a, b := fp.Random(), fp.Random()
								as, bs := split(a, entryNK.n, (entryNK.k+1)/2), split(b, entryNK.n, (entryNK.k+1)/2)
								expected := a.Mul(b)

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(4)
										return proc.New(pid, []asm.Inst{
											asm.InstMove(mem.Offset(0), as[i]),
											asm.InstMove(mem.Offset(1), bs[i]),
											asm.InstGenerateRnTuple(mem.Offset(2), mem.Offset(3), 1),
											asm.InstMul(mem.Offset(0), mem.Offset(0), mem.Offset(1), mem.Offset(2), mem.Offset(3), 1),
											asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
											asm.InstExit(mem.Offset(0), 1),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())
										Expect(res.Value.Eq(expected)).To(BeTrue())
									})
							})

							tableNotGate := []struct {
								a, out algebra.FpElement
							}{
								{zero, one},
								{one, zero},
							}
							for _, entryNotGate := range tableNotGate {
								entryNotGate := entryNotGate

								It(fmt.Sprintf("should compute a not gate on a = %v", entryNotGate.a.Value()), func(doneT Done) {
									defer close(doneT)
									defer GinkgoRecover()

									pid := randomProcID()
									a := entryNotGate.a
									as := split(a, entryNK.n, (entryNK.k+1)/2)

									runProcess(
										entryNK.n, entryNK.k, entryCap.cap,
										entryFailureRate.failureRate,
										func(i int) proc.Proc {
											mem := asm.Alloc(1)
											return proc.New(pid, []asm.Inst{
												asm.InstMove(mem.Offset(0), as[i]),
												macro.BitwiseNot(mem.Offset(0), mem.Offset(0), 1, fp),
												asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
												asm.InstExit(mem.Offset(0), 1),
											})
										},
										func(i int, value asm.Value) {
											defer GinkgoRecover()

											res, ok := value.(asm.ValuePublic)
											Expect(ok).To(BeTrue())
											Expect(res.Value.Eq(entryNotGate.out)).To(BeTrue())
										})
								})
							}

							tableOrGate := []struct {
								a, b, out algebra.FpElement
							}{
								{zero, zero, zero},
								{zero, one, one},
								{one, zero, one},
								{one, one, one},
							}
							for _, entryOrGate := range tableOrGate {
								entryOrGate := entryOrGate

								It(fmt.Sprintf("should compute an or gate on a = %v, b = %v", entryOrGate.a.Value(), entryOrGate.b.Value()), func(doneT Done) {
									defer close(doneT)
									defer GinkgoRecover()

									pid := randomProcID()
									a, b := entryOrGate.a, entryOrGate.b
									as, bs := split(a, entryNK.n, (entryNK.k+1)/2), split(b, entryNK.n, (entryNK.k+1)/2)

									runProcess(
										entryNK.n, entryNK.k, entryCap.cap,
										entryFailureRate.failureRate,
										func(i int) proc.Proc {
											mem := asm.Alloc(4)
											return proc.New(pid, []asm.Inst{
												asm.InstMove(mem.Offset(0), as[i]),
												asm.InstMove(mem.Offset(1), bs[i]),
												asm.InstGenerateRnTuple(mem.Offset(2), mem.Offset(3), 1),
												macro.BitwiseOr(mem.Offset(0), mem.Offset(0), mem.Offset(1), mem.Offset(2), mem.Offset(3), 1),
												asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
												asm.InstExit(mem.Offset(0), 1),
											})
										},
										func(i int, value asm.Value) {
											defer GinkgoRecover()

											res, ok := value.(asm.ValuePublic)
											Expect(ok).To(BeTrue())
											Expect(res.Value.Eq(entryOrGate.out)).To(BeTrue())
										})
								})
							}

							tableXorGate := []struct {
								a, b, out algebra.FpElement
							}{
								{zero, zero, zero},
								{zero, one, one},
								{one, zero, one},
								{one, one, zero},
							}
							for _, entryXorGate := range tableXorGate {
								entryXorGate := entryXorGate

								It(fmt.Sprintf("should compute an or gate on a = %v, b = %v", entryXorGate.a.Value(), entryXorGate.b.Value()), func(doneT Done) {
									defer close(doneT)
									defer GinkgoRecover()

									pid := randomProcID()
									a, b := entryXorGate.a, entryXorGate.b
									as, bs := split(a, entryNK.n, (entryNK.k+1)/2), split(b, entryNK.n, (entryNK.k+1)/2)

									runProcess(
										entryNK.n, entryNK.k, entryCap.cap,
										entryFailureRate.failureRate,
										func(i int) proc.Proc {
											mem := asm.Alloc(4)
											return proc.New(pid, []asm.Inst{
												asm.InstMove(mem.Offset(0), as[i]),
												asm.InstMove(mem.Offset(1), bs[i]),
												asm.InstGenerateRnTuple(mem.Offset(2), mem.Offset(3), 1),
												macro.BitwiseXor(mem.Offset(0), mem.Offset(0), mem.Offset(1), mem.Offset(2), mem.Offset(3), 1),
												asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
												asm.InstExit(mem.Offset(0), 1),
											})
										},
										func(i int, value asm.Value) {
											defer GinkgoRecover()

											res, ok := value.(asm.ValuePublic)
											Expect(ok).To(BeTrue())
											Expect(res.Value.Eq(entryXorGate.out)).To(BeTrue())
										})
								})
							}

							tableAndGate := []struct {
								a, b, out algebra.FpElement
							}{
								{zero, zero, zero},
								{zero, one, zero},
								{one, zero, zero},
								{one, one, one},
							}
							for _, entryAndGate := range tableAndGate {
								entryAndGate := entryAndGate

								It(fmt.Sprintf("should compute an or gate on a = %v, b = %v", entryAndGate.a.Value(), entryAndGate.b.Value()), func(doneT Done) {
									defer close(doneT)
									defer GinkgoRecover()

									pid := randomProcID()
									a, b := entryAndGate.a, entryAndGate.b
									as, bs := split(a, entryNK.n, (entryNK.k+1)/2), split(b, entryNK.n, (entryNK.k+1)/2)

									runProcess(
										entryNK.n, entryNK.k, entryCap.cap,
										entryFailureRate.failureRate,
										func(i int) proc.Proc {
											mem := asm.Alloc(4)
											return proc.New(pid, []asm.Inst{
												asm.InstMove(mem.Offset(0), as[i]),
												asm.InstMove(mem.Offset(1), bs[i]),
												asm.InstGenerateRnTuple(mem.Offset(2), mem.Offset(3), 1),
												macro.BitwiseAnd(mem.Offset(0), mem.Offset(0), mem.Offset(1), mem.Offset(2), mem.Offset(3), 1),
												asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
												asm.InstExit(mem.Offset(0), 1),
											})
										},
										func(i int, value asm.Value) {
											defer GinkgoRecover()

											res, ok := value.(asm.ValuePublic)
											Expect(ok).To(BeTrue())
											Expect(res.Value.Eq(entryAndGate.out)).To(BeTrue())
										})
								})
							}

							tablePropGen := []struct {
								a, b, p, g algebra.FpElement
							}{
								{zero, zero, zero, zero},
								{zero, one, one, zero},
								{one, zero, one, zero},
								{one, one, zero, one},
							}
							for _, entryPropGenGate := range tablePropGen {
								entryPropGenGate := entryPropGenGate

								It(fmt.Sprintf("should correctly compute the propagator and generator on a = %v, b = %v", entryPropGenGate.a.Value(), entryPropGenGate.b.Value()), func(doneT Done) {
									defer close(doneT)
									defer GinkgoRecover()

									pid := randomProcID()
									a, b := entryPropGenGate.a, entryPropGenGate.b
									as, bs := split(a, entryNK.n, (entryNK.k+1)/2), split(b, entryNK.n, (entryNK.k+1)/2)

									runProcess(
										entryNK.n, entryNK.k, entryCap.cap,
										entryFailureRate.failureRate,
										func(i int) proc.Proc {
											mem := asm.Alloc(6)
											return proc.New(pid, []asm.Inst{
												asm.InstMove(mem.Offset(0), as[i]),
												asm.InstMove(mem.Offset(1), bs[i]),
												asm.InstGenerateRnTuple(mem.Offset(2), mem.Offset(4), 2),
												macro.BitwisePropGen(mem.Offset(0), mem.Offset(1), mem.Offset(0), mem.Offset(1), mem.Offset(2), mem.Offset(4), 1),
												asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
												asm.InstOpen(mem.Offset(1), mem.Offset(1), 1),
												asm.InstExit(mem.Offset(0), 2),
											})
										},
										func(i int, value asm.Value) {
											defer GinkgoRecover()

											res, ok := value.(asm.ValuePublic)
											Expect(ok).To(BeTrue())

											if i == 0 {
												Expect(res.Value.Eq(entryPropGenGate.p)).To(BeTrue())
											}
											if i == 1 {
												Expect(res.Value.Eq(entryPropGenGate.g)).To(BeTrue())
											}
										})
								})
							}

							tableCLA := []struct {
								p1, g1, p2, g2, pp, gg algebra.FpElement
							}{
								{zero, zero, zero, zero, zero, zero},
								{zero, zero, zero, one, zero, one},
								{zero, zero, one, zero, zero, zero},
								{zero, zero, one, one, zero, one},
								{zero, one, zero, zero, zero, zero},
								{zero, one, zero, one, zero, one},
								{zero, one, one, zero, zero, one},
								{zero, one, one, one, zero, one},
								{one, zero, zero, zero, zero, zero},
								{one, zero, zero, one, zero, one},
								{one, zero, one, zero, one, zero},
								{one, zero, one, one, one, one},
								{one, one, zero, zero, zero, zero},
								{one, one, zero, one, zero, one},
								{one, one, one, zero, one, one},
								{one, one, one, one, one, one},
							}
							for _, entryCLAGate := range tableCLA {
								entryCLAGate := entryCLAGate

								It(fmt.Sprintf("should correctly compute the CLA operation on p1, g1, p2, g2, pp, gg = %v, %v, %v, %v, %v, %v", entryCLAGate.p1.Value(), entryCLAGate.g1.Value(), entryCLAGate.p2.Value(), entryCLAGate.g2.Value(), entryCLAGate.pp.Value(), entryCLAGate.gg.Value()), func(doneT Done) {
									defer close(doneT)
									defer GinkgoRecover()

									pid := randomProcID()
									p1, g1, p2, g2 := entryCLAGate.p1, entryCLAGate.g1, entryCLAGate.p2, entryCLAGate.g2
									p1s, g1s, p2s, g2s := split(p1, entryNK.n, (entryNK.k+1)/2), split(g1, entryNK.n, (entryNK.k+1)/2), split(p2, entryNK.n, (entryNK.k+1)/2), split(g2, entryNK.n, (entryNK.k+1)/2)

									runProcess(
										entryNK.n, entryNK.k, entryCap.cap,
										entryFailureRate.failureRate,
										func(i int) proc.Proc {
											mem := asm.Alloc(10)
											return proc.New(pid, []asm.Inst{
												asm.InstMove(mem.Offset(0), p1s[i]),
												asm.InstMove(mem.Offset(1), g1s[i]),
												asm.InstMove(mem.Offset(2), p2s[i]),
												asm.InstMove(mem.Offset(3), g2s[i]),
												asm.InstGenerateRnTuple(mem.Offset(4), mem.Offset(7), 3),
												macro.BitwiseOpCLA(mem.Offset(0), mem.Offset(1), mem.Offset(0), mem.Offset(1), mem.Offset(2), mem.Offset(3), mem.Offset(4), mem.Offset(7), 1),
												asm.InstOpen(mem.Offset(0), mem.Offset(0), 2),
												asm.InstExit(mem.Offset(0), 2),
											})
										},
										func(i int, value asm.Value) {
											defer GinkgoRecover()

											res, ok := value.(asm.ValuePublic)
											Expect(ok).To(BeTrue())

											if i == 0 {
												Expect(res.Value.Eq(entryCLAGate.pp)).To(BeTrue())
											}
											if i == 1 {
												Expect(res.Value.Eq(entryCLAGate.gg)).To(BeTrue())
											}
										})
								})
							}

							tableBits := []struct {
								bits int
							}{
								{1}, {2}, {3}, {6}, {7}, {14}, {15}, {30}, {31},
							}
							for _, entryBits := range tableBits {
								entryBits := entryBits

								It(fmt.Sprintf("should correctly compute the carry out operation on a %v-bit number", entryBits.bits), func(doneT Done) {
									defer close(doneT)
									defer GinkgoRecover()

									pid := randomProcID()
									a := big.NewInt(0).SetUint64(rand.Uint64() % (1 << uint64(entryBits.bits)))
									b := big.NewInt(0).SetUint64(rand.Uint64() % (1 << uint64(entryBits.bits)))
									as := splitToBits(a, entryBits.bits, entryNK.n, (entryNK.k+1)/2)
									bs := splitToBits(b, entryBits.bits, entryNK.n, (entryNK.k+1)/2)

									runProcess(
										entryNK.n, entryNK.k, entryCap.cap,
										entryFailureRate.failureRate,
										func(i int) proc.Proc {
											mem := asm.Alloc(1)
											memA := asm.Alloc(entryBits.bits)
											memB := asm.Alloc(entryBits.bits)
											for j := 0; j < entryBits.bits; j++ {
												memA.Store(j, as[i][j])
												memB.Store(j, bs[i][j])
											}
											return proc.New(pid, []asm.Inst{
												macro.BitwiseCarryOut(mem.Offset(0), memA.Offset(0), memB.Offset(0), false, int(entryBits.bits), fp),
												asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
												asm.InstExit(mem.Offset(0), 1),
											})
										},
										func(i int, value asm.Value) {
											defer GinkgoRecover()

											res, ok := value.(asm.ValuePublic)
											Expect(ok).To(BeTrue())

											if big.NewInt(0).Add(a, b).Cmp(big.NewInt(0).SetUint64(1<<uint64(entryBits.bits))) >= 0 {
												Expect(res.Value.Eq(one)).To(BeTrue())
											} else {
												Expect(res.Value.Eq(zero)).To(BeTrue())
											}
										})
								}, 30)
							}

							It("should correctly compute bitwise LT on k bit numbers", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()
								k := uint64(32)
								a := big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << k))
								b := big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << k))
								aTemp := big.NewInt(0).Set(a)
								bTemp := big.NewInt(0).Set(b)

								aBits := make([]algebra.FpElement, k)
								bBits := make([]algebra.FpElement, k)
								for i := range aBits {
									ar := big.NewInt(0).Mod(aTemp, big.NewInt(2))
									br := big.NewInt(0).Mod(bTemp, big.NewInt(2))
									aBits[i] = fp.NewInField(ar)
									bBits[i] = fp.NewInField(br)
									aTemp.Div(aTemp, big.NewInt(2))
									bTemp.Div(bTemp, big.NewInt(2))
								}

								aVals := make([][]asm.ValuePrivate, entryNK.n)
								bVals := make([][]asm.ValuePrivate, entryNK.n)
								for i := range aVals {
									aVals[i] = make([]asm.ValuePrivate, k)
									bVals[i] = make([]asm.ValuePrivate, k)
								}

								for i := uint64(0); i < k; i++ {
									sharesA := split(aBits[i], uint64(entryNK.n), (entryNK.k+1)/2)
									sharesB := split(bBits[i], uint64(entryNK.n), (entryNK.k+1)/2)

									for j, share := range sharesA {
										aVals[j][i] = share
									}
									for j, share := range sharesB {
										bVals[j][i] = share
									}
								}

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(1)
										memA := asm.Alloc(int(k))
										memB := asm.Alloc(int(k))
										for j := 0; j < int(k); j++ {
											memA.Store(j, aVals[i][j])
											memB.Store(j, bVals[i][j])
										}
										return proc.New(pid, []asm.Inst{
											macro.BitwiseLT(mem.Offset(0), memA.Offset(0), memB.Offset(0), int(k), fp),
											asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
											asm.InstExit(mem.Offset(0), 1),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())

										if a.Cmp(b) == -1 {
											Expect(res.Value.Eq(fp.NewInField(big.NewInt(1)))).To(BeTrue())
										} else {
											Expect(res.Value.Eq(fp.NewInField(big.NewInt(0)))).To(BeTrue())
										}
									})
							}, 30)

							It("should compute integers modulo powers of two", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()
								bits := uint64(25)
								m := uint64(16)
								kappa := 5
								a := fp.NewInField(big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << (bits - 1))))

								negCase := randomBool()
								if negCase {
									a = a.Neg()
								}
								shares := split(a, uint64(entryNK.n), (entryNK.k+1)/2)

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(1)
										return proc.New(pid, []asm.Inst{
											asm.InstMove(mem, shares[i]),
											macro.Mod2M(mem, mem, int(bits), int(m), kappa, fp),
											asm.InstOpen(mem, mem, 1),
											asm.InstExit(mem, 1),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())

										twoPow := big.NewInt(0).SetUint64(uint64(1) << m)
										var mod *big.Int
										if negCase {
											mod = big.NewInt(0).Mod(big.NewInt(0).Neg(a.Neg().Value()), twoPow)
										} else {
											mod = big.NewInt(0).Mod(a.Value(), twoPow)
										}

										Expect(mod.Cmp(res.Value.Value())).To(Equal(0))
									})
							}, 30)

							It("should generate random bits", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(10)
										return proc.New(pid, []asm.Inst{
											macro.GenerateRandomBit(mem, 10, fp),
											asm.InstOpen(mem, mem, 10),
											asm.InstExit(mem, 10),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())

										if !res.Value.IsZero() {
											Expect(res.Value.IsOne()).To(BeTrue())
										}
									})
							})

							It("should compute the binary representation of a number", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()
								a := fp.NewInField(big.NewInt(0).SetUint64(uint64(rand.Uint32())))

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(32)
										return proc.New(pid, []asm.Inst{
											asm.InstMove(mem.Offset(0), asm.NewValuePublic(a)),
											macro.Bits(mem.Offset(0), mem.Offset(0), 32, fp),
											asm.InstExit(mem.Offset(0), 32),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())

										expectedBit := (a.Value().Uint64() % (1 << (uint64(i) + 1))) >> uint64(i)
										Expect(res.Value.Value().Uint64()).To(Equal(expectedBit))
									})
							})

							It("should compare integers", func(doneT Done) {
								defer close(doneT)
								defer GinkgoRecover()

								pid := randomProcID()
								bits := uint64(30)
								kappa := 1
								a := fp.NewInField(big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << (bits - 1))))
								b := fp.NewInField(big.NewInt(0).SetUint64(rand.Uint64() % (uint64(1) << (bits - 1))))
								sharesA := split(a, uint64(entryNK.n), (entryNK.k+1)/2)
								sharesB := split(b, uint64(entryNK.n), (entryNK.k+1)/2)

								runProcess(
									entryNK.n, entryNK.k, entryCap.cap,
									entryFailureRate.failureRate,
									func(i int) proc.Proc {
										mem := asm.Alloc(2)
										return proc.New(pid, []asm.Inst{
											asm.InstMove(mem.Offset(0), sharesA[i]),
											asm.InstMove(mem.Offset(1), sharesB[i]),
											macro.LT(mem.Offset(0), mem.Offset(0), mem.Offset(1), int(bits), kappa, fp),
											asm.InstOpen(mem.Offset(0), mem.Offset(0), 1),
											asm.InstExit(mem.Offset(0), 1),
										})
									},
									func(i int, value asm.Value) {
										defer GinkgoRecover()

										res, ok := value.(asm.ValuePublic)
										Expect(ok).To(BeTrue())

										if a.Value().Cmp(b.Value()) == -1 {
											Expect(res.Value.Eq(fp.NewInField(big.NewInt(1)))).To(BeTrue())
										} else {
											Expect(res.Value.Eq(fp.NewInField(big.NewInt(0)))).To(BeTrue())
										}
									})
							}, 30)
						})
					})
				}
			}
		}
	})

	Context("when creating messages", func() {
		It("should implement the message interface for all messages", func() {
			Exec{}.IsMessage()
			RemoteProcedureCall{}.IsMessage()
			Result{}.IsMessage()
		})
	})
})
