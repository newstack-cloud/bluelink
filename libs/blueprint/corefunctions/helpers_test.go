package corefunctions

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type functionCallArgsMock struct {
	args    []any
	callCtx provider.FunctionCallContext
}

func (f *functionCallArgsMock) Get(ctx context.Context, position int) (any, error) {
	return f.args[position], nil
}

func (f *functionCallArgsMock) GetVar(ctx context.Context, position int, target any) error {
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

func (f *functionCallArgsMock) GetMultipleVars(ctx context.Context, targets ...any) error {
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

func (f *functionCallArgsMock) Export(ctx context.Context) ([]any, error) {
	return f.args, nil
}

type functionCallContextMock struct {
	params          *core.ParamsImpl
	registry        *internal.FunctionRegistryMock
	callStack       function.Stack
	currentLocation *source.Meta
}

func (f *functionCallContextMock) Registry() provider.FunctionRegistry {
	return f.registry
}

func (f *functionCallContextMock) Params() core.BlueprintParams {
	return f.params
}

func (f *functionCallContextMock) NewCallArgs(args ...any) provider.FunctionCallArguments {
	return &functionCallArgsMock{args: args, callCtx: f}
}

func (f *functionCallContextMock) CallStackSnapshot() []*function.Call {
	// Take a copy of the current call stack.
	return f.callStack.Snapshot()
}

func (f *functionCallContextMock) CurrentLocation() *source.Meta {
	return f.currentLocation
}

type comparableInt int

func (c comparableInt) Equal(other any) bool {
	otherInt, ok := other.(comparableInt)
	if !ok {
		return false
	}
	return c == otherInt
}

func sortIfaceSlice(slice []interface{}) {
	slices.SortFunc(slice, func(a, b interface{}) int {
		// Normalise to a string for comparison,
		// This is only used in tests so the actual ordering
		// doesn't matter, only that it is consistent.
		aStr := fmt.Sprintf("%v", a)
		bStr := fmt.Sprintf("%v", b)
		if aStr < bStr {
			return -1
		}

		if aStr > bStr {
			return 1
		}

		return 0
	})
}

func sortStrSlice(slice []string) {
	slices.SortFunc(slice, func(a, b string) int {
		if a < b {
			return -1
		}

		if a > b {
			return 1
		}

		return 0
	})
}

type linkStateRetrieverMock struct {
	linkState map[string]state.LinkState
}

func (s *linkStateRetrieverMock) GetByName(ctx context.Context, instanceID string, linkName string) (state.LinkState, error) {
	linkState, ok := s.linkState[fmt.Sprintf("%s::%s", instanceID, linkName)]
	if !ok {
		return state.LinkState{}, fmt.Errorf("link state not found")
	}
	return linkState, nil
}
