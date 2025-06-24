package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func composeFunction() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: &function.Definition{
			Name:                 "compose",
			Summary:              "A higher-order function that combines N functions into a single function.",
			FormattedDescription: "A higher-order function that combines N functions into a single function, where the output of one function is passed in as the input to the previous function. The call order of the functions is from right to left.\n\n**Examples:**\n\n```plaintext\n${map(\n  datasources.network.subnets,\n  compose(to_upper, getattr(\"id\"))\n)}\n```",
			Parameters: []function.Parameter{
				&function.VariadicParameter{
					Label:       "functions",
					Description: "N functions to be composed together.",
					SingleType:  true,
					Type: &function.ValueTypeDefinitionFunction{
						Label:       "Function",
						Description: "A function that takes an input value and returns an output value that can be passed to the next function in the composition.",
						Definition: function.Definition{
							Parameters: []function.Parameter{
								&function.AnyParameter{
									Label:       "input",
									Description: "The input to the function",
								},
							},
							Return: &function.AnyReturn{
								Type:        function.ValueTypeAny,
								Description: "The output of the function.",
							},
						},
					},
				},
			},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Label: "ComposedFunction",
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.AnyParameter{
								Label:       "input",
								Description: "The input of the composed function, this must be of the same type of the input of the right-most function in the composition.",
							},
						},
						Return: &function.AnyReturn{
							Type:        function.ValueTypeAny,
							Description: "The output of the composed function, this must be the same type of the return value of the left-most function in the composition.",
						},
					},
				},
				Description: "A function that takes the input value of the right-most function and returns the output value of the left-most function.",
			},
		},
	}
}
