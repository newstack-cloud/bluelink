package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type CoalesceFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&CoalesceFunctionTestSuite{})

func (s *CoalesceFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *CoalesceFunctionTestSuite) Test_gets_first_non_none_value(c *C) {
	coalesceFunc := NewCoalesceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "coalesce",
	})
	output, err := coalesceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{
					core.GetNoneMarker(),
					core.GetNoneMarker(),
					"First non-none value",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "First non-none value")
}

func (s *CoalesceFunctionTestSuite) Test_gets_first_non_none_value_allowing_for_empty_strings(c *C) {
	coalesceFunc := NewCoalesceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "coalesce",
	})
	output, err := coalesceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{
					"",
					core.GetNoneMarker(),
					core.GetNoneMarker(),
					"last non-none value",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// An empty string is treated as a valid non-none value for coalesce.
	c.Assert(output.ResponseData, Equals, "")
}

func (s *CoalesceFunctionTestSuite) Test_gets_first_non_none_value_allowing_for_empty_arrays(c *C) {
	coalesceFunc := NewCoalesceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "coalesce",
	})
	output, err := coalesceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{
					[]any{},
					core.GetNoneMarker(),
					core.GetNoneMarker(),
					"last non-none value",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// An empty array is treated as a valid non-none value for coalesce.
	c.Assert(output.ResponseData, DeepEquals, []any{})
}

func (s *CoalesceFunctionTestSuite) Test_gets_first_non_none_value_allowing_for_empty_maps(c *C) {
	coalesceFunc := NewCoalesceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "coalesce",
	})
	output, err := coalesceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{
					map[string]any{},
					core.GetNoneMarker(),
					core.GetNoneMarker(),
					"last non-none value",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	// An empty map is treated as a valid non-none value for coalesce.
	c.Assert(output.ResponseData, DeepEquals, map[string]any{})
}

func (s *CoalesceFunctionTestSuite) Test_gets_last_none_value_if_all_values_are_none(c *C) {
	coalesceFunc := NewCoalesceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "coalesce",
	})
	output, err := coalesceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{
					core.GetNoneMarker(),
					core.GetNoneMarker(),
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, core.GetNoneMarker())
}

func (s *CoalesceFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	coalesceFunc := NewCoalesceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "coalesce",
	})
	_, err := coalesceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// At least one variadic argument is expected, not an empty slice.
				[]any{},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "no arguments passed to the `coalesce` function, at least one argument is expected")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "coalesce",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}
