package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type MaxFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&MaxFunctionTestSuite{})

func (s *MaxFunctionTestSuite) SetUpTest(c *C) {
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

func (s *MaxFunctionTestSuite) Test_returns_maximum_from_integers(c *C) {
	maxFunc := NewMaxFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "max",
	})
	output, err := maxFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{10, 5, 8, 3, 12},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, 12)
}

func (s *MaxFunctionTestSuite) Test_returns_maximum_from_floats(c *C) {
	maxFunc := NewMaxFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "max",
	})
	output, err := maxFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{10.5, 5.2, 8.9, 3.1, 12.7},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, 12.7)
}

func (s *MaxFunctionTestSuite) Test_returns_maximum_from_mixed_numbers(c *C) {
	maxFunc := NewMaxFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "max",
	})
	output, err := maxFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{10, 5.2, 8, 15.9, 12},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, 15.9)
}

func (s *MaxFunctionTestSuite) Test_returns_maximum_with_negative_numbers(c *C) {
	maxFunc := NewMaxFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "max",
	})
	output, err := maxFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{-10, -5, -8, -3, -12},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, -3)
}

func (s *MaxFunctionTestSuite) Test_returns_error_for_no_arguments(c *C) {
	maxFunc := NewMaxFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "max",
	})
	_, err := maxFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "max requires at least one argument")
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *MaxFunctionTestSuite) Test_returns_error_for_non_numeric_argument(c *C) {
	maxFunc := NewMaxFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "max",
	})
	_, err := maxFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{10, "not a number", 5},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument at index 1 must be a number (integer or float)")
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
