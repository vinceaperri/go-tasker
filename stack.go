package tasker

import (
	"errors"
)

// string_stack is a generic slice with stack operations.
type string_stack struct {
	stack []string
	count int
}

// push adds an element onto the top of the stack.
func (s *string_stack) push(e string) {
	s.stack = append(s.stack, e)
	s.count++
}

// pop removes and returns the element on top of the stack. It returns an
// error is no element can be removed.
func (s *string_stack) pop() (string, error) {
	if s.count == 0 {
		return "", errors.New("Stack is empty")
	}
	s.count--
	e := s.stack[s.count]
	s.stack = s.stack[:s.count]
	return e, nil
}

// new_string_stack returns a new empty stack.
func new_string_stack() *string_stack {
	return &string_stack{make([]string, 0), 0}
}
