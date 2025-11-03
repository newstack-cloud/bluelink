package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// NotFunction provides the implementation of
// a function that acts as a logical NOT operator.
type NotFunction struct {
	definition *function.Definition
}

// NewNotFunction creates a new instance of the NotFunction with
// a complete function definition.
func NewNotFunction() provider.Function {
	return &NotFunction{
		definition: &function.Definition{
			Description: "A function that negates a given boolean value.",
			FormattedDescription: "A function that negates a given boolean value.\n\n" +
				"**Examples:**\n\n" +
				"```\n${not(eq(variables.environment, \"prod\"))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "toNegate",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "boolean",
						Type:  function.ValueTypeBool,
					},
					Description: "The result of a boolean expression to negate.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "boolean",
					Type:  function.ValueTypeBool,
				},
				Description: "The result of negating the provided boolean value.",
			},
		},
	}
}

func (f *NotFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *NotFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	// Get argument as any to check for none marker
	toNegateAny, err := input.Arguments.Get(ctx, 0)
	if err != nil {
		return nil, err
	}

	// Treat none as falsy (false), so !none = true
	toNegate := false
	if !core.IsNoneMarker(toNegateAny) {
		var ok bool
		toNegate, ok = toNegateAny.(bool)
		if !ok {
			return nil, function.NewFuncCallError(
				"argument to `not` must be a boolean value",
				function.FuncCallErrorCodeInvalidInput,
				input.CallContext.CallStackSnapshot(),
			)
		}
	}

	return &provider.FunctionCallOutput{
		ResponseData: !toNegate,
	}, nil
}
