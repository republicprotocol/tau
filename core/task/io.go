package task

import (
	"reflect"

	"github.com/republicprotocol/oro-go/core/buffer"
)

type IO interface {
	Flush(done <-chan struct{}, reducer Reducer, children Children) bool

	WriteIn(message Message) bool
	WriteOut(message Message) bool

	InputBuffer() buffer.Buffer
	InputWriter() chan<- Message

	OutputBuffer() buffer.Buffer
	OutputWriter() chan<- Message
}

type inputOutput struct {
	ibuf buffer.Buffer
	r    chan Message

	obuf buffer.Buffer
	w    chan Message
}

func NewIO(cap int) IO {
	r := make(chan Message, cap)
	w := make(chan Message, cap)

	return &inputOutput{
		ibuf: buffer.New(cap),
		r:    r,

		obuf: buffer.New(cap),
		w:    w,
	}
}

func (io *inputOutput) Flush(done <-chan struct{}, reducer Reducer, children Children) bool {

	cases := []reflect.SelectCase{
		// Read from the done channel
		reflect.SelectCase{
			Chan: reflect.ValueOf(done),
			Dir:  reflect.SelectRecv,
		},

		// Read from own output buffer
		reflect.SelectCase{
			Chan: reflect.ValueOf(io.obuf.Peek()),
			Dir:  reflect.SelectRecv,
		},

		// Read from own reader
		reflect.SelectCase{
			Chan: reflect.ValueOf(io.r),
			Dir:  reflect.SelectRecv,
		},
	}

	for _, child := range children {
		cases = append(cases,
			// Read from child input buffer
			reflect.SelectCase{
				Chan: reflect.ValueOf(child.IO().InputBuffer().Peek()),
				Dir:  reflect.SelectRecv,
			},

			// Read from child writer
			reflect.SelectCase{
				Chan: reflect.ValueOf(child.IO().OutputWriter()),
				Dir:  reflect.SelectRecv,
			},
		)
	}

	chosen, recv, recvOk := reflect.Select(cases)

	// Done was selected
	if chosen == 0 || !recvOk {
		return false
	}

	// Select reading from own output buffer
	if chosen == 1 {
		select {
		case <-done:
			return false

		case io.w <- recv.Interface().(Message):
			return io.obuf.Dequeue()
		}
	}
	// Select reading from owner reader
	if chosen == 2 {
		if message := reducer.Reduce(recv.Interface().(Message)); message != nil {
			io.WriteOut(message)
		}
		return recvOk
	}

	// Select reading from one of the child input buffers
	if (chosen-3)%2 == 0 {
		child := children[(chosen-3)/2]
		select {
		case <-done:
			return false

		case child.IO().InputWriter() <- recv.Interface().(Message):
			return child.IO().InputBuffer().Dequeue()
		}
	}

	// Select reading from one of the child writers
	if message := reducer.Reduce(recv.Interface().(Message)); message != nil {
		io.WriteOut(message)
	}
	return recvOk
}

func (io *inputOutput) WriteIn(message Message) bool {
	return io.ibuf.Enqueue(message)
}

func (io *inputOutput) WriteOut(message Message) bool {
	return io.obuf.Enqueue(message)
}

func (io *inputOutput) InputBuffer() buffer.Buffer {
	return io.ibuf
}

func (io *inputOutput) InputWriter() chan<- Message {
	return io.r
}

func (io *inputOutput) OutputBuffer() buffer.Buffer {
	return io.obuf
}

func (io *inputOutput) OutputWriter() chan<- Message {
	return io.w
}
