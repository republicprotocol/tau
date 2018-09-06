package pedersen

import (
	"github.com/republicprotocol/smpc-go/core/vss/algebra"
)

// A Pedersen struct encapsulates the information needed to create and verify
// pedersen commitments. A particular instance contains the two generators g and
// h, which are used to create the commitments.
type Pedersen struct {
	g           algebra.FpElement
	h           algebra.FpElement
	secretField algebra.Fp
}

// New creates a new Pedersen struct from the generators g and h. No checking is
// done to ensure that these are correctly chosen; g and h need to be generators
// of a subgroup of Zp, where this subgroup has order q. Furthermore, p and q
// need to be prime, and q needs to divide p - 1 (this ensures that a subgroup
// of order q indeed exists inside Zp).
func New(g, h algebra.FpElement, field algebra.Fp) Pedersen {
	return Pedersen{g, h, field}
}

func (ped *Pedersen) SecretField() algebra.Fp {
	return ped.secretField
}

// Commit creates a Pedersen commitment for the value s and using the
// randomising term t. The commitment is (g^s)(h^t), where g and h are
// determined by the Pedersen scheme. If s and t are not able to be cast up into
// Zp, then it will panic (indirectly through the panic that will occur in
// FpElemet.AsField).
func (ped *Pedersen) Commit(s, t algebra.FpElement) algebra.FpElement {
	l := ped.g.Exp(s.AsField(ped.g.Field()))
	r := ped.h.Exp(t.AsField(ped.h.Field()))
	return l.Mul(r)
}

// Verify checks whether values s and t correspond to the given commitment. It
// will return true if the correspondance is correct, and false otherwise.
func (ped *Pedersen) Verify(s, t, commitment algebra.FpElement) bool {
	return ped.Commit(s, t).Eq(commitment)
}
