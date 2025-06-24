package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ToUpperFunction provides the implementation of
// a function that converts a string to uppercase.
type ToUpperFunction struct {
	definition *function.Definition
}

// NewToUpperFunction creates a new instance of the NewToUpperFunction with
// a complete function definition.
func NewToUpperFunction() provider.Function {
	return &ToUpperFunction{
		definition: &function.Definition{
			Description: "Converts all characters of a string to upper case.",
			FormattedDescription: "Converts all characters of a string to upper case.\n\n" +
				"**Examples:**\n\n" +
				"```\n${to_upper(values.cacheClusterConfig.hostName)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "input",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing the string to convert to upper case.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The input string with all characters converted to upper case.",
			},
		},
	}
}

func (f *ToUpperFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ToUpperFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStr string
	if err := input.Arguments.GetVar(ctx, 0, &inputStr); err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		ResponseData: strings.ToUpper(inputStr),
	}, nil
}
