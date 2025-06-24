package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// NamedArgument is a struct that represents a named argument.
// This must be passed into functions that expect named arguments
// such as the `object` function.
type NamedArgument struct {
	Name  string
	Value any
}

// ObjectFunction provides the implementation of
// a function that checks if a string has a suffix.
type ObjectFunction struct {
	definition *function.Definition
}

// NewObjectFunction creates a new instance of the ObjectFunction with
// a complete function definition.
func NewObjectFunction() provider.Function {
	return &ObjectFunction{
		definition: &function.Definition{
			Description: "Creates an object from named arguments.",
			FormattedDescription: "Creates an object from named arguments.\n\n" +
				"**Examples:**\n\n" +
				"```\n${object(id=\"subnet-1234\", label=\"Subnet 1234\")}\n```",
			Parameters: []function.Parameter{
				&function.VariadicParameter{
					Label: "attributes",
					Type: &function.ValueTypeDefinitionAny{
						Type:  function.ValueTypeAny,
						Label: "any",
					},
					Named: true,
					Description: "N named arguments that will be used to create an object/mapping. " +
						"When no arguments are passed, an empty object should be returned.",
				},
			},
			Return: &function.ObjectReturn{
				Description: "An object containing attributes that have been passed as named arguments.",
			},
		},
	}
}

func (f *ObjectFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ObjectFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var params []NamedArgument
	if err := input.Arguments.GetVar(ctx, 0, &params); err != nil {
		return nil, err
	}

	object := make(map[string]any)
	for _, param := range params {
		object[param.Name] = param.Value
	}

	return &provider.FunctionCallOutput{
		ResponseData: object,
	}, nil
}
