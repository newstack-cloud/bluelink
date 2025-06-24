package validation

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// IsWebURL validates that a string is a valid HTTP or HTTPS URL.
func IsWebURL() func(string, *core.ScalarValue) []*core.Diagnostic {
	return IsURL([]string{"http", "https"})
}

// IsHTTPSURL validates that a string is a valid HTTPS URL.
func IsHTTPSURL() func(string, *core.ScalarValue) []*core.Diagnostic {
	return IsURL([]string{"https"})
}

// IsHTTPURL validates that a string is a valid HTTP URL.
func IsHTTPURL() func(string, *core.ScalarValue) []*core.Diagnostic {
	return IsURL([]string{"http"})
}

// IsURL validates that a string is a valid URL with one of the allowed schemes.
func IsURL(
	allowedSchemes []string,
) func(string, *core.ScalarValue) []*core.Diagnostic {
	return func(fieldName string, value *core.ScalarValue) []*core.Diagnostic {
		if !core.IsScalarString(value) {
			return invalidTypeDiagnostics(
				fieldName,
				value,
				"a string",
			)
		}

		urlValue := core.StringValueFromScalar(value)

		if urlValue == "" {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"expected %q to be a url, not an empty string.",
						fieldName,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		url, err := url.Parse(urlValue)
		if err != nil {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%q must be a valid url, but got %q: %v.",
						fieldName,
						urlValue,
						err,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		if url.Host == "" {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be a valid url with a host, but got %q.",
						fieldName,
						urlValue,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		isSchemeAllowed := slices.Contains(
			allowedSchemes,
			url.Scheme,
		)
		if !isSchemeAllowed {
			return []*core.Diagnostic{
				{
					Level: core.DiagnosticLevelError,
					Message: fmt.Sprintf(
						"%s must be a valid url with a scheme of: %q, but got %q with scheme %q.",
						fieldName,
						strings.Join(allowedSchemes, ", "),
						urlValue,
						url.Scheme,
					),
					Range: toDiagnosticRange(value.SourceMeta, nil),
				},
			}
		}

		return nil
	}
}
