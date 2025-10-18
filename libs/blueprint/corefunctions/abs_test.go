package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type AbsFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&AbsFunctionTestSuite{})

func (s *AbsFunctionTestSuite) SetUpTest(c *C) {
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

func (s *AbsFunctionTestSuite) Test_returns_absolute_value_of_negative_integer(c *C) {
	absFunc := NewAbsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "abs",
	})
	output, err := absFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{-5},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, 5)
}

func (s *AbsFunctionTestSuite) Test_returns_absolute_value_of_positive_integer(c *C) {
	absFunc := NewAbsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "abs",
	})
	output, err := absFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{3},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, 3)
}

func (s *AbsFunctionTestSuite) Test_returns_absolute_value_of_negative_float(c *C) {
	absFunc := NewAbsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "abs",
	})
	output, err := absFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{-5.7},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, 5.7)
}

func (s *AbsFunctionTestSuite) Test_returns_absolute_value_of_positive_float(c *C) {
	absFunc := NewAbsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "abs",
	})
	output, err := absFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{3.14},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, 3.14)
}

func (s *AbsFunctionTestSuite) Test_returns_zero_for_zero(c *C) {
	absFunc := NewAbsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "abs",
	})
	output, err := absFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{0},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, 0)
}

func (s *AbsFunctionTestSuite) Test_returns_error_for_non_numeric_argument(c *C) {
	absFunc := NewAbsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "abs",
	})
	_, err := absFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args:    []any{"not a number"},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "input argument at index 0 must be a number (integer or float)")
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
