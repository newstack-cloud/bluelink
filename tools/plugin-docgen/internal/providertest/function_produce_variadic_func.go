package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func produceVariadicFuncFunction() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: &function.Definition{
			Name:                 "produce_variadic_func",
			Summary:              "Creates a function that takes a variadic number of arguments.",
			FormattedDescription: "Creates a function that takes a variadic number of arguments.\n\n**Examples:**\n\n```plaintext\n${produce_variadic_func()}\n```",
			Parameters:           []function.Parameter{},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Label: "VariadicArgsFunction",
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.VariadicParameter{
								Label:       "args",
								Description: "The variadic arguments.",
								SingleType:  true,
								Named:       true,
								Type: &function.ValueTypeDefinitionScalar{
									Type:        function.ValueTypeString,
									Label:       "string",
									Description: "A value of string type.",
								},
							},
						},
						Return: &function.ListReturn{
							ElementType: &function.ValueTypeDefinitionScalar{
								Type:        function.ValueTypeString,
								Label:       "string",
								Description: "A value of string type.",
							},
							Description: "The result of the function.",
						},
					},
				},
				Description: "A function that takes variadic arguments and returns a list of strings.",
			},
		},
	}
}
