package corefunctions

import (
	"context"
	"fmt"
	"reflect"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ReduceFunction provides the implementation of the
// core "reduce" function defined in the blueprint specification.
type ReduceFunction struct {
	definition *function.Definition
}

// NewReduceFunction creates a new instance of the ReduceFunction with
// a complete function definition.
func NewReduceFunction() provider.Function {
	return &ReduceFunction{
		definition: &function.Definition{
			Description: "Reduces a list of values to a single value using a function. There are no " +
				"core functions that have the signature of a reducer function, the \"reduce\" function is in the core spec" +
				" to provide the tools for implementations and end users to be able to apply more complex transformations of data in blueprints.",
			FormattedDescription: "Reduces a list of values to a single value using a function. There are no " +
				"core functions that have the signature of a reducer function, the `reduce` function is in the core spec" +
				" to provide the tools for implementations and end users to be able to apply more complex transformations of data in blueprints.\n\n" +
				"**Examples:**\n\n" +
				"```\n${reduce(\n  datasources.network.subnets,\n  extract_crucial_network_info,\n  object()\n)}\n```",
			Parameters: []function.Parameter{
				&function.ListParameter{
					Label: "items",
					ElementType: &function.ValueTypeDefinitionAny{
						Label:       "Any",
						Type:        function.ValueTypeAny,
						Description: "A value of any type, every element in the containing list must be of the same type.",
					},
					Description: "An array of items where all items are of the same type to reduce over.",
				},
				&function.FunctionParameter{
					Label: "reduceFunc",
					FunctionType: &function.ValueTypeDefinitionFunction{
						Definition: function.Definition{
							Parameters: []function.Parameter{
								&function.AnyParameter{
									Label: "accum",
									Description: "The accumulated value. This is the value that will be returned " +
										"as the result of the reduction after all items have been processed.",
								},
								&function.AnyParameter{
									Label:       "item",
									Description: "The item to to apply the reducer function to.",
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
								Description: "The single value that is the result of the reduction.",
							},
						},
					},
					Description: "A function that will be applied to each item in the array, accumulating a single value. " +
						"This function can optionally take an index as a third argument.",
				},
				&function.AnyParameter{
					Label:       "initial",
					Description: "The initial value to start the reduction with.",
				},
			},
			Return: &function.AnyReturn{
				Description: "The final value that is the result of the reduction.",
			},
		},
	}
}

func (f *ReduceFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ReduceFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var initial interface{}
	var items []interface{}
	var reduceFuncInfo provider.FunctionRuntimeInfo
	if err := input.Arguments.GetMultipleVars(ctx, &initial, &items, &reduceFuncInfo); err != nil {
		return nil, err
	}

	// It would be very costly to check the type of each item in the list
	// at this stage, so we will pass each item to the provided function
	// and trust the function will check the type and return an error
	// when it encounters an item of the wrong type.
	accum := initial
	for i, item := range items {
		callArgs := []interface{}{accum, item}
		if reduceFuncInfo.ArgsOffset == 2 {
			callArgs = append(callArgs, reduceFuncInfo.PartialArgs...)
		} else if reduceFuncInfo.ArgsOffset > 2 {
			return nil, function.NewFuncCallError(
				fmt.Sprintf(
					"invalid args offset defined for "+
						"the partially applied \"%s\" function, "+
						"this is an issue with the function used to "+
						"create the function value passed into reduce",
					reduceFuncInfo.FunctionName,
				),
				function.FuncCallErrorCodeInvalidArgsOffset,
				input.CallContext.CallStackSnapshot(),
			)
		} else {
			callArgs = append(reduceFuncInfo.PartialArgs, callArgs...)
		}

		// Add the index of the current item to the end of the call arguments.
		// The provided function does not have to use this argument.
		callArgs = append(callArgs, int64(i))

		output, err := input.CallContext.Registry().Call(
			ctx,
			reduceFuncInfo.FunctionName,
			&provider.FunctionCallInput{
				Arguments: input.CallContext.NewCallArgs(callArgs...),
			},
		)
		if err != nil {
			return nil, err
		}

		accumVal := reflect.ValueOf(accum)
		if accumVal.Kind() == reflect.Pointer {
			accumVal = accumVal.Elem()
		}
		resultVal := reflect.ValueOf(output.ResponseData)
		if resultVal.Kind() == reflect.Ptr {
			resultVal = resultVal.Elem()
		}

		if accumVal.Type() != resultVal.Type() {
			return nil, function.NewFuncCallError(
				fmt.Sprintf(
					"expected the \"%s\" reducer function to return a value of type %s, "+
						"but got a value of type %s",
					reduceFuncInfo.FunctionName,
					accumVal.Type(),
					resultVal.Type(),
				),
				function.FuncCallErrorCodeInvalidReturnType,
				input.CallContext.CallStackSnapshot(),
			)
		}

		accum = output.ResponseData
	}

	return &provider.FunctionCallOutput{
		ResponseData: accum,
	}, nil
}
