package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// TrimPrefixFunction provides the implementation of
// a function that trims a prefix from a string.
type TrimPrefixFunction struct {
	definition *function.Definition
}

// NewTrimPrefixFunction creates a new instance of the TrimPrefixFunction with
// a complete function definition.
func NewTrimPrefixFunction() provider.Function {
	return &TrimPrefixFunction{
		definition: &function.Definition{
			Description: "Removes a prefix from a string.",
			FormattedDescription: "Removes a prefix from a string.\n\n" +
				"**Examples:**\n\n" +
				"```\n${trimprefix(values.cacheClusterConfig.host, \"http://\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "input",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing the string to remove the prefix from.",
				},
				&function.ScalarParameter{
					Label: "prefix",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The prefix to remove from the string.",
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
	}
}

func (f *TrimPrefixFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *TrimPrefixFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStr string
	var prefix string
	if err := input.Arguments.GetMultipleVars(ctx, &inputStr, &prefix); err != nil {
		return nil, err
	}

	outputStr := strings.TrimPrefix(inputStr, prefix)

	return &provider.FunctionCallOutput{
		ResponseData: outputStr,
	}, nil
}
