package corefunctions

import (
	"context"
	"strings"

	"github.com/two-hundred/celerity/libs/blueprint/function"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
)

// JoinFunction provides the implementation of
// a function that joins an array of strings into a single
// string with a provided delimiter.
type JoinFunction struct {
	definition *function.Definition
}

// NewJoinFunction creates a new instance of the JoinFunction with
// a complete function definition.
func NewJoinFunction() provider.Function {
	return &JoinFunction{
		definition: &function.Definition{
			Description: "Joins an array of strings into a single string using a delimiter.",
			FormattedDescription: "Joins an array of strings into a single string using a delimiter.\n\n" +
				"**Examples:**\n\n" +
				"```\n${join(values.cacheClusterConfig.hosts, \",\")}\n```",
			Parameters: []function.Parameter{
				&function.ListParameter{
					Label: "strings",
					ElementType: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A reference or function call yielding a return value " +
						"representing an array of strings to join together.",
				},
				&function.ScalarParameter{
					Label: "delimiter",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The delimiter to join the strings with.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "A single string that is the result of joining the array of strings with the delimiter.",
			},
		},
	}
}

func (f *JoinFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *JoinFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputStrSlice []string
	var delimiter string
	if err := input.Arguments.GetMultipleVars(ctx, &inputStrSlice, &delimiter); err != nil {
		return nil, err
	}

	joined := strings.Join(inputStrSlice, delimiter)

	return &provider.FunctionCallOutput{
		ResponseData: joined,
	}, nil
}
