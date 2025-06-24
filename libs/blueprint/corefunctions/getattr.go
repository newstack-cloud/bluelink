package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// GetAttrFunction provides the implementation of the
// core "getattr" function defined in the blueprint specification.
type GetAttrFunction struct {
	definition *function.Definition
}

// NewGetAttrFunction creates a new instance of the GetAttrFunction with
// a complete function definition.
func NewGetAttrFunction() provider.Function {
	return &GetAttrFunction{
		definition: &function.Definition{
			Description: "A higher-order function that returns a function that extracts a named attribute from an object or a mapping.\n" +
				"This is useful in situations where you want to map an array of objects to an array of values of a specific attribute such as IDs.\n\n" +
				"It can also be used to extract a named attribute from a mapping but the \".\" or \"[]\" notation is more appropriate for this use case.\n" +
				"\"datasources.network.subnets[].id\" is more concise and readable than \"getattr(\\\"id\\\")(datasources.network.subnets[])\"",
			FormattedDescription: "A higher-order function that returns a function that extracts a named attribute from an object or a mapping.\n" +
				"This is useful in situations where you want to map an array of objects to an array of values of a specific attribute such as IDs.\n\n" +
				"**Examples:**\n\n" +
				"```\n${map(\ndatasources.network.subnets,\ncompose(getattr(\"id\"), getattr(\"definition\"))\n)}\n```\n" +
				"This example would take a list of subnets that would be of the following form:\n" +
				"```\n[\n{ \"definition\": { \"id\": \"subnet-1234\" }},\n{ \"definition\": { \"id\": \"subnet-5678\" }}\n]\n```\n" +
				"And return a list of IDs:\n```\n[\"subnet-1234\", \"subnet-5678\"]```\n",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "attrName",
					Type: &function.ValueTypeDefinitionScalar{
						Type: function.ValueTypeString,
					},
					Description: "The name of the attribute to extract from the object or mapping.",
				},
			},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Label: "func (object | mapping) -> any",
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.AnyParameter{
								UnionTypes: []function.ValueTypeDefinition{
									&function.ValueTypeDefinitionMap{
										Label: "map",
										ElementType: &function.ValueTypeDefinitionAny{
											Label: "any",
											Type:  function.ValueTypeAny,
										},
									},
									&function.ValueTypeDefinitionObject{
										Label: "object",
									},
								},
								Description: "A valid object or mapping to extract the attribute from.",
							},
						},
						Return: &function.AnyReturn{
							Type:        function.ValueTypeAny,
							Description: "The extracted attribute or mapping value.",
						},
					},
				},
				Description: "A function that takes an object or mapping and returns the value of the named attribute.",
			},
		},
	}
}

func (f *GetAttrFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *GetAttrFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var attrName string
	err := input.Arguments.GetVar(ctx, 0, &attrName)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "_getattr_exec",
			PartialArgs:  []interface{}{attrName},
			// The input string is passed as the first argument to the _getattr_exec function.
			ArgsOffset: 1,
		},
	}, nil
}
