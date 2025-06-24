package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func andFunction() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: &function.Definition{
			Name:    "and",
			Summary: "A function that acts as a logical AND operator on two boolean values.",
			FormattedDescription: "A function that acts as a logical AND operator on two boolean values." +
				"\n\n**Examples:**\n\n```plaintext\n${and(resources.orderApi.spec.isProd, eq(variables.environment, \"prod\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "a",
					Type: &function.ValueTypeDefinitionScalar{
						Type: function.ValueTypeBool,
					},
					Description: "The result of boolean expression A, the left-hand side of the AND operation.",
				},
				&function.ScalarParameter{
					Label: "b",
					Type: &function.ValueTypeDefinitionScalar{
						Type: function.ValueTypeBool,
					},
					Description: "The result of boolean expression B, the right-hand side of the AND operation.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Type: function.ValueTypeBool,
				},
				Description: "The result of the logical AND operation on the two boolean values.",
			},
		},
	}
}
