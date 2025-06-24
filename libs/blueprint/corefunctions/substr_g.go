package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// Substr_G_Function provides the implementation of the
// core "substr_g" function defined in the blueprint specification.
type Substr_G_Function struct {
	definition *function.Definition
}

// NewSubstr_G_Function creates a new instance of the Substr_G_Function with
// a complete function definition.
func NewSubstr_G_Function() provider.Function {
	return &Substr_G_Function{
		definition: &function.Definition{
			Description: "A composable version of the \"substr\" function that takes the start and end indexes " +
				"as static arguments and returns a function that takes the string to get the substring from.",
			FormattedDescription: "A composable version of the \"substr\" function that takes the start and end indexes " +
				"as static arguments and returns a function that takes the string to get the substring from.\n\n" +
				"**Examples:**\n\n" +
				"```\n${map(values.cacheClusterConfig.hosts, substr_g(0, 3))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "start",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "integer",
						Type:  function.ValueTypeInt64,
					},
					Description: "The index of the first character to include in the substring.",
				},
				&function.ScalarParameter{
					Label: "end",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "integer",
						Type:  function.ValueTypeInt64,
					},
					Optional: true,
					Description: "The index of the last character to include in the substring. " +
						"If not provided, the substring will include all characters from the start index to the end of the string.",
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
									"representing the string to extract the substring from.",
							},
						},
						Return: &function.ScalarReturn{
							Type: &function.ValueTypeDefinitionScalar{
								Label: "string",
								Type:  function.ValueTypeString,
							},
							Description: "The substring extracted from the provided string.",
						},
					},
				},
				Description: "A function that takes a string and returns the substring from the string using " +
					"the pre-configured start and end positions.",
			},
		},
	}
}

func (f *Substr_G_Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *Substr_G_Function) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var start int64
	err := input.Arguments.GetVar(ctx, 0, &start)
	if err != nil {
		return nil, err
	}

	var end int64
	if err := input.Arguments.GetVar(ctx, 1, &end); err != nil {
		end = int64(-1)
	}

	if end == -1 {
		return &provider.FunctionCallOutput{
			FunctionInfo: provider.FunctionRuntimeInfo{
				FunctionName: "substr",
				PartialArgs:  []interface{}{start},
				// The input string is passed as the first argument to the substr function.
				ArgsOffset: 1,
			},
		}, nil
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "substr",
			PartialArgs:  []interface{}{start, end},
			// The input string is passed as the first argument to the substr function.
			ArgsOffset: 1,
		},
	}, nil
}
