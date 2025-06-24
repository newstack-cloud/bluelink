package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// Contains_G_Function provides the implementation of the
// core "contains_g" function defined in the blueprint specification.
type Contains_G_Function struct {
	definition *function.Definition
}

// NewContains_G_Function creates a new instance of the Contains_G_Function with
// a complete function definition.
func NewContains_G_Function() provider.Function {
	return &Contains_G_Function{
		definition: &function.Definition{
			Description: "A composable version of the \"contains\" function that takes the search value as a " +
				"static argument and returns a function that takes the string or array to search in.",
			FormattedDescription: "A composable version of the \"contains\" function that takes the search value as a " +
				"static argument and returns a function that takes the string or array to search in.\n\n" +
				"**Examples:**\n\n" +
				"```\n${filter(\nvalues.cacheClusterConfig.hosts,\ncontains_g(\"bluelink.dev\")\n)}\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label:       "substring",
					Description: "The substring or value to search for in the string or array.",
				},
			},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.AnyParameter{
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
								},
								Description: "A valid string literal, reference or function call yielding a return value " +
									"representing a string or array to search.",
							},
						},
						Return: &function.ScalarReturn{
							Type: &function.ValueTypeDefinitionScalar{
								Label: "boolean",
								Type:  function.ValueTypeBool,
							},
							Description: "True, if the substring or value is found in the string or array, false otherwise.",
						},
					},
				},
				Description: "A function that takes a string or array and returns whether or not it contains the pre-configured search value.",
			},
		},
	}
}

func (f *Contains_G_Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *Contains_G_Function) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var needle interface{}
	err := input.Arguments.GetVar(ctx, 0, &needle)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "contains",
			PartialArgs:  []interface{}{needle},
			// The input string is passed as the first argument to the contains function.
			ArgsOffset: 1,
		},
	}, nil
}
