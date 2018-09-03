package buffer

type Message interface {
	IsMessage()
}

type Buffer struct {
	top      int
	free     int
	empty    bool
	messages []Message
}

func New(cap int) Buffer {
	return Buffer{
		top:      0,
		free:     0,
		empty:    true,
		messages: make([]Message, cap, cap),
	}
}

func (buffer *Buffer) Push(message Message) bool {
	if buffer.IsFull() {
		return false
	}

	buffer.messages[buffer.free] = message
	buffer.free = (buffer.free + 1) % len(buffer.messages)
	buffer.empty = false

	return true
}

func (buffer *Buffer) Pop() bool {
	if buffer.IsEmpty() {
		return false
	}

	buffer.top = (buffer.top + 1) % len(buffer.messages)
	buffer.empty = buffer.top == buffer.free

	return true
}

func (buffer *Buffer) Peek() <-chan Message {
	if buffer.IsEmpty() {
		return nil
	}

	peek := make(chan Message, 1)
	peek <- buffer.messages[buffer.top]

	return peek
}

func (buffer *Buffer) IsFull() bool {
	return buffer.top == buffer.free && !buffer.empty
}

func (buffer *Buffer) IsEmpty() bool {
	return buffer.empty
}
