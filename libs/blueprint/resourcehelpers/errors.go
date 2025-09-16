package resourcehelpers

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

const (
	// ErrorReasonCodeProviderResourceTypeNotFound is provided when the
	// reason for a blueprint spec load error is due to
	// the resource provider missing an implementation for a
	// specific resource type.
	ErrorReasonCodeProviderResourceTypeNotFound errors.ErrorReasonCode = "resource_type_not_found"
	// ErrorReasonCodeItemTypeProviderNotFound is provided when the
	// reason for a blueprint run error is due to the provider
	// for a specific resource type not being found.
	ErrorReasonCodeEmptyResourceSpecDefinition errors.ErrorReasonCode = "empty_resource_spec_definition"
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
		Err:        fmt.Errorf("provider %q not found for resource type %q", providerNamespace, resourceType),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryProviderMissing,
			ReasonCode: provider.ErrorReasonCodeItemTypeProviderNotFound,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeInstallProvider),
					Title:       "Install Provider",
					Description: fmt.Sprintf("Install the %s provider to support %s resources", providerNamespace, resourceType),
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeCheckResourceType),
					Title:       "Verify Resource Type",
					Description: "Check if the resource type name is correct",
					Priority:    2,
				},
			},
			Metadata: map[string]any{
				"providerNamespace": providerNamespace,
				"resourceType":      resourceType,
			},
		},
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

func errEmptyResourceSpecDefinition(
	resourceType string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeEmptyResourceSpecDefinition,
		Err: fmt.Errorf(
			"run failed as the resource spec definition for resource type %q is empty",
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
