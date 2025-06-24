package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// TrimSuffix_G_Function provides the implementation of the
// core "trimsuffix_g" function defined in the blueprint specification.
type TrimSuffix_G_Function struct {
	definition *function.Definition
}

// NewTrimSuffix_G_Function creates a new instance of the TrimSuffix_G_Function with
// a complete function definition.
func NewTrimSuffix_G_Function() provider.Function {
	return &TrimSuffix_G_Function{
		definition: &function.Definition{
			Description: "A composable version of the \"trimsuffix\" function that takes the suffix as a " +
				"static argument and returns a function that takes the string to remove the suffix from.",
			FormattedDescription: "A composable version of the \"trimsuffix\" function that takes the suffix as a " +
				"static argument and returns a function that takes the string to remove the suffix from.\n\n" +
				"**Examples:**\n\n" +
				"```\n${map(variables,cacheClusterConfig.hosts, trimsuffix_g(\"/config\"))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "suffix",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The suffix to remove from the string.",
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
									"representing an input string to remove the pre-configured suffix from.",
							},
						},
						Return: &function.ScalarReturn{
							Type: &function.ValueTypeDefinitionScalar{
								Label: "string",
								Type:  function.ValueTypeString,
							},
							Description: "The input string with the suffix removed.",
						},
					},
				},
				Description: "A function that takes a string and returns the string with " +
					"the pre-configured suffix removed.",
			},
		},
	}
}

func (f *TrimSuffix_G_Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *TrimSuffix_G_Function) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var suffix string
	err := input.Arguments.GetVar(ctx, 0, &suffix)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "trimsuffix",
			PartialArgs:  []interface{}{suffix},
			// The input string is passed as the first argument to the trimsuffix function.
			ArgsOffset: 1,
		},
	}, nil
}
