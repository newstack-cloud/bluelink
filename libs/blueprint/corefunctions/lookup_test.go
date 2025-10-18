package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type LookupFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&LookupFunctionTestSuite{})

func (s *LookupFunctionTestSuite) SetUpTest(c *C) {
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

func (s *LookupFunctionTestSuite) Test_gets_existing_value_in_map(c *C) {
	lookupFunc := NewLookupFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "lookup",
	})
	output, err := lookupFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				map[string]any{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
				"key2",
				"default",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "value2")
}

func (s *LookupFunctionTestSuite) Test_gets_default_value_if_key_is_not_found(c *C) {
	lookupFunc := NewLookupFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "lookup",
	})
	output, err := lookupFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// All values are empty, the last value should be returned.
				map[string]any{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
				"key4",
				"default",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, Equals, "default")
}

func (s *LookupFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	lookupFunc := NewLookupFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "lookup",
	})
	_, err := lookupFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				map[string]any{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
				// An integer is not allowed for a key.
				34292,
				"default",
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
			FunctionName: "lookup",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
