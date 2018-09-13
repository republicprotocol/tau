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
	Flush(done <-chan struct{}, callback func(Message) Message, channels ...Channel) bool
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

func (io *inputOutput) Flush(done <-chan struct{}, callback func(Message) Message, channels ...Channel) bool {

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
		return false
	}

	// Reading from the io output buffer was selected, so an element is dequeued
	// from the buffer and flushed to the io output
	if chosen == 1 {
		select {
		case <-done:
			return false

		case io.w <- recv.Interface().(Message):
			return io.buf.Dequeue()
		}
	}
	// Reading from the io input was selected, so an element is read from the io
	// input and returned
	if chosen == 2 {
		// TODO: Remove duplication!
		message := recv.Interface().(Message)
		if message = callback(message); message != nil {
			io.write(message)
		}
		return recvOk
	}

	// An input buffer was selected from one of the channels, so an element is
	// dequeued from the input buffer and flushed to the channel input
	if (chosen-3)%2 == 0 {
		ch := channels[(chosen-3)/2].(*channel)
		select {
		case <-done:
			return false

		case ch.w <- recv.Interface().(Message):
			return ch.buf.Dequeue()
		}
	}

	// TODO: Remove duplication!
	// An output was selected from one of the channels, so an element is read
	// from the channel output and returned
	message := recv.Interface().(Message)
	if message = callback(message); message != nil {
		io.write(message)
	}
	return recvOk
}

func (io *inputOutput) Channel() Channel {
	return io.ch
}

func (io *inputOutput) write(message Message) {
	if !io.buf.Enqueue(message) {
		log.Printf("[error] (io) buffer overflow")
	}
}
