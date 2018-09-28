package task

// A Reducer consumes input Messages, uses them to modify its state, and
// produces output Messages in response.
type Reducer interface {

	// Reduce a new state from the current state and the Message. An output
	// Message can also be returned in response to the input Message. If the
	// output Message is nil, it will be ignored.
	Reduce(Message) Message
}

// ReduceFunc is a function that directly implements the Reducer interface.
// Although it has no explicit state of its own, a ReduceFunc can be a closure
// that captures state.
type ReduceFunc func(Message) Message

// Reduce implements the Reducer interface for ReduceFunc.
func (f ReduceFunc) Reduce(message Message) Message {
	return f(message)
}
