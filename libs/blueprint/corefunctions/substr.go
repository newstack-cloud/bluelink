package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// SubstrFunction provides the implementation of
// a function that extracts a substring from a string.
type SubstrFunction struct {
	definition *function.Definition
}

// NewSubstrFunction creates a new instance of the SubstrFunction with
// a complete function definition.
func NewSubstrFunction() provider.Function {
	return &SubstrFunction{
		definition: &function.Definition{
			Description: "Extracts a substring from the given string.",
			FormattedDescription: "Extracts a substring from the given string.\n\n" +
				"**Examples:**\n\n" +
				"```\n${substr(values.cacheClusterConfig.host, 0, 3)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "input",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing the string to extract the substring from.",
				},
				&function.ScalarParameter{
					Label: "start",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "integer",
						Type:  function.ValueTypeInt64,
					},
					Description: "The index of the first character to include in the substring.",
				},
				&function.ScalarParameter{
					Label: "end",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "integer",
						Type:  function.ValueTypeInt64,
					},
					Optional: true,
					Description: "The index of the last character to include in the substring. " +
						"If not provided, the substring will include all characters from the start index to the end of the string.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The substring extracted from the provided string.",
			},
		},
	}
}

func (f *SubstrFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *SubstrFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStr string
	var start int64
	if err := input.Arguments.GetMultipleVars(ctx, &inputStr, &start); err != nil {
		return nil, err
	}

	var end int64
	if err := input.Arguments.GetVar(ctx, 2, &end); err != nil {
		end = int64(len(inputStr))
	}

	if start > end {
		return nil, function.NewFuncCallError(
			"start index cannot be greater than end index",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	if start < 0 || end < 0 {
		return nil, function.NewFuncCallError(
			"start and end indices cannot be negative",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	if start > int64(len(inputStr)-1) {
		return nil, function.NewFuncCallError(
			"start index cannot be greater than the last element index in the string",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	if end > int64(len(inputStr)) {
		return nil, function.NewFuncCallError(
			"end index cannot be greater than the length of the string",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: inputStr[start:end],
	}, nil
}
