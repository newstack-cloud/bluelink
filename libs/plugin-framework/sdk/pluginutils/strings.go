package pluginutils

import (
	"strings"
	"unicode"

	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// StripNonAlphaNumericChars strips all non-alphanumeric characters from a string.
func StripNonAlphaNumericChars(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}

// StringToSubstitutions converts a string to a StringOrSubstitutions with a single StringValue.
func StringToSubstitutions(s string) *substitutions.StringOrSubstitutions {
	return &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				StringValue: &s,
			},
		},
	}
}
