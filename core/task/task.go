package task

import (
	"github.com/republicprotocol/co-go"
)

type Reduce func(Message) Message

type Task interface {

	// Run the Task until its terminates. This blocks the current goroutine. The
	// done channel can be closed to signal to the Task that it should
	// terminate, however the Task can also terminate without the done channel
	// being closed.
	Run(done <-chan struct{})

	Send(Message)
}

type Children []Task

type task struct {
	io       IO
	reduce   Reduce
	children Children
}

func New(cap int, reduce Reduce, children ...Task) Task {
	return &task{NewIO(cap), reduce, children}
}

func (task *task) Run(done <-chan struct{}) {
	co.ParBegin(
		func() {
			for task.io.Flush(done, task.reduce) {
			}
		},
		func() {
			co.ParForAll(task.children, func(i int) {
				task.children[i].Run(done)
			})
		})
}

func (task *task) Send(message Message) {
	task.io.Channel().Send(message)
}
