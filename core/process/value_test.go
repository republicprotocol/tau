package process_test

import (
	"math/big"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/process"
	"github.com/republicprotocol/smpc-go/core/vss/shamir"
)

var _ = Describe("Values", func() {
	buildValuePublic := func(i *big.Int) ValuePublic {
		return ValuePublic{
			Int: i,
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

	generateShare := func(index uint64, i *big.Int) shamir.Share {
		return shamir.Share{
			Index: index,
			Value: i,
		}
	}

	Context("building values", func() {
		It("public value", func() {
			Expect(buildValuePublic(big.NewInt(0))).NotTo(BeNil())
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
			v1 := buildValuePublic(big.NewInt(2))
			v2 := buildValuePublic(big.NewInt(3))
			ret := v1.Add(v2)
			retVal := ret.(ValuePublic)
			Expect(retVal.Int.Cmp(big.NewInt(5))).To(Equal(0))
		})

		It("lhs: private value & rhs: public value", func() {
			v1 := buildValuePrivate(generateShare(1, big.NewInt(5)))
			v2 := buildValuePublic(big.NewInt(2))
			ret := v1.Add(v2)
			retVal := ret.(ValuePrivate)
			Expect(retVal.Share.Value.Cmp(big.NewInt(7))).To(Equal(0))
		})

		It("lhs: public value & rhs: private value", func() {
			v1 := buildValuePublic(big.NewInt(3))
			v2 := buildValuePrivate(generateShare(1, big.NewInt(5)))
			ret := v1.Add(v2)
			retVal := ret.(ValuePrivate)
			Expect(retVal.Share.Value.Cmp(big.NewInt(8))).To(Equal(0))
		})

		It("lhs: private value & rhs: private value", func() {
			v1 := buildValuePrivate(generateShare(1, big.NewInt(6)))
			v2 := buildValuePrivate(generateShare(1, big.NewInt(7)))
			ret := v1.Add(v2)
			retVal := ret.(ValuePrivate)
			Expect(retVal.Share.Value.Cmp(big.NewInt(13))).To(Equal(0))
		})
	})

})
