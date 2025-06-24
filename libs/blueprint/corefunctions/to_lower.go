package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ToLowerFunction provides the implementation of
// a function that converts a string to uppercase.
type ToLowerFunction struct {
	definition *function.Definition
}

// NewToLowerFunction creates a new instance of the NewToLowerFunction with
// a complete function definition.
func NewToLowerFunction() provider.Function {
	return &ToLowerFunction{
		definition: &function.Definition{
			Description: "Converts all characters of a string to lower case.",
			FormattedDescription: "Converts all characters of a string to lower case.\n\n" +
				"**Examples:**\n\n" +
				"```\n${to_lower(values.cacheClusterConfig.hostId)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "input",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing the string to convert to lower case.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The input string with all characters converted to lower case.",
			},
		},
	}
}

func (f *ToLowerFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ToLowerFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStr string
	if err := input.Arguments.GetVar(ctx, 0, &inputStr); err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		ResponseData: strings.ToLower(inputStr),
	}, nil
}
