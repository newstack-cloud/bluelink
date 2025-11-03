package corefunctions

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// Base64DecodeFunction provides the implementation of
// a function that decodes a base64 string to a byte array.
type Base64DecodeFunction struct {
	definition *function.Definition
}

// NewBase64DecodeFunction creates a new instance of the Base64DecodeFunction with
// a complete function definition.
func NewBase64DecodeFunction() provider.Function {
	return &Base64DecodeFunction{
		definition: &function.Definition{
			Description: "A function that decodes a Base64-encoded string back to binary data using the standard RFC 4648 encoding (including padding).",
			FormattedDescription: "A function that decodes a Base64-encoded string back to binary data using the standard " +
				"[RFC 4648](https://datatracker.ietf.org/doc/rfc4648/) encoding (including padding).\n\n" +
				"**Examples:**\n\n" +
				"```\n${base64decode(\"SGVsbG8gV29ybGQ=\")\n```\n\n" +
				"Decoding and parsing JSON:\n" +
				"```\n${jsondecode(utf8(base64decode(variables.encodedConfig)))}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "base64String",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A Base64-encoded string to decode back to binary data.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "byte array",
					Type:  function.ValueTypeBytes,
				},
				Description: "The decoded binary data.",
			},
		},
	}
}

func (f *Base64DecodeFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *Base64DecodeFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	// Get argument as any to check for none marker
	valueAny, err := input.Arguments.Get(ctx, 0)
	if err != nil {
		return nil, err
	}

	// If input is none, propagate none
	if core.IsNoneMarker(valueAny) {
		return &provider.FunctionCallOutput{
			ResponseData: core.GetNoneMarker(),
		}, nil
	}

	var value string
	if err := input.Arguments.GetVar(ctx, 0, &value); err != nil {
		return nil, err
	}

	valueBytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, function.NewFuncCallError(
			fmt.Sprintf("unable to decode base64 string: %s", err.Error()),
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: valueBytes,
	}, nil
}
