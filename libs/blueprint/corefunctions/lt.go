package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// LtFunction provides the implementation of
// a function that checks if a number is less than another.
type LtFunction struct {
	definition *function.Definition
}

// NewLtFunction creates a new instance of the LtFunction with
// a complete function definition.
func NewLtFunction() provider.Function {
	return &LtFunction{
		definition: &function.Definition{
			Description: "A function that determines whether a number is less than another number.",
			FormattedDescription: "A function that determines whether a number is less than another number.\n\n" +
				"**Examples:**\n\n" +
				"```\n${lt(len(datasources.network.subnets), 10)}\n```",
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
					Description: "\"a\" in the expression \"a < b\".",
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
					Description: "\"b\" in the expression \"a < b\".",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "boolean",
					Type:  function.ValueTypeBool,
				},
				Description: "True, if the first number is less than the second number, false otherwise.",
			},
		},
	}
}

func (f *LtFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *LtFunction) Call(
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
			ResponseData: *info.lhs.intVal < *info.rhs.intVal,
		}, nil
	}

	if info.lhs.floatVal != nil && info.rhs.floatVal != nil {
		return &provider.FunctionCallOutput{
			ResponseData: *info.lhs.floatVal < *info.rhs.floatVal,
		}, nil
	}

	if info.lhs.intVal != nil && info.rhs.floatVal != nil {
		return &provider.FunctionCallOutput{
			ResponseData: float64(*info.lhs.intVal) < *info.rhs.floatVal,
		}, nil
	}

	return &provider.FunctionCallOutput{
		ResponseData: *info.lhs.floatVal < float64(*info.rhs.intVal),
	}, nil
}
