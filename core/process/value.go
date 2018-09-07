package process

import (
	"github.com/republicprotocol/smpc-go/core/vss/algebra"

	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

// Value is the interface of any struct that can be pushed on to the stack.
type Value interface {
	IsValue()
}

// ValuePublic is a public constant, that can be pushed on to the stack.
type ValuePublic struct {
	Value algebra.FpElement
}

func (lhs ValuePublic) Add(rhs Value) (ret Value) {
	switch rhs := rhs.(type) {
	case ValuePublic:
		ret = ValuePublic{
			lhs.Value.Add(rhs.Value),
		}

	case ValuePrivate:
		ret = ValuePrivate{
			Share: shamir.Share{
				Index: rhs.Share.Index,
				Value: lhs.Value.Add(rhs.Share.Value),
			},
		}
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
		ret = ValuePrivate{
			Share: shamir.Share{
				Index: lhs.Share.Index,
				Value: lhs.Share.Value.Add(rhs.Value),
			},
		}

	case ValuePrivate:
		ret = ValuePrivate{
			Share: shamir.Share{
				Index: lhs.Share.Index,
				Value: lhs.Share.Value.Add(rhs.Share.Value),
			},
		}
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
