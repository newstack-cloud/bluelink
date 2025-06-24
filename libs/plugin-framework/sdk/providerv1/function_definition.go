package providerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// FunctionDefinition is a template to be used for defining functions
// when creating provider plugins.
// It provides a structure that allows you to define the signature and behaviour
// of a function.
// This implements the `provider.Function` interface and can be used in the same way
// as any other function implementation used in a provider plugin.
type FunctionDefinition struct {

	// The definition that describes the signature of the function along with
	// useful information that can be used in tooling and documentation.
	Definition *function.Definition

	// Provides the behaviour for when the function plugin is called.
	CallFunc func(
		ctx context.Context,
		input *provider.FunctionCallInput,
	) (*provider.FunctionCallOutput, error)
}

func (f *FunctionDefinition) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.Definition,
	}, nil
}

func (f *FunctionDefinition) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	if f.CallFunc == nil {
		functionName := getFunctionName(f.Definition)
		return nil, errFunctionCallFunctionMissing(functionName)
	}

	return f.CallFunc(ctx, input)
}

func getFunctionName(definition *function.Definition) string {
	if definition == nil || definition.Name == "" {
		return "anonymous function"
	}

	return definition.Name
}
