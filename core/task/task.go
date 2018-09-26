package task

import (
	"github.com/republicprotocol/co-go"
)

// A Reducer consumes input Messages, uses them to modify its state, and
// produces output Messages in response.
type Reducer interface {

	// Reduce a new state from the current state and the Message. An output
	// Message can also be returned in response to the input Message. If the
	// output Message is nil, it will be ignored.
	Reduce(Message) Message
}

type reducer struct {
	reduce func(Message) Message
}

// NewReducer returns a Reducer that uses a higher-order function as the reduce
// function. The Reducer has no state of its own, however the higher-order
// function can capture state if it is a closure.
func NewReducer(reduce func(Message) Message) Reducer {
	return &reducer{reduce}
}

func (reducer *reducer) Reduce(message Message) Message {
	return reducer.reduce(message)
}

// A Task is an independently executing actor. It can only communicate with
// other Task, and can only do so by consuming and producing Messages. A Task
// can receive Messages from its parent Task, and can send Messages to its
// children Tasks. All Messages output by a Task are returned to its parent
// Task.
type Task interface {

	// Run the Task until its terminates. The done channel can be closed to
	// signal to the Task that it should terminate, however the Task can also
	// terminate without the done channel being closed. Running a Task will
	// drive all input/output with its parent and children. This blocks the
	// current goroutine.
	Run(done <-chan struct{})

	// Send a Message to the Task. Sending a Message to a Task should only be
	// done by the parent Task. This will never block.
	Send(Message)

	// IO returns the IO object used by the Task to handle input/output with its
	// parent.
	IO() IO
}

// Tasks is a slice.
type Tasks []Task

// Children is a slice used to store the children of a Task.
type Children []Task

type task struct {
	io       IO
	reducer  Reducer
	children Children
}

// New returns a Task that uses a Reducer to handle Messages sent by its parent
// Task, and received as responses from its children. All Messages returned from
// the Reducer will be output to the parent. The Task will use an IO object to
// drive the input/output of Messages to/from its parent and children.
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
