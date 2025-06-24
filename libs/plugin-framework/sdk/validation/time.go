package validation

import (
	"fmt"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// IsDayOfTheWeek validates that a string is a valid day of the week.
// The `ignoreCase` parameter determines whether the validation is case-insensitive.
func IsDayOfTheWeek(ignoreCase bool) func(string, *core.ScalarValue) []*core.Diagnostic {
	return StringInList(
		[]string{
			"Monday",
			"Tuesday",
			"Wednesday",
			"Thursday",
			"Friday",
			"Saturday",
			"Sunday",
		},
		ignoreCase,
	)
}

// IsMonth validates that a string is a valid month of the year.
// The `ignoreCase` parameter determines whether the validation is case-insensitive.
func IsMonth(ignoreCase bool) func(string, *core.ScalarValue) []*core.Diagnostic {
	return StringInList(
		[]string{
			"January",
			"February",
			"March",
			"April",
			"May",
			"June",
			"July",
			"August",
			"September",
			"October",
			"November",
			"December",
		},
		ignoreCase,
	)
}

// IsShortMonth validates that a string is a valid short month of the year.
func IsShortMonth(ignoreCase bool) func(string, *core.ScalarValue) []*core.Diagnostic {
	return StringInList(
		[]string{
			"Jan",
			"Feb",
			"Mar",
			"Apr",
			"May",
			"Jun",
			"Jul",
			"Aug",
			"Sep",
			"Oct",
			"Nov",
			"Dec",
		},
		ignoreCase,
	)
}

// IsRFC3339 validates that a string is a valid RFC3339 date-time format.
func IsRFC3339() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringValue := core.StringValueFromScalar(value)

		if _, err := time.Parse(time.RFC3339, stringValue); err != nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be a valid RFC3339 date-time, but got %q: %v.",
						fieldName,
						core.StringValueFromScalar(value),
						err,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}
