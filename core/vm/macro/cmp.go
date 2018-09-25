package macro

import (
	"math/big"

	"github.com/republicprotocol/oro-go/core/vm/asm"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
)

func GenerateRandomBit(dst asm.Memory, n int, field algebra.Fp) asm.Inst {
	// We need (q+1)/4, where q is the prime determining the field. This is
	// equivalent to (q-3)/4 + 1. We can get q-3 in the field because it is
	// simply -3, and we can perform the division by using the fact that since
	// q-3 is divisible by 4, multiplication by the (field) inverse of 4 is
	// equivalent to normal division.
	e := field.NewInField(big.NewInt(3)).Neg()
	one := field.NewInField(big.NewInt(1))
	twoInv := field.NewInField(big.NewInt(2)).Inv()
	fourInv := field.NewInField(big.NewInt(4)).Inv()
	e = e.Mul(fourInv)
	e = e.Add(field.NewInField(big.NewInt(1)))

	tmp1 := asm.Alloc(n)
	tmp2 := asm.Alloc(n)
	for i := 0; i < n; i++ {
		tmp2.Store(i, asm.NewValuePublic(e))
	}
	tmp3 := asm.Alloc(n)
	for i := 0; i < n; i++ {
		tmp3.Store(i, asm.NewValuePublic(one))
	}
	tmp4 := asm.Alloc(n)
	for i := 0; i < n; i++ {
		tmp4.Store(i, asm.NewValuePublic(twoInv))
	}

	code := []asm.Inst{
		asm.InstGenerateRn(dst, n),
		asm.InstMulOpen(tmp1, dst, dst, n),
		asm.InstExp(tmp2, tmp1, tmp2, n),
		asm.InstInv(tmp2, tmp2, n),
		asm.InstMul(tmp2, dst, tmp2, nil, nil, n),
		asm.InstAdd(tmp2, tmp3, tmp2, n),
		asm.InstMul(dst, tmp2, tmp4, nil, nil, n),
	}
	return asm.InstMacro(code)
}

func Bits(dst, src asm.Memory, bits int, field algebra.Fp) asm.Inst {
	two := asm.NewValuePublic(field.NewInField(big.NewInt(2)))

	tmp1 := asm.Alloc(1, two)
	tmp2 := asm.Alloc(1, two.Inv())
	tmp3 := asm.Alloc(1)

	code := []asm.Inst{
		asm.InstCopy(tmp3, src, 1),
	}

	for i := 0; i < bits; i++ {
		c := []asm.Inst{
			asm.InstMod(dst.Offset(i), tmp3, tmp1, 1),
			asm.InstSub(tmp3, tmp3, dst.Offset(i), 1),
			asm.InstMul(tmp3, tmp3, tmp2, nil, nil, 1),
		}
		code = append(code, c...)
	}

	return asm.InstMacro(code)
}

func Mod2M(dst, src asm.Memory, bits, m, kappa int, field algebra.Fp) asm.Inst {
	twoPowerBits := asm.NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(1 << uint(bits-1))))
	twoPowerM := asm.NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(1 << uint(m))))

	tmpBits := asm.Alloc(m)
	tmpRandBits := asm.Alloc(bits + kappa)
	tmpTwoPowerM := asm.Alloc(1, twoPowerM)

	tmp1 := asm.Alloc(1, twoPowerBits)
	tmp2 := asm.Alloc(1)
	tmp3 := asm.Alloc(1)
	tmp4 := asm.Alloc(bits + kappa)
	for i := 0; i < bits+kappa; i++ {
		tmp4.Store(i, asm.NewValuePublic(field.NewInField(big.NewInt(int64(1<<uint(i))))))
	}

	code := []asm.Inst{
		// Random bits
		GenerateRandomBit(tmpRandBits, bits+kappa, field),

		// Random bits multiplied by powers of two
		asm.InstMul(tmp4, tmp4, tmpRandBits, nil, nil, bits+kappa),

		// Random number from first m bits
		asm.InstAdd(asm.NewAddrIter(tmp4, 0), asm.NewAddrIter(tmp4, 0), asm.NewAddrIter(tmp4.Offset(1), 1), m-1),
		asm.InstCopy(tmp2, tmp4, 1),

		// Random number from all bits
		asm.InstAdd(asm.NewAddrIter(tmp4, 0), asm.NewAddrIter(tmp4, 0), asm.NewAddrIter(tmp4.Offset(m), 1), bits+kappa-m),
		asm.InstCopy(tmp3, tmp4, 1),

		// Mod2m
		asm.InstAdd(tmp1, tmp1, src, 1),
		asm.InstAdd(tmp1, tmp1, tmp3, 1),
		asm.InstOpen(tmp1, tmp1, 1),
		asm.InstMod(tmp1, tmp1, tmpTwoPowerM, 1),
		Bits(tmpBits, tmp1, m, field),
		BitwiseLT(tmp4, tmpBits, tmpRandBits, m, field),
		asm.InstMul(tmp4, tmp4, tmpTwoPowerM, nil, nil, 1),
		asm.InstAdd(tmp4, tmp4, tmp1, 1),
		asm.InstSub(dst, tmp4, tmp2, 1),
	}

	return asm.InstMacro(code)
}

func Truncate(dst, src asm.Memory, bits, m, kappa int, field algebra.Fp) asm.Inst {
	twoPowerM := asm.NewValuePublic(field.NewInField(big.NewInt(0).SetUint64(1 << uint(m))))

	tmp1 := asm.Alloc(1)
	tmp2 := asm.Alloc(1, twoPowerM.Inv())

	code := []asm.Inst{
		Mod2M(tmp1, src, bits, m, kappa, field),
		asm.InstSub(tmp1, src, tmp1, 1),
		asm.InstMul(dst, tmp1, tmp2, nil, nil, 1),
	}
	return asm.InstMacro(code)
}

func LTZ(dst, src asm.Memory, bits, kappa int, field algebra.Fp) asm.Inst {
	code := []asm.Inst{
		Truncate(dst, src, bits, bits-1, kappa, field),
		asm.InstNeg(dst, dst, 1),
	}
	return asm.InstMacro(code)
}

func LT(dst, lhs, rhs asm.Memory, bits, kappa int, field algebra.Fp) asm.Inst {
	code := []asm.Inst{
		asm.InstSub(dst, lhs, rhs, 1),
		LTZ(dst, dst, bits, kappa, field),
	}
	return asm.InstMacro(code)
}
