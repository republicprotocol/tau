package process

import (
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"
)

// Value is the interface of any struct that can be pushed on to the stack.
type Value interface {
	IsValue()
}

// ValuePublic is a public constant, that can be pushed on to the stack.
type ValuePublic struct {
	Value algebra.FpElement
}

func NewValuePublic(n algebra.FpElement) ValuePublic {
	return ValuePublic{n}
}

func (lhs ValuePublic) Add(rhs Value) (ret Value) {
	switch rhs := rhs.(type) {
	case ValuePublic:
		ret = ValuePublic{
			lhs.Value.Add(rhs.Value),
		}

	case ValuePrivate:
		ret = ValuePrivate{
			Share: shamir.New(rhs.Share.Index(), lhs.Value.Add(rhs.Share.Value())),
		}
	default:
		panic("unimplemented")
	}
	return
}

func (lhs ValuePublic) Neg() Value {
	return ValuePublic{lhs.Value.Neg()}

}

func (lhs ValuePublic) Sub(rhs Value) (ret Value) {
	switch rhs := rhs.(type) {
	case ValuePublic:
		ret = ValuePublic{
			lhs.Value.Sub(rhs.Value),
		}

	case ValuePrivate:
		ret = ValuePrivate{
			Share: shamir.New(rhs.Share.Index(), lhs.Value.Sub(rhs.Share.Value())),
		}
	default:
		panic("unimplemented")
	}
	return
}

func (lhs ValuePublic) Exp(rhs ValuePublic) (ret ValuePublic) {
	return ValuePublic{lhs.Value.Exp(rhs.Value)}
}

func (lhs ValuePublic) IsValue() {
}

type ValuePrivate struct {
	Share shamir.Share
}

func NewValuePrivate(share shamir.Share) ValuePrivate {
	return ValuePrivate{share}
}

func (lhs ValuePrivate) Add(rhs Value) (ret Value) {
	switch rhs := rhs.(type) {

	case ValuePublic:
		ret = ValuePrivate{
			Share: shamir.New(lhs.Share.Index(), lhs.Share.Value().Add(rhs.Value)),
		}

	case ValuePrivate:
		if lhs.Share.Index() != rhs.Share.Index() {
			panic("private addition: index mismatch")
		}
		ret = ValuePrivate{
			Share: shamir.New(lhs.Share.Index(), lhs.Share.Value().Add(rhs.Share.Value())),
		}
	default:
		panic("unimplemented")
	}
	return
}

func (lhs ValuePrivate) Neg() Value {
	return ValuePrivate{shamir.New(lhs.Share.Index(), lhs.Share.Value().Neg())}

}

func (lhs ValuePrivate) Sub(rhs Value) (ret Value) {
	switch rhs := rhs.(type) {

	case ValuePublic:
		ret = ValuePrivate{
			Share: shamir.New(lhs.Share.Index(), lhs.Share.Value().Sub(rhs.Value)),
		}

	case ValuePrivate:
		if lhs.Share.Index() != rhs.Share.Index() {
			panic("private addition: index mismatch")
		}
		ret = ValuePrivate{
			Share: shamir.New(lhs.Share.Index(), lhs.Share.Value().Sub(rhs.Share.Value())),
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
