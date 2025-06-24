package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type KeysFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&KeysFunctionTestSuite{})

func (s *KeysFunctionTestSuite) SetUpTest(c *C) {
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

func (s *KeysFunctionTestSuite) Test_extracts_keys_from_map(c *C) {
	keysFunc := NewKeysFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "keys",
	})
	output, err := keysFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputSlice, isSlice := output.ResponseData.([]string)
	c.Assert(isSlice, Equals, true)
	expected := []string{"key1", "key2", "key3"}
	sortStrSlice(expected)
	sortStrSlice(outputSlice)
	c.Assert(outputSlice, DeepEquals, expected)
}

func (s *KeysFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	keysFunc := NewKeysFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "keys",
	})
	_, err := keysFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Expected a map of strings to interfaces, not an integer.
				213426322,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "argument at index 0 is of type int, but target is of type map")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "keys",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
