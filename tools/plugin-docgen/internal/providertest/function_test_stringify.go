package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func stringifyFunction() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: &function.Definition{
			Name:                 "stringify",
			Summary:              "Stringifies a value that can be one of a number of types.",
			FormattedDescription: "Stringifies a value that can be one of a number of types.\n\n**Examples:**\n\n```plaintext\n${stringify(variables.someVariable)}\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label:       "value",
					Description: "The value to stringify.",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionScalar{
							Type:        function.ValueTypeString,
							Label:       "string",
							Description: "A value of string type.",
						},
						&function.ValueTypeDefinitionScalar{
							Type:        function.ValueTypeInt64,
							Label:       "int64",
							Description: "A value of int64 type.",
						},
						&function.ValueTypeDefinitionScalar{
							Type:        function.ValueTypeFloat64,
							Label:       "float64",
							Description: "A value of float64 type.",
						},
						&function.ValueTypeDefinitionScalar{
							Type:        function.ValueTypeBool,
							Label:       "boolean",
							Description: "A value of boolean type.",
						},
						&function.ValueTypeDefinitionFunction{
							Label: "FunctionToStringify",
							Definition: function.Definition{
								Parameters: []function.Parameter{
									&function.ScalarParameter{
										Label:       "input",
										Description: "The input to the function.",
										Type: &function.ValueTypeDefinitionScalar{
											Type: function.ValueTypeString,
										},
									},
								},
								Return: &function.MapReturn{
									ElementType: &function.ValueTypeDefinitionScalar{
										Type:        function.ValueTypeString,
										Label:       "string",
										Description: "A value of string type.",
									},
									Description: "The output of the function.",
								},
							},
						},
					},
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Type:        function.ValueTypeString,
					Label:       "string",
					Description: "A value of string type.",
				},
				Description: "The stringified value.",
			},
		},
	}
}
