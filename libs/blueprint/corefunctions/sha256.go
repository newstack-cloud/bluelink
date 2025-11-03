package corefunctions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// SHA256Function provides the implementation of
// a function that computes the SHA-256 hash of input data.
type SHA256Function struct {
	definition *function.Definition
}

// NewSHA256Function creates a new instance of the SHA256Function with
// a complete function definition.
func NewSHA256Function() provider.Function {
	return &SHA256Function{
		definition: &function.Definition{
			Description: "A function that computes the SHA-256 hash of the input data and returns it as a hexadecimal string.",
			FormattedDescription: "A function that computes the SHA-256 hash of the input data and returns it as a hexadecimal string.\n\n" +
				"**Examples:**\n\n" +
				"Computing hash of a string:\n" +
				"```\n${sha256(\"Hello World\")}\n```\n\n" +
				"Computing hash of binary data:\n" +
				"```\n${sha256(base64decode(datasources.certificate.certificateBody))}\n```\n\n" +
				"Using hash for resource naming:\n" +
				"```\nmy-bucket-${sha256(variables.environment)}\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label: "data",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionScalar{
							Label: "string",
							Type:  function.ValueTypeString,
						},
						&function.ValueTypeDefinitionScalar{
							Label: "bytes",
							Type:  function.ValueTypeBytes,
						},
					},
					Description: "The data to compute the hash for. This can be a string or a byte array.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The SHA-256 hash as a hexadecimal string.",
			},
		},
	}
}

func (f *SHA256Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *SHA256Function) Call(
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

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, function.NewFuncCallError(
			fmt.Sprintf("input argument at index 0 must be a string or byte array, got %T", value),
			function.FuncCallErrorCodeInvalidArgumentType,
			input.CallContext.CallStackSnapshot(),
		)
	}

	hash := sha256.Sum256(data)
	return &provider.FunctionCallOutput{
		ResponseData: hex.EncodeToString(hash[:]),
	}, nil
}
