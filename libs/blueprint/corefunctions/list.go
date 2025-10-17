package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ListFunction provides the implementation of
// a function that checks if a string has a suffix.
type ListFunction struct {
	definition *function.Definition
}

// NewListFunction creates a new instance of the ListFunction with
// a complete function definition.
func NewListFunction() provider.Function {
	return &ListFunction{
		definition: &function.Definition{
			Description: "Creates a list of values from arguments of the same type.",
			FormattedDescription: "Creates a list of values from arguments of the same type.\n\n" +
				"**Examples:**\n\n" +
				"```\n${list(\"item1\",\"item2\",\"item3\",\"item4\")}\n```",
			Parameters: []function.Parameter{
				&function.VariadicParameter{
					Label: "values",
					Type: &function.ValueTypeDefinitionAny{
						Type:  function.ValueTypeAny,
						Label: "any",
					},
					Description: "N arguments of the same type that will be used to create a list.",
				},
			},
			Return: &function.ListReturn{
				ElementType: &function.ValueTypeDefinitionAny{
					Label: "any",
					Type:  function.ValueTypeAny,
				},
				Description: "An array of values that have been passed as arguments.",
			},
		},
	}
}

func (f *ListFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ListFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var params []any
	if err := input.Arguments.GetVar(ctx, 0, &params); err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		// Make a copy of the params slice.
		// Operationally, the `list` function does not do anything special,
		// it provides a way to create a list of values from the end-user perspective,
		// variadic arguments are already passed as a slice of values to the function instead of
		// individual arguments as there is no way of knowing how many arguments will be passed.
		ResponseData: append([]any{}, params...),
	}, nil
}
