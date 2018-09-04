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

func (f Fp) Eq(g Fp) bool {
	return f.prime.Cmp(g.prime) == 0
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

func (f Fp) NewInField(value *big.Int) FpElement {
	if value.Sign() == -1 || value.Cmp(f.prime) != -1 {
		panic("cannot create field element from value outside of [0, p)")
	}
	return FpElement{
		f.prime,
		value,
	}
}

// InField checks whether a given integer is in the field. This will be the case
// when the integer is positive and less than the prime defining the field.
func (f Fp) InField(x *big.Int) bool {
	return x.Cmp(f.prime) == -1 && x.Sign() != -1
}

// Random returns a random element in the given field.
func (f Fp) Random() FpElement {
	// This should never return an error because it is impossible to construct a
	// field with a prime that is not positive
	r, _ := rand.Int(rand.Reader, f.prime)

	return f.NewInField(r)
}
