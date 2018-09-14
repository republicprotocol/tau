package task

import (
	"github.com/republicprotocol/co-go"
)

type Reducer interface {
	Reduce(Message) Message
}

type reducer struct {
	reduce func(Message) Message
}

func NewReducer(reduce func(Message) Message) Reducer {
	return &reducer{reduce}
}

func (reducer *reducer) Reduce(message Message) Message {
	return reducer.reduce(message)
}

type Task interface {

	// Run the Task until its terminates. This blocks the current goroutine. The
	// done channel can be closed to signal to the Task that it should
	// terminate, however the Task can also terminate without the done channel
	// being closed.
	Run(done <-chan struct{})

	Send(Message)

	IO() IO
}

type Children []Task

type task struct {
	io       IO
	reducer  Reducer
	children Children
}

func New(io IO, reducer Reducer, children ...Task) Task {
	return &task{io, reducer, children}
}

func (task *task) Run(done <-chan struct{}) {
	co.ParBegin(
		func() {
			for task.io.Flush(done, task.reducer, task.children) {
			}
		},
		func() {
			co.ParForAll(task.children, func(i int) {
				task.children[i].Run(done)
			})
		})
}

func (task *task) Send(message Message) {
	task.io.WriteIn(message)
}

func (task *task) IO() IO {
	return task.io
}
