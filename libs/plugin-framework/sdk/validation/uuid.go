package validation

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// IsUUID validates that a string is a valid UUID.
func IsUUID() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringValue := core.StringValueFromScalar(value)
		if _, err := uuid.Parse(stringValue); err != nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid UUID, but got %q.",
						fieldName,
						stringValue,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}
