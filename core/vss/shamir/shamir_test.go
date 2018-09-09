package shamir_test

import (
	"math/big"
	"math/rand"

	"github.com/republicprotocol/oro-go/core/vss/algebra"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/vss/algebra"
	. "github.com/republicprotocol/oro-go/core/vss/shamir"
)

var _ = Describe("Shamir secret sharing", func() {
	const Trials = 10

	// randomDegree yields a random degree for constructing a polynomial, in a
	// small range of values.
	randomDegree := func(prime *big.Int) uint {
		r := uint64(rand.Uint32() % 17)
		if prime.Cmp(big.NewInt(int64(r))) != 1 {
			r %= prime.Uint64()
		}
		return uint(r)
	}

	Context("when splitting a secret into shares", func() {
		DescribeTable("it should panic if n is too small", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				if prime.Uint64() == 2 {
					// Not possible to share correctly
					continue
				}
				secret := field.Random()
				k := randomDegree(prime) + 1
				n := rand.Uint64() % uint64(k)
				poly := algebra.NewRandomPolynomial(field, k-1, secret)

				Expect(func() { Split(poly, n) }).To(Panic())
			}
		},
			PrimeEntries...,
		)
	})

	Context("when reconstructing secrets", func() {
		DescribeTable("it should panic if the list of shares is empty", func(prime *big.Int) {
			emptyShares := make(Shares, 0)
			Expect(func() { Join(emptyShares) }).To(Panic())
		},
			PrimeEntries...,
		)

		It("should return an error if shares are in different fields", func() {
			field := NewField(big.NewInt(2))
			otherField := NewField(big.NewInt(3))
			shares := make(Shares, 2)
			shares[0] = New(1, field.Random())
			shares[1] = New(2, otherField.Random())
			_, err := Join(shares)
			Expect(err).To(HaveOccurred())
		})

		DescribeTable("the shares should join back into the original secret", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				if prime.Uint64() == 2 {
					// Not possible to share correctly
					continue
				}
				secret := field.Random()
				n := uint64(24)
				k := randomDegree(prime) + 1
				poly := algebra.NewRandomPolynomial(field, k-1, secret)

				shares := Split(poly, n)

				actual, _ := Join(shares)
				Expect(actual.Eq(secret)).To(BeTrue())

				for i := uint64(0); i < n-uint64(k); i++ {
					actual, _ := Join(shares[i : i+uint64(k)])
					Expect(actual.Eq(secret)).To(BeTrue())
				}
			}
		},
			PrimeEntries...,
		)

		Context("when performing share arithmetic", func() {
			It("should panic when adding shares in different fields", func() {
				field := NewField(big.NewInt(2))
				otherField := NewField(big.NewInt(3))
				share := New(1, field.Random())
				otherShare := New(2, otherField.Random())
				Expect(func() { share.Add(otherShare) }).To(Panic())
			})

			DescribeTable("the sharing should be commutable with addition", func(prime *big.Int) {
				field := NewField(prime)

				for i := 0; i < Trials; i++ {
					if prime.Uint64() == 2 {
						// Not possible to share correctly
						continue
					}
					secretA := field.Random()
					secretB := field.Random()
					n := uint64(24)
					k := randomDegree(prime) + 1
					polyA := algebra.NewRandomPolynomial(field, k-1, secretA)
					polyB := algebra.NewRandomPolynomial(field, k-1, secretB)

					sharesA := Split(polyA, n)
					sharesB := Split(polyB, n)

					secret := secretA.Add(secretB)

					shares := make(Shares, n)
					for i := range shares {
						shares[i] = sharesA[i].Add(sharesB[i])
					}

					actual, _ := Join(shares)
					Expect(actual.Eq(secret)).To(BeTrue())

					for i := uint64(0); i < n-uint64(k); i++ {
						actual, _ := Join(shares[i : i+uint64(k)])
						Expect(actual.Eq(secret)).To(BeTrue())
					}
				}
			},
				PrimeEntries...,
			)
		})

		It("should panic when subtracting shares in different fields", func() {
			field := NewField(big.NewInt(2))
			otherField := NewField(big.NewInt(3))
			share := New(1, field.Random())
			otherShare := New(2, otherField.Random())
			Expect(func() { share.Sub(otherShare) }).To(Panic())
		})

		DescribeTable("the sharing should be commutable with subtraction", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				if prime.Uint64() == 2 {
					// Not possible to share correctly
					continue
				}
				secretA := field.Random()
				secretB := field.Random()
				n := uint64(24)
				k := randomDegree(prime) + 1
				polyA := algebra.NewRandomPolynomial(field, k-1, secretA)
				polyB := algebra.NewRandomPolynomial(field, k-1, secretB)

				sharesA := Split(polyA, n)
				sharesB := Split(polyB, n)

				secret := secretA.Sub(secretB)

				shares := make(Shares, n)
				for i := range shares {
					shares[i] = sharesA[i].Sub(sharesB[i])
				}

				actual, _ := Join(shares)
				Expect(actual.Eq(secret)).To(BeTrue())

				for i := uint64(0); i < n-uint64(k); i++ {
					actual, _ := Join(shares[i : i+uint64(k)])
					Expect(actual.Eq(secret)).To(BeTrue())
				}
			}
		},
			PrimeEntries...,
		)

		It("should panic when multiplying shares in different fields", func() {
			field := NewField(big.NewInt(2))
			otherField := NewField(big.NewInt(3))
			share := New(1, field.Random())
			otherShare := New(2, otherField.Random())
			Expect(func() { share.Mul(otherShare) }).To(Panic())
		})

		DescribeTable("the sharing should be commutable with multiplication when reconstructing with double the threshold", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				if prime.Uint64() == 2 {
					// Not possible to share correctly
					continue
				}
				secretA := field.Random()
				secretB := field.Random()
				n := uint64(40)
				k := randomDegree(prime) + 1
				polyA := algebra.NewRandomPolynomial(field, k-1, secretA)
				polyB := algebra.NewRandomPolynomial(field, k-1, secretB)

				sharesA := Split(polyA, n)
				sharesB := Split(polyB, n)

				secret := secretA.Mul(secretB)

				shares := make(Shares, n)
				for i := range shares {
					shares[i] = sharesA[i].Mul(sharesB[i])
				}

				actual, _ := Join(shares)
				Expect(actual.Eq(secret)).To(BeTrue())

				for i := uint64(0); i < n-uint64(2*k); i++ {
					actual, _ := Join(shares[i : i+uint64(2*k)])
					Expect(actual.Eq(secret)).To(BeTrue())
				}
			}
		},
			PrimeEntries...,
		)
	})
})

// PrimeEntries is a list of table entries of random prime numbers less than
// 2^64
var PrimeEntries = []TableEntry{
	Entry("for the field defined by the prime 2", big.NewInt(2)),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{5, 255, 255, 255, 255, 255, 255, 254, 159})),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{255, 255, 255, 255, 255, 255, 255, 197})),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{59, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 218, 189})),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 97})),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{33, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 230, 231})),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 67})),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{4, 201, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 245, 91, 7})),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 253, 199})),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{5, 169, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 253, 173, 71})),
	Entry("for the field defined by a large prime", big.NewInt(0).SetBytes([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 151})),
	Entry("for the field defined by the prime 11415648579556416673", big.NewInt(0).SetUint64(uint64(11415648579556416673))),
	Entry("for the field defined by the prime 10891814531730287201", big.NewInt(0).SetUint64(uint64(10891814531730287201))),
	Entry("for the field defined by the prime 2173186581265841101", big.NewInt(0).SetUint64(uint64(2173186581265841101))),
	Entry("for the field defined by the prime 8037833094411151351", big.NewInt(0).SetUint64(uint64(8037833094411151351))),
	Entry("for the field defined by the prime 160889637713534993", big.NewInt(0).SetUint64(uint64(160889637713534993))),
	Entry("for the field defined by the prime 2598439422723623851", big.NewInt(0).SetUint64(uint64(2598439422723623851))),
	Entry("for the field defined by the prime 15063151627087255057", big.NewInt(0).SetUint64(uint64(15063151627087255057))),
	Entry("for the field defined by the prime 5652006400289677651", big.NewInt(0).SetUint64(uint64(5652006400289677651))),
	Entry("for the field defined by the prime 1075037556033023437", big.NewInt(0).SetUint64(uint64(1075037556033023437))),
	Entry("for the field defined by the prime 4383237663223642961", big.NewInt(0).SetUint64(uint64(4383237663223642961))),
	Entry("for the field defined by the prime 11491288605849083743", big.NewInt(0).SetUint64(uint64(11491288605849083743))),
	Entry("for the field defined by the prime 18060401258323832179", big.NewInt(0).SetUint64(uint64(18060401258323832179))),
	Entry("for the field defined by the prime 2460931945023125813", big.NewInt(0).SetUint64(uint64(2460931945023125813))),
	Entry("for the field defined by the prime 14720243597953921717", big.NewInt(0).SetUint64(uint64(14720243597953921717))),
	Entry("for the field defined by the prime 11460698326622148979", big.NewInt(0).SetUint64(uint64(11460698326622148979))),
	Entry("for the field defined by the prime 7289555056001917459", big.NewInt(0).SetUint64(uint64(7289555056001917459))),
	Entry("for the field defined by the prime 10819520547428938847", big.NewInt(0).SetUint64(uint64(10819520547428938847))),
	Entry("for the field defined by the prime 17087033667620041241", big.NewInt(0).SetUint64(uint64(17087033667620041241))),
	Entry("for the field defined by the prime 11897098582950941981", big.NewInt(0).SetUint64(uint64(11897098582950941981))),
	Entry("for the field defined by the prime 14162389779744880153", big.NewInt(0).SetUint64(uint64(14162389779744880153))),
	Entry("for the field defined by the prime 3341353876108302833", big.NewInt(0).SetUint64(uint64(3341353876108302833))),
	Entry("for the field defined by the prime 2421057993123425237", big.NewInt(0).SetUint64(uint64(2421057993123425237))),
	Entry("for the field defined by the prime 6099033893113295747", big.NewInt(0).SetUint64(uint64(6099033893113295747))),
	Entry("for the field defined by the prime 9119102700930783271", big.NewInt(0).SetUint64(uint64(9119102700930783271))),
	Entry("for the field defined by the prime 11701444041617194927", big.NewInt(0).SetUint64(uint64(11701444041617194927))),
	Entry("for the field defined by the prime 6492121780466656261", big.NewInt(0).SetUint64(uint64(6492121780466656261))),
	Entry("for the field defined by the prime 1719187971393348791", big.NewInt(0).SetUint64(uint64(1719187971393348791))),
	Entry("for the field defined by the prime 7128898183300867241", big.NewInt(0).SetUint64(uint64(7128898183300867241))),
	Entry("for the field defined by the prime 10448609340017805841", big.NewInt(0).SetUint64(uint64(10448609340017805841))),
	Entry("for the field defined by the prime 5250106197074512951", big.NewInt(0).SetUint64(uint64(5250106197074512951))),
	Entry("for the field defined by the prime 12523635873138238501", big.NewInt(0).SetUint64(uint64(12523635873138238501))),
	Entry("for the field defined by the prime 6179856695580003673", big.NewInt(0).SetUint64(uint64(6179856695580003673))),
	Entry("for the field defined by the prime 14312226640074246223", big.NewInt(0).SetUint64(uint64(14312226640074246223))),
	Entry("for the field defined by the prime 2656168198924335947", big.NewInt(0).SetUint64(uint64(2656168198924335947))),
	Entry("for the field defined by the prime 15282215154228341597", big.NewInt(0).SetUint64(uint64(15282215154228341597))),
	Entry("for the field defined by the prime 5862491744359797091", big.NewInt(0).SetUint64(uint64(5862491744359797091))),
	Entry("for the field defined by the prime 10930389297127849337", big.NewInt(0).SetUint64(uint64(10930389297127849337))),
	Entry("for the field defined by the prime 15453819937382700221", big.NewInt(0).SetUint64(uint64(15453819937382700221))),
	Entry("for the field defined by the prime 8587765603082695229", big.NewInt(0).SetUint64(uint64(8587765603082695229))),
	Entry("for the field defined by the prime 6499635665205708017", big.NewInt(0).SetUint64(uint64(6499635665205708017))),
	Entry("for the field defined by the prime 9522904300687004989", big.NewInt(0).SetUint64(uint64(9522904300687004989))),
	Entry("for the field defined by the prime 6754377453775717483", big.NewInt(0).SetUint64(uint64(6754377453775717483))),
	Entry("for the field defined by the prime 10278941889065878913", big.NewInt(0).SetUint64(uint64(10278941889065878913))),
	Entry("for the field defined by the prime 4119057578904911521", big.NewInt(0).SetUint64(uint64(4119057578904911521))),
	Entry("for the field defined by the prime 2695278052346845627", big.NewInt(0).SetUint64(uint64(2695278052346845627))),
	Entry("for the field defined by the prime 2898709949625550547", big.NewInt(0).SetUint64(uint64(2898709949625550547))),
	Entry("for the field defined by the prime 14640846616444411459", big.NewInt(0).SetUint64(uint64(14640846616444411459))),
	Entry("for the field defined by the prime 8775965213363272289", big.NewInt(0).SetUint64(uint64(8775965213363272289))),
	Entry("for the field defined by the prime 7695258118026415753", big.NewInt(0).SetUint64(uint64(7695258118026415753))),
	Entry("for the field defined by the prime 9112974089849462297", big.NewInt(0).SetUint64(uint64(9112974089849462297))),
	Entry("for the field defined by the prime 14662204281882267989", big.NewInt(0).SetUint64(uint64(14662204281882267989))),
	Entry("for the field defined by the prime 4999606432544782237", big.NewInt(0).SetUint64(uint64(4999606432544782237))),
	Entry("for the field defined by the prime 8961999239135894533", big.NewInt(0).SetUint64(uint64(8961999239135894533))),
	Entry("for the field defined by the prime 14602672531347032081", big.NewInt(0).SetUint64(uint64(14602672531347032081))),
	Entry("for the field defined by the prime 14606570603637462067", big.NewInt(0).SetUint64(uint64(14606570603637462067))),
	Entry("for the field defined by the prime 3662715635181767911", big.NewInt(0).SetUint64(uint64(3662715635181767911))),
	Entry("for the field defined by the prime 15528677330235002987", big.NewInt(0).SetUint64(uint64(15528677330235002987))),
	Entry("for the field defined by the prime 17549052314223638287", big.NewInt(0).SetUint64(uint64(17549052314223638287))),
	Entry("for the field defined by the prime 14793342612719440001", big.NewInt(0).SetUint64(uint64(14793342612719440001))),
	Entry("for the field defined by the prime 1110258828067568087", big.NewInt(0).SetUint64(uint64(1110258828067568087))),
	Entry("for the field defined by the prime 8321432222762641111", big.NewInt(0).SetUint64(uint64(8321432222762641111))),
	Entry("for the field defined by the prime 2099085051126463573", big.NewInt(0).SetUint64(uint64(2099085051126463573))),
	Entry("for the field defined by the prime 17684615516776485691", big.NewInt(0).SetUint64(uint64(17684615516776485691))),
	Entry("for the field defined by the prime 5581192723150425841", big.NewInt(0).SetUint64(uint64(5581192723150425841))),
	Entry("for the field defined by the prime 12295043986397223823", big.NewInt(0).SetUint64(uint64(12295043986397223823))),
	Entry("for the field defined by the prime 4590971551517707183", big.NewInt(0).SetUint64(uint64(4590971551517707183))),
	Entry("for the field defined by the prime 6667954438606055873", big.NewInt(0).SetUint64(uint64(6667954438606055873))),
	Entry("for the field defined by the prime 11257624651846941287", big.NewInt(0).SetUint64(uint64(11257624651846941287))),
	Entry("for the field defined by the prime 11269427064747885857", big.NewInt(0).SetUint64(uint64(11269427064747885857))),
	Entry("for the field defined by the prime 10832662390615802801", big.NewInt(0).SetUint64(uint64(10832662390615802801))),
	Entry("for the field defined by the prime 1149178208693899297", big.NewInt(0).SetUint64(uint64(1149178208693899297))),
	Entry("for the field defined by the prime 7776311754824701427", big.NewInt(0).SetUint64(uint64(7776311754824701427))),
	Entry("for the field defined by the prime 12138619704493513207", big.NewInt(0).SetUint64(uint64(12138619704493513207))),
	Entry("for the field defined by the prime 11715817198039041233", big.NewInt(0).SetUint64(uint64(11715817198039041233))),
	Entry("for the field defined by the prime 8776823877387205793", big.NewInt(0).SetUint64(uint64(8776823877387205793))),
	Entry("for the field defined by the prime 900483851285056369", big.NewInt(0).SetUint64(uint64(900483851285056369))),
	Entry("for the field defined by the prime 10565010275733687859", big.NewInt(0).SetUint64(uint64(10565010275733687859))),
	Entry("for the field defined by the prime 3598475899888315571", big.NewInt(0).SetUint64(uint64(3598475899888315571))),
	Entry("for the field defined by the prime 609292139725849487", big.NewInt(0).SetUint64(uint64(609292139725849487))),
	Entry("for the field defined by the prime 2512663778109890407", big.NewInt(0).SetUint64(uint64(2512663778109890407))),
	Entry("for the field defined by the prime 5356705606915059847", big.NewInt(0).SetUint64(uint64(5356705606915059847))),
	Entry("for the field defined by the prime 4926920292130371833", big.NewInt(0).SetUint64(uint64(4926920292130371833))),
	Entry("for the field defined by the prime 15588936261527250763", big.NewInt(0).SetUint64(uint64(15588936261527250763))),
	Entry("for the field defined by the prime 17674364459850493807", big.NewInt(0).SetUint64(uint64(17674364459850493807))),
	Entry("for the field defined by the prime 15010913622986786653", big.NewInt(0).SetUint64(uint64(15010913622986786653))),
	Entry("for the field defined by the prime 17165846626530660623", big.NewInt(0).SetUint64(uint64(17165846626530660623))),
	Entry("for the field defined by the prime 13953789782321853637", big.NewInt(0).SetUint64(uint64(13953789782321853637))),
	Entry("for the field defined by the prime 9875187539480118827", big.NewInt(0).SetUint64(uint64(9875187539480118827))),
	Entry("for the field defined by the prime 9411830831698978339", big.NewInt(0).SetUint64(uint64(9411830831698978339))),
	Entry("for the field defined by the prime 2181702112484780533", big.NewInt(0).SetUint64(uint64(2181702112484780533))),
	Entry("for the field defined by the prime 15314636212339236139", big.NewInt(0).SetUint64(uint64(15314636212339236139))),
	Entry("for the field defined by the prime 511205612465019343", big.NewInt(0).SetUint64(uint64(511205612465019343))),
	Entry("for the field defined by the prime 8113765242226142771", big.NewInt(0).SetUint64(uint64(8113765242226142771))),
	Entry("for the field defined by the prime 8891182210143952699", big.NewInt(0).SetUint64(uint64(8891182210143952699))),
	Entry("for the field defined by the prime 6315655006279877437", big.NewInt(0).SetUint64(uint64(6315655006279877437))),
	Entry("for the field defined by the prime 8364339317215443659", big.NewInt(0).SetUint64(uint64(8364339317215443659))),
	Entry("for the field defined by the prime 1207853845318533811", big.NewInt(0).SetUint64(uint64(1207853845318533811))),
	Entry("for the field defined by the prime 11869971765257449303", big.NewInt(0).SetUint64(uint64(11869971765257449303))),
	Entry("for the field defined by the prime 17490095259054169019", big.NewInt(0).SetUint64(uint64(17490095259054169019))),
	Entry("for the field defined by the prime 7590272435001495331", big.NewInt(0).SetUint64(uint64(7590272435001495331))),
}
