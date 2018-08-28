package vss

import (
	"crypto/rand"
	"math/big"
)

type Polynomial struct {
	coefficients []*big.Int
}

func New(coefficients []*big.Int) Polynomial {
	return Polynomial{coefficients}
}

func NewRandom(degree int, max *big.Int) Polynomial {
	var err error
	coefficients := make([]*big.Int, degree+1)
	for i := 0; i < degree; i++ {
		coefficients[i], err = rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
	}

	// Make sure that the leading coefficient is not zero, otherwise the
	// polynomial would have a smaller degree
	zero := big.NewInt(0)
	for coefficients[degree].Cmp(zero) == 0 {
		coefficients[degree], err = rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
	}

	return Polynomial{coefficients}
}

func NewRandomWithSecret(degree int, max, secret *big.Int) Polynomial {
	var err error
	coefficients := make([]*big.Int, degree+1)

	// The secret is the constant term
	coefficients[0] = secret

	// The rest of the coefficients are random
	for i := 1; i < degree; i++ {
		coefficients[i], err = rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
	}

	// Make sure that the leading coefficient is not zero, otherwise the
	// polynomial would have a smaller degree
	zero := big.NewInt(0)
	for coefficients[degree].Cmp(zero) == 0 {
		coefficients[degree], err = rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
	}

	return Polynomial{coefficients}
}

func (p *Polynomial) Degree() int {
	return len(p.coefficients)
}

func (p *Polynomial) Evaluate(x *big.Int) *big.Int {
	accum := big.NewInt(0).Set(p.coefficients[p.Degree()])

	for i := p.Degree() - 1; i >= 0; i-- {
		accum.Mul(accum, p.coefficients[i])
		accum.Mod()
	}
}
