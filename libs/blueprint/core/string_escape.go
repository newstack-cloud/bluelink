package core

import "regexp"

func escapeRegexpSpecialChars(input string) string {
	return regexp.QuoteMeta(input)
}
