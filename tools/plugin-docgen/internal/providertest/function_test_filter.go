package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func filterFunction() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: &function.Definition{
			Name:                 "filter",
			Summary:              "Filters a list of values based on a predicate function.",
			FormattedDescription: "Filters a list of values based on a predicate function.\n\n**Examples:**\n\n```plaintext\n${filter(\n  datasources.network.subnets,\n  has_prefix_g(\"subnet-402948-\")\n)}\n```",
			Parameters: []function.Parameter{
				&function.ListParameter{
					Label:       "items",
					Description: "An array of items where all items are of the same type to filter.",
					ElementType: &function.ValueTypeDefinitionAny{
						Label:       "any",
						Description: "A value of any type, every element in the containing list must be of the same type.",
					},
				},
				&function.FunctionParameter{
					Label:       "filterFunc",
					Description: "The predicate function to check if each item in the list meets a certain criteria.",
					FunctionType: &function.ValueTypeDefinitionFunction{
						Definition: function.Definition{
							Parameters: []function.Parameter{
								&function.AnyParameter{
									Label:       "item",
									Description: "The item to check.",
								},
								&function.ScalarParameter{
									Type: &function.ValueTypeDefinitionScalar{
										Type: function.ValueTypeInt64,
									},
									Label:       "index",
									Description: "The index of the item in the list.",
									Optional:    true,
								},
							},
							Return: &function.ScalarReturn{
								Type: &function.ValueTypeDefinitionScalar{
									Type:  function.ValueTypeBool,
									Label: "boolean",
								},
								Description: "Whether or not the item meets the criteria.",
							},
						},
					},
				},
			},
			Return: &function.ListReturn{
				ElementType: &function.ValueTypeDefinitionAny{
					Label:       "any",
					Description: "A value of any type, every element in the returned list must be of the same type.",
				},
				Description: "The list of values that remain after applying the filter.",
			},
		},
	}
}
