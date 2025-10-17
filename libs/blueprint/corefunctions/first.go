package corefunctions

import (
	"context"
	"reflect"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// FirstFunction provides the implementation of
// a function that returns the first value that is not empty
// from a list of values.
type FirstFunction struct {
	definition *function.Definition
}

// NewFirstFunction creates a new instance of the FirstFunction with
// a complete function definition.
func NewFirstFunction() provider.Function {
	return &FirstFunction{
		definition: &function.Definition{
			Description: "A function that returns the first non-empty value from a list of arguments. " +
				"Empty values include empty strings (\"\"), and empty arrays.",
			FormattedDescription: "A function that returns the first non-empty value from a list of arguments. " +
				"Empty values include empty strings (\"\"), and empty arrays.\n\n" +
				"**Examples:**\n\n" +
				"```\n${first(list(\"\", \"item2\", [], \"item4\"))}\n```",
			Parameters: []function.Parameter{
				&function.VariadicParameter{
					Label: "values",
					Type: &function.ValueTypeDefinitionAny{
						Type:  function.ValueTypeAny,
						Label: "any",
					},
					Description: "N arguments of any type that will be used to find the first non-empty value.",
				},
			},
			Return: &function.AnyReturn{
				Type:        function.ValueTypeAny,
				Description: "The first non-empty value from the list of arguments or the last value if all values are empty.",
			},
		},
	}
}

func (f *FirstFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *FirstFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var params []any
	if err := input.Arguments.GetVar(ctx, 0, &params); err != nil {
		return nil, err
	}

	if len(params) == 0 {
		return nil, function.NewFuncCallError(
			"no arguments passed to the `first` function, at least one argument is expected",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	for _, param := range params {
		reflectParam := reflect.ValueOf(param)
		isSlice := reflect.ValueOf(param).Kind() == reflect.Slice
		if param != nil && param != "" && (isSlice && reflectParam.Len() > 0) {
			return &provider.FunctionCallOutput{
				ResponseData: param,
			}, nil
		}
	}

	return &provider.FunctionCallOutput{
		ResponseData: params[len(params)-1],
	}, nil
}
