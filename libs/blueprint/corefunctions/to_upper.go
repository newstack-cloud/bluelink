package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
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
			"argument to `to_upper` must be a string",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: strings.ToUpper(inputStr),
	}, nil
}
