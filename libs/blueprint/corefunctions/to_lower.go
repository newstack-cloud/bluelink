package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
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
			"argument to `to_lower` must be a string",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: strings.ToLower(inputStr),
	}, nil
}
