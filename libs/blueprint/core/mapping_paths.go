package core

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// GetPathValue retrieves a value from a MappingNode using a path.
// This will return an error if the provided path is invalid and will
// return a nil MappingNode if the path does not exist in the given node.
//
// A path supports the following acessors:
//
// - "." for fields
// - "[\"<field>\"]" for fields with special characters
// - "[<index>]" for array items
//
// "$" represents the root of the path and must always be the first character
// in the path.
//
// Example:
//
//	GetPathValue("$[\"cluster.v1\"].config.endpoints[0]", node, 3)
func GetPathValue(path string, node *MappingNode, maxTraverseDepth int) (*MappingNode, error) {
	parsedPath, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	current := node
	pathExists := true
	i := 0
	maxDepth := int(math.Min(float64(maxTraverseDepth), float64(len(parsedPath))))
	for pathExists && current != nil && i < maxDepth {
		pathItem := parsedPath[i]
		if pathItem.FieldName != "" && current.Fields != nil {
			current = current.Fields[pathItem.FieldName]
		} else if pathItem.ArrayIndex != nil && current.Items != nil {
			current = current.Items[*pathItem.ArrayIndex]
		} else if IsNilMappingNode(current) {
			pathExists = false
		}

		i += 1
	}

	if maxDepth < len(parsedPath) {
		return nil, nil
	}

	return current, nil
}

// Represents a single item in a path used to access
// values in a MappingNode.
type pathItem struct {
	FieldName  string
	ArrayIndex *int
}

func parsePath(path string) ([]*pathItem, error) {

	if len(path) == 0 || path[0] != '$' {
		return nil, errInvalidMappingPath(path, nil)
	}

	pathWithoutRoot := path[1:]
	if len(pathWithoutRoot) == 0 {
		// "$" is a valid path to the root of the node.
		return []*pathItem{}, nil
	}

	return parsePathItems(pathWithoutRoot)
}

func parsePathItems(pathWithoutRoot string) ([]*pathItem, error) {
	pathItems := []*pathItem{}

	i := 0
	prevChar := ' '
	inFieldNameAccessor := false
	inStringLiteral := false
	inOpenBracket := false
	inArrayIndexAccessor := false
	currentItemStr := ""
	var err error
	for i < len(pathWithoutRoot) && err == nil {
		char, width := utf8.DecodeRuneInString(pathWithoutRoot[i:])
		if isDotAccessor(char, inOpenBracket) {
			inFieldNameAccessor = true
			currentItemStr, err = takeCurrentItem(
				&pathItems,
				currentItemStr,
				inFieldNameAccessor,
				inArrayIndexAccessor,
			)
		} else if isAccessorOpenBracket(char, inStringLiteral) {
			inOpenBracket = true
			currentItemStr, err = takeCurrentItem(
				&pathItems,
				currentItemStr,
				inFieldNameAccessor,
				inArrayIndexAccessor,
			)
			// "[" marks the end of the previous path item where the
			// previous path item was accessed via dot notation.
			// (e.g. the end of endpoints in config.endpoints[0])
			inFieldNameAccessor = false
		} else if isAccessorCloseBracket(char, inOpenBracket, inStringLiteral) {
			inOpenBracket = false
			currentItemStr, err = takeCurrentItem(
				&pathItems,
				currentItemStr,
				inFieldNameAccessor,
				inArrayIndexAccessor,
			)
		} else if isStringLiteralDelimiter(char, prevChar, inOpenBracket) {
			inStringLiteral = !inStringLiteral
			inFieldNameAccessor, currentItemStr, err = tryTakeCurrentItemEndOfStringLiteral(
				&pathItems,
				currentItemStr,
				inFieldNameAccessor,
				inArrayIndexAccessor,
				inStringLiteral,
			)
		} else if isFirstDigitOfArrayIndex(char, prevChar, inOpenBracket, inStringLiteral) {
			inArrayIndexAccessor = true
			currentItemStr += string(char)
		} else if inFieldNameAccessor || inArrayIndexAccessor {
			currentItemStr += string(char)
		}
		i += width
		prevChar = char
	}

	if len(currentItemStr) > 0 {
		_, err = takeCurrentItem(
			&pathItems,
			currentItemStr,
			inFieldNameAccessor,
			inArrayIndexAccessor,
		)
	}

	if err != nil || inOpenBracket {
		return nil, errInvalidMappingPath(
			fmt.Sprintf("$%s", pathWithoutRoot),
			err,
		)
	}

	return pathItems, nil
}

func isDotAccessor(char rune, inOpenBracket bool) bool {
	return char == '.' && !inOpenBracket
}

func isAccessorOpenBracket(char rune, inStringLiteral bool) bool {
	return char == '[' && !inStringLiteral
}

func isStringLiteralDelimiter(char rune, prevChar rune, inOpenBracket bool) bool {
	return char == '"' && prevChar != '\\' && inOpenBracket
}

func isFirstDigitOfArrayIndex(
	char rune,
	prevChar rune,
	inOpenBracket bool,
	inStringLiteral bool,
) bool {
	return unicode.IsDigit(char) &&
		prevChar == '[' &&
		inOpenBracket &&
		!inStringLiteral
}

func isAccessorCloseBracket(
	char rune,
	inOpenBracket bool,
	inStringLiteral bool,
) bool {
	return char == ']' && inOpenBracket && !inStringLiteral
}

func tryTakeCurrentItemEndOfStringLiteral(
	pathItems *[]*pathItem,
	currentItemStr string,
	inFieldNameAccessor bool,
	inArrayIndexAccessor bool,
	inStringLiteral bool,
) (bool, string, error) {
	if inStringLiteral {
		// A string literal is a field name accessor,
		// if we are in a string literal, we should
		// treat the current character as a part of
		// a field name.
		return true, currentItemStr, nil
	}

	currentItemStr, err := takeCurrentItem(
		pathItems,
		currentItemStr,
		inFieldNameAccessor,
		inArrayIndexAccessor,
	)

	return false, currentItemStr, err
}

func takeCurrentItem(
	pathItems *[]*pathItem,
	currentItemStr string,
	inFieldNameAccessor bool,
	inArrayIndexAccessor bool,
) (string, error) {
	if len(currentItemStr) == 0 {
		return currentItemStr, nil
	}

	if inFieldNameAccessor {
		*pathItems = append(*pathItems, &pathItem{
			// Unescape quotes in the field name.
			FieldName: strings.Replace(currentItemStr, "\\\"", "\"", -1),
		})
		// Reset the current item string.
		return "", nil
	} else if inArrayIndexAccessor {
		index, err := strconv.Atoi(currentItemStr)
		if err != nil {
			return currentItemStr, err
		}
		*pathItems = append(*pathItems, &pathItem{
			ArrayIndex: &index,
		})
		// Reset the current item string.
		return "", nil
	}

	return currentItemStr, nil
}
