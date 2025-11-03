package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// TrimSuffixFunction provides the implementation of
// a function that trims a prefix from a string.
type TrimSuffixFunction struct {
	definition *function.Definition
}

// NewTrimSuffixFunction creates a new instance of the TrimSuffixFunction with
// a complete function definition.
func NewTrimSuffixFunction() provider.Function {
	return &TrimSuffixFunction{
		definition: &function.Definition{
			Description: "Removes a suffix from a string.",
			FormattedDescription: "Removes a suffix from a string.\n\n" +
				"**Examples:**\n\n" +
				"```\n${trimsuffix(values.cacheClusterConfig.host, \":3000\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "input",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing the string to remove the suffix from.",
				},
				&function.ScalarParameter{
					Label: "suffix",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The suffix to remove from the string.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The input string with the suffix removed.",
			},
		},
	}
}

func (f *TrimSuffixFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *TrimSuffixFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	// Get first argument as any to check for none marker
	inputAny, err := input.Arguments.Get(ctx, 0)
	if err != nil {
		return nil, err
	}

	// If input is none, propagate none
	if core.IsNoneMarker(inputAny) {
		return &provider.FunctionCallOutput{
			ResponseData: core.GetNoneMarker(),
		}, nil
	}

	var inputStr string
	var suffix string
	if err := input.Arguments.GetMultipleVars(ctx, &inputStr, &suffix); err != nil {
		return nil, err
	}

	outputStr := strings.TrimSuffix(inputStr, suffix)

	return &provider.FunctionCallOutput{
		ResponseData: outputStr,
	}, nil
}
