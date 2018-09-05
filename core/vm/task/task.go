package task

// A Task consumes inputs from an IO, and produces outputs to the IO. The Run
// method is typically called from within a background goroutine.
type Task interface {

	// IO returns the input/output refernce used to write/read messages to the
	// Task. This is not the same as the IO used internally.
	IO() IO

	// Run the Task until termination on the current goroutine. The done channel
	// is closed when the Task should terminate, however the Task can terminate
	// without the done channel being closed.
	Run(done <-chan struct{})
}
