package subengine

import (
	"context"
	"fmt"
	"reflect"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

type functionCallDependencies struct {
	scopedRegistry provider.FunctionRegistry
	callCtx        provider.FunctionCallContext
}

func createFunctionCallDependencies(
	rootRegistry provider.FunctionRegistry,
	params bpcore.BlueprintParams,
	location *source.Meta,
) *functionCallDependencies {
	stack := function.NewStack()
	scopedFunctionRegistry := rootRegistry.ForCallContext(stack)
	functionCallContext := NewFunctionCallContext(
		stack,
		scopedFunctionRegistry,
		params,
		location,
	)
	return &functionCallDependencies{
		scopedRegistry: scopedFunctionRegistry,
		callCtx:        functionCallContext,
	}
}

type functionCallArgs struct {
	args    []any
	callCtx provider.FunctionCallContext
}

func newFunctionCallArgs(args []any, callCtx provider.FunctionCallContext) *functionCallArgs {
	return &functionCallArgs{
		args,
		callCtx,
	}
}

func (f *functionCallArgs) Get(ctx context.Context, position int) (any, error) {
	return f.args[position], nil
}

func (f *functionCallArgs) GetVar(ctx context.Context, position int, target any) error {
	val := reflect.ValueOf(target)
	if position >= len(f.args) {
		return function.NewFuncCallError(
			fmt.Sprintf("argument at index %d not found", position),
			function.FuncCallErrorCodeFunctionCall,
			f.callCtx.CallStackSnapshot(),
		)
	}

	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return function.NewFuncCallError(
			"target to read argument into is not a pointer",
			function.FuncCallErrorCodeInvalidArgumentType,
			f.callCtx.CallStackSnapshot(),
		)
	}

	argVal := reflect.ValueOf(f.args[position])
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
			f.callCtx.CallStackSnapshot(),
		)
	}

	val.Elem().Set(reflect.ValueOf(f.args[position]))
	return nil
}

func (f *functionCallArgs) GetMultipleVars(ctx context.Context, targets ...any) error {
	for i := 0; i < len(f.args); i += 1 {
		if i < len(targets) {
			targetVal := reflect.ValueOf(targets[i])
			if targetVal.Kind() != reflect.Ptr {
				return function.NewFuncCallError(
					fmt.Sprintf("target at index %d to read argument into is not a pointer", i),
					function.FuncCallErrorCodeInvalidArgumentType,
					f.callCtx.CallStackSnapshot(),
				)
			}

			argVal := reflect.ValueOf(f.args[i])
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
					f.callCtx.CallStackSnapshot(),
				)
			}
			targetVal.Elem().Set(argVal)
		}
	}

	if len(targets) > len(f.args) {
		expectedText := fmt.Sprintf("%d arguments expected", len(targets))
		if len(targets) == 1 {
			expectedText = "1 argument expected"
		}
		argsText := fmt.Sprintf(", but %d arguments were passed into function", len(f.args))
		if len(f.args) == 1 {
			argsText = ", but 1 argument was passed into function"
		}

		return function.NewFuncCallError(
			fmt.Sprintf(
				"%s%s",
				expectedText,
				argsText,
			),
			function.FuncCallErrorCodeFunctionCall,
			f.callCtx.CallStackSnapshot(),
		)
	}
	return nil
}

func (f *functionCallArgs) Export(ctx context.Context) ([]any, error) {
	return f.args, nil
}

type functionCallContext struct {
	stack    function.Stack
	registry provider.FunctionRegistry
	params   bpcore.BlueprintParams
	location *source.Meta
}

// NewFunctionCallContext creates a new call context that can be passed into function
// plugin calls and can be used to generate a function call arguments service that
// function plugins can call to retrieve arguments from.
func NewFunctionCallContext(
	stack function.Stack,
	registry provider.FunctionRegistry,
	params bpcore.BlueprintParams,
	location *source.Meta,
) provider.FunctionCallContext {
	return &functionCallContext{
		stack:    stack,
		registry: registry,
		params:   params,
		location: location,
	}
}

func (c *functionCallContext) Registry() provider.FunctionRegistry {
	return c.registry
}

func (c *functionCallContext) Params() bpcore.BlueprintParams {
	return c.params
}

func (c *functionCallContext) NewCallArgs(args ...any) provider.FunctionCallArguments {
	return newFunctionCallArgs(args, c)
}

func (c *functionCallContext) CallStackSnapshot() []*function.Call {
	return c.stack.Snapshot()
}

func (c *functionCallContext) CurrentLocation() *source.Meta {
	return c.location
}

type resolvedFunctionCallValue struct {
	value    *bpcore.MappingNode
	function provider.FunctionRuntimeInfo
}
