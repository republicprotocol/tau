package pedersen

import (
	"github.com/republicprotocol/smpc-go/core/vss/algebra"
)

type Pedersen struct {
	g algebra.FpElement
	h algebra.FpElement
}

func New(g, h algebra.FpElement) Pedersen {
	return Pedersen{g, h}
}

func (ped *Pedersen) Commit(s, t algebra.FpElement) algebra.FpElement {
	l := ped.g.Exp(s.AsField(ped.g.Field()))
	r := ped.h.Exp(t.AsField(ped.h.Field()))
	return l.Mul(r)
}

func (ped *Pedersen) Verify(s, t, commitment algebra.FpElement) bool {
	return ped.Commit(s, t).Eq(commitment)
}
