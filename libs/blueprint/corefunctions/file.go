package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// FileFunction provides the implementation of
// a function that reads binary data from a file path.
type FileFunction struct {
	definition *function.Definition
}

// NewFileFunction creates a new instance of the FileFunction with
// a complete function definition.
func NewFileFunction() provider.Function {
	return &FileFunction{
		definition: &function.Definition{
			Description: "A function that reads binary data from a file path (local or remote) and returns it as raw bytes.",
			FormattedDescription: "A function that reads binary data from a file path (local or remote) and returns it as raw bytes.\n\n" +
				"**Examples:**\n\n" +
				"Reading local files:\n" +
				"```\n${file(\"certificates/server.pem\")}\n```\n" +
				"```\n${file(\"dist/function.zip\")}\n```\n\n" +
				"Reading remote files:\n" +
				"```\n${file(\"s3://my-bucket/certificates/server.pem\")}\n```\n\n" +
				"**Note:** When the output of `file()` is used as the value of a blueprint element field, " +
				"it will be automatically UTF-8 encoded by default.",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "path",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "The path to the file to read (local path or URI).",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "binary",
					Type:  function.ValueTypeBytes,
				},
				Description: "The raw binary data from the file.",
			},
		},
	}
}

func (f *FileFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *FileFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var path string
	if err := input.Arguments.GetVar(ctx, 0, &path); err != nil {
		return nil, err
	}

	// Use the file source registry to read the file
	// This allows host applications to register custom handlers
	// for different URI schemes (s3://, gs://, etc.)
	fileSourceRegistry := input.CallContext.FileSourceRegistry()
	data, err := fileSourceRegistry.ReadFile(ctx, path)
	if err != nil {
		return nil, function.NewFuncCallError(
			fmt.Sprintf("unable to read file at path %q: %s", path, err.Error()),
			function.FuncCallErrorCodeFunctionCall,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: data,
	}, nil
}
