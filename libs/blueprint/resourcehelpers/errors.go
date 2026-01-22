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
		Err:        fmt.Errorf("provider or transformer %q not found for resource type %q", providerNamespace, resourceType),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryProvider,
			ReasonCode: provider.ErrorReasonCodeItemTypeProviderNotFound,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeInstallProvider),
					Title:       "Install Provider or Transformer",
					Description: fmt.Sprintf("Install the %s provider or transformer to support %s resources", providerNamespace, resourceType),
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
				"category":          "resource",
				"itemType":          resourceType,
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
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryResourceType,
			ReasonCode: ErrorReasonCodeProviderResourceTypeNotFound,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeCheckResourceType),
					Title:       "Check Resource Type",
					Description: "Verify the resource type name is correct",
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeUpdateProvider),
					Title:       "Update Provider",
					Description: "Update to a newer version that may support this resource type",
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

func errAbstactResourceTypeNotFound(
	resourceType string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeAbstractResourceTypeNotFound,
		Err: fmt.Errorf(
			"run failed as the abstract resource with type %q was not found in any of the loaded transformers",
			resourceType,
		),
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryResourceType,
			ReasonCode: ErrorReasonCodeAbstractResourceTypeNotFound,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeCheckAbstractResourceType),
					Title:       "Check Abstract Resource Type",
					Description: "Verify the abstract resource type name is correct",
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeCheckResourceType),
					Title:       "Check Resource Type",
					Description: "Verify the resource type name is correct",
					Priority:    2,
				},
			},
			Metadata: map[string]any{
				"abstractResourceType": resourceType,
			},
		},
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
