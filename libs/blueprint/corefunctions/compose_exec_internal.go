package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ComposeExecFunction provides the implementation of the internal
// function that executes a composed function.
// All higher-order functions require a named function that can be called
// at execution time.
// This should only be used for executing composed functions,
// and should only be directly referenced in the compose function
// implementation.
type ComposeExecFunction struct {
	definition *function.Definition
}

// NewComposeExecFunction creates a new instance of the internal ComposeExecFunction
// that is used to execute functions that have been composed together.
func NewComposeExecFunction() provider.Function {
	return &ComposeExecFunction{
		definition: &function.Definition{
			Internal: true,
		},
	}
}

func (f *ComposeExecFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ComposeExecFunction) Call(
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
	for i := len(functions) - 1; i >= 0; i -= 1 {
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
