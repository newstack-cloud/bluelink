package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type ValsFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&ValsFunctionTestSuite{})

func (s *ValsFunctionTestSuite) SetUpTest(c *C) {
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

func (s *ValsFunctionTestSuite) Test_extracts_keys_from_map(c *C) {
	keysFunc := NewValsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "vals",
	})
	output, err := keysFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
					"key4": 403212,
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputSlice, isSlice := output.ResponseData.([]interface{})
	c.Assert(isSlice, Equals, true)
	expected := []interface{}{"value1", "value2", "value3", 403212}
	sortIfaceSlice(expected)
	sortIfaceSlice(outputSlice)
	c.Assert(outputSlice, DeepEquals, expected)
}

func (s *ValsFunctionTestSuite) Test_returns_func_error_for_invalid_input(c *C) {
	keysFunc := NewValsFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "vals",
	})
	_, err := keysFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// Expected a map of strings to interfaces, not an integer.
				913826325,
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
			FunctionName: "vals",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}
