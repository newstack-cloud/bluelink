package provider

import (
	"context"

	"github.com/two-hundred/celerity/libs/blueprint/errors"
	"github.com/two-hundred/celerity/libs/blueprint/function"
)

// FunctionRegistry provides a way to retrieve function plugins
// to call from other functions.
// Instead of returning a function directly, a registry allows
// calling functions through the registry as a proxy to allow
// for adding calls to a stack along with other context-specific
// enhancements that may be needed.
type FunctionRegistry interface {
	// Call allows calling a function in the registry by name.
	Call(ctx context.Context, functionName string, input *FunctionCallInput) (*FunctionCallOutput, error)

	// GetDefinition returns the definition of a function
	// in the registry that includes allowed parameters and return types.
	GetDefinition(
		ctx context.Context,
		functionName string,
		input *FunctionGetDefinitionInput,
	) (*FunctionGetDefinitionOutput, error)

	// HasFunction checks if a function is available in the registry.
	HasFunction(ctx context.Context, functionName string) (bool, error)

	// ListFunctions retrieves a list of all the functions avaiable
	// in the registry.
	ListFunctions(ctx context.Context) ([]string, error)
}

type functionRegistryFromProviders struct {
	providers             map[string]Provider
	functionProviderCache map[string]Provider
	functionCache         map[string]Function
	functionNames         []string
	callStack             function.Stack
}

// NewFunctionRegistry creates a new FunctionRegistry from a map of providers,
// matching against providers based on the the list of functions that a provider
// exposes.
func NewFunctionRegistry(
	providers map[string]Provider,
) FunctionRegistry {
	return &functionRegistryFromProviders{
		providers:             providers,
		functionProviderCache: map[string]Provider{},
		functionCache:         map[string]Function{},
		functionNames:         []string{},
		callStack:             function.NewStack(),
	}
}

func (r *functionRegistryFromProviders) Call(
	ctx context.Context,
	functionName string,
	input *FunctionCallInput,
) (*FunctionCallOutput, error) {
	functionImpl, err := r.getFunction(ctx, functionName)
	if err != nil {
		return nil, err
	}

	r.callStack.Push(&function.Call{
		FunctionName: functionName,
		Location:     input.CallContext.CurrentLocation(),
	})

	output, err := functionImpl.Call(ctx, input)
	r.callStack.Pop()
	return output, err
}

func (r *functionRegistryFromProviders) GetDefinition(
	ctx context.Context,
	functionName string,
	input *FunctionGetDefinitionInput,
) (*FunctionGetDefinitionOutput, error) {
	functionImpl, err := r.getFunction(ctx, functionName)
	if err != nil {
		return nil, err
	}

	return functionImpl.GetDefinition(ctx, input)
}

func (r *functionRegistryFromProviders) HasFunction(ctx context.Context, functionName string) (bool, error) {
	functionImpl, err := r.getFunction(ctx, functionName)
	if err != nil {
		if runErr, isRunErr := err.(*errors.RunError); isRunErr {
			if runErr.ReasonCode == ErrorReasonCodeProviderFunctionNotFound ||
				runErr.ReasonCode == ErrorReasonCodeFunctionNotFound {
				return false, nil
			}
		}
		return false, err
	}
	return functionImpl != nil, nil
}

func (r *functionRegistryFromProviders) ListFunctions(ctx context.Context) ([]string, error) {
	if len(r.functionNames) > 0 {
		return r.functionNames, nil
	}

	functionNames := []string{}
	for _, provider := range r.providers {
		functions, err := provider.ListFunctions(ctx)
		if err != nil {
			return nil, err
		}

		functionNames = append(functionNames, functions...)
	}

	r.functionNames = functionNames

	return functionNames, nil
}

func (r *functionRegistryFromProviders) getFunction(ctx context.Context, functionName string) (Function, error) {
	function, cached := r.functionCache[functionName]
	if cached {
		return function, nil
	}

	funcProvider, funcProviderCached := r.functionProviderCache[functionName]
	if !funcProviderCached {
		err := r.registerProviderFunctions(ctx)
		if err != nil {
			return nil, err
		}
		funcProvider, funcProviderCached = r.functionProviderCache[functionName]
		if !funcProviderCached {
			return nil, errFunctionNotFound(functionName)
		}
	}

	providerNamespace, err := funcProvider.Namespace(ctx)
	if err != nil {
		return nil, err
	}

	functionImpl, err := funcProvider.Function(ctx, functionName)
	if err != nil {
		return nil, errFunctionNotFoundInProvider(functionName, providerNamespace)
	}
	r.functionCache[functionName] = functionImpl
	return functionImpl, nil
}

func (r *functionRegistryFromProviders) registerProviderFunctions(ctx context.Context) error {
	for _, provider := range r.providers {
		functions, err := provider.ListFunctions(ctx)
		if err != nil {
			return err
		}

		for _, functionName := range functions {
			if providedBy, alreadyProvided := r.functionProviderCache[functionName]; alreadyProvided {
				err := handleFunctionProviderConflict(ctx, functionName, providedBy, provider)
				if err != nil {
					return err
				}
			}
			r.functionProviderCache[functionName] = provider
		}
	}
	return nil
}

func handleFunctionProviderConflict(
	ctx context.Context,
	functionName string,
	providedBy Provider,
	provider Provider,
) error {
	if providedBy != provider {
		providerNamespace, err := provider.Namespace(ctx)
		if err != nil {
			return err
		}

		providedByNamespace, err := providedBy.Namespace(ctx)
		if err != nil {
			return err
		}

		return errFunctionAlreadyProvided(
			functionName,
			providerNamespace,
			providedByNamespace,
		)
	}

	return nil
}
