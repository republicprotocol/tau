package task

type Task interface {
	Channel() Channel

	// Run the Task until its terminates. This blocks the current goroutine. The
	// done channel can be closed to signal to the Task that it should
	// terminate, however the Task can also terminate without the done channel
	// being closed.
	Run(done <-chan struct{})
}
