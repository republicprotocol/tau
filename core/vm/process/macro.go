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
	code := Code{
		InstGenerateRnTuple(tmp1, tmp2),    // rand
		InstMul(dst, lhs, rhs, tmp1, tmp2), // ab
		InstSub(dst, lhs, dst),             // b - ab
		InstAdd(dst, rhs, dst),             // a + b - ab
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
	code := Code{
		MacroBitwiseXor(propDst, lhs, rhs),
		MacroBitwiseAnd(genDst, lhs, rhs),
	}
	return InstMacro(code)
}

func MacroBitwiseOpCLA(propDst, genDst, prop1, gen1, prop2, gen2 Addr) Inst {
	tmp1 := new(Value)
	code := Code{
		MacroBitwiseAnd(propDst, prop1, prop2),
		MacroBitwiseAnd(tmp1, gen1, prop2),
		MacroBitwiseOr(genDst, gen2, tmp1),
	}
	return InstMacro(code)
}

func MacroBitwiseCOut(dst, src Addr, field algebra.Fp, bits uint) Inst {

	size := unsafe.Sizeof(interface{}(nil))
	dstPtr := unsafe.Pointer(dst)
	srcPtr := unsafe.Pointer(src)

	code := make(Code, 0)
	for i := uint(0); i < bits; i++ {
		c := Code{
			MacroBitwisePropGen(
				(*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(2*i))),
				(*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(2*i+1))),
				(*Value)(unsafe.Pointer(uintptr(srcPtr)+size*uintptr(2*i))),
				(*Value)(unsafe.Pointer(uintptr(srcPtr)+size*uintptr(2*i+1))),
			),
		}
		code = append(code, c...)
	}

	for i := bits / 2; i > 0; i /= 2 {
		for j := uint(0); j < i; j++ {
			c := Code{
				MacroBitwiseOpCLA(
					(*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(2*j))),
					(*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(2*j+1))),
					(*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(4*j))),
					(*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(4*j+1))),
					(*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(4*j+2))),
					(*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(4*j+3))),
				),
			}
			code = append(code, c...)
		}
	}

	return InstMacro(code)
}

func MacroBitwiseLT(dst, src Addr, field algebra.Fp, bits uint) Inst {
	panic("unimplemented")
}
