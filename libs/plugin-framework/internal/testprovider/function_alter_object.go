package testprovider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func functionAlterObject() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: AlterObjectFunctionDefinition(),
		CallFunc:   alterObject,
	}
}

// AlterObjectFunctionDefinition returns the definition of the function that
// alters an object of a specified type.
func AlterObjectFunctionDefinition() *function.Definition {
	return &function.Definition{
		Name:        "alter_object",
		Description: "Alter an object of a specific type.",
		Parameters: []function.Parameter{
			&function.ObjectParameter{
				Label:           "items",
				ObjectValueType: objectValueType(),
				Description:     "The complex object to alter.",
			},
		},
		Return: &function.ObjectReturn{
			ObjectValueType: objectValueType(),
			Description:     "The altered object.",
		},
	}
}

func objectValueType() *function.ValueTypeDefinitionObject {
	return &function.ValueTypeDefinitionObject{
		Label: "complex_object",
		AttributeTypes: map[string]function.AttributeType{
			"components": {
				Type: &function.ValueTypeDefinitionList{
					Label: "list of values that are strings or integers",
					ElementType: &function.ValueTypeDefinitionAny{
						Type:  function.ValueTypeAny,
						Label: "string or int",
						UnionTypes: []function.ValueTypeDefinition{
							&function.ValueTypeDefinitionScalar{
								Label: "string",
								Type:  function.ValueTypeString,
							},
							&function.ValueTypeDefinitionScalar{
								Label: "int",
								Type:  function.ValueTypeInt64,
							},
						},
					},
				},
				AllowNullValue: false,
			},
			"nestedObject": {
				Type: &function.ValueTypeDefinitionObject{
					Label: "nested_object",
					AttributeTypes: map[string]function.AttributeType{
						"nestedString": {
							Type: &function.ValueTypeDefinitionScalar{
								Label: "string",
								Type:  function.ValueTypeString,
							},
							AllowNullValue: true,
						},
					},
				},
			},
			"tags": {
				Type: &function.ValueTypeDefinitionMap{
					Label: "map of strings",
					ElementType: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
				},
			},
		},
	}
}

func alterObject(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var object map[string]any
	if err := input.Arguments.GetVar(ctx, 0, &object); err != nil {
		return nil, err
	}

	// Do nothing, this function isn't tested for its functionality,
	// just for its definition.
	return &provider.FunctionCallOutput{
		ResponseData: object,
	}, nil
}
