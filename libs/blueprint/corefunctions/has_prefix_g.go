package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// HasPrefix_G_Function provides the implementation of the
// core "has_prefix_g" function defined in the blueprint specification.
type HasPrefix_G_Function struct {
	definition *function.Definition
}

// NewHasPrefix_G_Function creates a new instance of the HasPrefix_G_Function with
// a complete function definition.
func NewHasPrefix_G_Function() provider.Function {
	return &HasPrefix_G_Function{
		definition: &function.Definition{
			Description: "A composable version of the \"has_prefix\" function that takes the prefix as a " +
				"static argument and returns a function that takes the string to check for the prefix in.",
			FormattedDescription: "A composable version of the \"has_prefix\" function that takes the prefix as a " +
				"static argument and returns a function that takes the string to check for the prefix in.\n\n" +
				"**Examples:**\n\n" +
				"```\n${filter(\nvalues.cacheClusterConfig.hosts,\nhas_prefix_g(\"http://\")\n)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "prefix",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The prefix to check for at the start of the string.",
				},
			},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.ScalarParameter{
								Label: "input",
								Type: &function.ValueTypeDefinitionScalar{
									Label: "string",
									Type:  function.ValueTypeString,
								},
								Description: "A valid string literal, reference or function call yielding a return value " +
									"representing an input string to check for the pre-configured prefix in.",
							},
						},
						Return: &function.ScalarReturn{
							Type: &function.ValueTypeDefinitionScalar{
								Label: "boolean",
								Type:  function.ValueTypeBool,
							},
							Description: "True, if the string starts with the prefix, false otherwise.",
						},
					},
				},
				Description: "A function that takes a string and returns whether or not it has the pre-configured prefix.",
			},
		},
	}
}

func (f *HasPrefix_G_Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *HasPrefix_G_Function) Call(
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
			FunctionName: "has_prefix",
			PartialArgs:  []interface{}{prefix},
			// The input string is passed as the first argument to the hasprefix function.
			ArgsOffset: 1,
		},
	}, nil
}
