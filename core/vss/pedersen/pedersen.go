package pedersen

import (
	"errors"
	"math/big"
)

var (
	// ErrNilArguments can be returned by Verify and signifies that one or more
	// of the arguments given to Verify were nil.
	ErrNilArguments = errors.New("nil arguments")

	// ErrUnacceptableCommitment can be returned by Verify and signifies that
	// the commitment was not accepted.
	ErrUnacceptableCommitment = errors.New("unacceptable commitment")
)

// A Pedersen struct provides functionality for creating pedersen commitments.
// The variables are named as they appear in the paper; q (a prime) is the order
// of the subgroup of the multiplicative group Zp (where p is a prime such that
// q divides p - 1), and g and h are elements of this subgroup such that
// log_g(h) is not known.
type Pedersen struct {
	p *big.Int
	q *big.Int
	g *big.Int
	h *big.Int
}

// New returns a new Pedersen struct that is used to create pedersen
// commitments. It performs a basic divisibility check that is required for the
// primes p and q.
func New(p, q, g, h *big.Int) (ped Pedersen, err error) {
	if p == nil || q == nil || g == nil || h == nil {
		err = errors.New("nil arguments")
		return
	}
	if big.NewInt(0).Mod(p.Sub(p, big.NewInt(1)), q).Cmp(big.NewInt(0)) != 0 {
		err = errors.New("q does not divide p - 1")
		return
	}
	ped = Pedersen{
		p,
		q,
		g,
		h,
	}
	return
}

// GroupOrder returns q, the order of the subgroup of the multiplicative group Zp.
func (ped *Pedersen) GroupOrder() *big.Int {
	return ped.p
}

// SubgroupOrder returns q, the order of the subgroup of the multiplicative group Zp.
func (ped *Pedersen) SubgroupOrder() *big.Int {
	return ped.q
}

// Commit takes a secret, s, and a randomising, t, number and produces a
// pedersen commitment (g^s)(h^t). If either of the arguments are nil, the
// function will return nil.
func (ped *Pedersen) Commit(s, t *big.Int) *big.Int {
	if s == nil || t == nil {
		return nil
	}
	l := big.NewInt(0).Exp(ped.g, s, ped.p)
	r := big.NewInt(0).Exp(ped.h, t, ped.p)
	l.Mul(l, r)
	l.Mod(l, ped.p)
	return l
}

// Verify checks whether a given commitment correctly corresponds to the secret
// s and randomising number t. Verify returns an error, where an ErrNilArguments
// error corresponds to the case that one or more of the arguments are nil,
// ErrUnacceptableCommitment corresponds to the case that s and t do not
// correctly correspond to the commitment, and a nil error means that the
// commitment was accepted.
func (ped *Pedersen) Verify(s, t, commitment *big.Int) error {
	if s == nil || t == nil || commitment == nil {
		return ErrNilArguments
	} else if ped.Commit(s, t).Cmp(commitment) != 0 {
		return ErrUnacceptableCommitment
	}
	return nil
}
