package algebra_test

import (
	"log"
	"math/big"
	mathrand "math/rand"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vss/algebra"
)

var _ = Describe("Polynomial", func() {
	const Trials = 100

	// randomDegree yields a random degree for constructing a polynomial, in a
	// small range of values.
	randomDegree := func() uint {
		return uint(mathrand.Uint32() % 17)
	}

	Context("when getting polynomial coefficients", func() {
		DescribeTable("it should return the correct coefficients", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree()
				coefficients := make([]*big.Int, degree+1)

				for i := range coefficients {
					coefficients[i] = field.Random()
				}
				poly := NewPolynomial(&field, coefficients)
				actualCoefficients := poly.Coefficients()

				for i := range coefficients {
					Expect(coefficients[i].Cmp(actualCoefficients[i])).To(Equal(0))
				}
			}
		},
			PrimeEntries...,
		)
	})

	Context("when explicitly constructing a polynomial", func() {
		DescribeTable("it should panic when any of the given coefficients are not field elements", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree()
				var nonFieldIndex uint
				if degree == 0 {
					nonFieldIndex = 0
				} else {
					nonFieldIndex = randomDegree() % degree
				}

				coefficients := make([]*big.Int, degree+1)
				for i := 0; i <= int(degree); i++ {
					if i == int(nonFieldIndex) {
						coefficients[i] = RandomNotInField(&field)
					} else {
						coefficients[i] = field.Random()
					}
				}

				Expect(func() { NewPolynomial(&field, coefficients) }).To(Panic())
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when the given coefficients are field elements", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree()
				coefficients := make([]*big.Int, degree+1)
				for i := 0; i <= int(degree); i++ {
					coefficients[i] = field.Random()
				}

				Expect(func() { NewPolynomial(&field, coefficients) }).ToNot(Panic())
			}
		},
			PrimeEntries...,
		)
	})

	Context("when constructing a random polynomial", func() {
		DescribeTable("it should panic when secret has length greater than one", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree()
				// Utilise the randomness from randomDegree for picking the length of secret
				length := randomDegree() + 2
				secret := make([]*big.Int, length)
				for i := 0; i < int(length); i++ {
					secret[i] = field.Random()
				}

				Expect(func() { NewRandomPolynomial(&field, degree, secret...) }).To(Panic())
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when secret has length zero or one", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree()

				if RandomBool() {
					Expect(func() { NewRandomPolynomial(&field, degree) }).ToNot(Panic())
				} else {
					secret := field.Random()
					poly := new(Polynomial)

					Expect(func() { *poly = NewRandomPolynomial(&field, degree, secret) }).ToNot(Panic())
					if prime.Cmp(big.NewInt(2)) == 0 {
						log.Printf("secret: %v, poly: %+v", secret, *poly)
					}
					Expect(poly.Evaluate(big.NewInt(0)).Cmp(secret)).To(Equal(0))
				}

			}
		},
			PrimeEntries...,
		)
	})

	Context("when computing the degree of a polynomial", func() {
		DescribeTable("it should correctly compute the degree for a random polynomial of given degree", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree()
				poly := NewRandomPolynomial(&field, degree)

				Expect(poly.Degree()).To(Equal(degree))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("the degree should be the same with leading zeros", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree()
				zeros := make([]*big.Int, randomDegree())
				poly := NewRandomPolynomial(&field, degree)
				paddedCoefficients := make([]*big.Int, int(degree)+len(zeros)+1)

				for i, c := range poly.Coefficients() {
					paddedCoefficients[i] = big.NewInt(0).Set(c)
				}
				for i := int(degree) + 1; i < len(paddedCoefficients); i++ {
					paddedCoefficients[i] = big.NewInt(0)
				}
				paddedPoly := NewPolynomial(&field, paddedCoefficients)

				Expect(poly.Degree()).To(Equal(paddedPoly.Degree()))
			}
		},
			PrimeEntries...,
		)
	})

	Context("when evaluating a polynomial at a point", func() {
		DescribeTable("it should panic when the point is not in the field", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				poly := NewRandomPolynomial(&field, randomDegree())
				x := RandomNotInField(&field)

				Expect(func() { poly.Evaluate(x) }).To(Panic())
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when the point is in the field", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				poly := NewRandomPolynomial(&field, randomDegree())
				x := field.Random()

				coefficients := poly.Coefficients()

				// Manual evaluation
				accum := big.NewInt(0)
				term := big.NewInt(0)
				for i, c := range coefficients {
					term.Exp(x, big.NewInt(int64(i)), prime)
					term.Mul(term, c)
					accum.Add(accum, term)
					accum.Mod(accum, prime)
				}

				Expect(poly.Evaluate(x).Cmp(accum)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)
	})
})