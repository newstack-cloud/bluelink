package utils

import (
	"fmt"
	"os"
	"regexp"
)

// ExpandEnv expands the environment variables in the given string.
// This supports windows and unix style environment variable expansion.
func ExpandEnv(input string) string {
	finalString := regexp.MustCompile(`%[^%]+%`).ReplaceAllStringFunc(
		input,
		func(match string) string {
			return fmt.Sprintf("${%s}", match[1:len(match)-1])
		},
	)
	return os.ExpandEnv(finalString)
}
