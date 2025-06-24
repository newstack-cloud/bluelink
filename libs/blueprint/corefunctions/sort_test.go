package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type SortFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&SortFunctionTestSuite{})

func (s *SortFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{
				"compare_sort_val": newCompareSortValFunction(),
			},
			CallStack: s.callStack,
		},
		callStack: s.callStack,
	}
}

func (s *SortFunctionTestSuite) Test_sorts_values(c *C) {

	sortFunc := NewSortFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "sort",
	})
	output, err := sortFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{
					&testSortVal{cost: 85},
					&testSortVal{cost: 10},
					&testSortVal{cost: 100},
					&testSortVal{cost: 50},
					&testSortVal{cost: 25},
					&testSortVal{cost: 1605},
					&testSortVal{cost: 5},
					&testSortVal{cost: 1},
					&testSortVal{cost: -5},
					&testSortVal{cost: 1},
				},
				provider.FunctionRuntimeInfo{
					FunctionName: "compare_sort_val",
					PartialArgs:  []any{},
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, IsNil)
	outputSlice, isSlice := output.ResponseData.([]interface{})
	c.Assert(isSlice, Equals, true)
	c.Assert(outputSlice, DeepEquals, []interface{}{
		&testSortVal{cost: -5},
		&testSortVal{cost: 1},
		&testSortVal{cost: 1},
		&testSortVal{cost: 5},
		&testSortVal{cost: 10},
		&testSortVal{cost: 25},
		&testSortVal{cost: 50},
		&testSortVal{cost: 85},
		&testSortVal{cost: 100},
		&testSortVal{cost: 1605},
	})
}

func (s *SortFunctionTestSuite) Test_returns_func_error_for_invalid_item_in_list(c *C) {
	sortFunc := NewSortFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "sort",
	})
	_, err := sortFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{
					&testSortVal{cost: 85},
					&testSortVal{cost: 10},
					// 10434 is not an expected test sort val.
					10434,
					&testSortVal{cost: 100},
				},
				provider.FunctionRuntimeInfo{
					FunctionName: "compare_sort_val",
				},
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})

	c.Assert(err, NotNil)
	funcErr, isFuncErr := err.(*function.FuncCallError)
	c.Assert(isFuncErr, Equals, true)
	c.Assert(funcErr.Message, Equals, "invalid item type provided in list")
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "compare_sort_val",
		},
		{
			FunctionName: "sort",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgumentType)
}

func (s *SortFunctionTestSuite) Test_returns_func_error_for_invalid_args_offset(c *C) {
	sortFunc := NewSortFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "sort",
	})
	_, err := sortFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				[]interface{}{
					&testSortVal{cost: 85},
					&testSortVal{cost: 10},
					&testSortVal{cost: 100},
				},
				provider.FunctionRuntimeInfo{
					FunctionName: "compare_sort_val",
					// 20 is not a valid args offset for the compare_sort_val function.
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
		"invalid args offset defined for the comparison function",
	)
	c.Assert(funcErr.CallStack, DeepEquals, []*function.Call{
		{
			FunctionName: "sort",
		},
	})
	c.Assert(funcErr.Code, Equals, function.FuncCallErrorCodeInvalidArgsOffset)
}

type testSortVal struct {
	cost int
}

func newCompareSortValFunction() provider.Function {
	return &compareSortValFunction{}
}

type compareSortValFunction struct{}

func (e *compareSortValFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		// Definition is an empty stub as it is not used in the tests.
		Definition: &function.Definition{},
	}, nil
}

func (e *compareSortValFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var itemAIface interface{}
	var itemBIface interface{}
	if err := input.Arguments.GetMultipleVars(ctx, &itemAIface, &itemBIface); err != nil {
		return nil, err
	}

	itemA, isItemA := itemAIface.(*testSortVal)
	itemB, isItemB := itemBIface.(*testSortVal)
	if !isItemA || !isItemB {
		return nil, function.NewFuncCallError(
			"invalid item type provided in list",
			function.FuncCallErrorCodeInvalidArgumentType,
			input.CallContext.CallStackSnapshot(),
		)
	}

	result := 0
	if itemA.cost < itemB.cost {
		result = -1
	} else if itemA.cost > itemB.cost {
		result = 1
	}

	return &provider.FunctionCallOutput{
		ResponseData: int64(result),
	}, nil
}
