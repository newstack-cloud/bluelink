package corefunctions

import (
	"context"
	"math"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// AbsFunction provides the implementation of
// a function that returns the absolute value of a number.
type AbsFunction struct {
	definition *function.Definition
}

// NewAbsFunction creates a new instance of the AbsFunction with
// a complete function definition.
func NewAbsFunction() provider.Function {
	return &AbsFunction{
		definition: &function.Definition{
			Description: "A function that returns the absolute value of a number.",
			FormattedDescription: "A function that returns the absolute value of a number.\n\n" +
				"**Examples:**\n\n" +
				"Getting absolute value:\n" +
				"```\n${abs(-5)}\n```\n" +
				"```\n${abs(3)}\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
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
					Description: "The number to get the absolute value of.",
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
				Description: "The absolute value of the input number.",
			},
		},
	}
}

func (f *AbsFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *AbsFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var value any
	if err := input.Arguments.GetVar(ctx, 0, &value); err != nil {
		return nil, err
	}

	var floatVal float64
	isInt := false

	switch v := value.(type) {
	case int:
		floatVal = float64(v)
		isInt = true
	case int64:
		floatVal = float64(v)
		isInt = true
	case float64:
		floatVal = v
	default:
		return nil, function.NewFuncCallError(
			"input argument at index 0 must be a number (integer or float)",
			function.FuncCallErrorCodeInvalidArgumentType,
			input.CallContext.CallStackSnapshot(),
		)
	}

	absVal := math.Abs(floatVal)

	// Return the same type as input
	if isInt {
		return &provider.FunctionCallOutput{
			ResponseData: int(absVal),
		}, nil
	}

	return &provider.FunctionCallOutput{
		ResponseData: absVal,
	}, nil
}
