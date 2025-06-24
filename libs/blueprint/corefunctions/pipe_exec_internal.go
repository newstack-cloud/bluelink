package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// PipeExecFunction provides the implementation of the internal
// function that executes a piped function.
// All higher-order functions require a named function that can be called
// at execution time.
// This should only be used for executing piped functions,
// and should only be directly referenced in the pipe function
// implementation.
type PipeExecFunction struct {
	definition *function.Definition
}

// NewPipeExecFunction creates a new instance of the internal PipeExecFunction
// that is used to execute functions that have been piped together.
func NewPipeExecFunction() provider.Function {
	return &PipeExecFunction{
		definition: &function.Definition{
			Internal: true,
		},
	}
}

func (f *PipeExecFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *PipeExecFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var inputVal interface{}
	var functions []provider.FunctionRuntimeInfo
	err := input.Arguments.GetMultipleVars(ctx, &inputVal, &functions)
	if err != nil {
		return nil, err
	}

	current := inputVal
	for i := 0; i < len(functions); i += 1 {
		currentFunc := functions[i]
		funcName := currentFunc.FunctionName
		args := []any{current}
		if functions[i].ArgsOffset > 1 {
			return nil, function.NewFuncCallError(
				fmt.Sprintf(
					"invalid args offset defined for the partially applied function \"%s\"",
					funcName,
				),
				function.FuncCallErrorCodeInvalidArgsOffset,
				input.CallContext.CallStackSnapshot(),
			)
		} else if functions[i].ArgsOffset == 1 {
			args = append(args, currentFunc.PartialArgs...)
		} else {
			args = append(currentFunc.PartialArgs, args...)
		}

		output, err := input.CallContext.Registry().Call(ctx, funcName, &provider.FunctionCallInput{
			Arguments:   input.CallContext.NewCallArgs(args...),
			CallContext: input.CallContext,
		})
		if err != nil {
			return nil, err
		}

		current = output.ResponseData
	}

	return &provider.FunctionCallOutput{
		ResponseData: current,
	}, nil
}
