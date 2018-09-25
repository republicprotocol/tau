package stack_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/oro-go/core/collection/stack"
)

var _ = Describe("Stack", func() {

	buildEmptyStack := func(cap int) Stack {
		return New(cap)
	}

	buildHalfFullStack := func(cap int) Stack {
		stack := New(cap)
		for i := 0; i < cap/2; i++ {
			stack.Push(struct{}{})
		}
		return stack
	}

	buildFullStack := func(cap int) Stack {
		stack := New(cap)
		for i := 0; i < cap; i++ {
			stack.Push(struct{}{})
		}
		return stack
	}

	table := []struct {
		cap int
	}{
		// Skip capacity 1 because a half buffer cannot be created
		{2}, {4}, {16}, {64}, {256}, {1024},
	}

	for _, entry := range table {
		entry := entry

		Context("when the stack is full", func() {

			Context("when checking the stack state", func() {
				It("should be full", func() {
					stack := buildFullStack(entry.cap)
					Expect(stack.IsFull()).To(BeTrue())
				})

				It("should not be empty", func() {
					stack := buildFullStack(entry.cap)
					Expect(stack.IsEmpty()).To(BeFalse())
				})
			})

			Context("when pushing an element", func() {
				It("should return a stack overflow", func() {
					stack := buildFullStack(entry.cap)
					err := stack.Push(struct{}{})
					Expect(err).To(Equal(ErrStackOverflow))
				})
			})

			Context("when popping elements", func() {
				It("should return an element until it is empty", func() {
					stack := buildFullStack(entry.cap)
					for i := 0; i < entry.cap; i++ {
						elem, err := stack.Pop()
						Expect(err).To(BeNil())
						Expect(elem).ToNot(BeNil())
					}
					Expect(stack.IsEmpty()).To(BeTrue())
				})
			})
		})

		Context("when the stack is empty", func() {
			Context("when checking the stack state", func() {
				It("should not be full ", func() {
					stack := buildEmptyStack(entry.cap)
					Expect(stack.IsFull()).To(BeFalse())
				})

				It("should be empty", func() {
					stack := buildEmptyStack(entry.cap)
					Expect(stack.IsEmpty()).To(BeTrue())
				})
			})

			Context("when pushing elements", func() {
				It("should store elements until it is full", func() {
					stack := buildEmptyStack(entry.cap)
					for i := 0; i < entry.cap; i++ {
						err := stack.Push(struct{}{})
						Expect(err).To(BeNil())
					}
					Expect(stack.IsFull()).To(BeTrue())
				})
			})

			Context("when popping an element", func() {
				It("should return a stack underflow", func() {
					stack := buildEmptyStack(entry.cap)
					elem, err := stack.Pop()
					Expect(err).To(Equal(ErrStackUnderflow))
					Expect(elem).To(BeNil())
				})
			})

			Context("when pushing an popping", func() {
				It("should pop all elements in reverse order", func() {
					stack := buildEmptyStack(entry.cap)

					for i := 0; i < entry.cap; i++ {
						err := stack.Push(i)
						Expect(err).To(BeNil())
					}
					Expect(stack.IsFull()).To(BeTrue())
					Expect(stack.IsEmpty()).To(BeFalse())

					for i := 0; i < entry.cap; i++ {
						elem, err := stack.Pop()
						Expect(err).To(BeNil())
						Expect(elem).To(Equal(entry.cap - (i + 1)))
					}
					Expect(stack.IsFull()).To(BeFalse())
					Expect(stack.IsEmpty()).To(BeTrue())
				})
			})
		})

		Context("when the stack is half full", func() {
			Context("when checking the stack state", func() {
				It("should not be full ", func() {
					stack := buildHalfFullStack(entry.cap)
					Expect(stack.IsFull()).To(BeFalse())
				})

				It("should not be empty", func() {
					stack := buildHalfFullStack(entry.cap)
					Expect(stack.IsEmpty()).To(BeFalse())
				})
			})

			Context("when pushing elements", func() {
				It("should store elements until it is full", func() {
					stack := buildHalfFullStack(entry.cap)
					for i := 0; i < entry.cap/2; i++ {
						err := stack.Push(struct{}{})
						Expect(err).To(BeNil())
					}
					Expect(stack.IsFull()).To(BeTrue())
				})
			})

			Context("when popping elements", func() {
				It("should return an element until it is empty", func() {
					stack := buildHalfFullStack(entry.cap)
					for i := 0; i < entry.cap/2; i++ {
						elem, err := stack.Pop()
						Expect(err).To(BeNil())
						Expect(elem).ToNot(BeNil())
					}
					Expect(stack.IsEmpty()).To(BeTrue())
				})
			})

			Context("when pushing an popping", func() {
				It("should pop all elements in reverse order", func() {
					stack := buildHalfFullStack(entry.cap)

					for i := 0; i < entry.cap/2; i++ {
						err := stack.Push(i)
						Expect(err).To(BeNil())
					}
					Expect(stack.IsFull()).To(BeTrue())
					Expect(stack.IsEmpty()).To(BeFalse())

					for i := 0; i < entry.cap/2; i++ {
						elem, err := stack.Pop()
						Expect(err).To(BeNil())
						Expect(elem).To(Equal(entry.cap/2 - (i + 1)))
					}
					Expect(stack.IsFull()).To(BeFalse())
					Expect(stack.IsEmpty()).To(BeFalse())
				})
			})
		})

	}

	Context("when building a stack with zero capacity", func() {
		It("should panic", func() {
			Expect(func() { New(0) }).To(Panic())
		})
	})
})
