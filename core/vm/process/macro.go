package process

import (
	"fmt"
	"log"
	"math/big"

	"github.com/republicprotocol/oro-go/core/vss/algebra"
)

func MacroNot(field algebra.Fp) Inst {
	code := Code{
		InstNeg(),
		InstPush(NewValuePublic(field.NewInField(big.NewInt(1)))),
		InstAdd(),
	}
	return InstMacro(code)
}

func MacroOr() Inst {
	code := Code{
		InstCopy(2),      // ...a, b, a, b]
		InstGenerateRn(), // ...a, b, a, b, r]
		InstMul(),        // ...a, b, ab]
		InstSub(),        // ...a, b - ab]
		InstAdd(),        // ...a + b - ab]
	}
	return InstMacro(code)
}

func MacroXor() Inst {
	code := Code{
		InstSub(),
		InstCopy(1),
		InstGenerateRn(),
		InstMul(),
	}
	return InstMacro(code)
}

func MacroAnd() Inst {
	code := Code{
		InstGenerateRn(),
		InstMul(),
	}
	return InstMacro(code)
}

func MacroBatchPush(values ...Value) Inst {
	code := make(Code, len(values))
	for i, value := range values {
		code[i] = InstPush(value)
	}
	return InstMacro(code)
}

func MacroBatchStore(offset Addr, count uint64) Inst {
	code := make(Code, count)
	for i := range code {
		code[i] = InstStore(offset + Addr(i))
	}
	return InstMacro(code)
}

func MacroBatchLoad(offset Addr, count uint64) Inst {
	code := make(Code, count)
	for i := range code {
		code[i] = InstLoad(offset + Addr(count-1-uint64(i)))
	}
	return InstMacro(code)
}

func MacroSwap() Inst {
	// ...x, y]
	// ...y, x]
	code := Code{
		InstStore(0), // ...x]
		InstStore(1), // ...]
		InstLoad(0),  // ...y]
		InstLoad(1),  // ...y, x]
	}
	return InstMacro(code)
}

func MacroPropGen() Inst {
	// ...a, b]
	// ...p, g]
	code := Code{
		InstCopy(2),  // ...a, b, a, b]
		MacroAnd(),   // ...a, b, a & b]
		InstStore(0), // ...a, b]
		MacroXor(),   // ...a ^ b]
		InstLoad(0),  // ...a ^ b, a & b]
	}
	return InstMacro(code)
}

func MacroOpCLA() Inst {
	// ...p1, g1, p2, g2]
	// ...P, G]
	code := Code{
		InstStore(2),     // ...p1, g1, p2]
		MacroSwap(),      // ...p1, p2, g1]
		InstLoad(2),      // ...p1, p2, g1, g2]
		MacroSwap(),      // ...p1, p2, g2, g1]
		InstLoadStack(2), // ...p1, p2, g2, g1, p2]
		MacroAnd(),       // ...p1, p2, g2, g1 & p2]
		MacroOr(),        // ...p1, p2, g2 | (g1 & p2)]
		InstStore(0),     // ...p1, p2]
		MacroAnd(),       // ...p1 & p2]
		InstLoad(0),      // ...p1 & p2, g2 | (g1 & p2)]
	}
	return InstMacro(code)
}

func MacroBitwiseCOut(field algebra.Fp, offset Addr, bits uint) Inst {
	code := make(Code, 0)
	for i := uint(0); i < bits; i++ {
		c := Code{
			InstLoad(offset + Addr(2*i)),    // ...ai]
			InstLoad(offset + Addr(2*i+1)),  // ...ai, bi]
			MacroPropGen(),                  // ...pi, gi]
			InstStore(offset + Addr(2*i+1)), // ...pi]
			InstStore(offset + Addr(2*i)),   // ...]
		}
		code = append(code, c...)
	}

	for i := bits / 2; i > 0; i /= 2 {
		for j := uint(0); j < i; j++ {
			c := Code{
				InstLoad(offset + Addr(4*j)),    // ...pj]
				InstLoad(offset + Addr(4*j+1)),  // ...pj, gj]
				InstLoad(offset + Addr(4*j+2)),  // ...pj, gj, p{j+1}]
				InstLoad(offset + Addr(4*j+3)),  // ...pj, gj, p{j+1}, g{j+1}]
				MacroOpCLA(),                    // ...Pj, Gj]
				InstStore(offset + Addr(2*j+1)), // ...Pj]
				InstStore(offset + Addr(2*j)),   // ...]
			}
			code = append(code, c...)
		}
	}

	code = append(code, InstLoad(offset+Addr(1))) // ...G]

	return InstMacro(code)
}

func MacroCmp64(field algebra.Fp, offset Addr) Inst {
	str := "[debug] memory operations:\n"
	code := make(Code, 0, 320+4+7*64+2)
	for i := 0; i < 64; i++ {
		c := Code{
			InstLoad(offset + Addr(2*i)),    // ...ai]
			InstLoad(offset + Addr(2*i+1)),  // ...ai, bi]
			MacroNot(field),                 // ...ai, !bi]
			MacroPropGen(),                  // ...pi, gi]
			InstStore(offset + Addr(2*i+1)), // ...pi]
			InstStore(offset + Addr(2*i)),   // ...]
		}
		code = append(code, c...)
		str += fmt.Sprintf("loading a%v from %v\n", i, offset+Addr(2*i))
		str += fmt.Sprintf("loading b%v from %v\n", i, offset+Addr(2*i+1))
		str += fmt.Sprintf("storing g%v at %v\n", i, offset+Addr(2*i+1))
		str += fmt.Sprintf("storing p%v at %v\n", i, offset+Addr(2*i))
	}

	c := Code{
		InstLoad(offset),            // ...p0]
		InstLoad(offset + Addr(1)),  // ...p0, g0]
		MacroOr(),                   // ...p0 | g0]
		InstStore(offset + Addr(1)), // ...]
	}
	code = append(code, c...)
	str += fmt.Sprintf("loading p0 from %v\n", offset)
	str += fmt.Sprintf("loading g0 from %v\n", offset+Addr(1))
	str += fmt.Sprintf("storing g0 at %v\n", offset+Addr(1))

	for i := 32; i > 0; i /= 2 {
		for j := 0; j < i; j++ {
			c := Code{
				InstLoad(offset + Addr(4*j)),    // ...pj]
				InstLoad(offset + Addr(4*j+1)),  // ...pj, gj]
				InstLoad(offset + Addr(4*j+2)),  // ...pj, gj, p{j+1}]
				InstLoad(offset + Addr(4*j+3)),  // ...pj, gj, p{j+1}, g{j+1}]
				MacroOpCLA(),                    // ...Pj, Gj]
				InstStore(offset + Addr(2*j+1)), // ...Pj]
				InstStore(offset + Addr(2*j)),   // ...]
			}
			code = append(code, c...)
			str += fmt.Sprintf("loading p%v from %v\n", j, offset+Addr(4*j))
			str += fmt.Sprintf("loading g%v from %v\n", j, offset+Addr(4*j+1))
			str += fmt.Sprintf("loading p%v from %v\n", j+1, offset+Addr(4*j+2))
			str += fmt.Sprintf("loading g%v from %v\n", j+1, offset+Addr(4*j+3))
			str += fmt.Sprintf("storing G%v at %v\n", j, offset+Addr(2*j+1))
			str += fmt.Sprintf("storing P%v at %v\n", j, offset+Addr(2*j))
		}
	}

	c = Code{
		InstLoad(offset + Addr(1)), // ...G]
		MacroNot(field),            // ..!G]
	}
	code = append(code, c...)
	str += fmt.Sprintf("loading G from %v\n", offset+Addr(1))
	log.Println(str)

	return InstMacro(code)
}