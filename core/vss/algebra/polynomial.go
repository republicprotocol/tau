package algebra

import (
	"math/big"
)

// A Polynomial struct represents a polynomial over a field Fp.
type Polynomial struct {
	coefficients []FpElement
}

// Coefficients returns the coefficients of the polynomial. This provides direct
// access to the struct field, and so modifications here will directly modify
// the polynomial object itself.
func (p Polynomial) Coefficients() []FpElement {
	return p.coefficients
}

// NewPolynomial creates a new polynomial over the given field with the given
// coefficients. If any of the coefficients are not in the field, then the
// function will panic.
func NewPolynomial(coefficients []FpElement) Polynomial {
	if len(coefficients) == 0 {
		panic("cannot construct a polynomial without coefficients")
	}
	if !coefficients[0].FieldContains(big.NewInt(int64(len(coefficients)) - 1)) {
		panic("polynomial cannot have degree greater than the order of the field")
	}
	if !SliceFieldEq(coefficients) {
		panic("coefficients must all be in the same field")
	}
	return Polynomial{coefficients}
}

// NewRandomPolynomial creates a new polynomial over the given field with random
// coefficients. A secret can be optionally provided, and will set the constant
// term to be that secret. If more than one argument for secret is given, the
// function will panic. The coefficient for the x^degree term is gauranteed to
// be non-zero, so that the polynomial is always of the given degree.
func NewRandomPolynomial(field Fp, degree uint, secret ...FpElement) Polynomial {
	if !field.Contains(big.NewInt(int64(degree))) {
		panic("polynomial cannot have degree greater than the order of the field")
	}
	if len(secret) > 1 {
		panic("maximum of one secret")
	}

	coefficients := make([]FpElement, degree+1)
	for i := 0; i <= int(degree); i++ {
		if i == 0 && len(secret) != 0 {
			coefficients[0] = secret[0]
			continue
		}
		coefficients[i] = field.Random()
	}

	// Make sure that the leading coefficient is not zero, otherwise the
	// polynomial would have a smaller degree
	for coefficients[degree].IsZero() {
		if degree == 0 {
			// Allow the zero polynomial when degree 0 is given
			break
		}
		coefficients[degree] = field.Random()
	}

	return NewPolynomial(coefficients)
}

// Degree returns the degree of the polynomial. The zero polynomial is
// considered to have degree 0.
func (p Polynomial) Degree() (degree uint) {
	degree = uint(len(p.coefficients)) - 1
	for p.coefficients[degree].IsZero() && degree != 0 {
		degree--
	}
	return degree
}

// Evaluate computes the value of the polynomial at the given point. If the
// given point is not in the field, the function will panic.
func (p Polynomial) Evaluate(x FpElement) FpElement {
	if !p.coefficients[0].FieldEq(x) {
		panic("cannot evaluate polynomial at a point not in the field")
	}
	accum := p.coefficients[p.Degree()]

	for i := int(p.Degree()) - 1; i >= 0; i-- {
		accum = accum.Mul(x)
		accum = accum.Add(p.coefficients[i])
	}

	return accum
}
