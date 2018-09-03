package vss

import (
	"math/big"

	"github.com/republicprotocol/smpc-go/core/vss/algebra"
	"github.com/republicprotocol/smpc-go/core/vss/pedersen"
	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

// A VerifiableShare struct contains the shares and broadcast message required to
// verify a Shamir sharing. sShare is the actual share that is to be used, and
// tShare is the obscuring share used for the Pedersen commitment.
type VerifiableShare struct {
	broadcast []*big.Int
	sShare    shamir.Share
	tShare    shamir.Share
}

// VerifiableShares is a list of VerifiableShare structs.
type VerifiableShares []VerifiableShare

// Share creates verifiable shares for a given secret. The Pedersen commitment
// scheme that will be used is given by ped, and the threshold for the secret
// sharing is given by k. The indicies for the secret sharing are given by
// indices. If the inputs are malformed (e.g. the secret is not in the field,
// the pedersen scheme is not correct) then the sharing will not work.
func Share(ped *pedersen.Pedersen, secret *big.Int, k uint, indices []uint64) VerifiableShares {
	field := algebra.NewField(ped.SubgroupOrder())
	polyF := algebra.NewRandomPolynomial(&field, k-1, secret)
	polyFCoeffs := polyF.Coefficients()
	polyG := algebra.NewRandomPolynomial(&field, k-1)
	polyGCoeffs := polyG.Coefficients()

	commitments := make([]*big.Int, k)
	for i := range commitments {
		commitments[i] = ped.Commit(polyFCoeffs[i], polyGCoeffs[i])
	}

	sShares := shamir.Split(&polyF, indices)
	tShares := shamir.Split(&polyG, indices)

	shares := make(VerifiableShares, len(indices))
	for i := range indices {
		shares[i] = VerifiableShare{commitments, sShares[i], tShares[i]}
	}

	return shares
}

// Verify takes a verifiable share and confirms or denies whether it is correct.
func Verify(ped *pedersen.Pedersen, share VerifiableShare) bool {
	expected := ped.Commit(share.sShare.Value, share.tShare.Value)
	actual := evaluate(ped, share.broadcast, share.sShare.Index)

	return expected.Cmp(actual) == 0
}

// evaluate performs the polynomial evaluation in the exponents of the
// commitments in the broadcast field of a verifiable share.
func evaluate(ped *pedersen.Pedersen, broadcast []*big.Int, index uint64) *big.Int {
	field := algebra.NewField(ped.GroupOrder())
	subfield := algebra.NewField(ped.SubgroupOrder())

	value := big.NewInt(0)
	base := big.NewInt(0).SetUint64(index)
	power := big.NewInt(1)
	ret := big.NewInt(1)
	for j, ej := range broadcast {
		if j != 0 {
			subfield.Mul(power, base, power)
		}

		value.Exp(ej, power, ped.GroupOrder())
		field.Mul(ret, value, ret)
	}

	return ret
}
