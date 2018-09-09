package process_test

import (
	"math/big"

	"github.com/republicprotocol/smpc-go/core/vss/algebra"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/process"
)

var _ = Describe("Stack", func() {
	buildValuePublic := func(v algebra.FpElement) ValuePublic {
		return ValuePublic{
			Value: v,
		}
	}

	buildEmptyStack := func(size int) Stack {
		return NewStack(size)
	}

	buildFullStack := func(size int, v Value) Stack {
		stack := NewStack(size)
		for i := 0; i < size; i++ {
			stack.Push(v)
		}
		return stack
	}

	Context("read operations", func() {
		It("is empty", func() {
			stack := buildEmptyStack(10)
			Expect(stack.IsEmpty()).To(BeTrue())
		})

		It("is full", func() {
			val := buildValuePublic(algebra.NewFpElement(big.NewInt(1), big.NewInt(7)))
			stack := buildFullStack(10, val)
			Expect(stack.IsFull()).To(BeTrue())
		})
	})

	Context("write operations", func() {
		It("push", func() {
			val := buildValuePublic(algebra.NewFpElement(big.NewInt(1), big.NewInt(7)))
			stack := buildEmptyStack(10)
			for i := 0; i < 10; i++ {
				err := stack.Push(val)
				Expect(err).To(BeNil())
			}
		})

		It("pop", func() {
			val := buildValuePublic(algebra.NewFpElement(big.NewInt(1), big.NewInt(7)))
			stack := buildFullStack(10, val)
			for i := 0; i < 10; i++ {
				val2, err := stack.Pop()
				Expect(err).To(BeNil())
				value := val2.(ValuePublic)
				Expect(value.Value.Eq(val.Value)).To(BeTrue())
			}
		})

		It("push and pop", func() {
			aVal := buildValuePublic(algebra.NewFpElement(big.NewInt(1), big.NewInt(7)))
			bVal := buildValuePublic(algebra.NewFpElement(big.NewInt(2), big.NewInt(7)))
			stack := buildEmptyStack(32)
			err := stack.Push(bVal)
			Expect(err).Should(BeNil())
			err = stack.Push(aVal)
			Expect(err).Should(BeNil())
			valA, err := stack.Pop()
			Expect(err).Should(BeNil())
			valueA := valA.(ValuePublic)
			Expect(valueA.Value.Eq(aVal.Value)).Should(BeTrue())
			valB, err := stack.Pop()
			Expect(err).Should(BeNil())
			valueB := valB.(ValuePublic)
			Expect(valueB.Value.Eq(bVal.Value)).Should(BeTrue())
		})
	})

	Context("erroneous operations", func() {
		It("stack underflow", func() {
			stack := buildEmptyStack(10)
			for i := 0; i < 10; i++ {
				_, err := stack.Pop()
				Expect(err).Should(Equal(ErrStackUnderflow))
			}
		})

		It("stack overflow", func() {
			val := buildValuePublic(algebra.NewFpElement(big.NewInt(1), big.NewInt(7)))
			stack := buildFullStack(10, val)
			for i := 0; i < 10; i++ {
				err := stack.Push(val)
				Expect(err).Should(Equal(ErrStackOverflow))
			}
		})
	})

})
