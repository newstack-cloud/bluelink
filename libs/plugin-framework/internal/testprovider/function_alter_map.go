package testprovider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func functionAlterMap() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: AlterMapFunctionDefinition(),
		CallFunc:   alterMap,
	}
}

// AlterMapFunctionDefinition returns the definition of the function that
// alters a map of string values.
func AlterMapFunctionDefinition() *function.Definition {
	return &function.Definition{
		Name:        "alter_map",
		Description: "Alters a map of strings.",
		Parameters: []function.Parameter{
			&function.MapParameter{
				Label: "items",
				ElementType: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "A map of strings to alter.",
			},
		},
		Return: &function.MapReturn{
			ElementType: &function.ValueTypeDefinitionScalar{
				Label: "string",
				Type:  function.ValueTypeString,
			},
			Description: "The altered map of strings.",
		},
	}
}

func alterMap(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var mapping map[string]string
	if err := input.Arguments.GetVar(ctx, 0, &mapping); err != nil {
		return nil, err
	}

	// Do nothing, this function isn't tested for its functionality,
	// just for its definition.
	return &provider.FunctionCallOutput{
		ResponseData: mapping,
	}, nil
}
