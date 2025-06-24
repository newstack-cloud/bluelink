package corefunctions

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type FilterFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
	suite.Suite
}

func (s *FilterFunctionTestSuite) SetupTest() {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{
				"has_prefix": NewHasPrefixFunction(),
			},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *FilterFunctionTestSuite) Test_filters_items_correctly() {

	filterFunc := NewFilterFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "filter",
	})
	output, err := filterFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{"core_a", "core_b", "aux_1", "aux_2", "core_c", "aux_3"},
				provider.FunctionRuntimeInfo{
					FunctionName: "has_prefix",
					PartialArgs: []any{
						"core_",
					},
					ArgsOffset: 1,
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().NoError(err)
	outputSlice, isStrSlice := output.ResponseData.([]interface{})
	s.Assert().True(isStrSlice)
	s.Assert().Equal([]interface{}{"core_a", "core_b", "core_c"}, outputSlice)
}

func (s *FilterFunctionTestSuite) Test_returns_func_error_for_invalid_item_in_list() {
	filterFunc := NewFilterFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "filter",
	})
	_, err := filterFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				// 5 is not a string, the has_prefix function should return an error.
				[]interface{}{"item_a", 5, "item_c"},
				provider.FunctionRuntimeInfo{
					FunctionName: "has_prefix",
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

	s.Require().Error(err)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	s.Assert().True(isFuncErr)
	s.Assert().Equal("argument at index 0 is of type int, but target is of type string", funcErr.Message)
	s.Assert().Equal(
		[]*function.Call{
			{
				FunctionName: "has_prefix",
			},
			{
				FunctionName: "filter",
			},
		},
		funcErr.CallStack,
	)
	s.Assert().Equal(function.FuncCallErrorCodeInvalidArgumentType, funcErr.Code)
}

func (s *FilterFunctionTestSuite) Test_returns_func_error_for_invalid_args_offset() {
	filterFunc := NewFilterFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "filter",
	})
	_, err := filterFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{"item_a", "item_c"},
				provider.FunctionRuntimeInfo{
					FunctionName: "has_prefix",
					PartialArgs: []any{
						"item_",
					},
					// 20 is not a valid args offset for the has_prefix function.
					ArgsOffset: 20,
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	s.Require().Error(err)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	s.Assert().True(isFuncErr)
	s.Assert().Equal(
		"invalid args offset defined for the partially applied \"has_prefix\""+
			" function, this is an issue with the function used to create the function value passed into filter",
		funcErr.Message,
	)
	s.Assert().Equal(
		[]*function.Call{{FunctionName: "filter"}},
		funcErr.CallStack,
	)
	s.Assert().Equal(
		function.FuncCallErrorCodeInvalidArgsOffset,
		funcErr.Code,
	)
}

func TestFilterFunctionTestSuite(t *testing.T) {
	suite.Run(t, new(FilterFunctionTestSuite))
}
