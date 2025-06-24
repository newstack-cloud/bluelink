package includes

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
)

const (
	// ErrorReasonCodeInvalidPath is an error that is returned when the path
	// provided to the file system child resolver is invalid.
	ErrorReasonCodeInvalidPath errors.ErrorReasonCode = "invalid_path"
	// ErrorReasonCodeBlueprintNotFound is an error that is returned when the
	// blueprint file is not found in a child resolver.
	ErrorReasonCodeBlueprintNotFound errors.ErrorReasonCode = "blueprint_not_found"
	// ErrorReasonCodePermissions is an error that is returned when the
	// file system child resolver encounters a permission error.
	ErrorReasonCodePermissions errors.ErrorReasonCode = "permission_error"
	// ErrorReasonCodeInvalidMetadata is an error that is returned when the
	// metadata provided to a child resolver is not valid based on the resolver's
	// requirements. (e.g. missing bucket field for an S3 include)
	ErrorReasonCodeInvalidMetadata errors.ErrorReasonCode = "invalid_metadata"
	// ErrorReasonCodeResolveFailure is an error that is returned when a child
	// resolver fails to resolve a child blueprint for a reason specific to the
	// resolver implementation.
	ErrorReasonCodeResolveFailure errors.ErrorReasonCode = "resolve_failure"
)

func ErrInvalidPath(includeName string, resolverName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInvalidPath,
		Err: fmt.Errorf(
			"[include.%s]: invalid path found, path value must be a string for the %s "+
				"child resolver, the provided value is either empty or not a string",
			includeName,
			resolverName,
		),
	}
}

func ErrBlueprintNotFound(includeName string, path string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeBlueprintNotFound,
		Err: fmt.Errorf(
			"[include.%s]: blueprint not found at path: %s",
			includeName,
			path,
		),
	}
}

func ErrPermissions(includeName string, path string, err error) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodePermissions,
		Err: fmt.Errorf(
			"[include.%s]: permission error encountered while reading blueprint at path: %s: %w",
			includeName,
			path,
			err,
		),
	}
}

func ErrInvalidMetadata(includeName, message string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInvalidMetadata,
		Err: fmt.Errorf(
			"[include.%s]: %s",
			includeName,
			message,
		),
	}
}

func ErrResolveFailure(includeName string, err error) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeResolveFailure,
		Err: fmt.Errorf(
			"[include.%s]: failed to resolve child blueprint: %w",
			includeName,
			err,
		),
	}
}
