package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// GetElemFunction provides the implementation of the
// core "getelem" function defined in the blueprint specification.
type GetElemFunction struct {
	definition *function.Definition
}

// NewGetElemFunction creates a new instance of the GetElemFunction with
// a complete function definition.
func NewGetElemFunction() provider.Function {
	return &GetElemFunction{
		definition: &function.Definition{
			Description: "A higher-order function that returns a function that extracts an element from an array.\n" +
				"This is useful in situations where you want to map a two-dimensional array to an array of values of a specific element.\n\n" +
				"It can also be used to extract values from an array but the \"[]\" notation is more appropriate for this use case.\n" +
				"\"datasources.network.subnets[2]\" is more concise and readable than \"getelem(2)(datasources.network.subnets)\"",
			FormattedDescription: "A higher-order function that returns a function that extracts an array of values of a specific element.\n" +
				"This is useful in situations where you want to map a two-dimensional array to an array of values of a specific element.\n\n" +
				"**Examples:**\n\n" +
				"```\n${map(\ndatasources.network.subnets,\ncompose(getattr(\"id\"), getelem(0))\n)}\n```\n" +
				"This example would take a list of subnets that would be of the following form:\n" +
				"```[\n  [{ \"id\": \"subnet-1234\", \"label\": \"Subnet 1234\" }, \"10.0.0.0/16\"]," +
				"\n  [{ \"id\": \"subnet-5678\", \"label\": \"Subnet 5678\" }, \"172.31.0.0/16\"]\n]```\n" +
				"And return a list of IDs:\n```\n[\"subnet-1234\", \"subnet-5678\"]```\n",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "elemIndex",
					Type: &function.ValueTypeDefinitionScalar{
						Type: function.ValueTypeInt64,
					},
					Description: "The index of the element to extract.",
				},
			},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Label: "func (array) -> any",
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.ListParameter{
								Label: "array",
								ElementType: &function.ValueTypeDefinitionAny{
									Type: function.ValueTypeAny,
								},
							},
						},
						Return: &function.AnyReturn{
							Type:        function.ValueTypeAny,
							Description: "The extracted element at the provided index.",
						},
					},
				},
				Description: "A function that takes an array and returns the value at the provided position.",
			},
		},
	}
}

func (f *GetElemFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *GetElemFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var attrIndex int64
	err := input.Arguments.GetVar(ctx, 0, &attrIndex)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "_getelem_exec",
			PartialArgs:  []interface{}{attrIndex},
			// The input string is passed as the first argument to the _getelem_exec function.
			ArgsOffset: 1,
		},
	}, nil
}
