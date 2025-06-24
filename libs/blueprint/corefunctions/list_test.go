package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type ListFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&ListFunctionTestSuite{})

func (s *ListFunctionTestSuite) SetUpTest(c *C) {
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

func (s *ListFunctionTestSuite) Test_creates_list(c *C) {
	listFunc := NewListFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "list",
	})
	output, err := listFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{1, 3, 5, 9, 10},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputSlice, isSlice := output.ResponseData.([]interface{})
	c.Assert(isSlice, Equals, true)
	// Operationally, this just returns the same value that was passed
	// in as the first argument.
	// The list function exists for end-users to be able to create a list
	// that can be passed around in a blueprint, where operationally variadic
	// arguments should always be passed in to functions as slices of interfaces
	// as functions won't know how many arguments they will receive.
	c.Assert(outputSlice, DeepEquals, []interface{}{1, 3, 5, 9, 10})
}

func (s *ListFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	listFunc := NewListFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "list",
	})
	_, err := listFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Array of interfaces are expected, not an int.
				4024932,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(
		funcErr.Message,
		Equals,
		"argument at index 0 is of type int, but target is of type slice",
	)
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "list",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
