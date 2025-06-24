package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// Replace_G_Function provides the implementation of the
// core "replace_g" function defined in the blueprint specification.
type Replace_G_Function struct {
	definition *function.Definition
}

// NewReplace_G_Function creates a new instance of the Replace_G_Function with
// a complete function definition.
func NewReplace_G_Function() provider.Function {
	return &Replace_G_Function{
		definition: &function.Definition{
			Description: "A composable version of the \"replace\" function that takes the search and replace substrings " +
				"as static arguments and returns a function that takes the string to replace the substrings in.",
			FormattedDescription: "A composable version of the \"replace\" function that takes the search and replace substrings " +
				"as static arguments and returns a function that takes the string to replace the substrings in.\n\n" +
				"**Examples:**\n\n" +
				"```\n${map(values.cacheClusterConfig.hosts, replace_g(\"http://\", \"https://\"))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "searchFor",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The \"search\" substring to replace.",
				},
				&function.ScalarParameter{
					Label: "replaceWith",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The substring to replace the \"search\" substring with.",
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
									"representing an input string that contains a substring that needs replacing.",
							},
						},
						Return: &function.ScalarReturn{
							Type: &function.ValueTypeDefinitionScalar{
								Label: "string",
								Type:  function.ValueTypeString,
							},
							Description: "The input string with all occurrences of the \"search\" substring" +
								" replaced with the \"replace\" substring.",
						},
					},
				},
				Description: "A function that takes a string and returns the substring from the string using " +
					"the pre-configured \"search\" and \"replace\" strings.",
			},
		},
	}
}

func (f *Replace_G_Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *Replace_G_Function) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var search string
	var replace string
	err := input.Arguments.GetMultipleVars(ctx, &search, &replace)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "replace",
			PartialArgs:  []interface{}{search, replace},
			// The input string is passed as the first argument to the replace function.
			ArgsOffset: 1,
		},
	}, nil
}
