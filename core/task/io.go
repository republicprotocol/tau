package task

import (
	"fmt"
	"log"
	"reflect"

	"github.com/republicprotocol/oro-go/core/buffer"
)

type Channel interface {
	Send(message Message) bool
}

type channel struct {
	r   <-chan Message
	w   chan<- Message
	buf buffer.Buffer
}

func newChannel(r <-chan Message, w chan<- Message, buf buffer.Buffer) Channel {
	return &channel{r, w, buf}
}

func (ch *channel) Send(message Message) bool {
	return ch.buf.Enqueue(message)
}

type IO interface {
	Write(message Message)
	Flush(done <-chan struct{}, channels ...Channel) (Message, bool)
	Channel() Channel
}

type inputOutput struct {
	ch Channel

	r   <-chan Message
	w   chan<- Message
	buf buffer.Buffer
}

func NewIO(cap int) IO {
	r := make(chan Message, cap)
	w := make(chan Message, cap)

	return &inputOutput{
		newChannel(w, r, buffer.New(cap)),

		r,
		w,
		buffer.New(cap),
	}
}

func (io *inputOutput) Write(message Message) {
	if !io.buf.Enqueue(message) {
		log.Printf("[error] (io) buffer overflow")
	}
}

func (io *inputOutput) Flush(done <-chan struct{}, channels ...Channel) (Message, bool) {

	cases := []reflect.SelectCase{
		// Read from the done channel
		reflect.SelectCase{
			Chan: reflect.ValueOf(done),
			Dir:  reflect.SelectRecv,
		},

		// Read from own writer buffer
		reflect.SelectCase{
			Chan: reflect.ValueOf(io.buf.Peek()),
			Dir:  reflect.SelectRecv,
		},

		// Read from own reader
		reflect.SelectCase{
			Chan: reflect.ValueOf(io.r),
			Dir:  reflect.SelectRecv,
		},
	}

	for _, ch := range channels {
		if _, ok := ch.(*channel); !ok {
			panic(fmt.Sprintf("unexpected channel type %T", ch))
		}
		cases = append(cases,
			// Read from channel writer buffer
			reflect.SelectCase{
				Chan: reflect.ValueOf(ch.(*channel).buf.Peek()),
				Dir:  reflect.SelectRecv,
			},

			// Read from channel reader
			reflect.SelectCase{
				Chan: reflect.ValueOf(ch.(*channel).r),
				Dir:  reflect.SelectRecv,
			},
		)
	}

	chosen, recv, recvOk := reflect.Select(cases)

	// Done was selected
	if chosen == 0 || !recvOk {
		return nil, false
	}

	// Reading from the io output buffer was selected, so an element is dequeued
	// from the buffer and flushed to the io output
	if chosen == 1 {
		select {
		case <-done:
			return nil, false

		case io.w <- recv.Interface().(Message):
			return nil, io.buf.Dequeue()
		}
	}
	// Reading from the io input was selected, so an element is read from the io
	// input and returned
	if chosen == 2 {
		return recv.Interface().(Message), recvOk
	}

	// An input buffer was selected from one of the channels, so an element is
	// dequeued from the input buffer and flushed to the channel input
	if (chosen-3)%2 == 0 {
		ch := channels[(chosen - 3)].(*channel)
		select {
		case <-done:
			return nil, false

		case ch.w <- recv.Interface().(Message):
			return nil, ch.buf.Dequeue()
		}
	}

	// An output was selected from one of the channels, so an element is read
	// from the channel output and returned
	return recv.Interface().(Message), recvOk
}

func (io *inputOutput) Channel() Channel {
	return io.ch
}
