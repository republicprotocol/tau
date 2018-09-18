package process

import (
	"math/big"
	"unsafe"

	"github.com/republicprotocol/oro-go/core/vss/algebra"
)

func MacroBitwiseNot(dst, src Addr, field algebra.Fp) Inst {
	tmp1 := new(Value)
	code := Code{
		InstMove(tmp1, NewValuePublic(field.NewInField(big.NewInt(1)))),
		InstSub(dst, tmp1, src),
	}
	return InstMacro(code)
}

func MacroBitwiseOr(dst, lhs, rhs Addr) Inst {
	tmp1 := new(Value)
	tmp2 := new(Value)
	tmp3 := new(Value)
	code := Code{
		InstGenerateRnTuple(tmp1, tmp2),     // rand
		InstMul(tmp3, lhs, rhs, tmp1, tmp2), // ab
		InstSub(tmp3, rhs, tmp3),            // b - ab
		InstAdd(dst, lhs, tmp3),             // a + b - ab
	}
	return InstMacro(code)
}

func MacroBitwiseXor(dst, lhs, rhs Addr) Inst {
	tmp1 := new(Value)
	tmp2 := new(Value)
	code := Code{
		InstSub(dst, lhs, rhs),             // a - b
		InstGenerateRnTuple(tmp1, tmp2),    // rand
		InstMul(dst, dst, dst, tmp1, tmp2), // (a - b)^2
	}
	return InstMacro(code)
}

func MacroBitwiseAnd(dst, lhs, rhs Addr) Inst {
	tmp1 := new(Value)
	tmp2 := new(Value)
	code := Code{
		InstGenerateRnTuple(tmp1, tmp2),
		InstMul(dst, lhs, rhs, tmp1, tmp2),
	}
	return InstMacro(code)
}

func MacroBitwisePropGen(propDst, genDst, lhs, rhs Addr) Inst {
	tmp1 := new(Value)
	code := Code{
		MacroBitwiseXor(tmp1, lhs, rhs),
		MacroBitwiseAnd(genDst, lhs, rhs),
		InstCopy(propDst, tmp1),
	}
	return InstMacro(code)
}

func MacroBitwiseOpCLA(propDst, genDst, prop1, gen1, prop2, gen2 Addr) Inst {
	tmp1 := new(Value)
	tmp2 := new(Value)
	code := Code{
		MacroBitwiseAnd(tmp1, prop1, prop2),
		MacroBitwiseAnd(tmp2, gen1, prop2),
		MacroBitwiseOr(genDst, gen2, tmp2),
		InstCopy(propDst, tmp1),
	}
	return InstMacro(code)
}

func MacroBitwiseCOut(dst, lhs, rhs Addr, carry bool, field algebra.Fp, bits uint) Inst {

	size := unsafe.Sizeof(interface{}(nil))
	tmps := make([]Value, 2*bits)
	lhsPtr := unsafe.Pointer(lhs)
	rhsPtr := unsafe.Pointer(rhs)

	code := make(Code, 0)
	for i := uint(0); i < bits; i++ {
		c := Code{
			MacroBitwisePropGen(
				&tmps[2*i],
				&tmps[2*i+1],
				(*Value)(unsafe.Pointer(uintptr(lhsPtr)+size*uintptr(i))),
				(*Value)(unsafe.Pointer(uintptr(rhsPtr)+size*uintptr(i))),
			),
		}
		code = append(code, c...)
	}

	// If there is initial carry in, update the first generator
	if carry {
		code = append(code, MacroBitwiseOr(&tmps[1], &tmps[0], &tmps[1]))
	}

	for i := bits / 2; i > 0; i /= 2 {
		for j := uint(0); j < i; j++ {
			c := Code{
				MacroBitwiseOpCLA(
					&tmps[2*j],
					&tmps[2*j+1],
					&tmps[4*j],
					&tmps[4*j+1],
					&tmps[4*j+2],
					&tmps[4*j+3],
				),
			}
			code = append(code, c...)
		}
	}

	code = append(code, InstCopy(dst, &tmps[1]))

	return InstMacro(code)
}

func MacroBitwiseLT(dst, lhs, rhs Addr, field algebra.Fp, bits uint) Inst {

	size := unsafe.Sizeof(interface{}(nil))
	tmps := make([]Value, bits)
	rhsPtr := unsafe.Pointer(rhs)

	code := make(Code, 0)
	for i := uint(0); i < bits; i++ {
		code = append(code, MacroBitwiseNot(&tmps[i], (*Value)(unsafe.Pointer(uintptr(rhsPtr)+size*uintptr(i))), field))
	}

	code = append(code, MacroBitwiseCOut(dst, lhs, &tmps[0], true, field, bits))

	return InstMacro(code)
}
