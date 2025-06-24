package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// GetElemExecFunction provides the implementation of the internal
// function that executes a getelem function.
// All higher-order functions require a named function that can be called
// at execution time.
// This should only be used for executing the getelem function,
// and should only be directly referenced in the getelem function
// implementation.
type GetElemExecFunction struct {
	definition *function.Definition
}

// NewGetElemExecFunction creates a new instance of the internal GetElemExecFunction
// that is used to execute an attribute extraction function.
func NewGetElemExecFunction() provider.Function {
	return &GetElemExecFunction{
		definition: &function.Definition{
			Internal: true,
		},
	}
}

func (f *GetElemExecFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *GetElemExecFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputVal []interface{}
	var index int64
	err := input.Arguments.GetMultipleVars(ctx, &inputVal, &index)
	if err != nil {
		return nil, err
	}

	if len(inputVal) <= int(index) {
		return nil, function.NewFuncCallError(
			fmt.Sprintf("index %d out of bounds", index),
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &provider.FunctionCallOutput{
		ResponseData: inputVal[index],
	}, nil
}
