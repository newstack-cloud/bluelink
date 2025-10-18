package corefunctions

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// SHA1Function provides the implementation of
// a function that computes the SHA-1 hash of input data.
type SHA1Function struct {
	definition *function.Definition
}

// NewSHA1Function creates a new instance of the SHA1Function with
// a complete function definition.
func NewSHA1Function() provider.Function {
	return &SHA1Function{
		definition: &function.Definition{
			Description: "A function that computes the SHA-1 hash of the input data and returns it as a hexadecimal string.",
			FormattedDescription: "A function that computes the SHA-1 hash of the input data and returns it as a hexadecimal string.\n\n" +
				"**Examples:**\n\n" +
				"Computing SHA-1 hash:\n" +
				"```\n${sha1(\"Hello World\")}\n```\n\n" +
				"Using SHA-1 for data integrity:\n" +
				"```\n${sha1(values.configData)}\n```",
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
				Description: "The SHA-1 hash as a hexadecimal string.",
			},
		},
	}
}

func (f *SHA1Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *SHA1Function) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var value any
	if err := input.Arguments.GetVar(ctx, 0, &value); err != nil {
		return nil, err
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

	hash := sha1.Sum(data)
	return &provider.FunctionCallOutput{
		ResponseData: hex.EncodeToString(hash[:]),
	}, nil
}
