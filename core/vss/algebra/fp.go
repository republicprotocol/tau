package algebra

import (
	"math/big"
)

// Fp represents the field of integers modulo p where p is a prime. This field
// object takes *big.Ints and performs the modulo arithmetic on them,
// abstracting the field away from the elements that it operates on.
type Fp struct {
	prime *big.Int
}

// NewField returns a new field object. The field will be the integers modulo
// the given prime. If the given prime is not a positive number, the function
// will panic.
func NewField(prime *big.Int) Fp {
	// The prime must be a positive number
	if prime.Sign() != 1 {
		panic("prime must be a positive integer")
	}
	return Fp{prime}
}

// InField checks whether a given integer is in the field. This will be the case
// when the integer is positive and less than the prime defining the field.
func (f *Fp) InField(x *big.Int) bool {
	if x.Cmp(f.prime) != -1 {
		return false
	}
	return true
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
	c.Neg(a)
	c.Add(c, f.prime)
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
	check := c.ModInverse(a, f.prime)
	if check == nil {
		// This will only occurr when f.prime is in fact not a prime
		panic("field element not relatively prime to field prime")
	}
}

// Sub sets c = a - b
func (f *Fp) Sub(a, b, c *big.Int) {
	if !f.InField(a) || !f.InField(b) {
		panic("cannot subtract elements that are not in the field")
	}
	c.Sub(a, b)
	c.Mod(c, f.prime)
}

// Div sets c = a/b = a*(b^-1)
func (f *Fp) Div(a, b, c *big.Int) {
	if !f.InField(a) || !f.InField(b) {
		panic("cannot subtract elements that are not in the field")
	}
	binv := big.NewInt(0)
	f.MulInv(b, binv)
	f.Mul(a, binv, c)
}
