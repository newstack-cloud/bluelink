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
