package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/celerity/libs/blueprint/function"
	"github.com/newstack-cloud/celerity/libs/blueprint/provider"
)

// IndexFunction provides the implementation of
// a function that joins an array of strings into a single
// string with a provided delimiter.
type IndexFunction struct {
	definition *function.Definition
}

// NewIndexFunction creates a new instance of the IndexFunction with
// a complete function definition.
func NewIndexFunction() provider.Function {
	return &IndexFunction{
		definition: &function.Definition{
			Description: "Gets the first index of a substring in a given string.",
			FormattedDescription: "Gets the first index of a substring in a given string.\n\n" +
				"**Examples:**\n\n" +
				"```\n${index(values.cacheClusterConfig.host, \":3000\")}\n```",
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
				Description: "The index of the first occurence of the substring in the string. " +
					"This will be -1 if the substring is not found in the string.",
			},
		},
	}
}

func (f *IndexFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *IndexFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var haystack string
	var needle string
	if err := input.Arguments.GetMultipleVars(ctx, &haystack, &needle); err != nil {
		return nil, err
	}

	index := strings.Index(haystack, needle)

	return &provider.FunctionCallOutput{
		ResponseData: int64(index),
	}, nil
}
