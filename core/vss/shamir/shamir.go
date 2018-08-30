package shamir

import (
	"math/big"

	"github.com/republicprotocol/smpc-go/core/vss/algebra"
)

// Share represents a share of a secret that has been shared using Shamir secret
// sharing.
type Share struct {
	Index uint64
	Value *big.Int
}

// Shares is a slice of Share structs.
type Shares []Share

// Split takes a polynomial over a field and splits it into shares. The secret
// that is being split is the constant term of the polynomial. The zero index
// corresponds to the secret itself, and so if this is given in the list of
// indices the function will panic.
func Split(poly *algebra.Polynomial, indices []uint64) Shares {
	shares := make(Shares, len(indices))
	x := big.NewInt(0)

	for i := range shares {
		index := indices[i]
		if index == 0 {
			panic("a share cannot be the secret itself")
		}

		x.SetUint64(index)
		shares[i] = Share{index, poly.Evaluate(x)}
	}

	return shares
}

// Join reconstructs a secret from a set of shares. it is assumed that the given
// field is the same as the one that was used when constructing the shares. If
// not then the result, if successfully computed, will be undefined.
func Join(field *algebra.Fp, shares Shares) *big.Int {
	indices := make([]*big.Int, len(shares))
	for i, s := range shares {
		indices[i] = big.NewInt(0).SetUint64(s.Index)
	}

	accum := big.NewInt(0)
	lagrangeEval := big.NewInt(0)
	numerator := big.NewInt(0)
	denominator := big.NewInt(0)
	for _, s := range shares {
		lagrangeEval.SetUint64(1)
		for _, index := range indices {
			if s.Index == index.Uint64() {
				continue
			}

			field.Neg(index, numerator)
			field.Sub(s.Value, index, denominator)
			field.Div(numerator, denominator, numerator)

			field.Mul(lagrangeEval, numerator, lagrangeEval)
		}

		field.Add(accum, lagrangeEval, accum)
	}

	return accum
}
