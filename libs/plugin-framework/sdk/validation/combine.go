package validation

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// AllOf validates that a value passes all provided validation functions,
// if there is at least one error returned in the given diagnostics,
// validation will fail.
func AllOf(
	validators ...func(string, *core.ScalarValue) []*core.Diagnostic,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		var diagnostics []*core.Diagnostic
		for _, validator := range validators {
			diagnostics = append(diagnostics, validator(fieldName, value)...)
		}
		return diagnostics
	}
}

// OneOf validates that a value passes for at least one of the provided validators.
func OneOf(
	validators ...func(string, *core.ScalarValue) []*core.Diagnostic,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		collectedDiags := []*core.Diagnostic{}
		for _, validator := range validators {
			diagnostics := validator(fieldName, value)
			collectedDiags = append(collectedDiags, diagnostics...)
			// Ensure warnings are returned so they can be displayed,
			// even if there are no errors.
			diagnosticsWithoutErrors := commoncore.Filter(
				diagnostics,
				func(d *core.Diagnostic, _ int) bool {
					return d.Level != core.DiagnosticLevelError
				},
			)
			if len(diagnosticsWithoutErrors) == len(diagnostics) {
				return diagnosticsWithoutErrors
			}
		}

		finalDiags := []*core.Diagnostic{
			{
				Level: core.DiagnosticLevelError,
				Message: fmt.Sprintf(
					"%q did not pass any of the validation checks. It must pass at least one check.",
					fieldName,
				),
				Range: toDiagnosticRange(value.SourceMeta, nil),
			},
		}
		finalDiags = append(finalDiags, collectedDiags...)
		return finalDiags
	}
}
