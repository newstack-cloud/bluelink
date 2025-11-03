package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type ReplaceFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&ReplaceFunctionTestSuite{})

func (s *ReplaceFunctionTestSuite) SetUpTest(c *C) {
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

func (s *ReplaceFunctionTestSuite) Test_replaces_all_occurrences_of_substring(c *C) {
	replaceFunc := NewReplaceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "replace",
	})
	output, err := replaceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example1.com, https://example2.com, https://example3.com, https://example4.com",
				"https://",
				"www.",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputStr, isStr := output.ResponseData.(string)
	c.Assert(isStr, Equals, true)
	c.Assert(outputStr, Equals, "www.example1.com, www.example2.com, www.example3.com, www.example4.com")
}

func (s *ReplaceFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	replaceFunc := NewReplaceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "replace",
	})
	_, err := replaceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"https://example.com",
				"https://",
				// Missing the "replaceWith" string.
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "3 arguments expected, but 2 arguments were passed into function")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "replace",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeFunctionCall)
}

func (s *ReplaceFunctionTestSuite) Test_propagates_none_value(c *C) {
	replaceFunc := NewReplaceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "replace",
	})
	output, err := replaceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				core.GetNoneMarker(),
				"https://",
				"www.",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(core.IsNoneMarker(output.ResponseData), Equals, true)
}
