package algebra

import (
	"crypto/rand"
	"math/big"
)

// A Polynomial struct represents a polynomial over a field Fp.
type Polynomial struct {
	field        *Fp
	coefficients []*big.Int
}

// NewPoly creates a new polynomial over the given field with the given
// coefficients. If any of the coefficients are not in the field, then the
// function will panic.
func NewPoly(field *Fp, coefficients []*big.Int) Polynomial {
	for _, c := range coefficients {
		if !field.InField(c) {
			panic("coefficient must be a field element")
		}
	}
	return Polynomial{field, coefficients}
}

// NewRandomPoly creates a new polynomial over the given field with random
// coefficients. The coefficient for the x^degree term is gauranteed to be
// non-zero, so that the polynomial is always of the given degree.
func NewRandomPoly(field *Fp, degree int) Polynomial {
	var err error
	coefficients := make([]*big.Int, degree+1)
	for i := 0; i <= degree; i++ {
		coefficients[i], err = rand.Int(rand.Reader, field.prime)
		if err != nil {
			// This should never occur because it is impossible to construct a
			// field with a prime that is not positive
			panic(err)
		}
	}

	// Make sure that the leading coefficient is not zero, otherwise the
	// polynomial would have a smaller degree
	for coefficients[degree].Sign() == 0 {
		coefficients[degree], err = rand.Int(rand.Reader, field.prime)
		if err != nil {
			// This should never occur because it is impossible to construct a
			// field with a prime that is not positive
			panic(err)
		}
	}

	return Polynomial{field, coefficients}
}

// NewRandomWithSecret creates a new polynomial over the given field where the
// constant term is the given secret. If the secret is not an element of the
// field, the function will panic. The remaining coefficients are random field
// elements. The polynomial is gauranteed to have the given degree by ensuring
// that the x^degree term is not zero.
func NewRandomWithSecret(field *Fp, degree int, secret *big.Int) Polynomial {
	if !field.InField(secret) {
		panic("secret must be a field element")
	}

	var err error
	coefficients := make([]*big.Int, degree+1)

	// The secret is the constant term
	coefficients[0] = secret

	// The rest of the coefficients are random
	for i := 1; i <= degree; i++ {
		coefficients[i], err = rand.Int(rand.Reader, field.prime)
		if err != nil {
			// This should never occur because it is impossible to construct a
			// field with a prime that is not positive
			panic(err)
		}
	}

	// Make sure that the leading coefficient is not zero, otherwise the
	// polynomial would have a smaller degree
	for coefficients[degree].Sign() == 0 {
		coefficients[degree], err = rand.Int(rand.Reader, field.prime)
		if err != nil {
			// This should never occur because it is impossible to construct a
			// field with a prime that is not positive
			panic(err)
		}
	}

	return Polynomial{field, coefficients}
}

// Degree returns the degree of the polynomial.
func (p *Polynomial) Degree() (degree int) {
	degree = len(p.coefficients)
	for p.coefficients[degree].Sign() == 0 {
		degree--
	}
	return
}

// Evaluate computes the value of the polynomial at the given point. If the
// given point is not in the field, the function will panic.
func (p *Polynomial) Evaluate(x *big.Int) *big.Int {
	if !p.field.InField(x) {
		panic("cannot evaluate polynomial at a point not in the field")
	}
	accum := big.NewInt(0).Set(p.coefficients[p.Degree()])

	for i := p.Degree() - 1; i >= 0; i-- {
		p.field.Mul(accum, x, accum)
		p.field.Add(accum, p.coefficients[i], accum)
	}

	return accum
}
