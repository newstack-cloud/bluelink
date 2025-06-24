package validation

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// IntRange validates that an integer value is within a specified range.
func IntRange(
	minValue int,
	maxValue int,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarInt(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"an integer",
			)
		}

		intValue := core.IntValueFromScalar(value)
		if intValue < minValue || intValue > maxValue {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be between %d and %d, but got %d.",
						fieldName,
						minValue,
						maxValue,
						intValue,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// MaxInt validates that an integer value does not exceed a specified maximum value.
func MaxInt(
	maxValue int,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarInt(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"an integer",
			)
		}

		intValue := core.IntValueFromScalar(value)
		if intValue > maxValue {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be less than or equal to %d, but got %d.",
						fieldName,
						maxValue,
						intValue,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// MinInt validates that an integer value is at least a specified minimum value.
func MinInt(
	minValue int,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarInt(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"an integer",
			)
		}

		intValue := core.IntValueFromScalar(value)
		if intValue < minValue {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be at least %d, but got %d.",
						fieldName,
						minValue,
						intValue,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}
