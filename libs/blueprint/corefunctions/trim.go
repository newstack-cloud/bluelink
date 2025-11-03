package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
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
	// Get argument as any to check for none marker
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

	// Convert to string and perform operation
	inputStr, ok := inputAny.(string)
	if !ok {
		return nil, function.NewFuncCallError(
			"argument to `trim` must be a string",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: strings.TrimSpace(inputStr),
	}, nil
}
