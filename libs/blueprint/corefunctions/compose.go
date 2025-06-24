package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ComposeFunction provides the implementation of the
// core "compose" function defined in the blueprint specification.
type ComposeFunction struct {
	definition *function.Definition
}

// NewComposeFunction creates a new instance of the ComposeFunction with
// a complete function definition.
func NewComposeFunction() provider.Function {
	return &ComposeFunction{
		definition: &function.Definition{
			Description: "A higher-order function that combines N functions into a single function, where the output " +
				"of one function is passed in as the input to the previous function. The call order of the functions is from right to left.",
			FormattedDescription: "A higher-order function that combines N functions into a single function, where the output " +
				"of one function is passed in as the input to the previous function. The call order of the functions is from right to left.\n\n" +
				"**Examples:**\n\n" +
				"```\n${map(\n  datasources.network.subnets,\n  compose(to_upper, getattr(\"id\"))\n)}\n```",
			Parameters: []function.Parameter{
				&function.VariadicParameter{
					Label: "functions",
					Type: &function.ValueTypeDefinitionFunction{
						Definition: function.Definition{
							Parameters: []function.Parameter{
								&function.AnyParameter{
									Label:       "any",
									Description: "The input to the function.",
								},
							},
							Return: &function.AnyReturn{
								Type:        function.ValueTypeAny,
								Description: "The output of the function.",
							},
						},
					},
					Description: "N functions to be composed together.",
				},
			},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Label: "func (any) -> any",
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.AnyParameter{
								Description: "The input to the composed function, " +
									"this must be the same type of the input of the right-most function in the composition.",
							},
						},
						Return: &function.AnyReturn{
							Type: function.ValueTypeAny,
							Description: "The output of the composed function, this must be the same type of the " +
								"return value of the left-most function in the composition.",
						},
					},
				},
				Description: "A function that takes the input value of the right-most function " +
					"and returns the output value of the left-most function.",
			},
		},
	}
}

func (f *ComposeFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ComposeFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var functions []provider.FunctionRuntimeInfo
	err := input.Arguments.GetVar(ctx, 0, &functions)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		FunctionInfo: provider.FunctionRuntimeInfo{
			FunctionName: "_compose_exec",
			PartialArgs:  []interface{}{functions},
			// The input value is passed as the first argument to the _compose_exec function.
			ArgsOffset: 1,
		},
	}, nil
}
