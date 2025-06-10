package resourcehelpers

import (
	"fmt"

	"github.com/newstack-cloud/celerity/libs/blueprint/errors"
	"github.com/newstack-cloud/celerity/libs/blueprint/provider"
)

const (
	// ErrorReasonCodeProviderResourceTypeNotFound is provided when the
	// reason for a blueprint spec load error is due to
	// the resource provider missing an implementation for a
	// specific resource type.
	ErrorReasonCodeProviderResourceTypeNotFound errors.ErrorReasonCode = "resource_type_not_found"
	// ErrorReasonCodeMultipleRunErrors is provided when the reason
	// for a blueprint run error is due to multiple errors
	// occurring during the run.
	ErrorReasonCodeMultipleRunErrors errors.ErrorReasonCode = "multiple_run_errors"
	// ErrorReasonCodeAbstractResourceTypeNotFound is provided when the
	// reason for a blueprint run error is due to an abstract resource
	// type not being found in any of the loaded transformers.
	ErrorReasonCodeAbstractResourceTypeNotFound errors.ErrorReasonCode = "abstract_resource_type_not_found"
)

func errResourceTypeProviderNotFound(
	providerNamespace string,
	resourceType string,
) error {
	return &errors.RunError{
		ReasonCode: provider.ErrorReasonCodeItemTypeProviderNotFound,
		Err: fmt.Errorf(
			"run failed as the provider with namespace %q was not found for resource type %q",
			providerNamespace,
			resourceType,
		),
	}
}

func errProviderResourceTypeNotFound(
	resourceType string,
	providerNamespace string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeProviderResourceTypeNotFound,
		Err: fmt.Errorf(
			"run failed as the provider with namespace %q does not have an implementation for resource type %q",
			providerNamespace,
			resourceType,
		),
	}
}

func errAbstactResourceTypeNotFound(
	resourceType string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeAbstractResourceTypeNotFound,
		Err: fmt.Errorf(
			"run failed as the abstract resource with type %q was not found in any of the loaded transformers",
			resourceType,
		),
	}
}

func errMultipleRunErrors(
	errs []error,
) error {
	return &errors.RunError{
		ReasonCode:  ErrorReasonCodeMultipleRunErrors,
		Err:         fmt.Errorf("run failed due to multiple errors"),
		ChildErrors: errs,
	}
}
