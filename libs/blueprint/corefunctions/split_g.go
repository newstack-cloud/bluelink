package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// Split_G_Function provides the implementation of the
// core "split_g" function defined in the blueprint specification.
type Split_G_Function struct {
	definition *function.Definition
}

// NewSplit_G_Function creates a new instance of the Split_G_Function with
// a complete function definition.
func NewSplit_G_Function() provider.Function {
	return &Split_G_Function{
		definition: &function.Definition{
			Description: "A composable version of the \"split\" function that takes the delimiter as a " +
				"static argument and returns a function that takes the string to split.",
			FormattedDescription: "A composable version of the \"split\" function that takes the delimiter as a " +
				"static argument and returns a function that takes the string to split.\n\n" +
				"**Examples:**\n\n" +
				"```\n${flatmap(values.cacheClusterConfig.multiClusterHosts, split_g(\",\"))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "delimiter",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The delimiter to split the string by.",
				},
			},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.ScalarParameter{
								Type: &function.ValueTypeDefinitionScalar{
									Label: "string",
									Type:  function.ValueTypeString,
								},
								Description: "A valid string literal, reference or function call yielding a return value " +
									"representing an input string to split.",
							},
						},
						Return: &function.ListReturn{
							ElementType: &function.ValueTypeDefinitionScalar{
								Label: "string",
								Type:  function.ValueTypeString,
							},
							Description: "An array of substrings that have been split by the delimiter.",
						},
					},
				},
				Description: "A function that takes a string and returns an array of " +
					"strings as a result of splitting a given string by a pre-configured delimiter.",
			},
		},
	}
}

func (f *Split_G_Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *Split_G_Function) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var delimiter string
	err := input.Arguments.GetVar(ctx, 0, &delimiter)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "split",
			PartialArgs:  []interface{}{delimiter},
			// The input string is passed as the first argument to the split function.
			ArgsOffset: 1,
		},
	}, nil
}
