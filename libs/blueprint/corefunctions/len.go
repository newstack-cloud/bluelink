package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// LenFunction provides the implementation of
// a function that gets the length of a string,
// array or mapping.
type LenFunction struct {
	definition *function.Definition
}

// NewLenFunction creates a new instance of the LenFunction with
// a complete function definition.
func NewLenFunction() provider.Function {
	return &LenFunction{
		definition: &function.Definition{
			Description: "Get the length of a string, array, or mapping.",
			FormattedDescription: "Get the length of a string, array, or mapping.\n\n" +
				"**Examples:**\n\n" +
				"```\n${len(values.cacheClusterConfig.endpoints)}\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label: "element",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionScalar{
							Label: "string",
							Type:  function.ValueTypeString,
						},
						&function.ValueTypeDefinitionList{
							Label: "array",
							ElementType: &function.ValueTypeDefinitionAny{
								Label: "any",
								Type:  function.ValueTypeAny,
							},
						},
						&function.ValueTypeDefinitionMap{
							Label: "mapping",
							ElementType: &function.ValueTypeDefinitionAny{
								Label: "any",
								Type:  function.ValueTypeAny,
							},
						},
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing the string, array, or mapping to get the length of.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "integer",
					Type:  function.ValueTypeInt64,
				},
				Description: "The length of the string, array, or mapping. " +
					"For a string, the length is the number of characters. " +
					"For a mapping, the length is the number of key value pairs.",
			},
		},
	}
}

func (f *LenFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *LenFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var value interface{}
	if err := input.Arguments.GetVar(ctx, 0, &value); err != nil {
		return nil, err
	}

	strVal, isStrVal := value.(string)
	if isStrVal {
		return &provider.FunctionCallOutput{
			ResponseData: int64(len(strVal)),
		}, nil
	}

	sliceVal, isSliceVal := value.([]interface{})
	if isSliceVal {
		return &provider.FunctionCallOutput{
			ResponseData: int64(len(sliceVal)),
		}, nil
	}

	mapVal, isMapVal := value.(map[string]interface{})
	if isMapVal {
		return &provider.FunctionCallOutput{
			ResponseData: int64(len(mapVal)),
		}, nil
	}

	return nil, function.NewFuncCallError(
		fmt.Sprintf("invalid input type, expected string, array or mapping, received: %T", value),
		function.FuncCallErrorCodeInvalidInput,
		input.CallContext.CallStackSnapshot(),
	)
}
