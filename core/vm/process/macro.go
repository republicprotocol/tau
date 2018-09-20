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

	remaining := bits
	for remaining != 1 {
		pairs := remaining / 2
		for j := uint(0); j < pairs; j++ {
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

		if remaining%2 == 1 {
			code = append(code,
				InstCopy(&tmps[2*pairs], &tmps[4*pairs]),
				InstCopy(&tmps[2*pairs+1], &tmps[4*pairs+1]),
			)

			remaining = (remaining + 1) / 2
		} else {
			remaining /= 2
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
		code = append(code,
			MacroBitwiseNot(
				&tmps[i],
				(*Value)(unsafe.Pointer(uintptr(rhsPtr)+size*uintptr(i))),
				field,
			))
	}

	code = append(code,
		MacroBitwiseCOut(dst, lhs, &tmps[0], true, field, bits),
		MacroBitwiseNot(dst, dst, field),
	)

	return InstMacro(code)
}

func MacroRandBit(dst Addr, field algebra.Fp) Inst {

	tmp1 := new(Value)
	tmp2 := new(Value)

	// We need (q+1)/4, where q is the prime determining the field. This is
	// equivalent to (q-3)/4 + 1. We can get q-3 in the field because it is
	// simply -3, and we can perform the division by using the fact that since
	// q-3 is divisible by 4, multiplication by the (field) inverse of 4 is
	// equivalent to normal division.
	e := field.NewInField(big.NewInt(3)).Neg()
	twoInv := field.NewInField(big.NewInt(2)).Inv()
	fourInv := field.NewInField(big.NewInt(4)).Inv()
	e = e.Mul(fourInv)
	e = e.Add(field.NewInField(big.NewInt(1)))

	code := Code{
		InstGenerateRn(dst),
		InstMulOpen(tmp1, dst, dst),
		InstMove(tmp2, NewValuePublic(e)),
		InstExp(tmp2, tmp1, tmp2),
		InstInv(tmp2, tmp2),
		InstMulPub(tmp2, dst, tmp2),
		InstMove(tmp1, NewValuePublic(field.NewInField(big.NewInt(1)))),
		InstAdd(tmp2, tmp1, tmp2),
		InstMove(tmp1, NewValuePublic(twoInv)),
		InstMulPub(dst, tmp2, tmp1),
	}
	return InstMacro(code)
}

func MacroBits(dst, src Addr, bits uint64, field algebra.Fp) Inst {

	size := unsafe.Sizeof(interface{}(nil))
	dstPtr := unsafe.Pointer(dst)
	tmp1 := new(Value)
	tmp2 := new(Value)
	tmp3 := new(Value)

	two := NewValuePublic(field.NewInField(big.NewInt(2)))

	code := Code{
		InstMove(tmp1, two),
		InstMove(tmp2, two),
		InstInv(tmp2, tmp2),
		InstCopy(tmp3, src),
	}

	for i := uint64(0); i < bits; i++ {
		c := Code{
			InstMod((*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(i))), tmp3, tmp1),
			InstSub(tmp3, tmp3, (*Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(i)))),
			InstMulPub(tmp3, tmp3, tmp2),
		}
		code = append(code, c...)
	}

	return InstMacro(code)
}

func MacroMod2m(dst, src Addr, bits, m, kappa uint64, field algebra.Fp) Inst {

	tmp1 := new(Value)
	tmp2 := new(Value)
	tmp3 := new(Value)
	tmp4 := new(Value)
	tmpBits := make([]Value, m)
	tmpRandBits := make([]Value, bits+kappa)

	zero := NewValuePublic(field.NewInField(big.NewInt(0)))
	two := NewValuePublic(field.NewInField(big.NewInt(2)))
	twoPowerBits := NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(uint64(1) << (bits - 1))))
	twoPowerM := NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(uint64(1) << m)))

	code := Code{
		InstMove(tmp1, two),
		InstMove(tmp2, zero),
		InstMove(tmp3, zero),
	}

	// Generate the needed random bits
	for i := range tmpRandBits {
		code = append(code, MacroRandBit(&tmpRandBits[i], field))
	}

	// Random number defined by the first m random bits
	for i := int(m) - 1; i >= 0; i-- {
		c := Code{
			InstMulPub(tmp2, tmp2, tmp1),
			InstAdd(tmp2, tmp2, &tmpRandBits[i]),
		}
		code = append(code, c...)
	}

	// Random number defined by all of the random bits
	for i := bits + kappa - 1; i >= m; i-- {
		c := Code{
			InstMulPub(tmp3, tmp3, tmp1),
			InstAdd(tmp3, tmp3, &tmpRandBits[i]),
		}
		code = append(code, c...)
	}
	code = append(code,
		InstMove(tmp1, twoPowerM),
		InstMulPub(tmp3, tmp3, tmp1),
		InstAdd(tmp3, tmp3, tmp2),
	)

	c := Code{
		InstMove(tmp1, twoPowerBits),
		InstAdd(tmp1, tmp1, src),
		InstAdd(tmp1, tmp1, tmp3),
		InstOpen(tmp1, tmp1),
		InstMove(tmp3, twoPowerM),
		InstMod(tmp1, tmp1, tmp3),
		MacroBits(&tmpBits[0], tmp1, m, field),
		MacroBitwiseLT(tmp4, &tmpBits[0], &tmpRandBits[0], field, uint(m)),
		InstMulPub(tmp4, tmp4, tmp3),
		InstAdd(tmp4, tmp4, tmp1),
		InstSub(dst, tmp4, tmp2),
	}
	code = append(code, c...)

	return InstMacro(code)
}

func MacroTrunc(dst, src Addr, bits, m, kappa uint64, field algebra.Fp) Inst {

	tmp1 := new(Value)
	tmp2 := new(Value)

	twoPowerM := NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(uint64(1) << m)))

	code := Code{
		MacroMod2m(tmp1, src, bits, m, kappa, field),
		InstMove(tmp2, twoPowerM),
		InstInv(tmp2, tmp2),
		InstSub(tmp1, src, tmp1),
		InstMulPub(dst, tmp1, tmp2),
	}
	return InstMacro(code)
}

func MacroLTZ(dst, src Addr, bits, kappa uint64, field algebra.Fp) Inst {

	code := Code{
		MacroTrunc(dst, src, bits, bits-1, kappa, field),
		InstNeg(dst, dst),
	}
	return InstMacro(code)
}

func MacroLT(dst, lhs, rhs Addr, bits, kappa uint64, field algebra.Fp) Inst {

	code := Code{
		InstSub(dst, lhs, rhs),
		MacroLTZ(dst, dst, bits, kappa, field),
	}
	return InstMacro(code)
}
