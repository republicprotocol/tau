package shamir

import (
	"errors"
	"math/big"

	"github.com/republicprotocol/smpc-go/core/vss/algebra"
)

var ErrDifferentFields = errors.New("expected shares to all be in the same field")

// Share represents a share of a secret that has been shared using Shamir secret
// sharing.
type Share struct {
	Index uint64
	Value algebra.FpElement
}

// Shares is a slice of Share structs.
type Shares []Share

// Split takes a polynomial over a field and splits it into shares. The secret
// that is being split is the constant term of the polynomial. The zero index
// corresponds to the secret itself, and so if this is given in the list of
// indices the function will panic.
func Split(poly algebra.Polynomial, indices []uint64) Shares {
	// TODO: what if there are duplicate indices?
	shares := make(Shares, len(indices))

	for i := range shares {
		index := indices[i]
		if index == 0 {
			panic("a share cannot be the secret itself")
		}

		shares[i] = Share{index, poly.EvaluateUint64(index)}
	}

	return shares
}

func Join(shares Shares) (algebra.FpElement, error) {
	if len(shares) == 0 {
		panic("cannot join empty list of shares")
	}
	field := shares[0].Value.Field()
	for _, share := range shares {
		if !share.Value.InField(field) {
			return field.NewInField(big.NewInt(0)), ErrDifferentFields
		}
	}
	indices := make([]algebra.FpElement, len(shares))
	for i, s := range shares {
		indices[i] = field.NewInField(big.NewInt(0).SetUint64(s.Index))
	}

	secret := field.NewInField(big.NewInt(0))
	for i, s := range shares {
		numerator := field.NewInField(big.NewInt(1))
		denominator := field.NewInField(big.NewInt(1))
		for j, index := range indices {
			if j == i {
				continue
			}

			numerator = numerator.Mul(index)
			denominator = denominator.Mul(index.Sub(indices[i]))
		}

		secret = secret.Add(s.Value.Mul(numerator.Div(denominator)))
	}

	return secret, nil
}
