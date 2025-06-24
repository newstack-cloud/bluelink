package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// FlatMapFunction provides the implementation of the
// core "flatmap" function defined in the blueprint specification.
type FlatMapFunction struct {
	definition *function.Definition
}

// NewFlatMapFunction creates a new instance of the FlatMapFunction with
// a complete function definition.
func NewFlatMapFunction() provider.Function {
	return &FlatMapFunction{
		definition: &function.Definition{
			Description: "Maps a list of values to a new list of values using a function and flattens the result.",
			FormattedDescription: "Maps a list of values to a new list of values using a function and flattens the result.\n\n" +
				"**Examples:**\n\n" +
				"```\n${flatmap(\n  values.hosts,\n  split_g(\",\")\n)}\n```",
			Parameters: []function.Parameter{
				&function.ListParameter{
					Label: "items",
					ElementType: &function.ValueTypeDefinitionAny{
						Label:       "Any",
						Type:        function.ValueTypeAny,
						Description: "A value of any type, every element in the containing list must be of the same type.",
					},
					Description: "An array of items where all items are of the same type to map.",
				},
				&function.FunctionParameter{
					Label: "mapFunc",
					FunctionType: &function.ValueTypeDefinitionFunction{
						Definition: function.Definition{
							Parameters: []function.Parameter{
								&function.AnyParameter{
									Label:       "item",
									Description: "The item to transform.",
								},
								&function.ScalarParameter{
									Label:       "index",
									Description: "The index of the item in the list.",
									Type: &function.ValueTypeDefinitionScalar{
										Type: function.ValueTypeInt64,
									},
									Optional: true,
								},
							},
							Return: &function.AnyReturn{
								Type:        function.ValueTypeAny,
								Description: "The transformed item as a list that will be flattened.",
							},
						},
					},
					Description: "The function to apply to each element in the list.",
				},
			},
			Return: &function.ListReturn{
				ElementType: &function.ValueTypeDefinitionAny{
					Label:       "Any",
					Type:        function.ValueTypeAny,
					Description: "A value of any type, every element in the returned list must be of the same type.",
				},
				Description: "The flattened list of values after applying the mapping function and flattening results.",
			},
		},
	}
}

func (f *FlatMapFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *FlatMapFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var items []interface{}
	var flatMapFuncInfo provider.FunctionRuntimeInfo
	if err := input.Arguments.GetMultipleVars(ctx, &items, &flatMapFuncInfo); err != nil {
		return nil, err
	}

	// It would be very costly to check the type of each item in the list
	// at this stage, so we will pass each item to the provided function
	// and trust the function will check the type and return an error
	// when it encounters an item of the wrong type.
	newItems := []interface{}{}
	for i, item := range items {
		callArgs := []interface{}{item}
		if flatMapFuncInfo.ArgsOffset == 1 {
			callArgs = append(callArgs, flatMapFuncInfo.PartialArgs...)
		} else if flatMapFuncInfo.ArgsOffset > 1 {
			return nil, function.NewFuncCallError(
				fmt.Sprintf(
					"invalid args offset defined for "+
						"the partially applied \"%s\" function, "+
						"this is an issue with the function used to "+
						"create the function value passed into flatmap",
					flatMapFuncInfo.FunctionName,
				),
				function.FuncCallErrorCodeInvalidArgsOffset,
				input.CallContext.CallStackSnapshot(),
			)
		} else {
			callArgs = append(flatMapFuncInfo.PartialArgs, callArgs...)
		}

		// Add the index of the current item to the end of the call arguments.
		// The provided function does not have to use this argument.
		callArgs = append(callArgs, int64(i))

		output, err := input.CallContext.Registry().Call(
			ctx,
			flatMapFuncInfo.FunctionName,
			&provider.FunctionCallInput{
				Arguments:   input.CallContext.NewCallArgs(callArgs...),
				CallContext: input.CallContext,
			},
		)
		if err != nil {
			return nil, err
		}

		itemSlice, isItemSlice := output.ResponseData.([]interface{})
		if !isItemSlice {
			return nil, function.NewFuncCallError(
				fmt.Sprintf(
					"expected the function \"%s\" to return a list of items, but got a value of type %T",
					flatMapFuncInfo.FunctionName,
					output.ResponseData,
				),
				function.FuncCallErrorCodeInvalidReturnType,
				input.CallContext.CallStackSnapshot(),
			)
		}

		newItems = append(newItems, itemSlice...)
	}

	return &provider.FunctionCallOutput{
		ResponseData: newItems,
	}, nil
}
