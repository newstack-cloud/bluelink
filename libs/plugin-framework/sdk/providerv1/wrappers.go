package providerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
)

// Retryable wraps a function that only returns an error
// and makes it retryable in the plugin system if the error
// meets the provided retryable criteria.
// This is to be used inside the body of a plugin definition
// handler and wrapped around all the functionality that needs
// to be retried.
func Retryable[Arg any](
	function pluginutils.ContextFunc[Arg],
	isErrorRetryable func(error) bool,
) pluginutils.ContextFunc[Arg] {
	return func(ctx context.Context, arg Arg) error {
		err := function(ctx, arg)
		if err != nil {
			if isErrorRetryable(err) {
				return &provider.RetryableError{
					ChildError: err,
				}
			}
			return err
		}
		return nil
	}
}

// RetryableReturnValue wraps a function that returns a value and an error
// and makes it retryable in the plugin system.
// This is to be used inside the body of a plugin definition
// handler and wrapped around all the functionality that needs
// to be retried.
func RetryableReturnValue[Arg any, Value any](
	function pluginutils.ContextFuncReturnValue[Arg, Value],
	isErrorRetryable func(error) bool,
) pluginutils.ContextFuncReturnValue[Arg, Value] {
	return func(ctx context.Context, arg Arg) (Value, error) {
		val, err := function(ctx, arg)
		if err != nil {
			if isErrorRetryable(err) {
				return val, &provider.RetryableError{
					ChildError: err,
				}
			}
			return val, err
		}
		return val, nil
	}
}
