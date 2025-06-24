package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func createSpecificObjectTypeFunction() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: &function.Definition{
			Name:                 "create_specific_object_type",
			Summary:              "Creates a specific object type.",
			FormattedDescription: "Creates a specific object type.\n\n**Examples:**\n\n```plaintext\n${create_specific_object_type(\n  \"value1\",\n  \"value2\"\n)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "field1",
					Type: &function.ValueTypeDefinitionScalar{
						Type:  function.ValueTypeString,
						Label: "string",
					},
					Description: "The value of field1.",
				},
				&function.ScalarParameter{
					Label: "field2",
					Type: &function.ValueTypeDefinitionScalar{
						Type:  function.ValueTypeString,
						Label: "string",
					},
					Description: "The value of field2.",
				},
			},
			Return: &function.ObjectReturn{
				ObjectValueType: &function.ValueTypeDefinitionObject{
					Label: "SpecificObjectType",
					AttributeTypes: map[string]function.AttributeType{
						"field1": {
							Type: &function.ValueTypeDefinitionScalar{
								Type:        function.ValueTypeString,
								Description: "The value of field1.",
							},
							AllowNullValue: true,
						},
						"field2": {
							Type: &function.ValueTypeDefinitionObject{
								Label: "NestedObjectType",
								AttributeTypes: map[string]function.AttributeType{
									"nestedField": {
										Type: &function.ValueTypeDefinitionScalar{
											Type:        function.ValueTypeString,
											Description: "The value of the nested field.",
										},
										AllowNullValue: false,
									},
								},
								Description: "The value of field2.",
							},
							AllowNullValue: false,
						},
					},
					Required: []string{"field1", "field2"},
				},
				Description: "The specific object type.",
			},
		},
	}
}
