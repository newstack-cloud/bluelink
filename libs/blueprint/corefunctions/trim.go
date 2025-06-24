package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// TrimFunction provides the implementation of
// a function that trims leading and trailing whitespace
// from a string.
type TrimFunction struct {
	definition *function.Definition
}

// NewTrimFunction creates a new instance of the NewTrimFunction with
// a complete function definition.
func NewTrimFunction() provider.Function {
	return &TrimFunction{
		definition: &function.Definition{
			Description: "Removes leading and trailing whitespace from a string.",
			FormattedDescription: "Removes leading and trailing whitespace from a string.\n\n" +
				"**Examples:**\n\n" +
				"```\n${trim(values.cacheClusterConfig.host)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "input",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing the string to trim.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The input string with all leading and trailing whitespace removed.",
			},
		},
	}
}

func (f *TrimFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *TrimFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStr string
	if err := input.Arguments.GetVar(ctx, 0, &inputStr); err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		ResponseData: strings.TrimSpace(inputStr),
	}, nil
}
