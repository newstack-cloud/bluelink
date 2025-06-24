package corefunctions

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type MapFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&MapFunctionTestSuite{})

func (s *MapFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{
				"trimprefix": NewTrimPrefixFunction(),
			},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *MapFunctionTestSuite) Test_map_over_values(c *C) {

	mapFunc := NewMapFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "map",
	})
	output, err := mapFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{"item_a", "item_b", "item_c"},
				provider.FunctionRuntimeInfo{
					FunctionName: "trimprefix",
					PartialArgs: []any{
						"item_",
					},
					ArgsOffset: 1,
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputSlice, isStrSlice := output.ResponseData.([]interface{})
	c.Assert(isStrSlice, Equals, true)
	c.Assert(outputSlice, DeepEquals, []interface{}{"a", "b", "c"})
}

func (s *MapFunctionTestSuite) Test_returns_func_error_for_invalid_item_in_list(c *C) {
	mapFunc := NewMapFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "map",
	})
	_, err := mapFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// 5 is not a string, the trimprefix function should return an error.
				[]interface{}{"item_a", 5, "item_c"},
				provider.FunctionRuntimeInfo{
					FunctionName: "trimprefix",
					PartialArgs: []any{
						"item_",
					},
					ArgsOffset: 1,
				},
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
			FunctionName: "trimprefix",
		},
		{
			FunctionName: "map",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *MapFunctionTestSuite) Test_returns_func_error_for_invalid_args_offset(c *C) {
	mapFunc := NewMapFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "map",
	})
	_, err := mapFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// 5 is not a string, the trimprefix function should return an error.
				[]interface{}{"item_a", 5, "item_c"},
				provider.FunctionRuntimeInfo{
					FunctionName: "trimprefix",
					PartialArgs: []any{
						"item_",
					},
					// 20 is not a valid args offset for the trimprefix function.
					ArgsOffset: 20,
				},
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
		"invalid args offset defined for the partially applied \"trimprefix\""+
			" function, this is an issue with the function used to create the function value passed into map",
	)
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "map",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgsOffset)
}
