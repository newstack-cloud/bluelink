package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// GetAttrExecFunction provides the implementation of the internal
// function that executes a getattr function.
// All higher-order functions require a named function that can be called
// at execution time.
// This should only be used for executing the getattr function,
// and should only be directly referenced in the getattr function
// implementation.
type GetAttrExecFunction struct {
	definition *function.Definition
}

// NewGetAttrExecFunction creates a new instance of the internal GetAttrExecFunction
// that is used to execute an attribute extraction function.
func NewGetAttrExecFunction() provider.Function {
	return &GetAttrExecFunction{
		definition: &function.Definition{
			Internal: true,
		},
	}
}

func (f *GetAttrExecFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *GetAttrExecFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputVal map[string]interface{}
	var attributeName string
	err := input.Arguments.GetMultipleVars(ctx, &inputVal, &attributeName)
	if err != nil {
		return nil, err
	}

	result, ok := inputVal[attributeName]
	if !ok {
		return nil, function.NewFuncCallError(
			fmt.Sprintf("attribute %q not found in object/mapping", attributeName),
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: result,
	}, nil
}
