package algebra

import (
	"fmt"
	"math/big"
)

// An FpElement represents an element of a finite field Fp defined by a prime p.
// Different FpElements, while the same type, can be in different fields as
// defined by their primes, and so care should be taken when using elements from
// more than one field.
type FpElement struct {
	prime, value *big.Int
}

func (lhs FpElement) String() string {
	return fmt.Sprintf("{p: %v, v: %v}", lhs.prime, lhs.value)
}

func (lhs FpElement) Field() Fp {
	return Fp{lhs.prime}
}

// NewFpElement creates a new field element directly from a value and a prime.
// If the value is not in the field defined by the prime, it will panic.
func NewFpElement(value, prime *big.Int) FpElement {
	if value.Sign() == -1 || value.Cmp(prime) != -1 {
		panic("cannot create a field element when the value is not in the field defined by the prime")
	}
	return FpElement{
		prime,
		value,
	}
}

// NewInSameField creates a new field element from a value in the same field as
// lhs. If the value is not in the same field as lhs, it will panic.
func (lhs FpElement) NewInSameField(value *big.Int) FpElement {
	if !lhs.FieldContains(value) {
		panic("cannot create field element from value outside of [0, p)")
	}
	return FpElement{
		lhs.prime,
		value,
	}
}

// Copy creates a copy of the field element lhs.
func (lhs FpElement) Copy() FpElement {
	return FpElement{
		lhs.prime,
		big.NewInt(0).Set(lhs.value),
	}
}

// FieldContains checks whether a value is the field of the field element lhs.
func (lhs FpElement) FieldContains(value *big.Int) bool {
	return value.Sign() != -1 && value.Cmp(lhs.prime) == -1
}

// InField checks whether a field element lhs is in the field f.
func (lhs FpElement) InField(f Fp) bool {
	return f.Contains(big.NewInt(0).Sub(lhs.prime, big.NewInt(1))) && !f.Contains(lhs.prime)
}

// FieldEq checks whether two field elements are in the same field.
func (lhs FpElement) FieldEq(rhs FpElement) bool {
	return lhs.prime.Cmp(rhs.prime) == 0
}

// Eq checks whether two field elements are both in the same field and have the
// same value.
func (lhs FpElement) Eq(rhs FpElement) bool {
	return lhs.prime.Cmp(rhs.prime) == 0 && lhs.value.Cmp(rhs.value) == 0
}

// SliceFieldEq checks whether all field elements in a given slice are in the
// same field. If the slice has length 0, it returns true.
func SliceFieldEq(s []FpElement) bool {
	if len(s) < 2 {
		return true
	}
	prime := s[0].prime

	for _, a := range s[1:] {
		if a.prime.Cmp(prime) != 0 {
			return false
		}
	}

	return true
}

// IsZero checks whether a field element is the zero element (additive
// identity) of the field.
func (lhs FpElement) IsZero() bool {
	return lhs.value.Sign() == 0
}

// IsOne checks whether a field element is the one element (multiplicative
// identity) of the field.
func (lhs FpElement) IsOne() bool {
	return lhs.value.Cmp(big.NewInt(1)) == 0
}

// Add returns the sum lhs + rhs. If lhs and rhs are not in the same field, it
// will panic.
func (lhs FpElement) Add(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot add two elements from different fields")
	}
	value := big.NewInt(0).Add(lhs.value, rhs.value)
	value = value.Mod(value, lhs.prime)
	return FpElement{
		lhs.prime,
		value,
	}
}

// Sub returns the subtraction lhs - rhs. If lhs and rhs are not in the same
// field, it will panic.
func (lhs FpElement) Sub(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot subtract two elements from different fields")
	}
	value := big.NewInt(0).Sub(lhs.value, rhs.value)
	if value.Sign() < 0 {
		value.Add(value, lhs.prime)
	}
	return FpElement{
		lhs.prime,
		value,
	}
}

// Mul returns the multiplication lhs * rhs. If lhs and rhs are not in the same
// field, it will panic.
func (lhs FpElement) Mul(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot multiply two elements from different fields")
	}
	value := big.NewInt(0).Mul(lhs.value, rhs.value)
	value = value.Mod(value, lhs.prime)
	return FpElement{
		lhs.prime,
		value,
	}
}

// Div returns the division lhs / rhs. If lhs and rhs are not in the same field,
// it will panic.
func (lhs FpElement) Div(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot divide two elements from different fields")
	}
	if rhs.IsZero() {
		panic("cannot divide by zero")
	}
	value := big.NewInt(0).ModInverse(rhs.value, lhs.prime)
	value = value.Mul(value, lhs.value)
	value = value.Mod(value, lhs.prime)
	return FpElement{
		lhs.prime,
		value,
	}
}

// Exp returns the exponentiation lhs ^ rhs. If lhs and rhs are not in the same
// field, it will panic.
func (lhs FpElement) Exp(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot exponentiate two elements from different fields")
	}
	value := big.NewInt(0).Exp(lhs.value, rhs.value, lhs.prime)
	return FpElement{
		lhs.prime,
		value,
	}
}

// Neg returns the negative (additive inverse) -lhs.
func (lhs FpElement) Neg() FpElement {
	value := big.NewInt(0).Sub(lhs.prime, lhs.value)
	return FpElement{
		lhs.prime,
		value,
	}
}

// Inv returns the multiplicative inverse lhs^{-1}.
func (lhs FpElement) Inv() FpElement {
	if lhs.IsZero() {
		panic("cannot find inverse of zero")
	}
	value := big.NewInt(0).ModInverse(lhs.value, lhs.prime)
	return FpElement{
		lhs.prime,
		value,
	}
}
