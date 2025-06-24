package testprovider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func functionAlterList() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: AlterListFunctionDefinition(),
		CallFunc:   alterList,
	}
}

// AlterListFunctionDefinition returns the definition of the function that
// alters a list of strings.
func AlterListFunctionDefinition() *function.Definition {
	return &function.Definition{
		Name:        "alter_list",
		Description: "Alters a list of strings.",
		Parameters: []function.Parameter{
			&function.ListParameter{
				Label: "items",
				ElementType: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "A list of strings to alter.",
			},
		},
		Return: &function.ListReturn{
			ElementType: &function.ValueTypeDefinitionScalar{
				Label: "string",
				Type:  function.ValueTypeString,
			},
			Description: "The altered list of strings.",
		},
	}
}

func alterList(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var items []string
	if err := input.Arguments.GetVar(ctx, 0, &items); err != nil {
		return nil, err
	}

	// Do nothing, this function isn't tested for its functionality,
	// just for its definition.
	return &provider.FunctionCallOutput{
		ResponseData: items,
	}, nil
}
