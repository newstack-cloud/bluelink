package corefunctions

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
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
	// Get the first argument as any to check if it's none
	firstArg, err := input.Arguments.Get(ctx, 0)
	if err != nil {
		return nil, err
	}

	// If the input array is none, return none
	if core.IsNoneMarker(firstArg) {
		return &provider.FunctionCallOutput{
			ResponseData: core.GetNoneMarker(),
		}, nil
	}

	var inputStrSlice []string
	var delimiter string
	if err := input.Arguments.GetMultipleVars(ctx, &inputStrSlice, &delimiter); err != nil {
		return nil, err
	}

	// There is no need to filter out none values from the slice before joining.
	// This is because GetMultipleVars converts to []string, so none markers would have
	// already been converted. We need to handle this at a different level.
	// For now, the filtering happens at the array resolution level in resolveInMappingNodeSlice.

	joined := strings.Join(inputStrSlice, delimiter)

	return &provider.FunctionCallOutput{
		ResponseData: joined,
	}, nil
}
