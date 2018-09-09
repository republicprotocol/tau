package task

// A Task consumes inputs from an IO, and produces outputs to the IO. The Run
// method is typically called from within a background goroutine.
type Task interface {

	// IO returns the input/output reference used to write/read messages to the
	// Task. This IO can be used to send messages to the Task, and to flush
	// messages written to output by the Task.
	//
	// The following example shows how to use a Task IO to send messages to a
	// Task, and flush messages from a Task.
	//
	// ```go
	// io := task.IO()
	//
	// // Send a message to `task`
	// io.Send(message)
	//
	// // Flush messages from `task`
	// Select(
	//     done,
	//     func(message buffer.Message) {
	//	       log.Printf("message received = %T", message)
	//     },
	//     io,
	// )
	// ```
	IO() IO

	// Run the Task until its terminates. This blocks the current goroutine. The
	// done channel can be closed to signal to the Task that it should
	// terminate, however the Task can also terminate without the done channel
	// being closed.
	Run(done <-chan struct{})
}
