package algebra_test

import (
	"crypto/rand"
	"math/big"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vss/algebra"
)

var _ = Describe("Finite field Elements", func() {
	const Trials = 100

	Context("when creating a new field element", func() {
		Context("from a value and a prime", func() {
			Context("when the value is not in the field", func() {
				DescribeTable("it should panic", func(prime *big.Int) {
					for i := 0; i < Trials; i++ {
						value := RandomNotInField(prime)
						Expect(func() { NewFpElement(value, prime) }).To(Panic())
					}
				},
					PrimeEntries...,
				)
			})

			Context("when the value is in the field", func() {
				DescribeTable("it should succeed", func(prime *big.Int) {
					for i := 0; i < Trials; i++ {
						value, _ := rand.Int(rand.Reader, prime)

						Expect(func() { NewFpElement(value, prime) }).ToNot(Panic())
					}
				},
					PrimeEntries...,
				)
			})
		})

		Context("from a value and an already existing field element", func() {
			Context("when the value is not in the field", func() {
				DescribeTable("it should panic", func(prime *big.Int) {
					field := NewField(prime)
					for i := 0; i < Trials; i++ {
						a := field.Random()
						value := RandomNotInField(prime)
						Expect(func() { a.NewInSameField(value) }).To(Panic())
					}
				},
					PrimeEntries...,
				)
			})

			Context("when the value is in the field", func() {
				DescribeTable("it should succeed", func(prime *big.Int) {
					field := NewField(prime)
					for i := 0; i < Trials; i++ {
						a := field.Random()
						value, _ := rand.Int(rand.Reader, prime)
						Expect(func() { a.NewInSameField(value) }).ToNot(Panic())
					}
				},
					PrimeEntries...,
				)
			})
		})
	})

	Context("when getting the underlying field", func() {
		DescribeTable("it should succeed", func(prime *big.Int) {
			field := NewField(prime)

			for i := 0; i < Trials; i++ {
				a := field.Random()
				Expect(a.Field().Eq(field)).To(BeTrue())
			}
		},
			PrimeEntries...,
		)
	})

	Context("when casting to a different field", func() {
		var prevField Fp
		firstEntry := true
		DescribeTable("it should succeed when the other field is at least as big, and fail when it is smaller", func(prime *big.Int) {
			field := NewField(prime)
			if firstEntry {
				prevField = field
				firstEntry = false
			} else {
				for i := 0; i < Trials; i++ {
					a := field.Random()
					Expect(a.Eq(a.AsField(field))).To(BeTrue())
					if field.LargerThan(prevField) {
						Expect(func() { a.AsField(prevField) }).To(Panic())
					} else {
						Expect(func() { a.AsField(prevField) }).ToNot(Panic())
					}
				}

				prevField = field
			}
		},
			PrimeEntries...,
		)
	})

	Context("when copying a field element", func() {
		DescribeTable("the new element should be equal to the oringal", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a := field.Random()
				b := a.Copy()
				Expect(a.Eq(b)).To(BeTrue())
			}
		},
			PrimeEntries...,
		)
	})

	Context("when checking if a value is in a field", func() {
		DescribeTable("it should succeed in determining containment", func(prime *big.Int) {
			field := NewField(prime)
			value := big.NewInt(0)
			for i := 0; i < Trials; i++ {
				a := field.Random()
				if RandomBool() {
					value = RandomNotInField(prime)
					Expect(a.FieldContains(value)).To(BeFalse())
				} else {
					value, _ = rand.Int(rand.Reader, prime)
					Expect(a.FieldContains(value)).To(BeTrue())
				}
			}
		},
			PrimeEntries...,
		)
	})

	Context("when checking if a field element is in a given field", func() {
		otherField := NewField(big.NewInt(7))
		DescribeTable("it should succeed in determining containment", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				if RandomBool() {
					a := field.Random()
					Expect(a.InField(field)).To(BeTrue())
					Expect(a.InField(otherField)).To(BeFalse())
				} else {
					a := otherField.Random()
					Expect(a.InField(field)).To(BeFalse())
					Expect(a.InField(otherField)).To(BeTrue())
				}
			}
		},
			PrimeEntries...,
		)
	})

	Context("when checking if two elements are in the same field", func() {
		otherField := NewField(big.NewInt(7))
		DescribeTable("it should succeed in the determination", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				a := field.Random()
				if RandomBool() {
					b := field.Random()
					Expect(a.FieldEq(b)).To(BeTrue())
				} else {
					b := otherField.Random()
					Expect(a.FieldEq(b)).To(BeFalse())
				}
			}
		},
			PrimeEntries...,
		)
	})

	Context("when checking if two field elements are equal", func() {
		DescribeTable("it should succeed in the determination", func(prime *big.Int) {
			for i := 0; i < Trials; i++ {
				a, _ := rand.Int(rand.Reader, prime)
				if RandomBool() {
					b := big.NewInt(0).Set(a)
					Expect(NewFpElement(a, prime).Eq(NewFpElement(b, prime))).To(BeTrue())
				} else {
					b, _ := rand.Int(rand.Reader, prime)
					if a.Cmp(b) == 0 {
						continue
					}
					Expect(NewFpElement(a, prime).Eq(NewFpElement(b, prime))).To(BeFalse())
				}
			}
		},
			PrimeEntries...,
		)
	})

	Context("when checking if a slice of field elements are all in the same field", func() {
		DescribeTable("it should succeed when they are in the same field", func(prime *big.Int) {
			field := NewField(prime)
			length, _ := rand.Int(rand.Reader, big.NewInt(10))
			elements := make([]FpElement, length.Uint64())

			for i := range elements {
				elements[i] = field.Random()
			}
			Expect(SliceFieldEq(elements)).To(BeTrue())
		},
			PrimeEntries...,
		)

		DescribeTable("it should fail when any element is in a different field", func(prime *big.Int) {
			field := NewField(prime)
			otherField := NewField(big.NewInt(7))
			length, _ := rand.Int(rand.Reader, big.NewInt(10))
			if length.Cmp(big.NewInt(2)) != -1 {
				diffIndex, _ := rand.Int(rand.Reader, length)
				elements := make([]FpElement, length.Uint64())

				for i := range elements {
					if uint64(i) == diffIndex.Uint64() {
						elements[i] = otherField.Random()
					} else {
						elements[i] = field.Random()
					}
				}

				Expect(SliceFieldEq(elements)).To(BeFalse())
			}
		},
			PrimeEntries...,
		)
	})

	Context("when checking if a field element is zero", func() {
		DescribeTable("it should succeed in the determination", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				if RandomBool() {
					a, _ := rand.Int(rand.Reader, prime)
					if a.Sign() == 0 {
						Expect(field.NewInField(a).IsZero()).To(BeTrue())
					} else {
						Expect(field.NewInField(a).IsZero()).To(BeFalse())
					}
				} else {
					a := big.NewInt(0)
					Expect(field.NewInField(a).IsZero()).To(BeTrue())
				}
			}
		},
			PrimeEntries...,
		)
	})

	Context("when checking if a field element is one", func() {
		DescribeTable("it should succeed in the determination", func(prime *big.Int) {
			field := NewField(prime)
			for i := 0; i < Trials; i++ {
				if RandomBool() {
					a, _ := rand.Int(rand.Reader, prime)
					if a.Cmp(big.NewInt(1)) == 0 {
						Expect(field.NewInField(a).IsOne()).To(BeTrue())
					} else {
						Expect(field.NewInField(a).IsOne()).To(BeFalse())
					}
				} else {
					a := big.NewInt(1)
					Expect(field.NewInField(a).IsOne()).To(BeTrue())
				}
			}
		},
			PrimeEntries...,
		)
	})

	Context("when doing arithmetic", func() {
		Context("when adding two field elements", func() {
			DescribeTable("it should panic when the lhs and rhs are not in the same field", func(prime *big.Int) {
				field := NewField(prime)
				otherField := NewField(big.NewInt(7))
				for i := 0; i < Trials; i++ {
					lhs := field.Random()
					rhs := otherField.Random()

					Expect(func() { lhs.Add(rhs) }).To(Panic())
				}
			},
				PrimeEntries...,
			)

			DescribeTable("it should succeed when the lhs and rhs are in the same field", func(prime *big.Int) {
				field := NewField(prime)
				for i := 0; i < Trials; i++ {
					lhs, _ := rand.Int(rand.Reader, prime)
					rhs, _ := rand.Int(rand.Reader, prime)
					expected := big.NewInt(0).Add(lhs, rhs)
					expected.Mod(expected, prime)

					Expect(field.NewInField(lhs).Add(field.NewInField(rhs)).Eq(field.NewInField(expected))).To(BeTrue())
				}
			},
				PrimeEntries...,
			)
		})

		Context("when subtracting two field elements", func() {
			DescribeTable("it should panic when the lhs and rhs are not in the same field", func(prime *big.Int) {
				field := NewField(prime)
				otherField := NewField(big.NewInt(7))
				for i := 0; i < Trials; i++ {
					lhs := field.Random()
					rhs := otherField.Random()

					Expect(func() { lhs.Sub(rhs) }).To(Panic())
				}
			},
				PrimeEntries...,
			)

			DescribeTable("it should succeed when the lhs and rhs are in the same field", func(prime *big.Int) {
				field := NewField(prime)
				for i := 0; i < Trials; i++ {
					lhs, _ := rand.Int(rand.Reader, prime)
					rhs, _ := rand.Int(rand.Reader, prime)
					expected := big.NewInt(0).Sub(lhs, rhs)
					if expected.Sign() == -1 {
						expected.Add(expected, prime)
					}

					Expect(field.NewInField(lhs).Sub(field.NewInField(rhs)).Eq(field.NewInField(expected))).To(BeTrue())
				}
			},
				PrimeEntries...,
			)
		})

		Context("when multiplying two field elements", func() {
			DescribeTable("it should panic when the lhs and rhs are not in the same field", func(prime *big.Int) {
				field := NewField(prime)
				otherField := NewField(big.NewInt(7))
				for i := 0; i < Trials; i++ {
					lhs := field.Random()
					rhs := otherField.Random()

					Expect(func() { lhs.Mul(rhs) }).To(Panic())
				}
			},
				PrimeEntries...,
			)

			DescribeTable("it should succeed when the lhs and rhs are in the same field", func(prime *big.Int) {
				field := NewField(prime)
				for i := 0; i < Trials; i++ {
					lhs, _ := rand.Int(rand.Reader, prime)
					rhs, _ := rand.Int(rand.Reader, prime)
					expected := big.NewInt(0).Mul(lhs, rhs)
					expected.Mod(expected, prime)

					Expect(field.NewInField(lhs).Mul(field.NewInField(rhs)).Eq(field.NewInField(expected))).To(BeTrue())
				}
			},
				PrimeEntries...,
			)
		})

		Context("when Dividing two field elements", func() {
			DescribeTable("it should panic when the lhs and rhs are not in the same field", func(prime *big.Int) {
				field := NewField(prime)
				otherField := NewField(big.NewInt(7))
				for i := 0; i < Trials; i++ {
					lhs := field.Random()
					rhs := otherField.Random()

					Expect(func() { lhs.Div(rhs) }).To(Panic())
				}
			},
				PrimeEntries...,
			)

			DescribeTable("it should panic when the rhs is zero", func(prime *big.Int) {
				field := NewField(prime)
				for i := 0; i < Trials; i++ {
					lhs := field.Random()
					rhs := field.NewInField(big.NewInt(0))

					Expect(func() { lhs.Div(rhs) }).To(Panic())
				}
			},
				PrimeEntries...,
			)

			DescribeTable("it should succeed when the lhs and rhs are in the same field", func(prime *big.Int) {
				field := NewField(prime)
				for i := 0; i < Trials; i++ {
					lhs, _ := rand.Int(rand.Reader, prime)
					rhs, _ := rand.Int(rand.Reader, prime)
					if rhs.Sign() == 0 {
						continue
					}
					expected := big.NewInt(0).ModInverse(rhs, prime)
					expected.Mul(lhs, expected)
					expected.Mod(expected, prime)

					Expect(field.NewInField(lhs).Div(field.NewInField(rhs)).Eq(field.NewInField(expected))).To(BeTrue())
				}
			},
				PrimeEntries...,
			)
		})

		Context("when exponentiating two field elements", func() {
			DescribeTable("it should panic when the lhs and rhs are not in the same field", func(prime *big.Int) {
				field := NewField(prime)
				otherField := NewField(big.NewInt(7))
				for i := 0; i < Trials; i++ {
					lhs := field.Random()
					rhs := otherField.Random()

					Expect(func() { lhs.Exp(rhs) }).To(Panic())
				}
			},
				PrimeEntries...,
			)

			DescribeTable("it should succeed when the lhs and rhs are in the same field", func(prime *big.Int) {
				field := NewField(prime)
				for i := 0; i < Trials; i++ {
					lhs, _ := rand.Int(rand.Reader, prime)
					rhs, _ := rand.Int(rand.Reader, prime)
					expected := big.NewInt(0).Exp(lhs, rhs, prime)

					Expect(field.NewInField(lhs).Exp(field.NewInField(rhs)).Eq(field.NewInField(expected))).To(BeTrue())
				}
			},
				PrimeEntries...,
			)
		})

		Context("when negating a field element", func() {
			DescribeTable("it should satisfy the additive inverse property", func(prime *big.Int) {
				field := NewField(prime)
				for i := 0; i < Trials; i++ {
					a := field.Random()

					Expect(a.Neg().Add(a).IsZero()).To(BeTrue())
				}
			},
				PrimeEntries...,
			)
		})

		Context("when inverting a field element", func() {
			DescribeTable("it should panic when the element is zero", func(prime *big.Int) {
				field := NewField(prime)
				for i := 0; i < Trials; i++ {
					a := field.NewInField(big.NewInt(0))

					Expect(func() { a.Inv() }).To(Panic())
				}
			},
				PrimeEntries...,
			)

			DescribeTable("it should satisfy the multiplicative inverse property", func(prime *big.Int) {
				field := NewField(prime)
				for i := 0; i < Trials; i++ {
					a := field.Random()
					if a.IsZero() {
						continue
					}

					Expect(a.Inv().Mul(a).IsOne()).To(BeTrue())
				}
			},
				PrimeEntries...,
			)
		})
	})
})
