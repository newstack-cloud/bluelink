package function

import (
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// Call holds information for a function call
// in a call stack.
type Call struct {
	// FilePath is the file path of the source blueprint
	// where the function call is located.
	// This is especially useful for debugging projects with multiple
	// blueprints or in a multi-stage validation/deployment process
	// where the blueprint is one of many files that could have caused
	// an error.
	FilePath     string
	FunctionName string
	// Location is derived from the location of the function
	// call in the source blueprint that is captured in the schema
	// and substitution parsing process.
	Location *source.Meta
}

// Stack is an interface for a stack of function calls.
type Stack interface {
	// Push a new function call onto the stack.
	Push(call *Call)
	// Pop the top function call from the stack.
	Pop() *Call
	// Snapshot returns a snapshot of the current stack.
	Snapshot() []*Call
	// Clone returns a copy of the stack.
	Clone() Stack
}

type stackImpl struct {
	calls []*Call
	// A mutex is required as plugin functions across
	// multiple coroutines can share the same call stack.
	mu sync.Mutex
}

// NewStack creates a new instance of a function call stack.
func NewStack() Stack {
	return &stackImpl{}
}

func (s *stackImpl) Push(call *Call) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = append(s.calls, call)
}

func (s *stackImpl) Pop() *Call {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.calls) == 0 {
		return nil
	}
	call := s.calls[len(s.calls)-1]
	s.calls = s.calls[:len(s.calls)-1]
	return call
}

func (s *stackImpl) Snapshot() []*Call {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := make([]*Call, len(s.calls))
	// Reverse the backing slice so the first call is at the top of the stack,
	// stack traces in errors will be printed in the order of the snapshot,
	// from top to bottom.
	for i := len(s.calls) - 1; i >= 0; i -= 1 {
		snapshot[len(s.calls)-1-i] = s.calls[i]
	}
	return snapshot
}

func (s *stackImpl) Clone() Stack {
	s.mu.Lock()
	defer s.mu.Unlock()

	clone := &stackImpl{
		calls: make([]*Call, len(s.calls)),
	}
	for i, call := range s.calls {
		clone.calls[i] = &Call{
			FilePath:     call.FilePath,
			FunctionName: call.FunctionName,
			Location:     call.Location,
		}
	}
	return clone
}
