package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// CoalesceFunction provides the implementation of
// a function that returns the first non-none value from a list of arguments.
// Unlike the `first` function, coalesce preserves other "empty" values
// like empty strings (""), empty arrays ([]), and false.
type CoalesceFunction struct {
	definition *function.Definition
}

// NewCoalesceFunction creates a new instance of the CoalesceFunction with
// a complete function definition.
func NewCoalesceFunction() provider.Function {
	return &CoalesceFunction{
		definition: &function.Definition{
			Description: "A function that returns the first non-none value from a list of arguments. " +
				"Unlike `first`, coalesce preserves other empty values like empty strings, empty arrays, and false.",
			FormattedDescription: "A function that returns the first non-none value from a list of arguments. " +
				"Unlike `first`, coalesce preserves other empty values like empty strings, empty arrays, and false.\n\n" +
				"**Examples:**\n\n" +
				"```\n${coalesce(variables.optionalValue, \"default\")}\n```\n\n" +
				"```\n${coalesce(datasources.config.data.setting, variables.setting, \"fallback\")}\n```",
			Parameters: []function.Parameter{
				&function.VariadicParameter{
					Label: "values",
					Type: &function.ValueTypeDefinitionAny{
						Type:  function.ValueTypeAny,
						Label: "any",
					},
					Description: "N arguments of any type. Returns the first argument that is not `none`, " +
						"or `none` if all arguments are `none`.",
				},
			},
			Return: &function.AnyReturn{
				Type:        function.ValueTypeAny,
				Description: "The first non-none value from the list of arguments, or `none` if all values are `none`.",
			},
		},
	}
}

func (f *CoalesceFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *CoalesceFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var params []any
	if err := input.Arguments.GetVar(ctx, 0, &params); err != nil {
		return nil, err
	}

	if len(params) == 0 {
		return nil, function.NewFuncCallError(
			"no arguments passed to the `coalesce` function, at least one argument is expected",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	// Return the first non-none value
	for _, param := range params {
		if !core.IsNoneMarker(param) {
			return &provider.FunctionCallOutput{
				ResponseData: param,
			}, nil
		}
	}

	// If all arguments are none, return none marker
	return &provider.FunctionCallOutput{
		ResponseData: core.GetNoneMarker(),
	}, nil
}
