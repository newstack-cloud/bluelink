package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type OrFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&OrFunctionTestSuite{})

func (s *OrFunctionTestSuite) SetUpTest(c *C) {
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

func (s *OrFunctionTestSuite) Test_applies_logical_or(c *C) {
	orFunc := NewOrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "or",
	})
	output, err := orFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				false,
				true,
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

func (s *OrFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	orFunc := NewOrFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "or",
	})
	_, err := orFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				true,
				// A boolean is expected here, not an integer.
				985043,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "right argument to `or` must be a boolean value")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "or",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidInput)
}
