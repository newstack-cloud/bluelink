package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type SubstrFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&SubstrFunctionTestSuite{})

func (s *SubstrFunctionTestSuite) SetUpTest(c *C) {
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

func (s *SubstrFunctionTestSuite) Test_extracts_substring_with_end_provided(c *C) {
	substrFunc := NewSubstrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "substr",
	})
	output, err := substrFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				int64(8),
				int64(15),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputStr, isStr := output.ResponseData.(string)
	c.Assert(isStr, Equals, true)
	c.Assert(outputStr, Equals, "example")
}

func (s *SubstrFunctionTestSuite) Test_extracts_substring_with_end_omitted(c *C) {
	substrFunc := NewSubstrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "substr",
	})
	output, err := substrFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				int64(8),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputStr, isStr := output.ResponseData.(string)
	c.Assert(isStr, Equals, true)
	c.Assert(outputStr, Equals, "example.com")
}

func (s *SubstrFunctionTestSuite) Test_returns_func_error_for_start_index_out_of_bounds(c *C) {
	SubstrFunc := NewSubstrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "substr",
	})
	_, err := SubstrFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				// Index 100 is out of bounds.
				int64(100),
				int64(200),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "start index cannot be greater than the last element index in the string")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "substr",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *SubstrFunctionTestSuite) Test_returns_func_error_for_negative_index(c *C) {
	SubstrFunc := NewSubstrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "substr",
	})
	_, err := SubstrFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				// Negative numbers are not allowed.
				int64(-5),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "start and end indices cannot be negative")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "substr",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *SubstrFunctionTestSuite) Test_returns_func_error_for_end_index_out_of_bounds(c *C) {
	SubstrFunc := NewSubstrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "substr",
	})
	_, err := SubstrFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				int64(3),
				// Index 100 is out of bounds.
				int64(100),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "end index cannot be greater than the length of the string")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "substr",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *SubstrFunctionTestSuite) Test_returns_func_error_for_start_index_greater_than_end_index(c *C) {
	SubstrFunc := NewSubstrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "substr",
	})
	_, err := SubstrFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				int64(8),
				int64(5),
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "start index cannot be greater than end index")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "substr",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}

func (s *SubstrFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	substrFunc := NewSubstrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "substr",
	})
	_, err := substrFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				// An integer is expected here, not a string.
				"5",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument at index 1 is of type string, but target is of type int64")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "substr",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
