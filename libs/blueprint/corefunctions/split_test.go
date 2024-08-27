package corefunctions

import (
	"context"

	"github.com/two-hundred/celerity/libs/blueprint/function"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type SplitFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&SplitFunctionTestSuite{})

func (s *SplitFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &blueprintParamsMock{},
		registry: &functionRegistryMock{
			functions: map[string]provider.Function{},
			callStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *SplitFunctionTestSuite) Test_splits_string_by_delimiter(c *C) {
	splitFunc := NewSplitFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "split",
	})
	output, err := splitFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example1.com,https://example2.com,https://example3.com,https://example4.com",
				",",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputStrSlice, isStrSlice := output.ResponseData.([]interface{})
	c.Assert(isStrSlice, Equals, true)
	c.Assert(outputStrSlice, DeepEquals, []interface{}{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
		"https://example4.com",
	})
}

func (s *SplitFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	splitFunc := NewSplitFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "split",
	})
	_, err := splitFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com,https://example2.com",
				// Missing the delimiter.
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "2 arguments expected, but 1 argument was passed into function")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "split",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeFunctionCall)
}
