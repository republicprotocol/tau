package task

import (
	"fmt"
	"reflect"

	"github.com/republicprotocol/oro-go/core/buffer"
)

type Message interface {
	IsMessage()
}

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

type IO struct {
	ch Channel

	r   <-chan Message
	w   chan<- Message
	buf buffer.Buffer
}

func NewIO(cap int) IO {
	r := make(chan Message, cap)
	w := make(chan Message, cap)

	return IO{
		newChannel(w, r, buffer.New(cap)),

		r,
		w,
		buffer.New(cap),
	}
}

func (io *IO) Write(message Message) bool {
	return io.buf.Enqueue(message)
}

func (io *IO) Flush(done <-chan struct{}, channels ...Channel) (Message, bool) {

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
	if chosen == 0 || !recvOk {
		return nil, false
	}
	if chosen == 1 {
		select {
		case <-done:
			return nil, false

		case io.w <- recv.Interface().(Message):
			return nil, io.buf.Dequeue()
		}
	}
	if chosen == 2 {
		return recv.Interface().(Message), recvOk
	}

	if (chosen-3)%2 == 0 {
		ch := channels[(chosen - 3)].(*channel)
		select {
		case <-done:
			return nil, false

		case ch.w <- recv.Interface().(Message):
			return nil, ch.buf.Dequeue()
		}
	}
	return recv.Interface().(Message), recvOk
}

func (io *IO) Channel() Channel {
	return io.ch
}
