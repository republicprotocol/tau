package algebra_test

import (
	"log"
	"math/big"
	"math/rand"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vss/algebra"
)

var _ = Describe("Polynomial", func() {
	const Trials = 100

	// randomDegree yields a random degree for constructing a polynomial, in a
	// small range of values.
	randomDegree := func(prime *big.Int) uint {
		r := uint64(rand.Uint32() % 17)
		if prime.Cmp(big.NewInt(int64(r))) != 1 {
			r %= prime.Uint64()
		}
		return uint(r)
	}

	Context("when getting polynomial coefficients", func() {
		DescribeTable("it should return the correct coefficients", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree(prime)
				coefficients := make([]FpElement, degree+1)

				for i := range coefficients {
					coefficients[i] = field.Random()
				}
				poly := NewPolynomial(coefficients)
				actualCoefficients := poly.Coefficients()

				Expect(len(coefficients)).To(Equal(len(actualCoefficients)))

				for i := range coefficients {
					Expect(coefficients[i].Eq(actualCoefficients[i])).To(BeTrue())
				}
			}
		},
			PrimeEntries...,
		)
	})

	Context("when explicitly constructing a polynomial", func() {
		DescribeTable("it should panic when there are no coefficients", func(prime *big.Int) {
			for i := 0; i < Trials; i++ {
				coefficients := make([]FpElement, 0)
				Expect(func() { NewPolynomial(coefficients) }).To(Panic())
			}
		},
			PrimeEntries...,
		)

		It("should panic when there are too many coefficients", func() {
			prime := big.NewInt(2)
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree(prime)
				degree += uint(prime.Uint64())
				coefficients := make([]FpElement, degree+1)
				for i := 0; i <= int(degree); i++ {
					coefficients[i] = field.Random()
				}

				Expect(func() { NewPolynomial(coefficients) }).To(Panic())
			}
		})

		DescribeTable("it should panic when any of the given coefficients are not in the same field as each other", func(prime *big.Int) {
			field := NewField(prime)
			otherField := NewField(big.NewInt(7))

			for i := 0; i < Trials; i++ {
				degree := randomDegree(prime)
				if degree == 0 {
					// This case will not panic
					continue
				}
				var nonFieldIndex uint
				if degree == 0 {
					nonFieldIndex = 0
				} else {
					nonFieldIndex = randomDegree(prime) % degree
				}

				coefficients := make([]FpElement, degree+1)
				for i := 0; i <= int(degree); i++ {
					if i == int(nonFieldIndex) {
						coefficients[i] = otherField.Random()
					} else {
						coefficients[i] = field.Random()
					}
				}

				Expect(func() { NewPolynomial(coefficients) }).To(Panic())
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when the given coefficients are in the same field", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree(prime)
				coefficients := make([]FpElement, degree+1)
				for i := 0; i <= int(degree); i++ {
					coefficients[i] = field.Random()
				}

				Expect(func() { NewPolynomial(coefficients) }).ToNot(Panic())
			}
		},
			PrimeEntries...,
		)
	})

	Context("when constructing a random polynomial", func() {
		DescribeTable("it should panic when the degree is too large", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree(prime)
				degree += uint(prime.Uint64())

				Expect(func() { NewRandomPolynomial(field, degree) }).To(Panic())
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should panic when secret has length greater than one", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree(prime)
				// Utilise the randomness from randomDegree for picking the length of secret
				length := randomDegree(prime) + 2
				secret := make([]FpElement, length)
				for i := 0; i < int(length); i++ {
					secret[i] = field.Random()
				}

				Expect(func() { NewRandomPolynomial(field, degree, secret...) }).To(Panic())
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when secret has length zero or one", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree(prime)

				if RandomBool() {
					Expect(func() { NewRandomPolynomial(field, degree) }).ToNot(Panic())
				} else {
					secret := field.Random()
					poly := new(Polynomial)

					Expect(func() { *poly = NewRandomPolynomial(field, degree, secret) }).ToNot(Panic())
					if prime.Cmp(big.NewInt(2)) == 0 {
						log.Printf("secret: %v, poly: %+v", secret, *poly)
					}
					Expect(poly.Evaluate(field.NewInField(big.NewInt(0))).Eq(secret)).To(BeTrue())
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
				degree := randomDegree(prime)
				poly := NewRandomPolynomial(field, degree)

				Expect(poly.Degree()).To(Equal(degree))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("the degree should be the same with leading zeros", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				degree := randomDegree(prime)
				pad := randomDegree(prime)
				if uint64(degree+pad+1) >= prime.Uint64() {
					// Ignore the case where the padded coefficient list is too
					// long
					continue
				}
				zeros := make([]FpElement, randomDegree(prime))
				poly := NewRandomPolynomial(field, degree)
				paddedCoefficients := make([]FpElement, int(degree)+len(zeros)+1)

				for i, c := range poly.Coefficients() {
					paddedCoefficients[i] = c.Copy()
				}
				for i := int(degree) + 1; i < len(paddedCoefficients); i++ {
					paddedCoefficients[i] = field.NewInField(big.NewInt(0))
				}
				paddedPoly := NewPolynomial(paddedCoefficients)

				Expect(poly.Degree()).To(Equal(paddedPoly.Degree()))
			}
		},
			PrimeEntries...,
		)
	})

	Context("when evaluating a polynomial at a point", func() {
		DescribeTable("it should panic when the point is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			otherField := NewField(big.NewInt(7))

			for i := 0; i < Trials; i++ {
				poly := NewRandomPolynomial(field, randomDegree(prime))
				x := otherField.Random()

				Expect(func() { poly.Evaluate(x) }).To(Panic())
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when the point is in the field", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				poly := NewRandomPolynomial(field, randomDegree(prime))
				x := field.Random()

				coefficients := poly.Coefficients()

				// Manual evaluation
				accum := field.NewInField(big.NewInt(0))
				term := field.NewInField(big.NewInt(0))
				for i, c := range coefficients {
					term = x.Exp(field.NewInField(big.NewInt(int64(i))))
					term = term.Mul(c)
					accum = accum.Add(term)
				}

				Expect(poly.Evaluate(x).Eq(accum)).To(BeTrue())
			}
		},
			PrimeEntries...,
		)
	})
})
