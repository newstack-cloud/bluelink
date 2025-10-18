package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// MinFunction provides the implementation of
// a function that returns the minimum value from a list of numbers.
type MinFunction struct {
	definition *function.Definition
}

// NewMinFunction creates a new instance of the MinFunction with
// a complete function definition.
func NewMinFunction() provider.Function {
	return &MinFunction{
		definition: &function.Definition{
			Description: "A function that returns the minimum value from a list of numbers.",
			FormattedDescription: "A function that returns the minimum value from a list of numbers.\n\n" +
				"**Examples:**\n\n" +
				"Finding minimum from multiple values:\n" +
				"```\n${min(10, 5, 8, 3)}\n```\n\n" +
				"Dynamic resource sizing:\n" +
				"```\n${min(variables.maxInstances, 10)}\n```",
			Parameters: []function.Parameter{
				&function.VariadicParameter{
					Label: "numbers",
					Type: &function.ValueTypeDefinitionAny{
						Type:  function.ValueTypeAny,
						Label: "number",
						UnionTypes: []function.ValueTypeDefinition{
							&function.ValueTypeDefinitionScalar{
								Label: "integer",
								Type:  function.ValueTypeInt64,
							},
							&function.ValueTypeDefinitionScalar{
								Label: "float",
								Type:  function.ValueTypeFloat64,
							},
						},
					},
					Description: "N arguments of type integer or float to find the minimum value from.",
				},
			},
			Return: &function.AnyReturn{
				Type: function.ValueTypeAny,
				UnionTypes: []function.ValueTypeDefinition{
					&function.ValueTypeDefinitionScalar{
						Label: "integer",
						Type:  function.ValueTypeInt64,
					},
					&function.ValueTypeDefinitionScalar{
						Label: "float",
						Type:  function.ValueTypeFloat64,
					},
				},
				Description: "The minimum value from the provided arguments.",
			},
		},
	}
}

func (f *MinFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *MinFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var numbers []any
	if err := input.Arguments.GetVar(ctx, 0, &numbers); err != nil {
		return nil, err
	}

	if len(numbers) == 0 {
		return nil, function.NewFuncCallError(
			"min requires at least one argument",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	// Convert all numbers to float64 for comparison
	hasFloat := false
	minVal := float64(0)

	for i, num := range numbers {
		var val float64
		switch v := num.(type) {
		case int:
			val = float64(v)
		case int64:
			val = float64(v)
		case float64:
			val = v
			hasFloat = true
		default:
			return nil, function.NewFuncCallError(
				fmt.Sprintf("argument at index %d must be a number (integer or float)", i),
				function.FuncCallErrorCodeInvalidArgumentType,
				input.CallContext.CallStackSnapshot(),
			)
		}

		if i == 0 || val < minVal {
			minVal = val
		}
	}

	// Return float if any input was float, otherwise return as integer
	if hasFloat {
		return &provider.FunctionCallOutput{
			ResponseData: minVal,
		}, nil
	}

	return &provider.FunctionCallOutput{
		ResponseData: int(minVal),
	}, nil
}
