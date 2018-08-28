package algebra

import (
	"errors"
	"math/big"
)

var (
	// ErrInvalidPrime signifies that a given prime was not positive.
	ErrInvalidPrime = errors.New("prime must be positive")

	// ErrLargeInteger signifies that an element was given that is larger than
	// the prime modulus for the field.
	ErrLargeInteger = errors.New("integer is larger than prime modulus")
)

// Fp represents a field element in the field of integers modulo p where p is a
// prime.
type Fp struct {
	element *big.Int
	prime   *big.Int
}

// New creates a new Fp from an integer and a prime, where the element is in the
// field of integers modulo that prime. When creating an Fp that is in the same
// field as a variable that is already accessible, it is recommended to
// construct the new Fp using NewInSameField. If the prime is not positive then
// and error will be returned. If the given integer is bigger than the prime,
// the integer will be appropraitely reduced modulo the prime, and an error will
// be returned.
func New(x *big.Int, prime big.Int) (ret Fp, err error) {
	// Check if the prime is valid
	if prime.Sign() != 1 {
		err = ErrInvalidPrime
		return
	}

	ret = Fp{x, &prime}
	if x.Cmp(&prime) == 1 {
		x.Mod(x, &prime)
		err = ErrLargeInteger
	}
	return
}

// NewInSameField creates a new Fp from an integer and a pointer to a field
// element that is in the target field. If the given integer is not in the
// field, the integer will first be reduced modulo p, where p is the prime
// defining the field, and an error will also be returned.
func NewInSameField(f Fp, x *big.Int) (ret Fp, err error) {
	ret = Fp{x, f.prime}
	if x.Cmp(f.prime) == 1 {
		x.Mod(x, f.prime)
		err = ErrLargeInteger
	}
	return
}

// Add implements the FieldElement interface
func (x *Fp) Add(a, b Fp) {
	x.element.Add(a.element, b.element)
	x.element.Mod(x.element, x.prime)
}

// Neg implements the FieldElement interface
func (x *Fp) Neg(a Fp) {
	x.element.Neg(x.element)
	x.element.Add(x.element, x.prime)
}

// Mul implements the FieldElement interface
func (x *Fp) Mul(a, b Fp) {
	x.element.Mul(a.element, b.element)
	x.element.Mod(x.element, x.prime)
}

// MulInv implements the FieldElement interface
func (x *Fp) MulInv(a Fp) {
	check := x.element.ModInverse(x.element, x.prime)
	if check == nil {
		panic("field element not relatively prime to field prime")
	}
}

// Sub sets x = a - b
func (x *Fp) Sub(a, b Fp) {
	x.element.Sub(a.element, b.element)
	x.element.Mod(x.element, x.prime)
}

// Div sets x = a*b = a*(b^-1)
func (x *Fp) Div(a, b Fp) {
	binv, _ := NewInSameField(b, big.NewInt(0))
	binv.MulInv(binv)
	x.Mul(a, binv)
}
