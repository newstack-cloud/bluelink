package provider

import (
	"context"

	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/function"
)

// Function is the interface for an implementation of a function
// that can be used in a blueprint "${..}" substitution.
type Function interface {
	// GetDefinition returns the definition of the function
	// that includes allowed parameters and return types.
	// This would usually be called during initialisation of a provider
	// to pre-fetch function definitions and cache them to validate
	// the returned values from plugin function calls.
	GetDefinition(ctx context.Context, input *FunctionGetDefinitionInput) (*FunctionGetDefinitionOutput, error)
	// Call is the function that is called when a function is used in a blueprint.
	// The function should return the result of the function call as a string.
	// Tools built on top of the framework should provide custom error types
	// that can be used to distinguish logical function call errors from other
	// errors that may occur during a function call.
	Call(ctx context.Context, input *FunctionCallInput) (*FunctionCallOutput, error)
}

// FunctionGetDefinitionInput provides the input data for retrieving
// the definition of a function.
type FunctionGetDefinitionInput struct {
	Params core.BlueprintParams
}

// FunctionGetDefinitionOutput provides the output data for retrieving
// the definition of a function.
type FunctionGetDefinitionOutput struct {
	Definition *function.Definition
}

// FunctionCallInput provides the input data needed for a substitution function
// to be called.
type FunctionCallInput struct {
	Arguments   FunctionCallArguments
	CallContext FunctionCallContext
}

// FunctionCallArguments provides a way to fetch the arguments passed
// to a function call.
type FunctionCallArguments interface {
	// Get retrieves the argument at the given position.
	Get(ctx context.Context, position int) (any, error)
	// GetVar writes the argument at the given position to the
	// provided target.
	GetVar(ctx context.Context, position int, target any) error
	// GetMultipleVars writes the arguments to the provided targets
	// in the order they were passed to the function.
	GetMultipleVars(ctx context.Context, targets ...any) error
}

// FunctionCallContext provides useful context for functions
// that allow access to "global" blueprint params and the ability
// to call other functions loaded into the provider.
// Functions should not be able to access directly access state,
// values from state should be resolved before calling a function.
type FunctionCallContext interface {
	// Registry retrieves the function registry that can be used to
	// call other functions from within a function.
	// This would be akin to importing functions from a module
	// to be called in a typical programming language.
	Registry() FunctionRegistry
	// Params retrieves the current context blueprint params
	// treated like global data from the perspective of function
	// implementations.
	Params() core.BlueprintParams
	// NewCallArgs creates a new instance of FunctionCallArguments
	// to be passed into a function call.
	NewCallArgs(args ...any) FunctionCallArguments
	// CallStackSnapshot provides a snapshot of the current call stack
	// to be used in error messages.
	CallStackSnapshot() []*function.Call
}

// FunctionRegistry provides a way to retrieve function plugins
// to call from other functions.
// Instead of returning a function directly, a registry allows
// calling functions through the registry as a proxy to allow
// for adding calls to a stack along with other context-specific
// enhancements that may be needed.
type FunctionRegistry interface {
	// Call allows calling a function in the registry by name.
	Call(ctx context.Context, functionName string, input *FunctionCallInput) (*FunctionCallOutput, error)
}

// FunctionCallOutput provides the output data from a substitution function
// call.
type FunctionCallOutput struct {
	ResponseData any
	FunctionInfo FunctionRuntimeInfo
}

// FunctionRuntimeInfo provides information about a function to be passed
// between functions.
// The blueprint function framework is designed to work across process boundaries
// so an actual function in memory can not be passed around, Instead, a function
// runtime info is used , this contains the function name to be called and pre-configured
// arguments that can be used when the function is eventually called.
//
// Higher-order functions can only use named functions for the return value
// as the function name is used to look up the function definition and combine
// the pre-configured arguments with the arguments passed to the function.
type FunctionRuntimeInfo struct {
	FunctionName string
	PartialArgs  []any
	// Specify the offset of the arguments in the partial arguments.
	// This should be rarely be used, but in the case where the captured
	// arguments to be "partially applied" are not the first arguments
	// in the function signature, this can be used to specify the offset
	// of the arguments in the partial arguments list.
	ArgsOffset int
}
