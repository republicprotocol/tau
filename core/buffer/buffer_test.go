package buffer_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/tau/core/buffer"
)

var _ = Describe("Buffer", func() {

	buildEmptyBuffer := func(cap int) Buffer {
		return New(cap)
	}

	buildHalfFullBuffer := func(cap int) Buffer {
		buf := New(cap)
		for i := 0; i < cap/2; i++ {
			buf.Enqueue(struct{}{})
		}
		return buf
	}

	buildFullBuffer := func(cap int) Buffer {
		buf := New(cap)
		for i := 0; i < cap; i++ {
			buf.Enqueue(struct{}{})
		}
		return buf
	}

	table := []struct {
		cap int
	}{
		// Skip capacity 1 because a half buffer cannot be created
		{2}, {4}, {16}, {64}, {256}, {1024},
	}

	for _, entry := range table {
		entry := entry

		Context("when the buffer is full", func() {

			Context("when checking the buffer state", func() {
				It("should be full", func() {
					buffer := buildFullBuffer(entry.cap)
					Expect(buffer.IsFull()).To(BeTrue())
				})

				It("should not be empty", func() {
					buffer := buildFullBuffer(entry.cap)
					Expect(buffer.IsEmpty()).To(BeFalse())
				})
			})

			Context("when enqueueing an element", func() {
				It("should return false", func() {
					buffer := buildFullBuffer(entry.cap)
					ok := buffer.Enqueue(struct{}{})
					Expect(ok).To(BeFalse())
				})
			})

			Context("when dequeueing elements", func() {
				It("should return an element until it is empty", func() {
					buffer := buildFullBuffer(entry.cap)
					for i := 0; i < entry.cap; i++ {
						peeker := buffer.Peek()
						ok := buffer.Dequeue()
						Expect(<-peeker).ToNot(BeNil())
						Expect(ok).To(BeTrue())
					}
					Expect(buffer.IsEmpty()).To(BeTrue())
				})
			})

			Context("when peeking", func() {
				It("should return a non-nil peeker", func() {
					buffer := buildFullBuffer(entry.cap)
					Expect(buffer.Peek()).ToNot(BeNil())
				})
			})
		})

		Context("when the buffer is empty", func() {
			Context("when checking the buffer state", func() {
				It("should not be full ", func() {
					buffer := buildEmptyBuffer(entry.cap)
					Expect(buffer.IsFull()).To(BeFalse())
				})

				It("should be empty", func() {
					buffer := buildEmptyBuffer(entry.cap)
					Expect(buffer.IsEmpty()).To(BeTrue())
				})
			})

			Context("when enqueueing elements", func() {
				It("should store elements until it is full", func() {
					buffer := buildEmptyBuffer(entry.cap)
					for i := 0; i < entry.cap; i++ {
						ok := buffer.Enqueue(struct{}{})
						Expect(ok).To(BeTrue())
					}
					Expect(buffer.IsFull()).To(BeTrue())
				})
			})

			Context("when dequeueing an element", func() {
				It("should return false", func() {
					buffer := buildEmptyBuffer(entry.cap)
					ok := buffer.Dequeue()
					Expect(ok).To(BeFalse())
				})
			})

			Context("when enqueueing an dequeueing", func() {
				It("should dequeue all elements in the same order", func() {
					buffer := buildEmptyBuffer(entry.cap)

					for i := 0; i < entry.cap; i++ {
						ok := buffer.Enqueue(i)
						Expect(ok).To(BeTrue())
					}
					Expect(buffer.IsFull()).To(BeTrue())
					Expect(buffer.IsEmpty()).To(BeFalse())

					for i := 0; i < entry.cap; i++ {
						peeker := buffer.Peek()
						ok := buffer.Dequeue()
						Expect(<-peeker).To(Equal(i))
						Expect(ok).To(BeTrue())
					}
					Expect(buffer.IsFull()).To(BeFalse())
					Expect(buffer.IsEmpty()).To(BeTrue())
				})
			})

			Context("when peeking", func() {
				It("should return a nil peeker", func() {
					buffer := buildEmptyBuffer(entry.cap)
					Expect(buffer.Peek()).To(BeNil())
				})
			})
		})

		Context("when the buffer is half full", func() {
			Context("when checking the buffer state", func() {
				It("should not be full ", func() {
					buffer := buildHalfFullBuffer(entry.cap)
					Expect(buffer.IsFull()).To(BeFalse())
				})

				It("should not be empty", func() {
					buffer := buildHalfFullBuffer(entry.cap)
					Expect(buffer.IsEmpty()).To(BeFalse())
				})
			})

			Context("when enqueueing elements", func() {
				It("should store elements until it is full", func() {
					buffer := buildHalfFullBuffer(entry.cap)
					for i := 0; i < entry.cap/2; i++ {
						ok := buffer.Enqueue(struct{}{})
						Expect(ok).To(BeTrue())
					}
					Expect(buffer.IsFull()).To(BeTrue())
				})
			})

			Context("when dequeueing elements", func() {
				It("should return an element until it is empty", func() {
					buffer := buildHalfFullBuffer(entry.cap)
					for i := 0; i < entry.cap/2; i++ {
						peeker := buffer.Peek()
						ok := buffer.Dequeue()
						Expect(<-peeker).ToNot(BeNil())
						Expect(ok).To(BeTrue())
					}
					Expect(buffer.IsEmpty()).To(BeTrue())
				})
			})

			Context("when enqueueing an dequeueing", func() {
				It("should dequeue all elements in the same order", func() {
					buffer := buildHalfFullBuffer(entry.cap)

					for i := 0; i < entry.cap/2; i++ {
						ok := buffer.Enqueue(i)
						Expect(ok).To(BeTrue())
					}
					Expect(buffer.IsFull()).To(BeTrue())
					Expect(buffer.IsEmpty()).To(BeFalse())

					for i := 0; i < entry.cap/2; i++ {
						peeker := buffer.Peek()
						ok := buffer.Dequeue()
						if i >= entry.cap/2 {
							Expect(<-peeker).To(Equal(i))
						}
						Expect(ok).To(BeTrue())
					}
					Expect(buffer.IsFull()).To(BeFalse())
					Expect(buffer.IsEmpty()).To(BeFalse())
				})
			})

			Context("when peeking", func() {
				It("should return a non-nil peeker", func() {
					buffer := buildHalfFullBuffer(entry.cap)
					Expect(buffer.Peek()).ToNot(BeNil())
				})
			})
		})
	}

	Context("when building a buffer with zero capacity", func() {
		It("should panic", func() {
			Expect(func() { New(0) }).To(Panic())
		})
	})
})
