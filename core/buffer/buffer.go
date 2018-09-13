package buffer

// An Element is enqueued and dequeued from a Buffer.
type Element interface {
}

// Peeker is a read-only channel that holds an Element. It is returned from a
// Buffer to peek at the Element that will be dequeued next.
type Peeker (<-chan Element)

// A Buffer is a FIFO queue of Elements with a limitied capacity. It will not
// Enqueue Elements when it is full, and will not Dequeue Elements when it is
// empty.
type Buffer interface {
	Peek() Peeker
	Enqueue(Element) bool
	Dequeue() bool
	IsFull() bool
	IsEmpty() bool
}

type buffer struct {
	top   int
	free  int
	empty bool
	elems []Element
}

// New returns a new Buffer with a limited capacity and zero runtime
// allocations. This function will panic if the capacity is less than, or equal,
// to zero.
func New(cap int) Buffer {
	if cap <= 0 {
		panic("buffer capacity must be greater than zero")
	}
	return &buffer{
		top:   0,
		free:  0,
		empty: true,
		elems: make([]Element, cap, cap),
	}
}

// Peek clones the Element at the front of the Buffer and returns a read-only
// channel that will produce this Element. The Element is not popped from the
// Buffer.
func (buf *buffer) Peek() Peeker {
	if buf.IsEmpty() {
		return nil
	}

	peek := make(chan Element, 1)
	peek <- buf.elems[buf.top]

	return peek
}

// Enqueue an Element onto the end of the Buffer. Returns true if the Buffer
// successfully enqueued the Element onto its internal queue, otherwise it
// returns false. The Buffer will fail to enqueue an Element when its internal
// queue is full.
func (buf *buffer) Enqueue(message Element) bool {
	if buf.IsFull() {
		return false
	}

	buf.elems[buf.free] = message
	buf.free = (buf.free + 1) % len(buf.elems)
	buf.empty = false

	return true
}

// Dequeue an Element from the front of the Buffer. Returns true if the Buffer
// successfully dequeued an Element from its internal queue, otherwise it
// returns false. The Buffer will fail to dequeue an Element when its internal
// queue is empty.
func (buf *buffer) Dequeue() bool {
	if buf.IsEmpty() {
		return false
	}

	buf.top = (buf.top + 1) % len(buf.elems)
	buf.empty = buf.top == buf.free

	return true
}

// IsFull returns true if the Buffer is full, otherwise it return false. If the
// Buffer is full, a call to `Buffer.Enqueue` will fail, otherwise it will
// succeed.
func (buf *buffer) IsFull() bool {
	return buf.top == buf.free && !buf.empty
}

// IsEmpty returns true if the Buffer is full, otherwise it return false. If the
// Buffer is empty, a call to `Buffer.Dequeue` will fail, otherwise it will
// succeed.
func (buf *buffer) IsEmpty() bool {
	return buf.empty
}
