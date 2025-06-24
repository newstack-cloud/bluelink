package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type LenFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&LenFunctionTestSuite{})

func (s *LenFunctionTestSuite) SetUpTest(c *C) {
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

func (s *LenFunctionTestSuite) Test_gets_length_of_string(c *C) {
	lenFunc := NewLenFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "len",
	})
	output, err := lenFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"This is an example string.",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputInt, isInt := output.ResponseData.(int64)
	c.Assert(isInt, Equals, true)
	c.Assert(outputInt, Equals, int64(26))
}

func (s *LenFunctionTestSuite) Test_gets_length_of_array(c *C) {
	lenFunc := NewLenFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "len",
	})
	output, err := lenFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]any{"This is", "an example", "array."},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputInt, isInt := output.ResponseData.(int64)
	c.Assert(isInt, Equals, true)
	c.Assert(outputInt, Equals, int64(3))
}

func (s *LenFunctionTestSuite) Test_gets_length_of_map(c *C) {
	lenFunc := NewLenFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "len",
	})
	output, err := lenFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				map[string]any{
					"key1": "This is",
					"key2": "an example",
					"key3": "array.",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputInt, isInt := output.ResponseData.(int64)
	c.Assert(isInt, Equals, true)
	c.Assert(outputInt, Equals, int64(3))
}

func (s *LenFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	lenFunc := NewLenFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "len",
	})
	_, err := lenFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// An integer is not allowed for a length check.
				785043,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "invalid input type, expected string, array or mapping, received: int")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "len",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}
