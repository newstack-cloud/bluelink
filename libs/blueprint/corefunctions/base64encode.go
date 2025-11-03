package corefunctions

import (
	"context"
	"encoding/base64"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// Base64EncodeFunction provides the implementation of
// a function that encodes a string to base64.
type Base64EncodeFunction struct {
	definition *function.Definition
}

// NewBase64EncodeFunction creates a new instance of the Base64EncodeFunction with
// a complete function definition.
func NewBase64EncodeFunction() provider.Function {
	return &Base64EncodeFunction{
		definition: &function.Definition{
			Description: "A function that encodes binary data to Base64 format using the standard RFC 4648 encoding (including padding).",
			FormattedDescription: "A function that encodes binary data to Base64 format using the standard " +
				"[RFC 4648](https://datatracker.ietf.org/doc/rfc4648/) encoding (including padding).\n\n" +
				"**Examples:**\n\n" +
				"```\n${base64encode(\"Encode me!\")\n```\n" +
				"```\n${base64encode(file(\"certificates/server.pem\"))\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label: "stringOrByteArray",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionScalar{
							Label: "string",
							Type:  function.ValueTypeString,
						},
						&function.ValueTypeDefinitionScalar{
							Label: "byte array",
							Type:  function.ValueTypeBytes,
						},
					},
					Description: "A string or byte array to encode to Base64.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The Base64 encoded string.",
			},
		},
	}
}

func (f *Base64EncodeFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *Base64EncodeFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var value any
	if err := input.Arguments.GetVar(ctx, 0, &value); err != nil {
		return nil, err
	}

	// If input is none, propagate none
	if core.IsNoneMarker(value) {
		return &provider.FunctionCallOutput{
			ResponseData: core.GetNoneMarker(),
		}, nil
	}

	var valueBytes []byte
	switch v := value.(type) {
	case string:
		valueBytes = []byte(v)
	case []byte:
		valueBytes = v
	default:
		return nil, function.NewFuncCallError(
			"input argument at index 0 must be a byte array or string",
			function.FuncCallErrorCodeInvalidArgumentType,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: base64.StdEncoding.EncodeToString(valueBytes),
	}, nil
}
