package algebra

import "math/big"

type FpElement struct {
	prime, value *big.Int
}

func NewFpElement(value, prime *big.Int) FpElement {
	if value.Sign() == -1 || value.Cmp(prime) != -1 {
		panic("cannot create a field element when the value is not in the field defined by the prime")
	}
	return FpElement{
		prime,
		value,
	}
}

func (a FpElement) NewInSameField(value *big.Int) FpElement {
	if !a.FieldContains(value) {
		panic("cannot create field element from value outside of [0, p)")
	}
	return FpElement{
		a.prime,
		value,
	}
}

func (a FpElement) Copy() FpElement {
	return FpElement{
		a.prime,
		big.NewInt(0).Set(a.value),
	}
}

func (a FpElement) FieldContains(value *big.Int) bool {
	return value.Sign() != -1 && value.Cmp(a.prime) == -1
}

func (a FpElement) InField(f Fp) bool {
	return f.Contains(big.NewInt(0).Sub(a.prime, big.NewInt(1))) && !f.Contains(a.prime)
}

func (a FpElement) FieldEq(b FpElement) bool {
	return a.prime.Cmp(b.prime) == 0
}

func (a FpElement) Eq(b FpElement) bool {
	return a.prime.Cmp(b.prime) == 0 && a.value.Cmp(b.value) == 0
}

func SliceFieldEq(s []FpElement) bool {
	if len(s) < 2 {
		return true
	}
	prime := s[0].prime

	for _, a := range s[1:] {
		if a.prime.Cmp(prime) != 0 {
			return false
		}
	}

	return true
}

func (a FpElement) IsZero() bool {
	return a.value.Sign() == 0
}

func (a FpElement) IsOne() bool {
	return a.value.Cmp(big.NewInt(1)) == 0
}

func (lhs FpElement) Add(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot add two elements from different fields")
	}
	value := big.NewInt(0).Add(lhs.value, rhs.value)
	value = value.Mod(value, lhs.prime)
	return FpElement{
		lhs.prime,
		value,
	}
}

func (lhs FpElement) Sub(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot subtract two elements from different fields")
	}
	value := big.NewInt(0).Sub(lhs.value, rhs.value)
	if value.Sign() < 0 {
		value.Add(value, lhs.prime)
	}
	return FpElement{
		lhs.prime,
		value,
	}
}

func (lhs FpElement) Mul(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot multiply two elements from different fields")
	}
	value := big.NewInt(0).Mul(lhs.value, rhs.value)
	value = value.Mod(value, lhs.prime)
	return FpElement{
		lhs.prime,
		value,
	}
}

func (lhs FpElement) Div(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot divide two elements from different fields")
	}
	if rhs.IsZero() {
		panic("cannot divide by zero")
	}
	value := big.NewInt(0).ModInverse(rhs.value, lhs.prime)
	value = value.Mul(value, lhs.value)
	value = value.Mod(value, lhs.prime)
	return FpElement{
		lhs.prime,
		value,
	}
}

func (lhs FpElement) Exp(rhs FpElement) FpElement {
	if !lhs.FieldEq(rhs) {
		panic("cannot exponentiate two elements from different fields")
	}
	value := big.NewInt(0).Exp(lhs.value, rhs.value, lhs.prime)
	return FpElement{
		lhs.prime,
		value,
	}
}

func (a FpElement) Neg() FpElement {
	value := big.NewInt(0).Sub(a.prime, a.value)
	return FpElement{
		a.prime,
		value,
	}
}

func (a FpElement) Inv() FpElement {
	if a.IsZero() {
		panic("cannot find inverse of zero")
	}
	value := big.NewInt(0).ModInverse(a.value, a.prime)
	return FpElement{
		a.prime,
		value,
	}
}
