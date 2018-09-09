package process_test

import (
	"math/big"

	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/vm/process"
)

var _ = FDescribe("Values", func() {
	buildValuePublic := func(v algebra.FpElement) ValuePublic {
		return ValuePublic{
			Value: v,
		}
	}

	buildValuePrivate := func(s shamir.Share) ValuePrivate {
		return ValuePrivate{
			Share: s,
		}
	}

	buildValuePrivareRn := func(ρ, σ shamir.Share) ValuePrivateRn {
		return ValuePrivateRn{
			Rho:   ρ,
			Sigma: σ,
		}
	}

	buildShare := func(index uint64, v algebra.FpElement) shamir.Share {
		return shamir.Share{
			Index: index,
			Value: v,
		}
	}

	Context("building values", func() {
		It("public value", func() {
			Expect(buildValuePublic(algebra.NewFpElement(big.NewInt(1), big.NewInt(7)))).NotTo(BeNil())
		})

		It("private value", func() {
			Expect(buildValuePrivate(shamir.Share{})).NotTo(BeNil())
		})

		It("private rn value", func() {
			Expect(buildValuePrivareRn(shamir.Share{}, shamir.Share{})).NotTo(BeNil())
		})
	})

	Context("add operations", func() {
		It("lhs: public value & rhs: public value", func() {
			v1 := buildValuePublic(algebra.NewFpElement(big.NewInt(2), big.NewInt(8113765242226142771)))
			v2 := buildValuePublic(algebra.NewFpElement(big.NewInt(3), big.NewInt(8113765242226142771)))
			ret := v1.Add(v2)
			retVal := ret.(ValuePublic)
			Expect(retVal.Value.Eq(algebra.NewFpElement(big.NewInt(5), big.NewInt(8113765242226142771)))).To(BeTrue())
		})

		It("lhs: private value & rhs: public value", func() {
			v1 := buildValuePrivate(buildShare(1, algebra.NewFpElement(big.NewInt(2), big.NewInt(8113765242226142771))))
			v2 := buildValuePublic(algebra.NewFpElement(big.NewInt(3), big.NewInt(8113765242226142771)))
			ret := v1.Add(v2)
			retVal := ret.(ValuePrivate)
			Expect(retVal.Share.Value.Eq(algebra.NewFpElement(big.NewInt(5), big.NewInt(8113765242226142771)))).To(BeTrue())
		})

		It("lhs: public value & rhs: private value", func() {
			v1 := buildValuePublic(algebra.NewFpElement(big.NewInt(5), big.NewInt(8113765242226142771)))
			v2 := buildValuePrivate(buildShare(1, algebra.NewFpElement(big.NewInt(4), big.NewInt(8113765242226142771))))
			ret := v1.Add(v2)
			retVal := ret.(ValuePrivate)
			Expect(retVal.Share.Value.Eq(algebra.NewFpElement(big.NewInt(9), big.NewInt(8113765242226142771)))).To(BeTrue())
		})

		It("lhs: private value & rhs: private value", func() {
			v1 := buildValuePrivate(buildShare(1, algebra.NewFpElement(big.NewInt(6), big.NewInt(8113765242226142771))))
			v2 := buildValuePrivate(buildShare(1, algebra.NewFpElement(big.NewInt(7), big.NewInt(8113765242226142771))))
			ret := v1.Add(v2)
			retVal := ret.(ValuePrivate)
			Expect(retVal.Share.Value.Eq(algebra.NewFpElement(big.NewInt(13), big.NewInt(8113765242226142771)))).To(BeTrue())
		})
	})

})
