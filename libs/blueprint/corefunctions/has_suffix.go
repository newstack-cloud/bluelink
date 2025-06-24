package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// HasSuffixFunction provides the implementation of
// a function that checks if a string has a suffix.
type HasSuffixFunction struct {
	definition *function.Definition
}

// NewHasSuffixFunction creates a new instance of the HasSuffixFunction with
// a complete function definition.
func NewHasSuffixFunction() provider.Function {
	return &HasSuffixFunction{
		definition: &function.Definition{
			Description: "Checks if a string ends with a given substring.",
			FormattedDescription: "Checks if a string ends with a given substring.\n\n" +
				"**Examples:**\n\n" +
				"```\n${has_suffix(values.cacheClusterConfig.host, \"/config\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "input",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing the string to check.",
				},
				&function.ScalarParameter{
					Label: "suffix",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The suffix to check for at the end of the string.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "boolean",
					Type:  function.ValueTypeBool,
				},
				Description: "True, if the string ends with the suffix, false otherwise.",
			},
		},
	}
}

func (f *HasSuffixFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *HasSuffixFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStr string
	var suffix string
	if err := input.Arguments.GetMultipleVars(ctx, &inputStr, &suffix); err != nil {
		return nil, err
	}

	hasSuffix := strings.HasSuffix(inputStr, suffix)

	return &provider.FunctionCallOutput{
		ResponseData: hasSuffix,
	}, nil
}
