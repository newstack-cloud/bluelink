package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// SplitFunction provides the implementation of
// a function that splits a string into an array of substrings based on a delimiter.
type SplitFunction struct {
	definition *function.Definition
}

// NewSplitFunction creates a new instance of the SplitFunction with
// a complete function definition.
func NewSplitFunction() provider.Function {
	return &SplitFunction{
		definition: &function.Definition{
			Description: "Splits a string into an array of substrings based on a delimiter.",
			FormattedDescription: "Splits a string into an array of substrings based on a delimiter.\n\n" +
				"**Examples:**\n\n" +
				"```\n${split(values.cacheClusterConfig.hosts, \",\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "input",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing an input string to split.",
				},
				&function.ScalarParameter{
					Label: "delimiter",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The delimiter to split the string by.",
				},
			},
			Return: &function.ListReturn{
				ElementType: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "An array of substrings that have been split by the delimiter.",
			},
		},
	}
}

func (f *SplitFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *SplitFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStr string
	var delimiter string
	if err := input.Arguments.GetMultipleVars(ctx, &inputStr, &delimiter); err != nil {
		return nil, err
	}

	split := strings.Split(inputStr, delimiter)
	splitInterfaceSlice := intoInterfaceSlice(split)

	return &provider.FunctionCallOutput{
		ResponseData: splitInterfaceSlice,
	}, nil
}
