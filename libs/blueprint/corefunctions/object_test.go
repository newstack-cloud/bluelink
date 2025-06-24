package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type ObjectFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&ObjectFunctionTestSuite{})

func (s *ObjectFunctionTestSuite) SetUpTest(c *C) {
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

func (s *ObjectFunctionTestSuite) Test_creates_object(c *C) {
	objectFunc := NewObjectFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "object",
	})
	output, err := objectFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]NamedArgument{
					{
						Name:  "Total",
						Value: 10,
					},
					{
						Name:  "Title",
						Value: "This is a title",
					},
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputMap, isMap := output.ResponseData.(map[string]interface{})
	c.Assert(isMap, Equals, true)
	c.Assert(outputMap, DeepEquals, map[string]interface{}{
		"Total": 10,
		"Title": "This is a title",
	})
}

func (s *ObjectFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	objectFunc := NewObjectFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "object",
	})
	_, err := objectFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Array of named arguments is expected, not an int.
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
			FunctionName: "object",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
