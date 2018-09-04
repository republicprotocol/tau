package program

import (
	"math/big"

	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

type Value interface {
	IsValue()
}

type ValuePublic struct {
	Int *big.Int
}

func (lhs ValuePublic) Add(rhs Value) (ret Value) {
	switch rhs := rhs.(type) {

	case ValuePublic:
		value := ValuePublic{
			Int: big.NewInt(0),
		}
		value.Int.Add(lhs.Int, rhs.Int)
		ret = value

	case ValuePrivate:
		value := ValuePrivate{
			Share: shamir.Share{
				Index: rhs.Share.Index,
				Value: big.NewInt(0),
			},
		}
		value.Share.Value.Add(lhs.Int, rhs.Share.Value)
		ret = value

	default:
		panic("unimplemented")
	}
	return
}

func (lhs ValuePublic) IsValue() {
}

type ValuePrivate struct {
	Share shamir.Share
}

func (lhs ValuePrivate) Add(rhs Value) (ret Value) {
	switch rhs := rhs.(type) {

	case ValuePublic:
		value := ValuePrivate{
			Share: shamir.Share{
				Index: lhs.Share.Index,
				Value: big.NewInt(0),
			},
		}
		value.Share.Value.Add(lhs.Share.Value, rhs.Int)
		ret = value

	case ValuePrivate:
		value := ValuePrivate{
			Share: shamir.Share{
				Index: lhs.Share.Index,
				Value: big.NewInt(0),
			},
		}
		value.Share.Value.Add(lhs.Share.Value, rhs.Share.Value)
		ret = value

	default:
		panic("unimplemented")
	}
	return
}

func (lhs ValuePrivate) IsValue() {
}

type ValuePrivateRn struct {
	Rho   shamir.Share
	Sigma shamir.Share
}

func (lhs ValuePrivateRn) IsValue() {
}
