package pluginutils

import (
	"strings"
	"unicode"
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
