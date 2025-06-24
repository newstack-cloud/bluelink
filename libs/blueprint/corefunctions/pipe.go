package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// PipeFunction provides the implementation of the
// core "pipe" function defined in the blueprint specification.
type PipeFunction struct {
	definition *function.Definition
}

// NewPipeFunction creates a new instance of the PipeFunction with
// a complete function definition.
func NewPipeFunction() provider.Function {
	return &PipeFunction{
		definition: &function.Definition{
			Description: "A higher-order function that combines N functions into a single function, where the output " +
				"of one function is passed in as the input to the next function. The call order of the functions is from left to right, " +
				"which is generally seen are more intuitive than the right to left order of \"compose\".",
			FormattedDescription: "A higher-order function that combines N functions into a single function, where the output " +
				"of one function is passed in as the input to the next function. The call order of the functions is from left to right, " +
				"which is generally seen are more intuitive than the right to left order of `compose`.\n\n" +
				"**Examples:**\n\n" +
				"```\n${map(\n  datasources.network.subnets,\n  pipe(getattr(\"id\"), to_upper)\n)}\n```",
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
					Description: "N functions to be piped together.",
				},
			},
			Return: &function.FunctionReturn{
				FunctionType: &function.ValueTypeDefinitionFunction{
					Label: "func (any) -> any",
					Definition: function.Definition{
						Parameters: []function.Parameter{
							&function.AnyParameter{
								Description: "The input to the piped function, " +
									"this must be the same type of the input of the left-most function when using pipe.",
							},
						},
						Return: &function.AnyReturn{
							Type: function.ValueTypeAny,
							Description: "The output of the piped function, this must be the same type of the " +
								"return value of the right-most function in the pipeline.",
						},
					},
				},
				Description: "A function that takes the input value of the left-most function " +
					"and returns the output value of the right-most function.",
			},
		},
	}
}

func (f *PipeFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *PipeFunction) Call(
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
			FunctionName: "_pipe_exec",
			PartialArgs:  []interface{}{functions},
			// The input value is passed as the first argument to the _pipe_exec function.
			ArgsOffset: 1,
		},
	}, nil
}
