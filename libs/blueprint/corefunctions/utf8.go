package corefunctions

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// UTF8Function provides the implementation of
// a function that converts binary data to UTF-8 encoded text.
type UTF8Function struct {
	definition *function.Definition
}

// NewUTF8Function creates a new instance of the UTF8Function with
// a complete function definition.
func NewUTF8Function() provider.Function {
	return &UTF8Function{
		definition: &function.Definition{
			Description: "A function that converts binary data to UTF-8 encoded text.",
			FormattedDescription: "A function that converts binary data to UTF-8 encoded text.\n\n" +
				"**When to Use:**\n\n" +
				"The `utf8()` function is only needed when the result is passed into another function call " +
				"for processing before the final output is set to a blueprint element field. " +
				"It is **not needed** if you are setting the result directly on a blueprint element field.\n\n" +
				"**Examples:**\n\n" +
				"When `utf8()` is needed (function chaining):\n" +
				"```\n${jsondecode(utf8(file(\"config.json\")))}\n```\n" +
				"```\n${sha256(utf8(file(\"data.txt\")))}\n```\n\n" +
				"When `utf8()` is NOT needed (direct assignment):\n" +
				"```yaml\n" +
				"resources:\n" +
				"  application:\n" +
				"    type: aws/ec2/instance\n" +
				"    spec:\n" +
				"      userData: ${file(\"scripts/init.sh\")}\n" +
				"```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Label: "binary",
					Type: &function.ValueTypeDefinitionScalar{
						Label: "binary",
						Type:  function.ValueTypeBytes,
					},
					Description: "The binary data to convert to UTF-8 text.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The UTF-8 encoded text representation of the binary data.",
			},
		},
	}
}

func (f *UTF8Function) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *UTF8Function) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var data []byte
	if err := input.Arguments.GetVar(ctx, 0, &data); err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		ResponseData: string(data),
	}, nil
}
