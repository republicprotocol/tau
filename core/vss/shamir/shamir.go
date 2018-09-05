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

// New returns a new share constructed from the given index and field element.
func New(index uint64, value algebra.FpElement) Share {
	return Share{index, value}
}

// Shares is a slice of Share structs.
type Shares []Share

// Split returns a list of n shares determined by the polynomial poly. The
// secret for this sharing will be the constant term in the polynomial. The
// indices that the polynomial will be evaluated at to create the shares will be
// sequence 1, 2, ..., n.
func Split(poly algebra.Polynomial, n uint64) Shares {
	if uint(n) <= poly.Degree() {
		panic("n is not large enough to allow reconstruction")
	}
	field := poly.Coefficients()[0].Field()
	shares := make(Shares, n)

	for i := range shares {
		index := uint64(i) + 1
		shares[i] = Share{index, poly.Evaluate(field.NewInField(big.NewInt(0).SetUint64(index)))}
	}

	return shares
}

// Join returns the secret defined by the given shares. If given an empty list
// of shares, it will panic. The reconstruction algorithm is agnostic to whether
// or not a set of shares are consistent and so if they are inconsistent then
// Join will produce different values for different subsets of shares.
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

func (share *Share) indexEq(other Share) bool {
	return share.index == other.index
}

// Add computes the share corresponding to the addition of the two secrets. If
// the indices of the two shares are different, it panics.
func (share *Share) Add(other Share) Share {
	if !share.indexEq(other) {
		panic("cannot add shares with different indices")
	}
	return New(share.index, share.value.Add(other.value))
}

// Sub computes the share corresponding to the addition of the two secrets. If
// the indices of the two shares are different, it panics.
func (share *Share) Sub(other Share) Share {
	if !share.indexEq(other) {
		panic("cannot subtract shares with different indices")
	}
	return New(share.index, share.value.Sub(other.value))
}

// Mul computes the share corresponding to the addition of the two secrets. If
// the indices of the two shares are different, it panics.
func (share *Share) Mul(other Share) Share {
	if !share.indexEq(other) {
		panic("cannot multiply shares with different indices")
	}
	return New(share.index, share.value.Mul(other.value))
}
