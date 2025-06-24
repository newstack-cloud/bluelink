package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// OrFunction provides the implementation of
// a function that acts as a logical OR operator.
type OrFunction struct {
	definition *function.Definition
}

// NewOrFunction creates a new instance of the OrFunction with
// a complete function definition.
func NewOrFunction() provider.Function {
	return &OrFunction{
		definition: &function.Definition{
			Description: "A function that acts as a logical OR operator on two boolean values.",
			FormattedDescription: "A function that acts as a logical OR operator on two boolean values.\n\n" +
				"**Examples:**\n\n" +
				"```\n${or(resources.orderApi.spec.isDev, eq(variables.environment, \"dev\"))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "a",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "boolean",
						Type:  function.ValueTypeBool,
					},
					Description: "The result of boolean expression A, the left-hand side of the OR operation.",
				},
				&function.ScalarParameter{
					Label: "b",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "boolean",
						Type:  function.ValueTypeBool,
					},
					Description: "The result of boolean expression B, the right-hand side of the OR operation.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "boolean",
					Type:  function.ValueTypeBool,
				},
				Description: "The result of the logical OR operation on the two boolean values.",
			},
		},
	}
}

func (f *OrFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *OrFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var lhs bool
	var rhs bool
	if err := input.Arguments.GetMultipleVars(ctx, &lhs, &rhs); err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		ResponseData: lhs || rhs,
	}, nil
}
