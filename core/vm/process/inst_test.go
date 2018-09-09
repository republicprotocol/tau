package process_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/process"
)

var _ = Describe("Instructions", func() {
	buildInstPush := func(value Value) InstPush {
		return InstPush{
			Value: value,
		}
	}

	buildInstAdd := func() InstAdd {
		return InstAdd{}
	}

	// buildInstMul := func(retReady bool, retCh <-chan shamir.Share, ret shamir.Share) InstMul {
	// 	return InstMul{
	// 		RetReady: retReady,
	// 		RetCh:    retCh,
	// 		Ret:      ret,
	// 	}
	// }

	// buildInstRand := func(ρReady, σReady bool, ρCh, σCh <-chan shamir.Share, ρ, σ shamir.Share) InstRand {
	// 	return InstRand{
	// 		RhoReady: ρReady,
	// 		RhoCh:    ρCh,
	// 		Rho:      ρ,

	// 		SigmaReady: σReady,
	// 		SigmaCh:    σCh,
	// 		Sigma:      σ,
	// 	}
	// }
	// buildInstOpen := func(retReady bool, retCh <-chan *big.Int, ret *big.Int) InstOpen {
	// 	return InstOpen{
	// 		RetReady: retReady,
	// 		RetCh:    retCh,
	// 		Ret:      ret,
	// 	}
	// }

	Context("building push instructions", func() {
		It("nil push instruction", func() {
			Expect(buildInstPush(nil)).NotTo(BeNil())
		})

		It("public value push instruction", func() {
			Expect(buildInstPush(ValuePublic{})).NotTo(BeNil())
		})

		It("private value push instruction", func() {
			Expect(buildInstPush(ValuePrivate{})).NotTo(BeNil())
		})

		It("private rn push instruction", func() {
			Expect(buildInstPush(ValuePrivateRn{})).NotTo(BeNil())
		})
	})

	Context("building add instruction", func() {
		It("building add instruction", func() {
			Expect(buildInstAdd()).NotTo(BeNil())
		})
	})

})
