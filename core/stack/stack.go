package stack

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrStackOverflow is returned when an Element is pushed to the Stack when
	// the Stack is already full.
	ErrStackOverflow = errors.New("stack overflow")

	// ErrStackUnderflow is returned when an Element is popped from the Stack
	// when the Stack is already empty.
	ErrStackUnderflow = errors.New("stack underflow")
)

// An Element is pushed and popped from a Stack.
type Element interface {
}

// A Stack is a LIFO queue of Elements with zero runtime memory allocations.
type Stack struct {
	cap   int
	free  int
	elems []Element
}

// New returns a Stack with a limited capacity of Elements.
func New(cap int) Stack {
	if cap <= 0 {
		panic("stack capacity must be greater than zero")
	}
	return Stack{
		cap:   cap,
		free:  0,
		elems: make([]Element, cap, cap),
	}
}

// Push an Element to the Stack. If the Stack is already full, the Element is
// not pushed and an ErrStackOverflow is returned.
func (stack *Stack) Push(elem Element) error {
	if stack.IsFull() {
		return ErrStackOverflow
	}

	stack.elems[stack.free] = elem
	stack.free = stack.free + 1

	return nil
}

// Pop an Element from the Stack. If the Stack is already empty, the Element
// returned will be nil and an ErrStackUnderflow is returned.
func (stack *Stack) Pop() (Element, error) {
	if stack.IsEmpty() {
		return nil, ErrStackUnderflow
	}

	stack.free = stack.free - 1
	elem := stack.elems[stack.free]

	return elem, nil
}

// IsFull returns true when the Stack is full, otherwise it returns false.
// Pushing to a full Stack will result in a stack overflow.
func (stack *Stack) IsFull() bool {
	return stack.free == stack.cap
}

// IsEmpty returns true when the Stack is empty, otherwise it returns false.
// Popping from an empty Stack will result in a stack underflow.
func (stack *Stack) IsEmpty() bool {
	return stack.free == 0
}

func (stack *Stack) String() string {
	types := []string{}
	for _, elem := range stack.elems {
		types = append(types, fmt.Sprintf("%T", elem))
	}
	return fmt.Sprintf("[%v]", strings.Join(types, ", "))
}
