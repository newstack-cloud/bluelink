package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type HasSuffixFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&HasSuffixFunctionTestSuite{})

func (s *HasSuffixFunctionTestSuite) SetUpTest(c *C) {
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

func (s *HasSuffixFunctionTestSuite) Test_has_suffix(c *C) {
	hasSuffixFunc := NewHasSuffixFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "has_suffix",
	})
	output, err := hasSuffixFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com/config",
				"/config",
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

func (s *HasSuffixFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	hasSuffixFunc := NewHasSuffixFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "has_suffix",
	})
	_, err := hasSuffixFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com/config/2",
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
			FunctionName: "has_suffix",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *HasSuffixFunctionTestSuite) Test_returns_false_for_none_value(c *C) {
	hasSuffixFunc := NewHasSuffixFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "has_suffix",
	})
	output, err := hasSuffixFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				core.GetNoneMarker(),
				"/config",
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
