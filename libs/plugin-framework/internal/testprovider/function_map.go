package testprovider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func functionMap() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: MapFunctionDefinition(),
		CallFunc:   mapFunc,
	}
}

func MapFunctionDefinition() *function.Definition {
	return &function.Definition{
		Description: "Maps a list of values to a new list of values using a provided function.",
		FormattedDescription: "Maps a list of values to a new list of values using a provided function.\n\n" +
			"**Examples:**\n\n" +
			"```\n${map(\n  datasources.network.subnets,\n  compose(to_upper, getattr(\"id\")\n)}\n```",
		Parameters: []function.Parameter{
			&function.ListParameter{
				Label: "items",
				ElementType: &function.ValueTypeDefinitionAny{
					Label:       "Any",
					Type:        function.ValueTypeAny,
					Description: "A value of any type, every element in the containing list must be of the same type.",
				},
				Description: "An array of items where all items are of the same type to map.",
			},
			&function.FunctionParameter{
				Label: "mapFunc",
				FunctionType: &function.ValueTypeDefinitionFunction{
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.AnyParameter{
								Label:       "item",
								Description: "The item to transform.",
							},
							&function.ScalarParameter{
								Label:       "index",
								Description: "The index of the item in the list.",
								Type: &function.ValueTypeDefinitionScalar{
									Type: function.ValueTypeInt64,
								},
								Optional: true,
							},
						},
						Return: &function.AnyReturn{
							Type:        function.ValueTypeAny,
							Description: "The transformed item.",
						},
					},
				},
				Description: "The function to apply to each element in the list.",
			},
		},
		Return: &function.ListReturn{
			ElementType: &function.ValueTypeDefinitionAny{
				Label:       "Any",
				Type:        function.ValueTypeAny,
				Description: "A value of any type, every element in the returned list must be of the same type.",
			},
			Description: "The list of values after applying the mapping function.",
		},
	}
}

func mapFunc(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var items []any
	var mapFuncInfo provider.FunctionRuntimeInfo
	if err := input.Arguments.GetMultipleVars(ctx, 0, &items, &mapFuncInfo); err != nil {
		return nil, err
	}

	// Do nothing, this function isn't tested for its functionality,
	// just for its definition.
	return &provider.FunctionCallOutput{
		ResponseData: items,
	}, nil
}
