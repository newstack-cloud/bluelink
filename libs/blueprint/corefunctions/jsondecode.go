package corefunctions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// JSONDecodeFunction provides the implementation of
// a function that decodes a JSON string.
type JSONDecodeFunction struct {
	definition *function.Definition
}

// NewJSONDecodeFunction creates a new instance of the JSONDecodeFunction with
// a complete function definition.
func NewJSONDecodeFunction() provider.Function {
	return &JSONDecodeFunction{
		definition: &function.Definition{
			Description: "Decodes a serialised json string into a primitive value, array or mapping.",
			FormattedDescription: "Decodes a serialised json string into a primitive value, array or mapping.\n\n" +
				"**Examples:**\n\n" +
				"```\n${jsondecode(variables.cacheClusterConfig)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "jsonString",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string literal, reference or function call yielding the json string to decode.",
				},
			},
			Return: &function.AnyReturn{
				Type:        function.ValueTypeAny,
				Description: "The decoded json string. This could be a primitive value, an array, or a mapping.",
			},
		},
	}
}

func (f *JSONDecodeFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *JSONDecodeFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var jsonStr string
	if err := input.Arguments.GetVar(ctx, 0, &jsonStr); err != nil {
		return nil, err
	}

	var output interface{}
	err := json.Unmarshal([]byte(jsonStr), &output)
	if err != nil {
		return nil, function.NewFuncCallError(
			fmt.Sprintf("unable to decode json string: %s", err.Error()),
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: output,
	}, nil
}
