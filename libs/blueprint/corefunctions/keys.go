package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// KeysFunction provides the implementation of
// a function that checks if a string has a suffix.
type KeysFunction struct {
	definition *function.Definition
}

// NewKeysFunction creates a new instance of the KeysFunction with
// a complete function definition.
func NewKeysFunction() provider.Function {
	return &KeysFunction{
		definition: &function.Definition{
			Description: "Produces an array of keys from a mapping or attribute names from an object.",
			FormattedDescription: "Produces an array of keys from a mapping or attribute names from an object.\n\n" +
				"**Examples:**\n\n" +
				"```\n${keys(datasources.network.subnets)}\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label: "objectOrMap",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionObject{
							Label: "object",
						},
						&function.ValueTypeDefinitionMap{
							Label: "mapping",
							ElementType: &function.ValueTypeDefinitionAny{
								Label: "any",
								Type:  function.ValueTypeAny,
							},
						},
					},
					Description: "A mapping to extract keys from or an object to extract attribute names from. " +
						"Mappings and objects are interchangeable up to the point of validating parameters and return values.",
				},
			},
			Return: &function.ListReturn{
				ElementType: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "An array of keys or attributes from the mapping or object.",
			},
		},
	}
}

func (f *KeysFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *KeysFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var mapping map[string]any
	if err := input.Arguments.GetVar(ctx, 0, &mapping); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}

	return &provider.FunctionCallOutput{
		ResponseData: keys,
	}, nil
}
