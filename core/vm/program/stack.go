package program

import "errors"

var (
	ErrStackOverflow  = errors.New("stack overflow")
	ErrStackUnderflow = errors.New("stack underflow")
)

type Stack struct {
	top    int
	free   int
	empty  bool
	values []Value
}

func NewStack(limit int) Stack {
	return Stack{
		top:    0,
		free:   0,
		empty:  true,
		values: make([]Value, limit, limit),
	}
}

func (stack *Stack) Push(value Value) error {
	if stack.IsFull() {
		return ErrStackOverflow
	}

	stack.values[stack.free] = value
	stack.free = (stack.free + 1) % len(stack.values)
	stack.empty = false

	return nil
}

func (stack *Stack) Pop() (Value, error) {
	if stack.IsEmpty() {
		return nil, ErrStackUnderflow
	}

	value := stack.values[stack.top]
	stack.top = (stack.top + 1) % len(stack.values)
	stack.empty = stack.top == stack.free

	return value, nil
}

func (stack *Stack) IsFull() bool {
	return stack.top == stack.free && !stack.empty
}

func (stack *Stack) IsEmpty() bool {
	return stack.empty
}
