package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ValsFunction provides the implementation of
// a function that checks if a string has a suffix.
type ValsFunction struct {
	definition *function.Definition
}

// NewValsFunction creates a new instance of the ValsFunction with
// a complete function definition.
func NewValsFunction() provider.Function {
	return &ValsFunction{
		definition: &function.Definition{
			Description: "Produces an array of values from a mapping or an object with known attributes. " +
				"This function is named \"vals\" to avoid conflicting with the \"values\" keyword used for blueprint values.",
			FormattedDescription: "Produces an array of values from a mapping or an object with known attributes. " +
				"_This function is named `vals` to avoid conflicting with the `values` keyword used for blueprint values._\n\n" +
				"**Examples:**\n\n" +
				"```\n${vals(datasources.network.subnets)}\n```",
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
					Description: "A mapping or object to extract values from. " +
						"Mappings and objects are interchangeable up to the point of validating parameters and return values.",
				},
			},
			Return: &function.ListReturn{
				ElementType: &function.ValueTypeDefinitionAny{
					Label: "any",
					Type:  function.ValueTypeAny,
				},
				Description: "An array of values extracted from the provided mapping or object.",
			},
		},
	}
}

func (f *ValsFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ValsFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var mapping map[string]interface{}
	if err := input.Arguments.GetVar(ctx, 0, &mapping); err != nil {
		return nil, err
	}

	vals := make([]interface{}, 0, len(mapping))
	for _, value := range mapping {
		vals = append(vals, value)
	}

	return &provider.FunctionCallOutput{
		ResponseData: vals,
	}, nil
}
