package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// FilterFunction provides the implementation of the
// core "filter" function defined in the blueprint specification.
type FilterFunction struct {
	definition *function.Definition
}

// NewFilterFunction creates a new instance of the FilterFunction with
// a complete function definition.
func NewFilterFunction() provider.Function {
	return &FilterFunction{
		definition: &function.Definition{
			Description: "Filters a list of values based on a predicate function.",
			FormattedDescription: "Filters a list of values based on a predicate function.\n\n" +
				"**Examples:**\n\n" +
				"```\n${filter(\n  datasources.network.subnets,\n  has_prefix_g(\"subnet-402948-\")\n)}\n```",
			Parameters: []function.Parameter{
				&function.ListParameter{
					Label: "items",
					ElementType: &function.ValueTypeDefinitionAny{
						Label:       "any",
						Type:        function.ValueTypeAny,
						Description: "A value of any type, every element in the containing list must be of the same type.",
					},
					Description: "An array of items where all items are of the same type to filter.",
				},
				&function.FunctionParameter{
					Label: "filterFunc",
					FunctionType: &function.ValueTypeDefinitionFunction{
						Definition: function.Definition{
							Parameters: []function.Parameter{
								&function.AnyParameter{
									Label:       "item",
									Description: "The item to check.",
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
							Return: &function.ScalarReturn{
								Type: &function.ValueTypeDefinitionScalar{
									Type: function.ValueTypeBool,
								},
								Description: "Whether or not the item meets the criteria.",
							},
						},
					},
					Description: "The predicate function to check if each item in the list meets a certain criteria.",
				},
			},
			Return: &function.ListReturn{
				ElementType: &function.ValueTypeDefinitionAny{
					Label:       "Any",
					Type:        function.ValueTypeAny,
					Description: "A value of any type, every element in the returned list must be of the same type.",
				},
				Description: "The list of values that remain after applying the filter.",
			},
		},
	}
}

func (f *FilterFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *FilterFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var items []interface{}
	var filterFuncInfo provider.FunctionRuntimeInfo
	if err := input.Arguments.GetMultipleVars(ctx, &items, &filterFuncInfo); err != nil {
		return nil, err
	}

	// It would be very costly to check the type of each item in the list
	// at this stage, so we will pass each item to the provided function
	// and trust the function will check the type and return an error
	// when it encounters an item of the wrong type.
	newItems := []interface{}{}
	for i, item := range items {
		callArgs := []interface{}{item}
		if filterFuncInfo.ArgsOffset == 1 {
			callArgs = append(callArgs, filterFuncInfo.PartialArgs...)
		} else if filterFuncInfo.ArgsOffset > 1 {
			return nil, function.NewFuncCallError(
				fmt.Sprintf(
					"invalid args offset defined for "+
						"the partially applied \"%s\" function, "+
						"this is an issue with the function used to "+
						"create the function value passed into filter",
					filterFuncInfo.FunctionName,
				),
				function.FuncCallErrorCodeInvalidArgsOffset,
				input.CallContext.CallStackSnapshot(),
			)
		} else {
			callArgs = append(filterFuncInfo.PartialArgs, callArgs...)
		}

		// Add the index of the current item to the end of the call arguments.
		// The provided function does not have to use this argument.
		callArgs = append(callArgs, int64(i))

		output, err := input.CallContext.Registry().Call(
			ctx,
			filterFuncInfo.FunctionName,
			&provider.FunctionCallInput{
				Arguments: input.CallContext.NewCallArgs(callArgs...),
			},
		)
		if err != nil {
			return nil, err
		}

		result, isBool := output.ResponseData.(bool)
		if !isBool {
			return nil, function.NewFuncCallError(
				fmt.Sprintf(
					"expected a boolean value from the predicate function, "+
						"but got a value of type %T",
					output.ResponseData,
				),
				function.FuncCallErrorCodeInvalidReturnType,
				input.CallContext.CallStackSnapshot(),
			)
		}
		if result {
			newItems = append(newItems, item)
		}
	}

	return &provider.FunctionCallOutput{
		ResponseData: newItems,
	}, nil
}
