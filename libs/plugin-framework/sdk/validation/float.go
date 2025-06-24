package validation

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// FloatRange validates that a float value is within a specified range.
func FloatRange(
	minValue float64,
	maxValue float64,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarFloat(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a float",
			)
		}

		floatValue := core.FloatValueFromScalar(value)
		if floatValue < minValue || floatValue > maxValue {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be between %s and %s, but got %s.",
						fieldName,
						// Wrap in scalar and convert to string to get only the precision
						// required for each float value.
						core.ScalarFromFloat(minValue).ToString(),
						core.ScalarFromFloat(maxValue).ToString(),
						value.ToString(),
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// MaxFloat validates that a float value does not exceed a specified maximum value.
func MaxFloat(
	maxValue float64,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarFloat(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a float",
			)
		}

		floatValue := core.FloatValueFromScalar(value)
		if floatValue > maxValue {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be less than or equal to %s, but got %s.",
						fieldName,
						core.ScalarFromFloat(maxValue).ToString(),
						value.ToString(),
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// MinFloat validates that a float value is at least a specified minimum value.
func MinFloat(
	minValue float64,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarFloat(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a float",
			)
		}

		floatValue := core.FloatValueFromScalar(value)
		if floatValue < minValue {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be at least %s, but got %s.",
						fieldName,
						core.ScalarFromFloat(minValue).ToString(),
						value.ToString(),
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}
