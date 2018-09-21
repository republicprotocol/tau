package proc_test

import (
	"math/big"

	"github.com/republicprotocol/oro-go/core/stack"
	"github.com/republicprotocol/oro-go/core/vss/algebra"
	"github.com/republicprotocol/oro-go/core/vss/shamir"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/vm/process"
)

var _ = FDescribe("Processes", func() {

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

	buildShare := func(index uint64, v algebra.FpElement) shamir.Share {
		return shamir.New(index, v)
	}

	add := func(a, b Value) Value {
		stack := stack.New(32)
		err := stack.Push(b)
		Expect(err).Should(BeNil())
		err = stack.Push(a)
		Expect(err).Should(BeNil())
		mem := Memory{}
		id := [32]byte{1}
		code := Code{InstAdd()}
		proc := New(id, stack, mem, code)
		proc.Exec()
		res, err := proc.Stack.Pop()
		Expect(err).Should(BeNil())
		return res
	}

	// mul := func(a, b Value) Value {
	// 	stack := NewStack(32)
	// 	err := stack.Push(b)
	// 	Expect(err).Should(BeNil())
	// 	err = stack.Push(a)
	// 	Expect(err).Should(BeNil())
	// 	mem := Memory{}
	// 	id := [32]byte{1}
	// 	code := []Inst{InstMul{}}
	// 	proc := New(id, stack, mem, code)
	// 	proc.Exec()
	// 	res, err := proc.Stack.Pop()
	// 	Expect(err).Should(BeNil())
	// 	return res
	// }

	Context("when adding two public variables", func() {
		It("should return the correct sum", func() {
			v1 := buildValuePublic(algebra.NewFpElement(big.NewInt(2), big.NewInt(8113765242226142771)))
			v2 := buildValuePublic(algebra.NewFpElement(big.NewInt(3), big.NewInt(8113765242226142771)))
			sum := algebra.NewFpElement(big.NewInt(5), big.NewInt(8113765242226142771))
			result := add(v1, v2).(ValuePublic)
			Expect(result.Value.Eq(sum)).Should(BeTrue())
		})
	})

	Context("when adding two private variables", func() {
		It("should return the correct share", func() {
			v1 := buildValuePrivate(buildShare(1, algebra.NewFpElement(big.NewInt(2), big.NewInt(8113765242226142771))))
			v2 := buildValuePrivate(buildShare(1, algebra.NewFpElement(big.NewInt(3), big.NewInt(8113765242226142771))))
			sum := algebra.NewFpElement(big.NewInt(5), big.NewInt(8113765242226142771))
			result := add(v1, v2).(ValuePrivate)
			Expect(result.Share.Value.Eq(sum)).Should(BeTrue())
		})

		It("should fail when the indices mismatch", func() {
			v1 := buildValuePrivate(buildShare(1, algebra.NewFpElement(big.NewInt(2), big.NewInt(8113765242226142771))))
			v2 := buildValuePrivate(buildShare(2, algebra.NewFpElement(big.NewInt(3), big.NewInt(8113765242226142771))))
			Expect(add(v1, v2).(ValuePrivate)).To(Panic())
		})
	})

	// Context("when multiplying two private variables", func() {
	// 	It("should return the correct share", func() {
	// 		v1 := buildValuePrivate(buildShare(1, algebra.NewFpElement(big.NewInt(2), big.NewInt(8113765242226142771))))
	// 		v2 := buildValuePrivate(buildShare(1, algebra.NewFpElement(big.NewInt(3), big.NewInt(8113765242226142771))))
	// 		product := algebra.NewFpElement(big.NewInt(6), big.NewInt(8113765242226142771))
	// 		result := mul(v1, v2).(ValuePrivate)
	// 		Expect(result.Share.Value.Eq(product)).Should(BeTrue())
	// 	})
	// })
})
