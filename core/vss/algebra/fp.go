package algebra

import (
	"crypto/rand"
	"math/big"
)

// Fp represents the field of integers modulo p where p is a prime. This field
// object takes *big.Ints and performs the modulo arithmetic on them,
// abstracting the field away from the elements that it operates on.
type Fp struct {
	prime *big.Int
}

// NewField returns a new field object. The field will be the integers modulo
// the given prime. If the given prime is probably not a prime, as determined by
// big.ProbablyPrime, then the function panics. If the prime is in fact a prime,
// then big.ProbablyPrime will always return true, and so for correctly inputs
// the function will never panic.
func NewField(prime *big.Int) Fp {
	// The prime must be a positive number
	if !prime.ProbablyPrime(32) {
		panic("given prime is probably not prime")
	}
	return Fp{prime}
}

// InField checks whether a given integer is in the field. This will be the case
// when the integer is positive and less than the prime defining the field.
func (f *Fp) InField(x *big.Int) bool {
	if x.Cmp(f.prime) != -1 || x.Sign() == -1 {
		return false
	}
	return true
}

// Random returns a random element in the given field.
func (f *Fp) Random() *big.Int {
	// This should never return an error because it is impossible to construct a
	// field with a prime that is not positive
	r, _ := rand.Int(rand.Reader, f.prime)

	return r
}

// Add sets c = a + b
func (f *Fp) Add(a, b, c *big.Int) {
	if !f.InField(a) || !f.InField(b) {
		panic("cannot add elements that are not in the field")
	}
	c.Add(a, b)
	c.Mod(c, f.prime)
}

// Neg sets c = -a
func (f *Fp) Neg(a, c *big.Int) {
	if !f.InField(a) {
		panic("cannot negate an element that is not in the field")
	}
	if a.Sign() != 0 {
		c.Neg(a)
		c.Add(c, f.prime)
	} else {
		c.Set(a)
	}
}

// Mul sets c = a*b
func (f *Fp) Mul(a, b, c *big.Int) {
	if !f.InField(a) || !f.InField(b) {
		panic("cannot multiply elements that are not in the field")
	}
	c.Mul(a, b)
	c.Mod(c, f.prime)
}

// MulInv sets c = a^-1
func (f *Fp) MulInv(a, c *big.Int) {
	if !f.InField(a) {
		panic("cannot find the inverse of an element that is not in the field")
	}
	if a.Sign() == 0 {
		panic("zero has no multiplicative inverse")
	}

	// This should never fail because it is not possible to construct a field
	// with a non-prime (with high probability)
	c.ModInverse(a, f.prime)
}

// Sub sets c = a - b
func (f *Fp) Sub(a, b, c *big.Int) {
	if !f.InField(a) || !f.InField(b) {
		panic("cannot subtract elements that are not in the field")
	}
	neg := big.NewInt(0)
	f.Neg(b, neg)
	f.Add(a, neg, c)
}

// Div sets c = a/b = a*(b^-1)
func (f *Fp) Div(a, b, c *big.Int) {
	if !f.InField(a) || !f.InField(b) {
		panic("cannot subtract elements that are not in the field")
	}
	inv := big.NewInt(0)
	f.MulInv(b, inv)
	f.Mul(a, inv, c)
}
