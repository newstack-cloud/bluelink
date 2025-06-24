package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// WorkingDirResolver is a function that resolves the current working directory
// from a blueprint function execution context.
type WorkingDirResolver func() (string, error)

// CWDFunction provides the implementation of
// a function that gets the user's current working directory.
type CWDFunction struct {
	definition        *function.Definition
	resolveWorkingDir WorkingDirResolver
}

// NewCWDFunction creates a new instance of the CWDFunction with
// a complete function definition.
func NewCWDFunction(resolveWorkingDir WorkingDirResolver) provider.Function {
	return &CWDFunction{
		resolveWorkingDir: resolveWorkingDir,
		definition: &function.Definition{
			Description: "A function that returns the user's current working directory as set on the host system, " +
				"by a tool implementing the spec or by the user.",
			FormattedDescription: "A function that returns the user's current working directory as set on the host system, " +
				"by a tool implementing the spec or by the user.\n\n" +
				"**Examples:**\n\n" +
				"```\n${cwd()}/blueprints/core-infra.blueprint.yaml\n```",
			Parameters: []function.Parameter{},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The current working directory of the user.",
			},
		},
	}
}

func (f *CWDFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *CWDFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	workingDir, err := f.resolveWorkingDir()
	if err != nil {
		return nil, function.NewFuncCallError(
			fmt.Sprintf("unable to get current working directory: %s", err.Error()),
			function.FuncCallErrorCodeFunctionCall,
			input.CallContext.CallStackSnapshot(),
		)
	}
	return &provider.FunctionCallOutput{
		ResponseData: workingDir,
	}, nil
}
