package process

import (
	"math/big"
	"unsafe"

	"github.com/republicprotocol/oro-go/core/vss/algebra"
)

// MacroGenerateRnTuplesN executes the generation of N random number tuples in
// parallel. The destination Memory allocation is expected to have 2N contiguous
// indices, where N ρ-σ tuples will be stored, where ρ-σ tuples are stored
// immediately adjacent.
func MacroGenerateRnTuplesN(dst Memory, n int) Inst {

	code := Code{}
	code = append(code, InstAsync())
	for i := 0; i < n; i++ {
		code = append(code, InstGenerateRnTuple(dst.At(2*i), 1))
	}
	code = append(code, InstAwait())

	return InstMacro(code)
}

func MacroBitwiseNot(dst, src Addr, field algebra.Fp) Inst {
	tmp1 := new(Value)
	code := Code{
		InstMove(tmp1, NewValuePublic(field.NewInField(big.NewInt(1)))),
		InstSub(dst, tmp1, src),
	}
	return InstMacro(code)
}

func MacroBitwiseOr(dst, lhs, rhs Addr) Inst {
	tmp1n2 := make([]Value, 2)
	tmp3 := new(Value)
	code := Code{
		InstGenerateRnTuple(&tmp1n2[0], 1),              // rand
		InstMul(tmp3, lhs, rhs, &tmp1n2[0], &tmp1n2[1]), // ab
		InstSub(tmp3, rhs, tmp3),                        // b - ab
		InstAdd(dst, lhs, tmp3),                         // a + b - ab
	}
	return InstMacro(code)
}

// MacroBitwiseOrN executes the equivalent of a MacroBitwiseOr on N bits in
// parallel. The destination, left-hand side, and right-hand side Memory
// allocations are expected to have N contiguous indices. The random number
// Memory allocation is expeted to have 2N contiguous indices, storing N ρ-σ
// tuples, where ρ-σ tuples are immediately adjacent.
func MacroBitwiseOrN(dst, lhs, rhs, rns Memory, n int) Inst {
	tmp := NewMemory(n)

	code := Code{}
	code = append(code, InstAsync())
	for i := 0; i < n; i++ {
		code = append(code, InstMul(tmp.At(i), lhs.At(i), rhs.At(i), rns.At(2*i), rns.At(2*i+1))) // ab
	}
	code = append(code, InstAwait())
	for i := 0; i < n; i++ {
		code = append(code,
			InstSub(tmp.At(i), rhs.At(i), tmp.At(i)), // b - ab
			InstAdd(dst.At(i), lhs.At(i), tmp.At(i)), // a + b - ab
		)
	}

	return InstMacro(code)
}

func MacroBitwiseXor(dst, lhs, rhs Addr) Inst {
	tmp1n2 := make([]Value, 2)
	code := Code{
		InstSub(dst, lhs, rhs),                         // a - b
		InstGenerateRnTuple(&tmp1n2[0], 1),             // rand
		InstMul(dst, dst, dst, &tmp1n2[0], &tmp1n2[1]), // (a - b)^2
	}
	return InstMacro(code)
}

func MacroBitwiseXorN(dst, lhs, rhs, rns Memory, n int) Inst {

	code := Code{}
	for i := 0; i < n; i++ {
		code = append(code, InstSub(dst.At(i), lhs.At(i), rhs.At(i))) // ab
	}
	code = append(code, InstAsync())
	for i := 0; i < n; i++ {
		code = append(code, InstMul(dst.At(i), dst.At(i), dst.At(i), rns.At(2*i), rns.At(2*i+1)))
	}
	code = append(code, InstAwait())

	return InstMacro(code)
}

func MacroBitwiseAnd(dst, lhs, rhs Addr) Inst {
	tmp1n2 := make([]Value, 2)
	code := Code{
		InstGenerateRnTuple(&tmp1n2[0], 1),
		InstMul(dst, lhs, rhs, &tmp1n2[0], &tmp1n2[1]),
	}
	return InstMacro(code)
}

func MacroBitwiseAndN(dst, lhs, rhs, rns Memory, n int) Inst {

	code := Code{}
	code = append(code, InstAsync())
	for i := 0; i < n; i++ {
		code = append(code, InstMul(dst.At(i), lhs.At(i), rhs.At(i), rns.At(2*i), rns.At(2*i+1)))
	}
	code = append(code, InstAwait())

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

func MacroBitwisePropGenN(propDst, genDst, lhs, rhs, rns Memory, n int) Inst {
	tmp := NewMemory(n)

	// XOR and AND
	code := Code{}
	for i := 0; i < n; i++ {
		code = append(code, InstSub(tmp.At(i), lhs.At(i), rhs.At(i)))
	}
	code = append(code, InstAsync())
	for i := 0; i < n; i++ {
		code = append(code,
			InstMul(tmp.At(i), tmp.At(i), tmp.At(i), rns.At(2*i), rns.At(2*i+1)),
			InstMul(genDst.At(i), lhs.At(i), rhs.At(i), rns.At(2*n+2*i), rns.At(2*n+2*i+1)),
		)
	}
	code = append(code, InstAwait())

	// COPY
	for i := 0; i < n; i++ {
		code = append(code, InstCopy(propDst.At(i), tmp.At(i)))
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

func MacroBitwiseOpCLAN(propDst, genDst, props, gens, rns Memory, n int) Inst {
	tmp1 := NewMemory(n)
	tmp2 := NewMemory(n)
	tmp3 := NewMemory(n)
	code := Code{}

	// ANDs
	code = append(code, InstAsync())
	for i := 0; i < n; i++ {
		code = append(code,
			InstMul(tmp1.At(i), props.At(2*i), props.At(2*i+1), rns.At(2*i), rns.At(2*i+1)),
			InstMul(tmp2.At(i), gens.At(2*i), props.At(2*i+1), rns.At(2*n+2*i), rns.At(2*n+2*i+1)),
		)
	}
	code = append(code, InstAwait())

	// OR
	code = append(code, InstAsync())
	for i := 0; i < n; i++ {
		code = append(code, InstMul(tmp3.At(i), gens.At(2*i+1), tmp2.At(i), rns.At(4*n+2*i), rns.At(4*n+2*i+1))) // ab
	}
	code = append(code, InstAwait())
	for i := 0; i < n; i++ {
		code = append(code,
			InstSub(tmp3.At(i), tmp2.At(i), tmp3.At(i)),       // b - ab
			InstAdd(genDst.At(i), gens.At(2*i+1), tmp3.At(i)), // a + b - ab
		)
	}

	// COPY
	for i := 0; i < n; i++ {
		code = append(code, InstCopy(propDst.At(i), tmp1.At(i)))
	}

	return InstMacro(code)
}

func MacroBitwiseCOut(dst, src Addr, field algebra.Fp, bits int) Inst {

	size := unsafe.Sizeof(interface{}(nil))
	dstPtr := unsafe.Pointer(dst)
	srcPtr := unsafe.Pointer(src)

	code := make(Code, 0)
	for i := 0; i < bits; i++ {
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
		for j := 0; j < i; j++ {
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

func MacroBitwiseCOutN(dst, lhs, rhs Memory, field algebra.Fp, bits int) Inst {

	rns := NewMemory(4*bits + 6*(bits-1))

	code := make(Code, 0)
	code = append(code,
		InstGenerateRnTuple(rns.At(0), 2*bits+3*(bits-1)),
		MacroBitwisePropGenN(
			dst[0:],
			dst[bits:],
			lhs[0:],
			rhs[0:],
			rns[0:],
			bits,
		),
	)

	j := 0
	for i := bits / 2; i > 0; i /= 2 {
		code = append(code,
			MacroBitwiseOpCLAN(
				dst[0:],
				dst[bits:],
				dst[0:],
				dst[bits:],
				rns[4*bits+j:],
				i,
			),
		)
		j += 6 * i
	}

	return InstMacro(code)
}

func MacroBitwiseLT(dst, src Addr, field algebra.Fp, bits uint) Inst {
	panic("unimplemented")
}
