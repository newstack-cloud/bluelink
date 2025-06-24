package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// SortFunction provides the implementation of the
// core "sort" function defined in the blueprint specification.
// This uses an implementation of quicksort to sort the list of items,
// this won't be as efficient as Go's built-in sort package,
// but is a good alternative with support for error handling and being able
// to sort based on a custom comparison provider function.
type SortFunction struct {
	definition *function.Definition
}

// NewSortFunction creates a new instance of the SortFunction with
// a complete function definition.
func NewSortFunction() provider.Function {
	return &SortFunction{
		definition: &function.Definition{
			Description: "Sorts a list of values based on comparison function. There are no core functions " +
				"that have the signature of a comparison function, the \"sort\" function is in the core spec " +
				"to provide the tools for implementations and end users to sort arrays of values on a custom comparison function.",
			FormattedDescription: "Sorts a list of values based on comparison function. There are no core functions " +
				"that have the signature of a comparison function, the `sort` function is in the core spec " +
				"to provide the tools for implementations and end users to sort arrays of values on a custom comparison function.\n\n" +
				"**Examples:**\n\n" +
				"```\n${sort(\n  datasources.network.subnets,\n  compare_cidr_ranges\n)}\n```",
			Parameters: []function.Parameter{
				&function.ListParameter{
					Label: "items",
					ElementType: &function.ValueTypeDefinitionAny{
						Label:       "Any",
						Type:        function.ValueTypeAny,
						Description: "A value of any type, every element in the containing list must be of the same type.",
					},
					Description: "An array of items where all items are of the same type to sort.",
				},
				&function.FunctionParameter{
					Label: "sortFunc",
					FunctionType: &function.ValueTypeDefinitionFunction{
						Definition: function.Definition{
							Parameters: []function.Parameter{
								&function.AnyParameter{
									Label:       "item a",
									Description: "An item in the list to compare.",
								},
								&function.AnyParameter{
									Label:       "item b",
									Description: "An item in the list to compare.",
								},
							},
							Return: &function.ScalarReturn{
								Type: &function.ValueTypeDefinitionScalar{
									Type: function.ValueTypeInt64,
								},
								Description: "a value less than 0 of item a is less than item b," +
									" 0 if they are equal, and a value greater than 0 if item a is greater than item b.",
							},
						},
					},
					Description: "The comparison function to sort the list of items.",
				},
			},
			Return: &function.ListReturn{
				ElementType: &function.ValueTypeDefinitionAny{
					Label:       "Any",
					Type:        function.ValueTypeAny,
					Description: "A value of any type, every element in the returned list must be of the same type.",
				},
				Description: "A sorted copy of the input list.",
			},
		},
	}
}

func (f *SortFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *SortFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var items []interface{}
	var comparisonFuncInfo provider.FunctionRuntimeInfo
	if err := input.Arguments.GetMultipleVars(ctx, &items, &comparisonFuncInfo); err != nil {
		return nil, err
	}

	// It would be very costly to check the type of each item in the list
	// at this stage, so we will pass items to the provided comparison function
	// and trust the function will check the type and return an error
	// when it encounters an item of the wrong type.
	sortCopy := append([]interface{}{}, items...)
	err := sortItems(ctx, sortCopy, comparisonFuncInfo, input.CallContext, 0, len(sortCopy)-1)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		ResponseData: sortCopy,
	}, nil
}

func sortItems(
	ctx context.Context,
	items []interface{},
	comparisonFuncInfo provider.FunctionRuntimeInfo,
	callCtx provider.FunctionCallContext,
	low int,
	high int,
) error {
	if low < high {
		p, err := partition(ctx, items, comparisonFuncInfo, callCtx, low, high)
		if err != nil {
			return err
		}

		err = sortItems(ctx, items, comparisonFuncInfo, callCtx, low, p-1)
		if err != nil {
			return err
		}

		err = sortItems(ctx, items, comparisonFuncInfo, callCtx, p+1, high)
		if err != nil {
			return err
		}
	}
	return nil
}

func partition(
	ctx context.Context,
	items []interface{},
	comparisonFuncInfo provider.FunctionRuntimeInfo,
	callCtx provider.FunctionCallContext,
	low int,
	high int,
) (int, error) {
	pivot := items[high]
	i := low
	j := low

	for i <= high {
		callArgs := []interface{}{items[i], pivot}
		if comparisonFuncInfo.ArgsOffset == 2 {
			callArgs = append(callArgs, comparisonFuncInfo.PartialArgs...)
		} else if comparisonFuncInfo.ArgsOffset > 2 {
			return 0, function.NewFuncCallError(
				"invalid args offset defined for the comparison function",
				function.FuncCallErrorCodeInvalidArgsOffset,
				callCtx.CallStackSnapshot(),
			)
		} else {
			callArgs = append(comparisonFuncInfo.PartialArgs, callArgs...)
		}

		output, err := callCtx.Registry().Call(
			ctx,
			comparisonFuncInfo.FunctionName,
			&provider.FunctionCallInput{
				Arguments:   callCtx.NewCallArgs(callArgs...),
				CallContext: callCtx,
			},
		)
		if err != nil {
			return 0, err
		}

		result, ok := output.ResponseData.(int64)
		if !ok {
			return 0, function.NewFuncCallError(
				"comparison function must return an integer",
				function.FuncCallErrorCodeInvalidReturnType,
				callCtx.CallStackSnapshot(),
			)
		}

		if result <= 0 {
			items[i], items[j] = items[j], items[i]
			j += 1
		}
		i += 1
	}

	return j - 1, nil
}
