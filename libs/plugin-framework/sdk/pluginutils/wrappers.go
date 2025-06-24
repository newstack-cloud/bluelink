package pluginutils

import (
	"context"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ContextFuncReturnValue is a function that takes a context,
// an argument and returns a value and an error.
type ContextFuncReturnValue[Arg any, Value any] func(context.Context, Arg) (Value, error)

// ContextFunc is a function that takes a context
// and an argument and returns an error.
type ContextFunc[Arg any] func(context.Context, Arg) error

// Retryable wraps a function that only returns an error
// and makes it retryable in the plugin system if the error
// meets the provided retryable criteria.
// This is to be used inside the body of a plugin definition
// handler and wrapped around all the functionality that needs
// to be retried.
func Retryable[Arg any](
	function ContextFunc[Arg],
	isErrorRetryable func(error) bool,
) ContextFunc[Arg] {
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
	function ContextFuncReturnValue[Arg, Value],
	isErrorRetryable func(error) bool,
) ContextFuncReturnValue[Arg, Value] {
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

// Timeout wraps a function that only returns an error
// and applies a timeout to it.
// Timeouts can be due to transient or permanent issues,
// to combine timeout and retry behaviour, wrap the timeout function with Retryable.
func Timeout[Arg any](function ContextFunc[Arg], timeout time.Duration) ContextFunc[Arg] {
	return func(ctx context.Context, arg Arg) error {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return function(timeoutCtx, arg)
	}
}

// TimeoutReturnValue wraps a function that returns a value and an error
// and applies a timeout to it.
// Timeouts can be due to transient or permanent issues,
// to retry a timeout error, wrap the timeout function with RetryableReturnValue.
func TimeoutReturnValue[Arg any, Value any](
	function ContextFuncReturnValue[Arg, Value],
	timeout time.Duration,
) ContextFuncReturnValue[Arg, Value] {
	return func(ctx context.Context, arg Arg) (Value, error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return function(timeoutCtx, arg)
	}
}
