package testutils

import (
	"context"
	"fmt"
	"reflect"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

type FunctionCallArgsMock struct {
	Args    []any
	CallCtx provider.FunctionCallContext
}

func (f *FunctionCallArgsMock) Get(ctx context.Context, position int) (any, error) {
	return f.Args[position], nil
}

func (f *FunctionCallArgsMock) GetVar(ctx context.Context, position int, target any) error {
	val := reflect.ValueOf(target)
	if position >= len(f.Args) {
		return function.NewFuncCallError(
			fmt.Sprintf("argument at index %d not found", position),
			function.FuncCallErrorCodeFunctionCall,
			f.CallCtx.CallStackSnapshot(),
		)
	}

	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return function.NewFuncCallError(
			"target to read argument into is not a pointer",
			function.FuncCallErrorCodeInvalidArgumentType,
			f.CallCtx.CallStackSnapshot(),
		)
	}

	argVal := reflect.ValueOf(f.Args[position])
	// Allow interface{} as a target type so that the caller can carry out type assertions
	// when an argument can be of multiple types.
	if targetVal.Elem().Kind() != reflect.Interface && targetVal.Elem().Kind() != argVal.Kind() {
		return function.NewFuncCallError(
			fmt.Sprintf(
				"argument at index %d is of type %s, but target is of type %s",
				position,
				argVal.Kind(),
				targetVal.Elem().Kind(),
			),
			function.FuncCallErrorCodeInvalidArgumentType,
			f.CallCtx.CallStackSnapshot(),
		)
	}

	val.Elem().Set(reflect.ValueOf(f.Args[position]))
	return nil
}

func (f *FunctionCallArgsMock) GetMultipleVars(ctx context.Context, targets ...any) error {
	for i := 0; i < len(f.Args); i += 1 {
		if i < len(targets) {
			targetVal := reflect.ValueOf(targets[i])
			if targetVal.Kind() != reflect.Ptr {
				return function.NewFuncCallError(
					fmt.Sprintf("target at index %d to read argument into is not a pointer", i),
					function.FuncCallErrorCodeInvalidArgumentType,
					f.CallCtx.CallStackSnapshot(),
				)
			}

			argVal := reflect.ValueOf(f.Args[i])
			// Allow interface{} as a target type so that the caller can carry out type assertions
			// when an argument can be of multiple types.
			if targetVal.Elem().Kind() != reflect.Interface && targetVal.Elem().Kind() != argVal.Kind() {
				return function.NewFuncCallError(
					fmt.Sprintf(
						"argument at index %d is of type %s, but target is of type %s",
						i,
						argVal.Kind(),
						targetVal.Elem().Kind(),
					),
					function.FuncCallErrorCodeInvalidArgumentType,
					f.CallCtx.CallStackSnapshot(),
				)
			}
			targetVal.Elem().Set(argVal)
		}
	}

	if len(targets) > len(f.Args) {
		expectedText := fmt.Sprintf("%d arguments expected", len(targets))
		if len(targets) == 1 {
			expectedText = "1 argument expected"
		}
		argsText := fmt.Sprintf(", but %d arguments were passed into function", len(f.Args))
		if len(f.Args) == 1 {
			argsText = ", but 1 argument was passed into function"
		}

		return function.NewFuncCallError(
			fmt.Sprintf(
				"%s%s",
				expectedText,
				argsText,
			),
			function.FuncCallErrorCodeFunctionCall,
			f.CallCtx.CallStackSnapshot(),
		)
	}
	return nil
}

func (f *FunctionCallArgsMock) Export(ctx context.Context) ([]any, error) {
	return f.Args, nil
}

type FunctionCallContextMock struct {
	CallCtxParams          *core.ParamsImpl
	CallCtxRegistry        provider.FunctionRegistry
	CallStack              function.Stack
	CallCtxCurrentLocation *source.Meta
}

func (f *FunctionCallContextMock) Registry() provider.FunctionRegistry {
	return f.CallCtxRegistry
}

func (f *FunctionCallContextMock) Params() core.BlueprintParams {
	return f.CallCtxParams
}

func (f *FunctionCallContextMock) NewCallArgs(args ...any) provider.FunctionCallArguments {
	return &FunctionCallArgsMock{Args: args, CallCtx: f}
}

func (f *FunctionCallContextMock) CallStackSnapshot() []*function.Call {
	// Take a copy of the current call stack.
	return f.CallStack.Snapshot()
}

func (f *FunctionCallContextMock) CurrentLocation() *source.Meta {
	return f.CallCtxCurrentLocation
}
