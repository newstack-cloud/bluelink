package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// AndFunction provides the implementation of
// a function that acts as a logical AND operator.
type AndFunction struct {
	definition *function.Definition
}

// NewAndFunction creates a new instance of the AndFunction with
// a complete function definition.
func NewAndFunction() provider.Function {
	return &AndFunction{
		definition: &function.Definition{
			Description: "A function that acts as a logical AND operator on two boolean values.",
			FormattedDescription: "A function that acts as a logical AND operator on two boolean values.\n\n" +
				"**Examples:**\n\n" +
				"```\n${and(resources.orderApi.spec.isProd, eq(variables.environment, \"prod\"))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "a",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "boolean",
						Type:  function.ValueTypeBool,
					},
					Description: "The result of boolean expression A, the left-hand side of the AND operation.",
				},
				&function.ScalarParameter{
					Label: "b",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "boolean",
						Type:  function.ValueTypeBool,
					},
					Description: "The result of boolean expression B, the right-hand side of the AND operation.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "boolean",
					Type:  function.ValueTypeBool,
				},
				Description: "The result of the logical AND operation on the two boolean values.",
			},
		},
	}
}

func (f *AndFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *AndFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	// Get arguments as any to check for none markers
	lhsAny, err := input.Arguments.Get(ctx, 0)
	if err != nil {
		return nil, err
	}
	rhsAny, err := input.Arguments.Get(ctx, 1)
	if err != nil {
		return nil, err
	}

	// Treat none as falsy (false) in AND logic
	lhs := false
	if !core.IsNoneMarker(lhsAny) {
		var ok bool
		lhs, ok = lhsAny.(bool)
		if !ok {
			return nil, function.NewFuncCallError(
				"left argument to `and` must be a boolean value",
				function.FuncCallErrorCodeInvalidInput,
				input.CallContext.CallStackSnapshot(),
			)
		}
	}

	rhs := false
	if !core.IsNoneMarker(rhsAny) {
		var ok bool
		rhs, ok = rhsAny.(bool)
		if !ok {
			return nil, function.NewFuncCallError(
				"right argument to `and` must be a boolean value",
				function.FuncCallErrorCodeInvalidInput,
				input.CallContext.CallStackSnapshot(),
			)
		}
	}

	return &provider.FunctionCallOutput{
		ResponseData: lhs && rhs,
	}, nil
}
