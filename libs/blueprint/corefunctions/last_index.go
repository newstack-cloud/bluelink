package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// LastIndexFunction provides the implementation of
// a function that joins an array of strings into a single
// string with a provided delimiter.
type LastIndexFunction struct {
	definition *function.Definition
}

// NewLastIndexFunction creates a new instance of the LastIndexFunction with
// a complete function definition.
func NewLastIndexFunction() provider.Function {
	return &LastIndexFunction{
		definition: &function.Definition{
			Description: "Gets the last index of a substring in a given string.",
			FormattedDescription: "Gets the last index of a substring in a given string.\n\n" +
				"**Examples:**\n\n" +
				"```\n${last_index(values.cacheClusterConfig.host, \":3000\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "searchIn",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing a string to search for the substring in.",
				},
				&function.ScalarParameter{
					Label: "substring",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The substring to search for in the string.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "integer",
					Type:  function.ValueTypeInt64,
				},
				Description: "The index of the last occurence of the substring in the string. " +
					"This will be -1 if the substring is not found in the string.",
			},
		},
	}
}

func (f *LastIndexFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *LastIndexFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var haystack string
	var needle string
	if err := input.Arguments.GetMultipleVars(ctx, &haystack, &needle); err != nil {
		return nil, err
	}

	index := strings.LastIndex(haystack, needle)

	return &provider.FunctionCallOutput{
		ResponseData: int64(index),
	}, nil
}
