package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// TrimPrefix_G_Function provides the implementation of the
// core "trimprefix_g" function defined in the blueprint specification.
type TrimPrefix_G_Function struct {
	definition *function.Definition
}

// NewTrimPrefix_G_Function creates a new instance of the TrimPrefix_G_Function with
// a complete function definition.
func NewTrimPrefix_G_Function() provider.Function {
	return &TrimPrefix_G_Function{
		definition: &function.Definition{
			Description: "A composable version of the \"trimprefix\" function that takes the prefix as a " +
				"static argument and returns a function that takes the string to remove the prefix from.",
			FormattedDescription: "A composable version of the \"trimprefix\" function that takes the prefix as a " +
				"static argument and returns a function that takes the string to remove the prefix from.\n\n" +
				"**Examples:**\n\n" +
				"```\n${map(variables,cacheClusterConfig.hosts, trimprefix_g(\"http://\"))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "prefix",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The prefix to remove from the string.",
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
									"representing an input string to remove the pre-configured prefix from.",
							},
						},
						Return: &function.ScalarReturn{
							Type: &function.ValueTypeDefinitionScalar{
								Label: "string",
								Type:  function.ValueTypeString,
							},
							Description: "The input string with the prefix removed.",
						},
					},
				},
				Description: "A function that takes a string and returns the string with " +
					"the pre-configured prefix removed.",
			},
		},
	}
}

func (f *TrimPrefix_G_Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *TrimPrefix_G_Function) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var prefix string
	err := input.Arguments.GetVar(ctx, 0, &prefix)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "trimprefix",
			PartialArgs:  []interface{}{prefix},
			// The input string is passed as the first argument to the trimprefix function.
			ArgsOffset: 1,
		},
	}, nil
}
