package algebra

import (
	"math/big"
)

// A Polynomial struct represents a polynomial over a field Fp.
type Polynomial struct {
	field        *Fp
	coefficients []*big.Int
}

// Coefficients returns the coefficients of the polynomial. This provides direct
// access to the struct field, and so modifications here will directly modify
// the polynomial object itself.
func (p *Polynomial) Coefficients() []*big.Int {
	return p.coefficients
}

// NewPolynomial creates a new polynomial over the given field with the given
// coefficients. If any of the coefficients are not in the field, then the
// function will panic.
func NewPolynomial(field *Fp, coefficients []*big.Int) Polynomial {
	for _, c := range coefficients {
		if !field.InField(c) {
			panic("coefficient must be a field element")
		}
	}
	return Polynomial{field, coefficients}
}

// NewRandomPolynomial creates a new polynomial over the given field with random
// coefficients. A secret can be optionally provided, and will set the constant
// term to be that secret. If more than one argument for secret is given, the
// function will panic. The coefficient for the x^degree term is gauranteed to
// be non-zero, so that the polynomial is always of the given degree.
func NewRandomPolynomial(field *Fp, degree uint, secret ...*big.Int) Polynomial {
	if len(secret) > 1 {
		panic("maximum of one secret")
	}

	coefficients := make([]*big.Int, degree+1)
	for i := 0; i <= int(degree); i++ {
		if i == 0 && len(secret) != 0 {
			coefficients[0] = secret[0]
			continue
		}
		coefficients[i] = field.Random()
	}

	// Make sure that the leading coefficient is not zero, otherwise the
	// polynomial would have a smaller degree
	for coefficients[degree].Sign() == 0 {
		if degree == 0 {
			// Allow the zero polynomial when degree 0 is given
			break
		}
		coefficients[degree] = field.Random()
	}

	return NewPolynomial(field, coefficients)
}

// Degree returns the degree of the polynomial. The zero polynomial is
// considered to have degree 0.
func (p *Polynomial) Degree() (degree uint) {
	degree = uint(len(p.coefficients)) - 1
	for p.coefficients[degree].Sign() == 0 && degree != 0 {
		degree--
	}
	return degree
}

// Evaluate computes the value of the polynomial at the given point. If the
// given point is not in the field, the function will panic.
func (p *Polynomial) Evaluate(x *big.Int) *big.Int {
	if !p.field.InField(x) {
		panic("cannot evaluate polynomial at a point not in the field")
	}
	accum := big.NewInt(0).Set(p.coefficients[p.Degree()])

	for i := int(p.Degree()) - 1; i >= 0; i-- {
		p.field.Mul(accum, x, accum)
		p.field.Add(accum, p.coefficients[i], accum)
	}

	return accum
}
