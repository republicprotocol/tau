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

func (lhs ValuePublic) Mul(rhs ValuePublic) Value {
	return ValuePublic{lhs.Value.Mul(rhs.Value)}
}

func (lhs ValuePublic) Exp(rhs ValuePublic) (ret ValuePublic) {
	return ValuePublic{lhs.Value.Exp(rhs.Value)}
}

func (lhs ValuePublic) Inv() Value {
	return ValuePublic{lhs.Value.Inv()}

}

func (lhs ValuePublic) Mod(rhs ValuePublic) Value {
	lhsVal := lhs.Value.Value()
	rhsVal := rhs.Value.Value()
	lhsVal.Mod(lhsVal, rhsVal)

	return ValuePublic{lhs.Value.NewInSameField(lhsVal)}

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

func (lhs ValuePrivate) Mul(rhs ValuePublic) Value {
	return ValuePrivate{
		Share: shamir.New(lhs.Share.Index(), lhs.Share.Value().Mul(rhs.Value)),
	}
}

func (lhs ValuePrivate) IsValue() {
}
