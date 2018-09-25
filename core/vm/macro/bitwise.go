package macro

import (
	"math/big"

	"github.com/republicprotocol/oro-go/core/vm/asm"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
)

func BitwiseNot(dst, src asm.Memory, n int, field algebra.Fp) asm.Inst {
	tmp := asm.Alloc(n)
	for i := 0; i < n; i++ {
		tmp.Store(i, asm.NewValuePublic(field.NewInField(big.NewInt(1))))
	}
	code := []asm.Inst{
		asm.InstSub(dst, tmp, src, n), // 1 - a
	}
	return asm.InstMacro(code)
}

func BitwiseOr(dst, lhs, rhs, ρs, σs asm.Memory, n int) asm.Inst {
	tmp := asm.Alloc(n)
	code := []asm.Inst{
		asm.InstMul(tmp, lhs, rhs, ρs, σs, n), // ab
		asm.InstSub(tmp, rhs, tmp, n),         // b - ab
		asm.InstAdd(dst, lhs, tmp, n),         // a + b - ab
	}
	return asm.InstMacro(code)
}

func BitwiseXor(dst, lhs, rhs, ρs, σs asm.Memory, n int) asm.Inst {
	code := []asm.Inst{
		asm.InstSub(dst, lhs, rhs, n),         // a - b
		asm.InstMul(dst, dst, dst, ρs, σs, n), // (a - b)^2
	}
	return asm.InstMacro(code)
}

func BitwiseAnd(dst, lhs, rhs, ρs, σs asm.Memory, n int) asm.Inst {
	code := []asm.Inst{
		asm.InstMul(dst, lhs, rhs, ρs, σs, n),
	}
	return asm.InstMacro(code)
}

func BitwisePropGen(pDst, gDst, lhs, rhs, ρs, σs asm.Memory, n int) asm.Inst {
	tmp := asm.Alloc(n)
	code := []asm.Inst{
		BitwiseXor(tmp, lhs, rhs, ρs, σs, n),
		BitwiseAnd(gDst, lhs, rhs, ρs.Offset(n), σs.Offset(n), n),
		asm.InstCopy(pDst, tmp, n),
	}
	return asm.InstMacro(code)
}

func BitwiseOpCLA(pDst, gDst, p1, g1, p2, g2, ρs, σs asm.Memory, n int) asm.Inst {
	tmp1 := asm.Alloc(n)
	tmp2 := asm.Alloc(n)
	code := []asm.Inst{
		BitwiseAnd(tmp1, p1, p2, ρs, σs, n),
		BitwiseAnd(tmp2, g1, p2, ρs.Offset(n), σs.Offset(n), n),
		BitwiseOr(gDst, g2, tmp2, ρs.Offset(2*n), σs.Offset(2*n), n),
		asm.InstCopy(pDst, tmp1, n),
	}
	return asm.InstMacro(code)
}

func BitwiseCarryOut(dst, lhs, rhs asm.Memory, carryIn bool, bits int, field algebra.Fp) asm.Inst {

	rnOffset := 0
	rnRequired := 2*bits + 3*(bits-1)
	if carryIn {
		rnRequired++
	}

	ρs := asm.Alloc(rnRequired)
	σs := asm.Alloc(rnRequired)
	props := asm.Alloc(bits)
	gens := asm.Alloc(bits)

	code := []asm.Inst{
		asm.InstGenerateRnTuple(ρs, σs, rnRequired),
		BitwisePropGen(
			props,
			gens,
			lhs,
			rhs,
			ρs, // This will consume 2N random numbers
			σs, // This will consume 2N random numbers
			bits,
		),
	}
	rnOffset += 2 * bits

	// If there is initial carry in then update the first generator
	if carryIn {
		code = append(code,
			BitwiseOr(
				gens,
				props,
				gens,
				ρs.Offset(rnOffset), // This will consume a random number
				σs.Offset(rnOffset), // This will consume a random number
				1,
			))
		rnOffset++
	}

	remaining := bits
	for remaining != 1 {
		pairs := remaining / 2

		code = append(code,
			BitwiseOpCLA(
				props,
				gens,
				asm.MemoryMapper(props, 2),
				asm.MemoryMapper(gens, 2),
				asm.MemoryMapper(props.Offset(1), 2),
				asm.MemoryMapper(gens.Offset(1), 2),
				ρs.Offset(rnOffset), // This will consume 3N random numbers
				σs.Offset(rnOffset), // This will consume 3N random numbers
				pairs,
			))
		rnOffset += 3 * pairs

		if remaining%2 == 1 {
			code = append(code,
				asm.InstCopy(props.Offset(pairs), props.Offset(2*pairs), 1),
				asm.InstCopy(gens.Offset(pairs), gens.Offset(2*pairs), 1),
			)
			remaining = (remaining + 1) / 2
		} else {
			remaining /= 2
		}
	}

	code = append(code, asm.InstCopy(dst, gens, 1))

	return asm.InstMacro(code)
}

func BitwiseLT(dst, lhs, rhs asm.Memory, bits int, field algebra.Fp) asm.Inst {

	tmp := asm.Alloc(bits)

	code := []asm.Inst{
		BitwiseNot(
			tmp,
			rhs,
			bits,
			field,
		),
		BitwiseCarryOut(dst, lhs, tmp, true, bits, field),
		BitwiseNot(dst, dst, 1, field),
	}

	return asm.InstMacro(code)
}
