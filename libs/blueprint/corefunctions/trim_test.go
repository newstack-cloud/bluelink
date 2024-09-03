package corefunctions

import (
	"context"

	"github.com/two-hundred/celerity/libs/blueprint/function"
	"github.com/two-hundred/celerity/libs/blueprint/internal"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type TrimFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&TrimFunctionTestSuite{})

func (s *TrimFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &blueprintParamsMock{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *TrimFunctionTestSuite) Test_trims_whitespace(c *C) {
	trimFunc := NewTrimFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "trim",
	})
	output, err := trimFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"  	https://example.com   \n   \n",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputStr, isStr := output.ResponseData.(string)
	c.Assert(isStr, Equals, true)
	c.Assert(outputStr, Equals, "https://example.com")
}

func (s *TrimFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	trimFunc := NewTrimFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "trim",
	})
	_, err := trimFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// A string is expected here, not an integer.
				183043,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument at index 0 is of type int, but target is of type string")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "trim",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
