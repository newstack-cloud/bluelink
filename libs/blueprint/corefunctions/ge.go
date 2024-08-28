package corefunctions

import (
	"context"

	"github.com/two-hundred/celerity/libs/blueprint/function"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
)

// GeFunction provides the implementation of
// a function that checks if a number is greater than or equal to another.
type GeFunction struct {
	definition *function.Definition
}

// NewGeFunction creates a new instance of the GeFunction with
// a complete function definition.
func NewGeFunction() provider.Function {
	return &GeFunction{
		definition: &function.Definition{
			Description: "A function that determines whether a number is greater than or equal to another number.",
			FormattedDescription: "A function that determines whether a number is greater than or equal to another number.\n\n" +
				"**Examples:**\n\n" +
				"```\n${ge(len(datasources.network.subnets), 10)}\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label: "a",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionScalar{
							Type: function.ValueTypeInt64,
						},
						&function.ValueTypeDefinitionScalar{
							Type: function.ValueTypeFloat64,
						},
					},
					Description: "\"a\" in the expression \"a >= b\".",
				},
				&function.AnyParameter{
					Label: "b",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionScalar{
							Type: function.ValueTypeInt64,
						},
						&function.ValueTypeDefinitionScalar{
							Type: function.ValueTypeFloat64,
						},
					},
					Description: "\"b\" in the expression \"a >= b\".",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "boolean",
					Type:  function.ValueTypeBool,
				},
				Description: "True, if the first number is greater than or equal to the second number, false otherwise.",
			},
		},
	}
}

func (f *GeFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *GeFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var lhs interface{}
	var rhs interface{}
	if err := input.Arguments.GetMultipleVars(ctx, &lhs, &rhs); err != nil {
		return nil, err
	}

	info, err := checkNumberTypes(input, lhs, rhs)
	if err != nil {
		return nil, err
	}

	if info.lhs.intVal != nil && info.rhs.intVal != nil {
		return &provider.FunctionCallOutput{
			ResponseData: *info.lhs.intVal >= *info.rhs.intVal,
		}, nil
	}

	if info.lhs.floatVal != nil && info.rhs.floatVal != nil {
		return &provider.FunctionCallOutput{
			ResponseData: *info.lhs.floatVal >= *info.rhs.floatVal,
		}, nil
	}

	if info.lhs.intVal != nil && info.rhs.floatVal != nil {
		return &provider.FunctionCallOutput{
			ResponseData: float64(*info.lhs.intVal) >= *info.rhs.floatVal,
		}, nil
	}

	return &provider.FunctionCallOutput{
		ResponseData: *info.lhs.floatVal >= float64(*info.rhs.intVal),
	}, nil
}
