package proc

import (
	"math/big"
	"unsafe"

	"github.com/republicprotocol/oro-go/core/vm/asm"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
)

func MacroBitwiseNot(dst, src asm.AddrIter, n int, field algebra.Fp) asm.Inst {
	tmp := asm.Alloc(n)
	for i := 0; i < n; i++ {
		tmp.Store(i, asm.NewValuePublic(field.NewInField(big.NewInt(1))))
	}
	code := []asm.Inst{
		asm.InstSub(dst.Addr, tmp, src.Addr, dst.Step, 1, src.Step, n), // 1 - a
	}
	return asm.InstMacro(code)
}

func MacroBitwiseOr(dst, lhs, rhs, ρs, σs asm.AddrIter, n int) asm.Inst {
	tmp := asm.Alloc(n)
	code := []asm.Inst{
		asm.InstMul(tmp, lhs.Addr, rhs.Addr, ρs.Addr, σs.Addr, 1, lhs.Step, rhs.Step, ρs.Step, σs.Step, n), // ab
		asm.InstSub(tmp, rhs.Addr, tmp, 1, rhs.Step, 1, n),                                                 // b - ab
		asm.InstAdd(dst.Addr, lhs.Addr, tmp, dst.Step, lhs.Step, 1, n),                                     // a + b - ab
	}
	return asm.InstMacro(code)
}

func MacroBitwiseXor(dst, lhs, rhs, ρs, σs asm.AddrIter, n int) asm.Inst {
	code := []asm.Inst{
		asm.InstSub(dst.Addr, lhs.Addr, rhs.Addr, dst.Step, lhs.Step, rhs.Step, n),                                     // a - b
		asm.InstMul(dst.Addr, dst.Addr, dst.Addr, ρs.Addr, σs.Addr, dst.Step, dst.Step, dst.Step, ρs.Step, σs.Step, n), // (a - b)^2
	}
	return asm.InstMacro(code)
}

func MacroBitwiseAnd(dst, lhs, rhs, ρs, σs asm.AddrIter, n int) asm.Inst {
	code := []asm.Inst{
		asm.InstMul(dst.Addr, lhs.Addr, rhs.Addr, ρs.Addr, σs.Addr, dst.Step, lhs.Step, rhs.Step, ρs.Step, σs.Step, n),
	}
	return asm.InstMacro(code)
}

func MacroBitwisePropGen(pDst, gDst, lhs, rhs, ρs, σs asm.AddrIter, n int) asm.Inst {
	tmp := asm.NewAddrIter(asm.Alloc(n), 1)
	code := []asm.Inst{
		MacroBitwiseXor(tmp, lhs, rhs, ρs, σs, n),
		MacroBitwiseAnd(gDst, lhs, rhs, ρs.Offset(n), σs.Offset(n), n),
		asm.InstCopy(pDst.Addr, tmp.Addr, pDst.Step, tmp.Step, n),
	}
	return asm.InstMacro(code)
}

func MacroBitwiseOpCLA(pDst, gDst, p1, g1, p2, g2, ρs, σs asm.AddrIter, n int) asm.Inst {
	tmp1 := asm.NewAddrIter(asm.Alloc(n), 1)
	tmp2 := asm.NewAddrIter(asm.Alloc(n), 1)
	code := []asm.Inst{
		MacroBitwiseAnd(tmp1, p1, p2, ρs, σs, n),
		MacroBitwiseAnd(tmp2, g1, p2, ρs.Offset(n), σs.Offset(n), n),
		MacroBitwiseOr(gDst, g2, tmp2, ρs.Offset(2*n), σs.Offset(2*n), n),
		asm.InstCopy(pDst.Addr, tmp1.Addr, pDst.Step, tmp1.Step, n),
	}
	return asm.InstMacro(code)
}

func MacroBitwiseCOut(dst, lhs, rhs asm.Addr, carry bool, field algebra.Fp, bits int) asm.Inst {

	size := unsafe.Sizeof(interface{}(nil))
	tmps := make([]asm.Value, 2*bits)
	lhsPtr := unsafe.Pointer(lhs)
	rhsPtr := unsafe.Pointer(rhs)

	code := make([]asm.Inst, 0)
	for i := 0; i < bits; i++ {
		c := []asm.Inst{
			MacroBitwisePropGen(
				&tmps[2*i],
				&tmps[2*i+1],
				(*asm.Value)(unsafe.Pointer(uintptr(lhsPtr)+size*uintptr(i))),
				(*asm.Value)(unsafe.Pointer(uintptr(rhsPtr)+size*uintptr(i))),
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
		for j := 0; j < pairs; j++ {
			c := []asm.Inst{
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
				asm.InstCopy(&tmps[2*pairs], &tmps[4*pairs], 1, 1),
				asm.InstCopy(&tmps[2*pairs+1], &tmps[4*pairs+1], 1, 1),
			)

			remaining = (remaining + 1) / 2
		} else {
			remaining /= 2
		}
	}

	code = append(code, asm.InstCopy(dst, &tmps[1], 1, 1))

	return asm.InstMacro(code)
}

func MacroBitwiseLT(dst, lhs, rhs asm.Addr, field algebra.Fp, bits int) asm.Inst {

	size := unsafe.Sizeof(interface{}(nil))
	tmps := make([]asm.Value, bits)
	rhsPtr := unsafe.Pointer(rhs)

	code := make([]asm.Inst, 0)
	for i := 0; i < bits; i++ {
		code = append(code,
			MacroBitwiseNot(
				&tmps[i],
				(*asm.Value)(unsafe.Pointer(uintptr(rhsPtr)+size*uintptr(i))),
				field,
			))
	}

	code = append(code,
		MacroBitwiseCOut(dst, lhs, &tmps[0], true, field, bits),
		MacroBitwiseNot(dst, dst, field),
	)

	return asm.InstMacro(code)
}

func MacroRandBit(dst asm.Addr, field algebra.Fp) asm.Inst {

	tmp1 := new(asm.Value)
	tmp2 := new(asm.Value)

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

	code := []asm.Inst{
		asm.InstGenerateRn(dst, 1),
		asm.InstMulOpen(tmp1, dst, dst),
		asm.InstMove(tmp2, asm.NewValuePublic(e)),
		asm.InstExp(tmp2, tmp1, tmp2),
		asm.InstInv(tmp2, tmp2),
		asm.InstMulPub(tmp2, dst, tmp2),
		asm.InstMove(tmp1, asm.NewValuePublic(field.NewInField(big.NewInt(1)))),
		asm.InstAdd(tmp2, tmp1, tmp2),
		asm.InstMove(tmp1, asm.NewValuePublic(twoInv)),
		asm.InstMulPub(dst, tmp2, tmp1),
	}
	return asm.InstMacro(code)
}

func MacroBits(dst, src asm.Addr, bits uint64, field algebra.Fp) asm.Inst {

	size := unsafe.Sizeof(interface{}(nil))
	dstPtr := unsafe.Pointer(dst)
	tmp1 := new(asm.Value)
	tmp2 := new(asm.Value)
	tmp3 := new(asm.Value)

	two := asm.NewValuePublic(field.NewInField(big.NewInt(2)))

	code := []asm.Inst{
		asm.InstMove(tmp1, two),
		asm.InstMove(tmp2, two),
		asm.InstInv(tmp2, tmp2),
		asm.InstCopy(tmp3, src, 1, 1),
	}

	for i := uint64(0); i < bits; i++ {
		c := []asm.Inst{
			asm.InstMod((*asm.Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(i))), tmp3, tmp1),
			asm.InstSub(tmp3, tmp3, (*asm.Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(i)))),
			asm.InstMulPub(tmp3, tmp3, tmp2),
		}
		code = append(code, c...)
	}

	return asm.InstMacro(code)
}

func MacroMod2m(dst, src asm.Addr, bits, m, kappa uint64, field algebra.Fp) asm.Inst {

	tmp1 := new(asm.Value)
	tmp2 := new(asm.Value)
	tmp3 := new(asm.Value)
	tmp4 := new(asm.Value)
	tmpBits := make([]asm.Value, m)
	tmpRandBits := make([]asm.Value, bits+kappa)

	zero := asm.NewValuePublic(field.NewInField(big.NewInt(0)))
	two := asm.NewValuePublic(field.NewInField(big.NewInt(2)))
	twoPowerBits := asm.NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(uint64(1) << (bits - 1))))
	twoPowerM := asm.NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(uint64(1) << m)))

	code := []asm.Inst{
		asm.InstMove(tmp1, two),
		asm.InstMove(tmp2, zero),
		asm.InstMove(tmp3, zero),
	}

	// Generate the needed random bits
	for i := range tmpRandBits {
		code = append(code, MacroRandBit(&tmpRandBits[i], field))
	}

	// Random number defined by the first m random bits
	for i := int(m) - 1; i >= 0; i-- {
		c := []asm.Inst{
			asm.InstMulPub(tmp2, tmp2, tmp1),
			asm.InstAdd(tmp2, tmp2, &tmpRandBits[i]),
		}
		code = append(code, c...)
	}

	// Random number defined by all of the random bits
	for i := bits + kappa - 1; i >= m; i-- {
		c := []asm.Inst{
			asm.InstMulPub(tmp3, tmp3, tmp1),
			asm.InstAdd(tmp3, tmp3, &tmpRandBits[i]),
		}
		code = append(code, c...)
	}
	code = append(code,
		asm.InstMove(tmp1, twoPowerM),
		asm.InstMulPub(tmp3, tmp3, tmp1),
		asm.InstAdd(tmp3, tmp3, tmp2),
	)

	c := []asm.Inst{
		asm.InstMove(tmp1, twoPowerBits),
		asm.InstAdd(tmp1, tmp1, src),
		asm.InstAdd(tmp1, tmp1, tmp3),
		asm.InstOpen(tmp1, tmp1),
		asm.InstMove(tmp3, twoPowerM),
		asm.InstMod(tmp1, tmp1, tmp3),
		MacroBits(&tmpBits[0], tmp1, m, field),
		MacroBitwiseLT(tmp4, &tmpBits[0], &tmpRandBits[0], field, int(m)),
		asm.InstMulPub(tmp4, tmp4, tmp3),
		asm.InstAdd(tmp4, tmp4, tmp1),
		asm.InstSub(dst, tmp4, tmp2),
	}
	code = append(code, c...)

	return asm.InstMacro(code)
}

func MacroTrunc(dst, src asm.Addr, bits, m, kappa uint64, field algebra.Fp) asm.Inst {

	tmp1 := new(asm.Value)
	tmp2 := new(asm.Value)

	twoPowerM := asm.NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(uint64(1) << m)))

	code := []asm.Inst{
		MacroMod2m(tmp1, src, bits, m, kappa, field),
		asm.InstMove(tmp2, twoPowerM),
		asm.InstInv(tmp2, tmp2),
		asm.InstSub(tmp1, src, tmp1),
		asm.InstMulPub(dst, tmp1, tmp2),
	}
	return asm.InstMacro(code)
}

func MacroLTZ(dst, src asm.Addr, bits, kappa uint64, field algebra.Fp) asm.Inst {

	code := []asm.Inst{
		MacroTrunc(dst, src, bits, bits-1, kappa, field),
		asm.InstNeg(dst, dst),
	}
	return asm.InstMacro(code)
}

func MacroLT(dst, lhs, rhs asm.Addr, bits, kappa uint64, field algebra.Fp) asm.Inst {

	code := []asm.Inst{
		asm.InstSub(dst, lhs, rhs),
		MacroLTZ(dst, dst, bits, kappa, field),
	}
	return asm.InstMacro(code)
}
