package proc

import (
	"math/big"

	"github.com/republicprotocol/oro-go/core/vm/asm"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
)

func MacroBitwiseNot(dst, src asm.Memory, n int, field algebra.Fp) asm.Inst {
	tmp := asm.NewAddrIter(asm.Alloc(n), 1)
	for i := 0; i < n; i++ {
		tmp.Store(i, asm.NewValuePublic(field.NewInField(big.NewInt(1))))
	}
	code := []asm.Inst{
		asm.InstSub(dst, tmp, src, n), // 1 - a
	}
	return asm.InstMacro(code)
}

func MacroBitwiseOr(dst, lhs, rhs, ρs, σs asm.Memory, n int) asm.Inst {
	tmp := asm.NewAddrIter(asm.Alloc(n), 1)
	code := []asm.Inst{
		asm.InstMul(tmp, lhs, rhs, ρs, σs, n), // ab
		asm.InstSub(tmp, rhs, tmp, n),         // b - ab
		asm.InstAdd(dst, lhs, tmp, n),         // a + b - ab
	}
	return asm.InstMacro(code)
}

func MacroBitwiseXor(dst, lhs, rhs, ρs, σs asm.Memory, n int) asm.Inst {
	code := []asm.Inst{
		asm.InstSub(dst, lhs, rhs, n),         // a - b
		asm.InstMul(dst, dst, dst, ρs, σs, n), // (a - b)^2
	}
	return asm.InstMacro(code)
}

func MacroBitwiseAnd(dst, lhs, rhs, ρs, σs asm.Memory, n int) asm.Inst {
	code := []asm.Inst{
		asm.InstMul(dst, lhs, rhs, ρs, σs, n),
	}
	return asm.InstMacro(code)
}

func MacroBitwisePropGen(pDst, gDst, lhs, rhs, ρs, σs asm.Memory, n int) asm.Inst {
	tmp := asm.NewAddrIter(asm.Alloc(n), 1)
	code := []asm.Inst{
		MacroBitwiseXor(tmp, lhs, rhs, ρs, σs, n),
		MacroBitwiseAnd(gDst, lhs, rhs, ρs.Offset(n), σs.Offset(n), n),
		asm.InstCopy(pDst, tmp, n),
	}
	return asm.InstMacro(code)
}

func MacroBitwiseOpCLA(pDst, gDst, p1, g1, p2, g2, ρs, σs asm.Memory, n int) asm.Inst {
	tmp1 := asm.NewAddrIter(asm.Alloc(n), 1)
	tmp2 := asm.NewAddrIter(asm.Alloc(n), 1)
	code := []asm.Inst{
		MacroBitwiseAnd(tmp1, p1, p2, ρs, σs, n),
		MacroBitwiseAnd(tmp2, g1, p2, ρs.Offset(n), σs.Offset(n), n),
		MacroBitwiseOr(gDst, g2, tmp2, ρs.Offset(2*n), σs.Offset(2*n), n),
		asm.InstCopy(pDst, tmp1, n),
	}
	return asm.InstMacro(code)
}

func MacroBitwiseCarryOut(dst, lhs, rhs asm.Memory, carryIn bool, bits int, field algebra.Fp) asm.Inst {

	rnRequired := 2 * bits
	rnRequired += 3 * (bits - 1)
	if carryIn {
		rnRequired++
	}

	ρs := asm.Alloc(rnRequired)
	σs := asm.Alloc(rnRequired)
	props := asm.Alloc(bits)
	gens := asm.Alloc(bits)

	code := []asm.Inst{
		asm.InstGenerateRnTuple(ρs, σs, rnRequired),
		MacroBitwisePropGen(
			props,
			gens,
			lhs,
			rhs,
			ρs, // This will consume 2N random numbers
			σs, // This will consume 2N random numbers
			bits,
		),
	}

	// If there is initial carry in then update the first generator
	if carryIn {
		code = append(code,
			MacroBitwiseOr(
				gens,
				props,
				gens,
				ρs, // This will consume a random number
				σs, // This will consume a random number
				1,
			))
	}

	rnOffset := 0
	remaining := bits
	for remaining != 1 {
		pairs := remaining / 2

		code = append(code,
			MacroBitwiseOpCLA(
				props,
				gens,
				asm.NewAddrIter(props, 2),
				asm.NewAddrIter(gens, 2),
				asm.NewAddrIter(props.Offset(1), 2),
				asm.NewAddrIter(gens.Offset(1), 2),
				ρs.Offset(rnOffset+2*bits+1), // This will consume 3N random numbers
				σs.Offset(rnOffset+2*bits+1), // This will consume 3N random numbers
				pairs,
			))
		rnOffset += 3 * pairs

		if remaining%2 == 1 {
			code = append(code,
				asm.InstCopy(props.Offset(2*pairs), props.Offset(4*pairs), 1),
				asm.InstCopy(gens.Offset(2*pairs), gens.Offset(4*pairs), 1),
			)
			remaining = (remaining + 1) / 2
		} else {
			remaining /= 2
		}
	}

	code = append(code, asm.InstCopy(dst, gens.Offset(1), 1))

	return asm.InstMacro(code)
}

// func MacroBitwiseLT(dst, lhs, rhs asm.AddrIter, bits int, field algebra.Fp) asm.Inst {

// 	size := unsafe.Sizeof(interface{}(nil))
// 	tmps := make([]asm.Value, bits)
// 	rhsPtr := unsafe.Pointer(rhs)

// 	code := make([]asm.Inst, 0)
// 	for i := 0; i < bits; i++ {
// 		code = append(code,
// 			MacroBitwiseNot(
// 				&tmps[i],
// 				(*asm.Value)(unsafe.Pointer(uintptr(rhsPtr)+size*uintptr(i))),
// 				field,
// 			))
// 	}

// 	code = append(code,
// 		MacroBitwiseCOut(dst, lhs, &tmps[0], true, field, bits),
// 		MacroBitwiseNot(dst, dst, field),
// 	)

// 	return asm.InstMacro(code)
// }

// func MacroRandBit(dst asm.Addr, field algebra.Fp) asm.Inst {

// 	tmp1 := new(asm.Value)
// 	tmp2 := new(asm.Value)

// 	// We need (q+1)/4, where q is the prime determining the field. This is
// 	// equivalent to (q-3)/4 + 1. We can get q-3 in the field because it is
// 	// simply -3, and we can perform the division by using the fact that since
// 	// q-3 is divisible by 4, multiplication by the (field) inverse of 4 is
// 	// equivalent to normal division.
// 	e := field.NewInField(big.NewInt(3)).Neg()
// 	twoInv := field.NewInField(big.NewInt(2)).Inv()
// 	fourInv := field.NewInField(big.NewInt(4)).Inv()
// 	e = e.Mul(fourInv)
// 	e = e.Add(field.NewInField(big.NewInt(1)))

// 	code := []asm.Inst{
// 		asm.InstGenerateRn(dst, 1),
// 		asm.InstMulOpen(tmp1, dst, dst),
// 		asm.InstMove(tmp2, asm.NewValuePublic(e)),
// 		asm.InstExp(tmp2, tmp1, tmp2),
// 		asm.InstInv(tmp2, tmp2),
// 		asm.InstMulPub(tmp2, dst, tmp2),
// 		asm.InstMove(tmp1, asm.NewValuePublic(field.NewInField(big.NewInt(1)))),
// 		asm.InstAdd(tmp2, tmp1, tmp2),
// 		asm.InstMove(tmp1, asm.NewValuePublic(twoInv)),
// 		asm.InstMulPub(dst, tmp2, tmp1),
// 	}
// 	return asm.InstMacro(code)
// }

// func MacroBits(dst, src asm.Addr, bits uint64, field algebra.Fp) asm.Inst {

// 	size := unsafe.Sizeof(interface{}(nil))
// 	dstPtr := unsafe.Pointer(dst)
// 	tmp1 := new(asm.Value)
// 	tmp2 := new(asm.Value)
// 	tmp3 := new(asm.Value)

// 	two := asm.NewValuePublic(field.NewInField(big.NewInt(2)))

// 	code := []asm.Inst{
// 		asm.InstMove(tmp1, two),
// 		asm.InstMove(tmp2, two),
// 		asm.InstInv(tmp2, tmp2),
// 		asm.InstCopy(tmp3, src, 1, 1),
// 	}

// 	for i := uint64(0); i < bits; i++ {
// 		c := []asm.Inst{
// 			asm.InstMod((*asm.Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(i))), tmp3, tmp1),
// 			asm.InstSub(tmp3, tmp3, (*asm.Value)(unsafe.Pointer(uintptr(dstPtr)+size*uintptr(i)))),
// 			asm.InstMulPub(tmp3, tmp3, tmp2),
// 		}
// 		code = append(code, c...)
// 	}

// 	return asm.InstMacro(code)
// }

// func MacroMod2m(dst, src asm.Addr, bits, m, kappa uint64, field algebra.Fp) asm.Inst {

// 	tmp1 := new(asm.Value)
// 	tmp2 := new(asm.Value)
// 	tmp3 := new(asm.Value)
// 	tmp4 := new(asm.Value)
// 	tmpBits := make([]asm.Value, m)
// 	tmpRandBits := make([]asm.Value, bits+kappa)

// 	zero := asm.NewValuePublic(field.NewInField(big.NewInt(0)))
// 	two := asm.NewValuePublic(field.NewInField(big.NewInt(2)))
// 	twoPowerBits := asm.NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(uint64(1) << (bits - 1))))
// 	twoPowerM := asm.NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(uint64(1) << m)))

// 	code := []asm.Inst{
// 		asm.InstMove(tmp1, two),
// 		asm.InstMove(tmp2, zero),
// 		asm.InstMove(tmp3, zero),
// 	}

// 	// Generate the needed random bits
// 	for i := range tmpRandBits {
// 		code = append(code, MacroRandBit(&tmpRandBits[i], field))
// 	}

// 	// Random number defined by the first m random bits
// 	for i := int(m) - 1; i >= 0; i-- {
// 		c := []asm.Inst{
// 			asm.InstMulPub(tmp2, tmp2, tmp1),
// 			asm.InstAdd(tmp2, tmp2, &tmpRandBits[i]),
// 		}
// 		code = append(code, c...)
// 	}

// 	// Random number defined by all of the random bits
// 	for i := bits + kappa - 1; i >= m; i-- {
// 		c := []asm.Inst{
// 			asm.InstMulPub(tmp3, tmp3, tmp1),
// 			asm.InstAdd(tmp3, tmp3, &tmpRandBits[i]),
// 		}
// 		code = append(code, c...)
// 	}
// 	code = append(code,
// 		asm.InstMove(tmp1, twoPowerM),
// 		asm.InstMulPub(tmp3, tmp3, tmp1),
// 		asm.InstAdd(tmp3, tmp3, tmp2),
// 	)

// 	c := []asm.Inst{
// 		asm.InstMove(tmp1, twoPowerBits),
// 		asm.InstAdd(tmp1, tmp1, src),
// 		asm.InstAdd(tmp1, tmp1, tmp3),
// 		asm.InstOpen(tmp1, tmp1),
// 		asm.InstMove(tmp3, twoPowerM),
// 		asm.InstMod(tmp1, tmp1, tmp3),
// 		MacroBits(&tmpBits[0], tmp1, m, field),
// 		MacroBitwiseLT(tmp4, &tmpBits[0], &tmpRandBits[0], field, int(m)),
// 		asm.InstMulPub(tmp4, tmp4, tmp3),
// 		asm.InstAdd(tmp4, tmp4, tmp1),
// 		asm.InstSub(dst, tmp4, tmp2),
// 	}
// 	code = append(code, c...)

// 	return asm.InstMacro(code)
// }

// func MacroTrunc(dst, src asm.Addr, bits, m, kappa uint64, field algebra.Fp) asm.Inst {

// 	tmp1 := new(asm.Value)
// 	tmp2 := new(asm.Value)

// 	twoPowerM := asm.NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(uint64(1) << m)))

// 	code := []asm.Inst{
// 		MacroMod2m(tmp1, src, bits, m, kappa, field),
// 		asm.InstMove(tmp2, twoPowerM),
// 		asm.InstInv(tmp2, tmp2),
// 		asm.InstSub(tmp1, src, tmp1),
// 		asm.InstMulPub(dst, tmp1, tmp2),
// 	}
// 	return asm.InstMacro(code)
// }

// func MacroLTZ(dst, src asm.Addr, bits, kappa uint64, field algebra.Fp) asm.Inst {

// 	code := []asm.Inst{
// 		MacroTrunc(dst, src, bits, bits-1, kappa, field),
// 		asm.InstNeg(dst, dst),
// 	}
// 	return asm.InstMacro(code)
// }

// func MacroLT(dst, lhs, rhs asm.Addr, bits, kappa uint64, field algebra.Fp) asm.Inst {

// 	code := []asm.Inst{
// 		asm.InstSub(dst, lhs, rhs),
// 		MacroLTZ(dst, dst, bits, kappa, field),
// 	}
// 	return asm.InstMacro(code)
// }
