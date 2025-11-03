package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// IfFunction provides the implementation of
// a function that provides conditional branching
// like the if-then-else construct in many programming languages.
type IfFunction struct {
	definition *function.Definition
}

// NewIfFunction creates a new instance of the IfFunction with
// a complete function definition.
func NewIfFunction() provider.Function {
	return &IfFunction{
		definition: &function.Definition{
			Description: "A function that returns one of two values based on a boolean condition.",
			FormattedDescription: "A function that returns one of two values based on a boolean condition.\n\n" +
				"**Examples:**\n\n" +
				"```\n${if(eq(variables.environment, \"prod\"), \"prod\", \"dev\")}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "condition",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "boolean",
						Type:  function.ValueTypeBool,
					},
					Description: "The boolean condition to evaluate.",
				},
				&function.AnyParameter{
					Label:       "then_value",
					Description: "The value to return if the condition is true.",
				},
				&function.AnyParameter{
					Label:       "else_value",
					Description: "The value to return if the condition is false.",
				},
			},
			Return: &function.AnyReturn{
				Type:        function.ValueTypeAny,
				Description: "The value returned based on the boolean condition.",
			},
		},
	}
}

func (f *IfFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *IfFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var conditionAny any
	var thenValue any
	var elseValue any
	if err := input.Arguments.GetMultipleVars(ctx, &conditionAny, &thenValue, &elseValue); err != nil {
		return nil, err
	}

	// Treat none as falsy - evaluate to false branch
	condition := false
	if !core.IsNoneMarker(conditionAny) {
		var ok bool
		condition, ok = conditionAny.(bool)
		if !ok {
			return nil, function.NewFuncCallError(
				"condition argument to `if` must be a boolean value",
				function.FuncCallErrorCodeInvalidInput,
				input.CallContext.CallStackSnapshot(),
			)
		}
	}

	if condition {
		return &provider.FunctionCallOutput{
			ResponseData: thenValue,
		}, nil
	}

	return &provider.FunctionCallOutput{
		ResponseData: elseValue,
	}, nil
}
