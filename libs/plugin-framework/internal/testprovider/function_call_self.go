package testprovider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func functionCallSelf() provider.Function {
	return &providerv1.FunctionDefinition{
		Definition: CallSelfFunctionDefinition(),
		CallFunc:   callSelf,
	}
}

// CallSelfFunctionDefinition returns the definition of a function that calls
// itself. This is used to test the mechanism that prevents infinite
// recursion in plugin function calls.
func CallSelfFunctionDefinition() *function.Definition {
	return &function.Definition{
		Name:        "call_self",
		Description: "A function that calls itself.",
		Parameters:  []function.Parameter{},
		Return: &function.ScalarReturn{
			Type: &function.ValueTypeDefinitionScalar{
				Label: "string",
				Type:  function.ValueTypeString,
			},
			Description: "A static string value.",
		},
	}
}

func callSelf(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	registry := input.CallContext.Registry()
	args := input.CallContext.NewCallArgs()
	_, err := registry.Call(
		ctx,
		"call_self",
		&provider.FunctionCallInput{
			Arguments: args,
			// We can safely pass through the same call context
			// to the registry, the registry will make sure
			// that this function call will be pushed to the stack.
			// It is important to make this clear to developers in
			// documentation and guides.
			CallContext: input.CallContext,
		},
	)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		ResponseData: "Some value",
	}, nil
}
