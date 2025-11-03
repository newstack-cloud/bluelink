package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type HasPrefixFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&HasPrefixFunctionTestSuite{})

func (s *HasPrefixFunctionTestSuite) SetUpTest(c *C) {
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

func (s *HasPrefixFunctionTestSuite) Test_has_prefix(c *C) {
	hasPrefixFunc := NewHasPrefixFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "has_prefix",
	})
	output, err := hasPrefixFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				"https://",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputBool, isBool := output.ResponseData.(bool)
	c.Assert(isBool, Equals, true)
	c.Assert(outputBool, Equals, true)
}

func (s *HasPrefixFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	hasPrefixFunc := NewHasPrefixFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "has_prefix",
	})
	_, err := hasPrefixFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				// A string is expected here, not an integer.
				785043,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument at index 1 is of type int, but target is of type string")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "has_prefix",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *HasPrefixFunctionTestSuite) Test_returns_false_for_none_value(c *C) {
	hasPrefixFunc := NewHasPrefixFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "has_prefix",
	})
	output, err := hasPrefixFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				core.GetNoneMarker(),
				"https://",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputBool, isBool := output.ResponseData.(bool)
	c.Assert(isBool, Equals, true)
	c.Assert(outputBool, Equals, false)
}
