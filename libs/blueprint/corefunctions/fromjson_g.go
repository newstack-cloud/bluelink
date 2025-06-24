package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// FromJSON_G_Function provides the implementation of the
// core "fromjson_g" function defined in the blueprint specification.
type FromJSON_G_Function struct {
	definition *function.Definition
}

// NewFromJSON_G_Function creates a new instance of the FromJSON_G_Function with
// a complete function definition.
func NewFromJSON_G_Function() provider.Function {
	return &FromJSON_G_Function{
		definition: &function.Definition{
			Description: "A composable version of the \"fromjson\" function that extracts a value from a serialised JSON string. " +
				"This uses json pointer notation to allow for the extraction of values from complex " +
				"serialised structures.",
			FormattedDescription: "A composable version of the `fromjson` function that extracts a value from a serialised JSON string. " +
				"This uses [json pointer notation](https://datatracker.ietf.org/doc/rfc6901/) " +
				"to allow for the extraction of values from complex serialised structures.\n\n" +
				"**Examples:**\n\n" +
				"```\n${map(variables.cacheClusterConfigDefs, fromjson_g(\"/host\"))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "jsonPointer",
					Type: &function.ValueTypeDefinitionScalar{
						Label:       "JSON pointer",
						Type:        function.ValueTypeString,
						Description: "A valid json pointer expression to extract the value from the json string.",
						FormattedDescription: "A valid [json pointer expression](https://datatracker.ietf.org/doc/rfc6901/) " +
							"to extract the value from the json string.",
					},
				},
			},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.ScalarParameter{
								Label: "jsonString",
								Type: &function.ValueTypeDefinitionScalar{
									Label: "JSON string",
									Type:  function.ValueTypeString,
									Description: "A valid string literal, reference or function" +
										" call yielding the json string to extract values from",
								},
							},
						},
						Return: &function.AnyReturn{
							Type: function.ValueTypeAny,
							Description: "The value extracted from the json string. " +
								"This can be a primitive value, an array, mapping or object",
						},
					},
				},
				Description: "A function that takes a json string and extracts a value from it using " +
					"the pre-configured json pointer expression.",
			},
		},
	}
}

func (f *FromJSON_G_Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *FromJSON_G_Function) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var jsonPtr string
	err := input.Arguments.GetVar(ctx, 0, &jsonPtr)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "fromjson",
			PartialArgs:  []interface{}{jsonPtr},
			// The JSON string is passed as the first argument to the fromjson function.
			ArgsOffset: 1,
		},
	}, nil
}
