package task

import (
	"log"
	"reflect"

	"github.com/republicprotocol/oro-go/core/buffer"
)

// IO is used to couple a `buffer.Reader` and `buffer.Writer`. It also provides
// buffered writing to the `buffer.Writer`. Generally, a Task will involve two
// IOs (1) to use internally to consume and produce messages for an external
// user, and (2) to expose to the external user so that the external user can
// consume and produce messages for the Task.
type IO struct {
	buf *buffer.Buffer
	r   buffer.Reader
	w   buffer.Writer
}

// NewIO returns a new IO. The `r` reader is used to consume `buffer.Messages`
// and the `w` writer is used to produce `buffer.Messages`. It is assumed that
// `r` has a respective `buffer.Writer` that is used to produce messages for the
// returned IO. Similarly, it is assumed that `w` has a respective
// `buffer.Reader` that is used to drain messages from the returned IO. Writing
// a message to the input of the IO will not result in that being produced on
// the IO output. To produce a message on the IO output, use the Send method. To
// consume a message from the IO input, use the Select function.
//
// In the following example, the read-only direction of `input` is passed to the
// IO (since the IO will read input messages from it), and the write-only
// direction of `output` is passed to the IO (since the IO will write output
// message to it).
//
// ```go
// buf := buffer.New(cap)
// input := buffer.NewReaderWriter(cap)
// output := buffer.NewReaderWriter(cap)
// io := NewIO(buf, input.Reader(), output.Writer())
// ```
func NewIO(buf *buffer.Buffer, r buffer.Reader, w buffer.Writer) IO {
	return IO{
		buf, r, w,
	}
}

// Send a `buffer.Message` to the output of the IO. The message will be buffered
// and will not be written until the message is flushed. The Select function can
// be used to flush an IO.
func (io IO) Send(message buffer.Message) {
	if !io.buf.Push(message) {
		log.Printf("[error] (io) buffer overflow")
	}
}

// Select an available read or write action from different IOs. A read action is
// available whenever an IO has message that can be read from its input
// `buffer.Reader`. A write action is available whenever an IO has a message
// buffered, waiting to be written to its output `buffer.Writer`. If a read
// action is selected, the IO will consume a `buffer.Message` from its input and
// invoke the `callback` function. If a write action is selected, the IO will
// flush a message to its output.
func Select(done <-chan struct{}, callback func(buffer.Message), ios ...IO) bool {
	cases := []reflect.SelectCase{
		// Read from the done channel
		reflect.SelectCase{
			Chan: reflect.ValueOf(done),
			Dir:  reflect.SelectRecv,
		},
	}

	for _, io := range ios {
		cases = append(cases,
			// Read from the output of the Runner
			reflect.SelectCase{
				Chan: reflect.ValueOf(io.r),
				Dir:  reflect.SelectRecv,
			},
			// Prepare to flush a Message to the input of the Runner
			reflect.SelectCase{
				Chan: reflect.ValueOf(io.buf.Peek()),
				Dir:  reflect.SelectRecv,
			},
		)
	}

	chosen, recv, recvOk := reflect.Select(cases)
	if chosen == 0 || !recvOk {
		return false
	}
	chosen--

	if chosen%2 == 0 {
		reflect.ValueOf(callback).Call([]reflect.Value{recv})
		return true
	}

	io := ios[chosen/2]
	cases = []reflect.SelectCase{
		// Read from the done channel
		reflect.SelectCase{
			Chan: reflect.ValueOf(done),
			Dir:  reflect.SelectRecv,
		},
		// Flush a Message to the input of the Runner
		reflect.SelectCase{
			Chan: reflect.ValueOf(io.w),
			Dir:  reflect.SelectSend,
			Send: recv,
		},
	}

	chosen, _, _ = reflect.Select(cases)
	if chosen == 0 {
		return false
	}

	if !io.buf.Pop() {
		log.Printf("[error] (io) buffer underflow")
	}
	return true
}
