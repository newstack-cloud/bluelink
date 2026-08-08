package deployconfig

import (
	"regexp"
	"strings"
)

var trailingCommaPattern = regexp.MustCompile(`,(\s*[}\]])`)

// Removes line and block comments and trailing commas so
// that JSONC content can be handed to encoding/json.
func stripJSONCComments(content string) string {
	result := strings.Builder{}

	index := 0
	for index < len(content) {
		if consumed, ok := skipComment(content, index); ok {
			if consumed == index {
				break
			}
			index = consumed
			continue
		}

		if content[index] == '"' {
			index = copyStringLiteral(content, index, &result)
			continue
		}

		result.WriteByte(content[index])
		index++
	}

	return trailingCommaPattern.ReplaceAllString(result.String(), "$1")
}

// Reports the index just past a comment starting at index.
// The returned index equals the input index when the comment is unterminated,
// signalling the caller to stop.
func skipComment(content string, index int) (int, bool) {
	if index+1 >= len(content) || content[index] != '/' {
		return index, false
	}

	if content[index+1] == '*' {
		end := strings.Index(content[index+2:], "*/")
		if end == -1 {
			return index, true
		}
		return index + end + 4, true
	}

	if content[index+1] == '/' {
		end := strings.IndexByte(content[index:], '\n')
		if end == -1 {
			return index, true
		}
		return index + end, true
	}

	return index, false
}

func copyStringLiteral(content string, index int, result *strings.Builder) int {
	result.WriteByte(content[index])
	index += 1

	for index < len(content) && content[index] != '"' {
		if content[index] == '\\' && index+1 < len(content) {
			result.WriteByte(content[index])
			index += 1
		}
		result.WriteByte(content[index])
		index += 1
	}

	if index < len(content) {
		result.WriteByte(content[index])
		index += 1
	}

	return index
}
