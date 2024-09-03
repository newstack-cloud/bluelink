package corefunctions

import (
	"context"

	"github.com/two-hundred/celerity/libs/blueprint/function"
	"github.com/two-hundred/celerity/libs/blueprint/internal"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type IndexFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&IndexFunctionTestSuite{})

func (s *IndexFunctionTestSuite) SetUpTest(c *C) {
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

func (s *IndexFunctionTestSuite) Test_finds_index_of_first_substring(c *C) {
	indexFunc := NewIndexFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "index",
	})
	output, err := indexFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"This is a test string for testing",
				"test",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputInt, isInt := output.ResponseData.(int64)
	c.Assert(isInt, Equals, true)
	c.Assert(outputInt, Equals, int64(10))
}

func (s *IndexFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	indexFunc := NewIndexFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "index",
	})
	_, err := indexFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Expected a string, not an integer.
				403928322,
				",",
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
			FunctionName: "index",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
