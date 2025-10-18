package corefunctions

import (
	"context"

	"github.com/google/uuid"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// UUIDFunction provides the implementation of
// a function that generates a version 4 UUID.
type UUIDFunction struct {
	definition *function.Definition
}

// NewUUIDFunction creates a new instance of the UUIDFunction with
// a complete function definition.
func NewUUIDFunction() provider.Function {
	return &UUIDFunction{
		definition: &function.Definition{
			Description: "A function that generates a version 4 UUID (random UUID) and returns it as a string.",
			FormattedDescription: "A function that generates a version 4 UUID (random UUID) and returns it as a string.\n\n" +
				"**Examples:**\n\n" +
				"Generating a UUID:\n" +
				"```\n${uuid()}\n```\n\n" +
				"Using UUID for unique resource names:\n" +
				"```\nresource-${uuid()}\n```",
			Parameters: []function.Parameter{},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "A version 4 UUID string.",
			},
		},
	}
}

func (f *UUIDFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *UUIDFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	newUUID := uuid.New()
	return &provider.FunctionCallOutput{
		ResponseData: newUUID.String(),
	}, nil
}
