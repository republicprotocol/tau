package vss

import (
	"math/big"

	"github.com/republicprotocol/smpc-go/core/vss/algebra"
	"github.com/republicprotocol/smpc-go/core/vss/pedersen"
	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

// A VShare is a Shamir share that can be verified to be correctly shared. The
// verification is based on Pedersen commitments.
type VShare struct {
	commitments []algebra.FpElement
	share, t    shamir.Share
}

// Share is a getter for the share field of a VShare struct.
func (vs *VShare) Share() shamir.Share {
	return vs.share
}

// SetShare is a setter for the share field of a VShare struct. The main purpose
// of this method is for testing malicious behaviour.
func (vs *VShare) SetShare(share shamir.Share) {
	vs.share = share
}

// SetCommitments is a setter for the commitment field of a VShare struct. The
// main purpose of this method is for testing malicious behaviour.
func (vs *VShare) SetCommitments(commitments []algebra.FpElement) {
	vs.commitments = commitments
}

// VShares is a list of VShare structs.
type VShares []VShare

// Share creates a list of verifiable shares given a Pedersen scheme, secret,
// and the n and k that define the threshold sharing scheme.
func Share(ped *pedersen.Pedersen, secret algebra.FpElement, n, k uint64) VShares {
	field := secret.Field()
	polyF := algebra.NewRandomPolynomial(field, uint(k-1), secret)
	polyFCoeffs := polyF.Coefficients()
	polyG := algebra.NewRandomPolynomial(field, uint(k-1))
	polyGCoeffs := polyG.Coefficients()

	commitments := make([]algebra.FpElement, k)
	for i := range commitments {
		commitments[i] = ped.Commit(polyFCoeffs[i], polyGCoeffs[i])
	}

	sShares := shamir.Split(polyF, n)
	tShares := shamir.Split(polyG, n)

	shares := make(VShares, n)
	for i := range shares {
		shares[i] = VShare{commitments, sShares[i], tShares[i]}
	}

	return shares
}

// Verify returns true if a verifiable share is correct in the Pedersen scheme,
// and false otherwise. If the list of commitments is empty (this cannot happen
// in the honest use case) then it will panic indirectly through evaluate().
func Verify(ped *pedersen.Pedersen, vshare VShare) bool {
	expected := ped.Commit(vshare.share.Value(), vshare.t.Value())
	actual := evaluate(ped, vshare.commitments, vshare.share)

	return expected.Eq(actual)
}

// The evaluate is a convenience function that computes the evaluation of the
// polynomial in the exponents of g and h from the Pedersen scheme.
func evaluate(ped *pedersen.Pedersen, commitments []algebra.FpElement, share shamir.Share) algebra.FpElement {
	if len(commitments) == 0 {
		panic("cannot verify against an empty list of commitments")
	}
	field := commitments[0].Field()
	subfield := share.Value().Field()

	base := subfield.NewInField(big.NewInt(0).SetUint64(share.Index()))
	power := subfield.NewInField(big.NewInt(1))
	ret := field.NewInField(big.NewInt(1))
	for j, ej := range commitments {
		if j != 0 {
			power = power.Mul(base)
		}

		ret = ret.Mul(ej.Exp(power.AsField(field)))
	}

	return ret
}
