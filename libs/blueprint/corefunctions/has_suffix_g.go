package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// HasSuffix_G_Function provides the implementation of the
// core "has_suffix_g" function defined in the blueprint specification.
type HasSuffix_G_Function struct {
	definition *function.Definition
}

// NewHasSuffix_G_Function creates a new instance of the HasSuffix_G_Function with
// a complete function definition.
func NewHasSuffix_G_Function() provider.Function {
	return &HasSuffix_G_Function{
		definition: &function.Definition{
			Description: "A composable version of the \"has_suffix\" function that takes the suffix as a " +
				"static argument and returns a function that takes the string to check for the suffix in.",
			FormattedDescription: "A composable version of the \"has_suffix\" function that takes the suffix as a " +
				"static argument and returns a function that takes the string to check for the suffix in.\n\n" +
				"**Examples:**\n\n" +
				"```\n${filter(\nvalues.cacheClusterConfig.hosts,\nhas_suffix_g(\"/config\")\n)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "suffix",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The suffix to check for at the end of the string.",
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
									"representing an input string to check for the pre-configured suffix in.",
							},
						},
						Return: &function.ScalarReturn{
							Type: &function.ValueTypeDefinitionScalar{
								Label: "boolean",
								Type:  function.ValueTypeBool,
							},
							Description: "True, if the string starts with the suffix, false otherwise.",
						},
					},
				},
				Description: "A function that takes a string and returns whether or not it has the pre-configured suffix.",
			},
		},
	}
}

func (f *HasSuffix_G_Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *HasSuffix_G_Function) Call(
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
			FunctionName: "has_suffix",
			PartialArgs:  []interface{}{suffix},
			// The input string is passed as the first argument to the has_suffix function.
			ArgsOffset: 1,
		},
	}, nil
}
