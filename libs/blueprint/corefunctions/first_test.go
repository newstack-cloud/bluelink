package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type FirstFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&FirstFunctionTestSuite{})

func (s *FirstFunctionTestSuite) SetUpTest(c *C) {
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

func (s *FirstFunctionTestSuite) Test_gets_first_non_empty_value(c *C) {
	firstFunc := NewFirstFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "first",
	})
	output, err := firstFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{
					"",
					[]string{},
					"First non-empty value",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "First non-empty value")
}

func (s *FirstFunctionTestSuite) Test_gets_last_empty_value_if_all_values_are_empty(c *C) {
	firstFunc := NewFirstFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "first",
	})
	output, err := firstFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// All values are empty, the last value should be returned.
				[]any{
					[]string{},
					[]string{},
					[]string{},
					"",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "")
}

func (s *FirstFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	firstFunc := NewFirstFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "first",
	})
	_, err := firstFunc.Call(context.TODO(), &provider.FunctionCallInput{
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
	c.Assert(funcErr.Message, Equals, "no arguments passed to the `first` function, at least one argument is expected")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "first",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}
