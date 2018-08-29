package algebra_test

import (
	"crypto/rand"
	"math/big"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vss/algebra"
)

var _ = Describe("Finite field Fp", func() {
	const Trials = 100

	Context("when constructing a field with a prime number", func() {
		DescribeTable("no panic is expected", func(prime *big.Int) {
			Expect(func() { NewField(prime) }).ToNot(Panic())
		},
			PrimeEntries...,
		)
	})

	Context("when constructing a field with a composite number", func() {
		DescribeTable("a panic is expected", func(composite *big.Int) {
			Expect(func() { NewField(composite) }).To(Panic())
		},
			CompositeEntries...,
		)
	})

	Context("when constructing a field with a negative number", func() {
		It("should panic", func(doneT Done) {
			defer close(doneT)

			for i := 0; i < Trials; i++ {
				negative, err := rand.Int(rand.Reader, big.NewInt(0).SetUint64(^uint64(0)))
				Expect(err).To(BeNil())

				negative.Neg(negative)
				Expect(func() { NewField(negative) }).To(Panic())
			}
		})
	})

	Context("when checking if an integer is an element of the field", func() {
		prime, _ := big.NewInt(0).SetString("11415648579556416673", 10)
		field := NewField(prime)

		Context("when the integer is too big", func() {
			It("should return false", func(doneT Done) {
				defer close(doneT)

				for i := 0; i < Trials; i++ {
					toobig, err := rand.Int(rand.Reader, big.NewInt(0).SetUint64(^uint64(0)))
					Expect(err).To(BeNil())

					toobig.Add(toobig, prime)
					Expect(field.InField(toobig)).To(BeFalse())
				}
			})
		})

		Context("when the integer is negative", func() {
			It("should return false", func(doneT Done) {
				defer close(doneT)

				for i := 0; i < Trials; i++ {
					negative, err := rand.Int(rand.Reader, big.NewInt(0).SetUint64(^uint64(0)))
					Expect(err).To(BeNil())

					negative.Neg(negative)
					Expect(field.InField(negative)).To(BeFalse())
				}
			})
		})

		Context("when the integer is in the field", func() {
			It("should return false", func(doneT Done) {
				defer close(doneT)

				for i := 0; i < Trials; i++ {
					correct, err := rand.Int(rand.Reader, prime)
					Expect(err).To(BeNil())

					Expect(field.InField(correct)).To(BeTrue())
				}
			})
		})
	})

	Context("when creating a random field element", func() {
		DescribeTable("no panic is expected", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				Expect(func() { field.Random() }).ToNot(Panic())
			}
		},
			PrimeEntries...,
		)
	})

	Context("when adding two numbers", func() {
		DescribeTable("it should panic when the first argument is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := RandomNotInField(&field), field.Random(), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Add(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should panic when the second argument is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := field.Random(), RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Add(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should panic when both arguments are not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := RandomNotInField(&field), RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Add(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when both arguments are in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b := field.Random(), field.Random()
				expected, actual := big.NewInt(0), big.NewInt(0)

				expected.Add(a, b)
				expected.Mod(expected, prime)
				field.Add(a, b, actual)

				Expect(actual.Cmp(expected)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

	})

	Context("when negating an element", func() {
		DescribeTable("it should panic when the element is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, oldc := RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Neg(a, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when the element is in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a := field.Random()
				aneg := big.NewInt(0).Set(a)
				field.Neg(a, aneg)
				res := big.NewInt(0)

				field.Add(a, aneg, res)
				Expect(res.Sign()).To(Equal(0))
			}
		},
			PrimeEntries...,
		)
	})

	Context("when multiplying two elements", func() {
		DescribeTable("it should panic when the first argument is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := RandomNotInField(&field), field.Random(), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Mul(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should panic when the second argument is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := field.Random(), RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Mul(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should panic when both arguments are not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := RandomNotInField(&field), RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Mul(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when both arguments are in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b := field.Random(), field.Random()
				expected, actual := big.NewInt(0), big.NewInt(0)

				expected.Mul(a, b)
				expected.Mod(expected, prime)
				field.Mul(a, b, actual)

				Expect(actual.Cmp(expected)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)
	})

	Context("when computing the multiplicative inverse of an element", func() {
		DescribeTable("i should panic when the element is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, oldc := RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.MulInv(a, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when the element is in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a := field.Random()
				if a.Sign() == 0 {
					// 0 has no multiplicative inverse
					continue
				}
				ainv := big.NewInt(0).Set(a)
				field.MulInv(a, ainv)
				res := big.NewInt(0)

				field.Mul(a, ainv, res)
				Expect(res.Cmp(big.NewInt(1))).To(Equal(0))
			}
		},
			PrimeEntries...,
		)
	})

	Context("when subtracting two elements", func() {
		DescribeTable("it should panic when the first argument is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := RandomNotInField(&field), field.Random(), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Sub(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should panic when the second argument is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := field.Random(), RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Sub(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should panic when both arguments are not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := RandomNotInField(&field), RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Sub(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when both arguments are in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b := field.Random(), field.Random()
				expected, actual := big.NewInt(0), big.NewInt(0)

				expected.Sub(a, b)
				if expected.Sign() == -1 {
					// Make sure that the expected value is positive
					expected.Add(expected, prime)
				}
				expected.Mod(expected, prime)
				field.Sub(a, b, actual)

				Expect(actual.Cmp(expected)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)
	})

	Context("when dividing two elements", func() {
		DescribeTable("it should panic when the first argument is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := RandomNotInField(&field), field.Random(), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Div(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should panic when the second argument is not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := field.Random(), RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Div(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should panic when both arguments are not in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b, oldc := RandomNotInField(&field), RandomNotInField(&field), field.Random()
				newc := big.NewInt(0).Set(oldc)

				Expect(func() { field.Div(a, b, newc) }).To(Panic())
				Expect(newc.Cmp(oldc)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)

		DescribeTable("it should succeed when both arguments are in the field", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a, b := field.Random(), field.Random()
				binv := big.NewInt(0).ModInverse(b, prime)
				expected, actual := big.NewInt(0), big.NewInt(0)

				expected.Mul(a, binv)
				expected.Mod(expected, prime)
				field.Div(a, b, actual)

				Expect(actual.Cmp(expected)).To(Equal(0))
			}
		},
			PrimeEntries...,
		)
	})
})
