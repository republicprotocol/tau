package buffer

// A Message is a unit of data stored in the Buffer.
type Message interface {

	// IsMessage is a marker method. A programmer must explicitly mark a type as
	// a Message by implementing this method. It does nothing, but types from
	// being erroneously used as a Message.
	IsMessage()
}

// A Reader is a read-only channel of Messages.
type Reader (<-chan Message)

// A Writer is a write-only channel of Messages.
type Writer (chan<- Message)

// A ReaderWriter is a bi-directional channel of Messages. It is recommended to
// typecast a ReaderWriter down into a Reader or a Writer before using it.
type ReaderWriter (chan Message)

// Reader returns the read-only direction of a ReaderWriter.
func (rw ReaderWriter) Reader() Reader {
	return (chan Message)(rw)
}

// Writer returns the write-only direction of a ReaderWriter.
func (rw ReaderWriter) Writer() Writer {
	return (chan Message)(rw)
}

// NewReaderWriter returns an asynchronous ReaderWriter with a capacity of
// `cap`. Using a `cap` of zero will return a synchronous ReaderWriter.
func NewReaderWriter(cap int) ReaderWriter {
	return make(ReaderWriter, cap)
}

// A Buffer is a FIFO queue of Messages with zero runtime allocations. It has a
// limited capacity and will not accept Messages while it is full.
type Buffer struct {
	top      int
	free     int
	empty    bool
	messages []Message
}

// New returns a new Buffer with a capacity of `cap`. This function will panic
// if `cap` is less than, or equal, to zero.
func New(cap int) Buffer {
	if cap <= 0 {
		panic("buffer capacity must be greater than zero")
	}
	return Buffer{
		top:      0,
		free:     0,
		empty:    true,
		messages: make([]Message, cap, cap),
	}
}

// Push a Message onto the end of the Buffer. Returns true if the Buffer
// successfully pushed the Message onto its internal queue, otherwise it returns
// false. The Buffer will fail to push a Message when its internal queue is
// full.
func (buf *Buffer) Push(message Message) bool {
	if buf.IsFull() {
		return false
	}

	buf.messages[buf.free] = message
	buf.free = (buf.free + 1) % len(buf.messages)
	buf.empty = false

	return true
}

// Pop a Message from the front of the Buffer. Returns true if the Buffer
// successfully popped a Message from its internal queue, otherwise it returns
// false. The Buffer will fail to pop a Message when its internal queue is
// empty.
func (buf *Buffer) Pop() bool {
	if buf.IsEmpty() {
		return false
	}

	buf.top = (buf.top + 1) % len(buf.messages)
	buf.empty = buf.top == buf.free

	return true
}

// Peek clones the Message at the front of the Buffer and returns a Reader that
// will produce this Message. The Message is not popped from the Buffer.
func (buf *Buffer) Peek() Reader {
	if buf.IsEmpty() {
		return nil
	}

	peek := NewReaderWriter(1)
	peek <- buf.messages[buf.top]

	return peek.Reader()
}

// IsFull returns true if the Buffer is full, otherwise it return false. If the
// Buffer is full, a call to `Buffer.Push` will fail, otherwise it will succeed.
func (buf *Buffer) IsFull() bool {
	return buf.top == buf.free && !buf.empty
}

// IsEmpty returns true if the Buffer is full, otherwise it return false. If the
// Buffer is empty, a call to `Buffer.Pop` will fail, otherwise it will succeed.
func (buf *Buffer) IsEmpty() bool {
	return buf.empty
}
