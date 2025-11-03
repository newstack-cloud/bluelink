package corefunctions

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// MD5Function provides the implementation of
// a function that computes the MD5 hash of input data.
type MD5Function struct {
	definition *function.Definition
}

// NewMD5Function creates a new instance of the MD5Function with
// a complete function definition.
func NewMD5Function() provider.Function {
	return &MD5Function{
		definition: &function.Definition{
			Description: "A function that computes the MD5 hash of the input data and returns it as a hexadecimal string.",
			FormattedDescription: "A function that computes the MD5 hash of the input data and returns it as a hexadecimal string.\n\n" +
				"**Examples:**\n\n" +
				"Computing MD5 hash:\n" +
				"```\n${md5(\"Hello World\")}\n```\n\n" +
				"Using MD5 for file integrity:\n" +
				"```\n${md5(file(\"config.json\"))}\n```",
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
				Description: "The MD5 hash as a hexadecimal string.",
			},
		},
	}
}

func (f *MD5Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *MD5Function) Call(
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

	hash := md5.Sum(data)
	return &provider.FunctionCallOutput{
		ResponseData: hex.EncodeToString(hash[:]),
	}, nil
}
