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
	index uint64
	value algebra.FpElement
}

func New(index uint64, value algebra.FpElement) Share {
	return Share{index, value}
}

// Shares is a slice of Share structs.
type Shares []Share

func Split(poly algebra.Polynomial, n uint64) Shares {
	if uint(n) <= poly.Degree() {
		panic("n is not large enough to allow reconstruction")
	}
	shares := make(Shares, n)

	for i := range shares {
		index := uint64(i) + 1
		shares[i] = Share{index, poly.EvaluateUint64(index)}
	}

	return shares
}

func Join(shares Shares) (algebra.FpElement, error) {
	if len(shares) == 0 {
		panic("cannot join empty list of shares")
	}
	field := shares[0].value.Field()
	indices := make([]algebra.FpElement, len(shares))
	for i, share := range shares {
		if !share.value.InField(field) {
			return field.NewInField(big.NewInt(0)), ErrDifferentFields
		}
		indices[i] = field.NewInField(big.NewInt(0).SetUint64(share.index))
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

		secret = secret.Add(s.value.Mul(numerator.Div(denominator)))
	}

	return secret, nil
}
