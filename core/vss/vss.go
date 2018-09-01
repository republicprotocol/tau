package vss

import (
	"math/big"

	"github.com/republicprotocol/smpc-go/core/vss/algebra"
	"github.com/republicprotocol/smpc-go/core/vss/pedersen"
	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

type VerifiableShare struct {
	broadcast []*big.Int
	sShare    shamir.Share
	tShare    shamir.Share
}

type VerifiableShares []VerifiableShare

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

func Verify(ped *pedersen.Pedersen, share VerifiableShare) bool {
	expected := ped.Commit(share.sShare.Value, share.tShare.Value)
	actual := evaluate(ped, share.broadcast, share.sShare.Index)

	// log.Printf("[debug] expected: %v", expected)
	// log.Printf("[debug] actual: %v", actual)

	return expected.Cmp(actual) == 0
}

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
