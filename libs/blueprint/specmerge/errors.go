package specmerge

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
)

const (
	// ErrorReasonCodeUnexpectedComputedField
	// is provided when the reason for an error
	// during deployment due to an unexpected computed field
	// being returned by a resource plugin implementation's deploy method.
	ErrorReasonCodeUnexpectedComputedField errors.ErrorReasonCode = "unexpected_computed_field"
	// ErrorReasonCodeApplyLinkProjection
	// is provided when a contribution a link has made to a resource
	// cannot be applied to the resource's spec.
	ErrorReasonCodeApplyLinkProjection errors.ErrorReasonCode = "apply_link_projection"
	// ErrorReasonCodeRemoveLinkProjection
	// is provided when a contribution a link has made to a resource
	// cannot be taken back out of the resource's spec.
	ErrorReasonCodeRemoveLinkProjection errors.ErrorReasonCode = "remove_link_projection"
)

func errUnexpectedComputedField(
	computedField string,
	resourceName string,
	expectedComputedFields []string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeUnexpectedComputedField,
		Err: fmt.Errorf(
			"unexpected computed field %q found in resource %q, "+
				"computed fields returned by the resource deploy method "+
				"can include the following: %v",
			computedField,
			resourceName,
			strings.Join(expectedComputedFields, ", "),
		),
	}
}

func errApplyLinkProjection(
	linkName string,
	resourceFieldPath string,
	err error,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeApplyLinkProjection,
		Err: fmt.Errorf(
			"failed to apply the contribution link %q makes to %q: %w",
			linkName,
			resourceFieldPath,
			err,
		),
	}
}

func errRemoveLinkProjection(
	linkName string,
	resourceFieldPath string,
	err error,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeRemoveLinkProjection,
		Err: fmt.Errorf(
			"failed to remove the contribution link %q makes to %q: %w",
			linkName,
			resourceFieldPath,
			err,
		),
	}
}
