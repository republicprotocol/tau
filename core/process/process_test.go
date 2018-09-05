package process_test

import (
	"math/big"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/process"
)

var _ = FDescribe("Processes", func() {
	simpleAdd := func(a, b *big.Int) *big.Int {
		aVal := ValuePublic{
			Int: a,
		}
		bVal := ValuePublic{
			Int: b,
		}
		stack := NewStack(32)
		err := stack.Push(bVal)
		Expect(err).Should(BeNil())
		err = stack.Push(aVal)
		Expect(err).Should(BeNil())
		mem := Memory{}
		id := [32]byte{1}
		code := []Inst{InstAdd{}}
		proc := New(id, stack, mem, code)
		proc.Exec()
		res, err := proc.Stack.Pop()
		Expect(err).Should(BeNil())
		ret := res.(ValuePublic)
		return ret.Int
	}

	It("simple public value add", func() {
		Expect(simpleAdd(big.NewInt(3), big.NewInt(2)).Cmp(big.NewInt(5))).Should(Equal(0))
	})

})
