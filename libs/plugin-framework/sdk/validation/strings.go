package validation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// StringLengthRange validates that a string value is within a specified length range.
// This counts characters, not bytes.
func StringLengthRange(
	minLength int,
	maxLength int,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		length := utf8.RuneCountInString(core.StringValueFromScalar(value))
		if length < minLength || length > maxLength {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be between %d and %d characters long, but got %d.",
						fieldName,
						minLength,
						maxLength,
						length,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// MaxStringLength validates that a string value does not exceed a specified maximum length.
// This counts characters, not bytes.
func MaxStringLength(
	maxLength int,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		length := utf8.RuneCountInString(core.StringValueFromScalar(value))
		if length > maxLength {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be no more than %d characters long, but got %d.",
						fieldName,
						maxLength,
						length,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// MinStringLength validates that a string value is at least a specified minimum length.
// This counts characters, not bytes.
func MinStringLength(
	minLength int,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		length := utf8.RuneCountInString(core.StringValueFromScalar(value))
		if length < minLength {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be at least %d characters long, but got %d.",
						fieldName,
						minLength,
						length,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// StringMatchesPattern validates that a string value matches a specified regular expression pattern.
func StringMatchesPattern(
	patternRegexp *regexp.Regexp,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		if !patternRegexp.MatchString(core.StringValueFromScalar(value)) {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must match the pattern %s.",
						fieldName,
						patternRegexp.String(),
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// StringDoesNotMatchPattern validates that a string value does not match a
// specified regular expression pattern.
func StringDoesNotMatchPattern(
	patternRegexp *regexp.Regexp,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		if patternRegexp.MatchString(core.StringValueFromScalar(value)) {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must not match the pattern %s.",
						fieldName,
						patternRegexp.String(),
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}
		return nil
	}
}

// StringInList validates that a string value is one of the specified allowed values.
func StringInList(
	allowedValues []string,
	ignoreCase bool,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringValue := core.StringValueFromScalar(value)
		inList := stringSliceContains(allowedValues, stringValue, ignoreCase)
		if !inList {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be one of %s, but got %q.",
						fieldName,
						strings.Join(allowedValues, ", "),
						stringValue,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}

// StringNotInList validates that a string value is not one of the specified disallowed values.
func StringNotInList(
	disallowedValues []string,
	ignoreCase bool,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringValue := core.StringValueFromScalar(value)
		inList := stringSliceContains(disallowedValues, stringValue, ignoreCase)
		if inList {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must not be one of %s, but got %q.",
						fieldName,
						strings.Join(disallowedValues, ", "),
						stringValue,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}

func stringSliceContains(
	allowedValues []string,
	value string,
	ignoreCase bool,
) bool {
	if !ignoreCase {
		return slices.Contains(allowedValues, value)
	}

	for _, allowedValue := range allowedValues {
		if strings.EqualFold(allowedValue, value) {
			return true
		}
	}

	return false
}

// StringDoesNotContainChars validates that a string value
// does not contain any of the specified unicode code points
// in the provided string.
func StringDoesNotContainChars(
	disallowedChars string,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringValue := core.StringValueFromScalar(value)
		if strings.ContainsAny(stringValue, disallowedChars) {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must not contain any of %q.",
						fieldName,
						disallowedChars,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}

// StringIsBase64 validates that a string value is a valid Base64-encoded string.
func StringIsBase64() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringValue := core.StringValueFromScalar(value)
		if _, err := base64.StdEncoding.DecodeString(stringValue); err != nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be a valid base64-encoded string, but got %q: %v.",
						fieldName,
						stringValue,
						err,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}

// StringIsJSON validates that a string value is valid JSON.
func StringIsJSON() func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		stringValue := core.StringValueFromScalar(value)
		if !json.Valid([]byte(stringValue)) {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be valid JSON, but got %q.",
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
