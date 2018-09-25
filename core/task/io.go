package task

import (
	"log"
	"reflect"

	"github.com/republicprotocol/oro-go/core/collection/buffer"
)

type IO interface {
	Flush(done <-chan struct{}, reducer Reducer, children Children) bool

	WriteIn(message Message) bool
	WriteOut(message Message) bool

	InputBuffer() buffer.Buffer
	InputWriter() chan<- Message

	OutputBuffer() buffer.Buffer
	OutputReader() <-chan Message
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
				Chan: reflect.ValueOf(child.IO().OutputReader()),
				Dir:  reflect.SelectRecv,
			},
		)
	}

	chosen, recv, recvOk := reflect.Select(cases)

	// Done was selected
	if chosen == 0 || !recvOk {
		return false
	}

	if _, ok := recv.Interface().(Message); !ok {
		return true
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
		message := io.reduceMessage(reducer, recv.Interface().(Message))
		if message != nil {
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
	message := io.reduceMessage(reducer, recv.Interface().(Message))
	if message != nil {
		io.WriteOut(message)
	}
	return recvOk
}

func (io *inputOutput) WriteIn(message Message) bool {
	ok := io.ibuf.Enqueue(message)
	if !ok {
		log.Printf("[error] (io, write) buffer overflow")
	}
	return ok
}

func (io *inputOutput) WriteOut(message Message) bool {
	ok := io.obuf.Enqueue(message)
	if !ok {
		log.Printf("[error] (io, write) buffer overflow")
	}
	return ok
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

func (io *inputOutput) OutputReader() <-chan Message {
	return io.w
}

func (io *inputOutput) reduceMessage(reducer Reducer, message Message) Message {
	// log.Printf("[debug] reducing message %T", message)
	if messages, ok := message.(MessageBatch); ok {
		for i := 0; i < len(messages); i++ {
			if messages[i] != nil {
				messages[i] = io.reduceMessage(reducer, messages[i])
			}
			if messages[i] == nil {
				messages[i] = messages[len(messages)-1]
				messages = messages[:len(messages)-1]
				i--
			}
		}
		if len(messages) == 0 {
			return nil
		}
		return messages
	}
	return reducer.Reduce(message)
}
