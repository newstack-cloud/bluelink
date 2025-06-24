package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type LastIndexFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&LastIndexFunctionTestSuite{})

func (s *LastIndexFunctionTestSuite) SetUpTest(c *C) {
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

func (s *LastIndexFunctionTestSuite) Test_finds_index_of_last_substring(c *C) {
	lastIndexFunc := NewLastIndexFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "last_index",
	})
	output, err := lastIndexFunc.Call(context.TODO(), &provider.FunctionCallInput{
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
	c.Assert(outputInt, Equals, int64(26))
}

func (s *LastIndexFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	lastIndexFunc := NewLastIndexFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "last_index",
	})
	_, err := lastIndexFunc.Call(context.TODO(), &provider.FunctionCallInput{
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
			FunctionName: "last_index",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
