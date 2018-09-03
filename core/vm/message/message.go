package message

type Message interface {
	IsMessage()
}

type Buffer struct {
	top      int
	free     int
	empty    bool
	messages []Message
}

func NewBuffer(cap int) Buffer {
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

func (buffer *Buffer) Pop() (Message, bool) {
	if buffer.IsEmpty() {
		return nil, false
	}

	message := buffer.messages[buffer.top]
	buffer.top = (buffer.top + 1) % len(buffer.messages)
	buffer.empty = buffer.top == buffer.free

	return message, true
}

func (buffer *Buffer) Peek() (Message, bool) {
	if buffer.IsEmpty() {
		return nil, false
	}

	message := buffer.messages[buffer.top]

	return message, true
}

func (buffer *Buffer) IsFull() bool {
	return buffer.top == buffer.free && !buffer.empty
}

func (buffer *Buffer) IsEmpty() bool {
	return buffer.empty
}
