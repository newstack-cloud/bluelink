package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type FlatMapFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&FlatMapFunctionTestSuite{})

func (s *FlatMapFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{
				"split":      NewSplitFunction(),
				"trimprefix": NewTrimPrefixFunction(),
			},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *FlatMapFunctionTestSuite) Test_map_over_values_and_flattens_results(c *C) {

	flatMapFunc := NewFlatMapFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "flatmap",
	})
	// Create a partially applied function that is equivalent to calling:
	// flatmap(list(...), split_g(","))
	split_G_Func := NewSplit_G_Function()
	splitFuncOutput, err := split_G_Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				",",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)

	output, err := flatMapFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{
					"item_a,item_b,item_c",
					"item_d,item_e,item_f",
					"item_g",
					"item_h,item_i",
				},
				splitFuncOutput.FunctionInfo,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputSlice, isSlice := output.ResponseData.([]interface{})
	c.Assert(isSlice, Equals, true)
	c.Assert(outputSlice, DeepEquals, []interface{}{
		"item_a",
		"item_b",
		"item_c",
		"item_d",
		"item_e",
		"item_f",
		"item_g",
		"item_h",
		"item_i",
	})
}

func (s *FlatMapFunctionTestSuite) Test_returns_func_error_for_invalid_item_in_list(c *C) {
	flatMapFunc := NewFlatMapFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "flatmap",
	})

	// Create a partially applied function that is equivalent to calling:
	// flatmap(list(...), split_g(","))
	split_G_Func := NewSplit_G_Function()
	splitFuncOutput, err := split_G_Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				",",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)

	_, err = flatMapFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// 5 is not a string, the split_g function should return an error.
				[]interface{}{"item_a", 5, "item_c"},
				splitFuncOutput.FunctionInfo,
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
			FunctionName: "split",
		},
		{
			FunctionName: "flatmap",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *FlatMapFunctionTestSuite) Test_returns_func_error_for_invalid_return_val(c *C) {
	flatMapFunc := NewFlatMapFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "flatmap",
	})

	// Create a partially applied function that is equivalent to calling:
	// flatmap(list(...), trimprefix_g("item_"))
	// trimprefix_g yields a partially applied function definition that returns a string,
	// so the flatmap function should return an error in processing the result.
	trimPrefix_G_Function := NewTrimPrefix_G_Function()
	trimPrefixFuncOutput, err := trimPrefix_G_Function.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"item_",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)

	_, err = flatMapFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{"item_a,item_b", "item_c"},
				trimPrefixFuncOutput.FunctionInfo,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "expected the function \"trimprefix\" to return a "+
		"list of items, but got a value of type string")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "flatmap",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidReturnType)
}

func (s *FlatMapFunctionTestSuite) Test_returns_func_error_for_invalid_args_offset(c *C) {
	flatMapFunc := NewFlatMapFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "flatmap",
	})

	split_G_Func := NewSplit_G_Function()
	splitFuncOutput, err := split_G_Func.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				",",
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)

	// 20 is not a valid args offset for the partially applied split function.
	splitFuncOutput.FunctionInfo.ArgsOffset = 20
	_, err = flatMapFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{"item_a,item_b", "item_c"},
				splitFuncOutput.FunctionInfo,
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
		"invalid args offset defined for the partially applied \"split\""+
			" function, this is an issue with the function used to create the function value passed into flatmap",
	)
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "flatmap",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgsOffset)
}
