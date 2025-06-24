package testprovider

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func functionTrimSpaceAndSuffix() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: TrimSpaceAndSuffixFunctionDefinition(),
		CallFunc:   trimSpaceAndSuffix,
	}
}

// TrimSpaceAndSuffixFunctionDefinition returns the definition of the function
// removes leading and trailing white space, then removes the specified suffix from the string.
func TrimSpaceAndSuffixFunctionDefinition() *function.Definition {
	return &function.Definition{
		Name:        "trim_space_and_suffix",
		Description: "Removes white space and a specified suffix from a string.",
		FormattedDescription: "Removes white space and specified suffix from a string.\n\n" +
			"**Examples:**\n\n" +
			"```\n${trim_space_and_suffix(values.cacheClusterConfig.host, \":3000\")}\n```",
		Parameters: []function.Parameter{
			&function.ScalarParameter{
				Label: "input",
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "A valid string literal, reference or function call yielding a return value " +
					"representing the string to remove white space and the given suffix from.",
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
			Description: "The input string with leading and trailing white space removed along with the suffix.",
		},
	}
}

func trimSpaceAndSuffix(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStr string
	var suffix string
	if err := input.Arguments.GetMultipleVars(ctx, &inputStr, &suffix); err != nil {
		return nil, err
	}

	spaceRemoved := strings.TrimSpace(inputStr)

	registry := input.CallContext.Registry()
	args := input.CallContext.NewCallArgs(spaceRemoved, suffix)
	trimSuffixResp, err := registry.Call(
		ctx,
		"trim_suffix",
		&provider.FunctionCallInput{
			Arguments:   args,
			CallContext: input.CallContext,
		},
	)
	if err != nil {
		return nil, err
	}

	return trimSuffixResp, nil
}
