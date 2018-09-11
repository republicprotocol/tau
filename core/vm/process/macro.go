package process

import (
	"math/big"

	"github.com/republicprotocol/oro-go/core/vss/algebra"
)

func Not(field algebra.Fp) Inst {
	code := Code{
		InstNeg(),
		InstPush(NewValuePublic(field.NewInField(big.NewInt(1)))),
		InstAdd(),
	}
	return InstMacro(code)
}

func Or() Inst {
	code := Code{
		InstCopy(2),
		InstGenerateRn(),
		InstMul(),
		InstSub(),
		InstAdd(),
	}
	return InstMacro(code)
}

func And() Inst {
	code := Code{
		InstGenerateRn(),
		InstMul(),
	}
	return InstMacro(code)
}
