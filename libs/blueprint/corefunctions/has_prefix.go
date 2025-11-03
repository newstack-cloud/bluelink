package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// HasPrefixFunction provides the implementation of
// a function that checks if a string has a prefix.
type HasPrefixFunction struct {
	definition *function.Definition
}

// NewHasPrefixFunction creates a new instance of the HasPrefixFunction with
// a complete function definition.
func NewHasPrefixFunction() provider.Function {
	return &HasPrefixFunction{
		definition: &function.Definition{
			Description: "Checks if a string starts with a given substring.",
			FormattedDescription: "Checks if a string starts with a given substring.\n\n" +
				"**Examples:**\n\n" +
				"```\n${has_prefix(values.cacheClusterConfig.host, \"http://\")}\n```",
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
					Label: "prefix",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The prefix to check for at the start of the string.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "boolean",
					Type:  function.ValueTypeBool,
				},
				Description: "True, if the string starts with the prefix, false otherwise.",
			},
		},
	}
}

func (f *HasPrefixFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *HasPrefixFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	// Get first argument as any to check for none marker
	inputAny, err := input.Arguments.Get(ctx, 0)
	if err != nil {
		return nil, err
	}

	// If input is none, return false (none strings don't have prefixes)
	if core.IsNoneMarker(inputAny) {
		return &provider.FunctionCallOutput{
			ResponseData: false,
		}, nil
	}

	var inputStr string
	var prefix string
	if err := input.Arguments.GetMultipleVars(ctx, &inputStr, &prefix); err != nil {
		return nil, err
	}

	hasPrefix := strings.HasPrefix(inputStr, prefix)

	return &provider.FunctionCallOutput{
		ResponseData: hasPrefix,
	}, nil
}
