package testprovider

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func functionTrimSuffix() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: TrimSuffixFunctionDefinition(),
		CallFunc:   trimSuffix,
	}
}

// TrimSuffixFunctionDefinition returns the definition of the function that removes a suffix from a string.
func TrimSuffixFunctionDefinition() *function.Definition {
	return &function.Definition{
		Name:        "trim_suffix",
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
	}
}

func trimSuffix(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
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
