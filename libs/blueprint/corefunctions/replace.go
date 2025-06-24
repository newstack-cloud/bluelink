package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ReplaceFunction provides the implementation of
// a function that replaces all occurrences of a substring in a string
// with another substring.
type ReplaceFunction struct {
	definition *function.Definition
}

// NewReplaceFunction creates a new instance of the ReplaceFunction with
// a complete function definition.
func NewReplaceFunction() provider.Function {
	return &ReplaceFunction{
		definition: &function.Definition{
			Description: "Replaces all occurrences of a substring in a string with another substring.",
			FormattedDescription: "Replaces all occurrences of a substring in a string with another substring.\n\n" +
				"**Examples:**\n\n" +
				"```\n${replace(values.cacheClusterConfig.host, \"http://\", \"https://\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "searchIn",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing an input string that contains a substring that needs replacing.",
				},
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
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The input string with all occurrences of the \"search\" substring" +
					" replaced with the \"replace\" substring.",
			},
		},
	}
}

func (f *ReplaceFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ReplaceFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStr string
	var toReplace string
	var replaceWith string
	if err := input.Arguments.GetMultipleVars(ctx, &inputStr, &toReplace, &replaceWith); err != nil {
		return nil, err
	}

	replaced := strings.ReplaceAll(inputStr, toReplace, replaceWith)

	return &provider.FunctionCallOutput{
		ResponseData: replaced,
	}, nil
}
