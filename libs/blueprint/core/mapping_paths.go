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
//	core.GetPathValue("$[\"cluster.v1\"].config.endpoints[0]", node, 3)
func GetPathValue(path string, node *MappingNode, maxTraverseDepth int) (*MappingNode, error) {
	parsedPath, err := parsePath(
		path,
		/* allowPatterns */ false,
	)
	if err != nil {
		return nil, err
	}

	current := node
	pathExists := true
	i := 0
	maxDepth := int(math.Min(float64(maxTraverseDepth), float64(len(parsedPath))))
	for pathExists && current != nil && i < maxDepth {
		pathItem := parsedPath[i]
		if pathItem.fieldName != "" && current.Fields != nil {
			current = current.Fields[pathItem.fieldName]
		} else if pathItem.arrayIndex != nil && current.Items != nil {
			current = current.Items[*pathItem.arrayIndex]
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

// InjectPathValue injects a value into a MappingNode using a path.
// This will return an error if the provided path is invalid
// or if the path is not reachable in the given node.
// Structures such as an arrays and field mappings will be created
// if they do not exist in the injectInto node and the path is valid.
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
//	core.InjectPathValue("$[\"cluster.v1\"].config.endpoints[0]", value, injectInto, 3)
func InjectPathValue(
	path string,
	value *MappingNode,
	injectInto *MappingNode,
	maxTraverseDepth int,
) error {
	return injectPathValue(
		path,
		value,
		injectInto,
		false, // replace
		maxTraverseDepth,
	)
}

// InjectPathValueReplace injects a value into a MappingNode using a path.
// This will return an error if the provided path is invalid
// or if the path is not reachable in the given node.
// Structures such as an arrays and field mappings will be created
// if they do not exist in the injectInto node and the path is valid.
//
// InjectPathValueReplace is similar to InjectPathValue,
// where the difference is that it replaces the existing value
// at the path if it exists, instead of skipping the injection.
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
//	core.InjectPathValueReplace("$[\"cluster.v1\"].config.endpoints[0]", value, injectInto, 3)
func InjectPathValueReplace(
	path string,
	value *MappingNode,
	injectInto *MappingNode,
	maxTraverseDepth int,
) error {
	return injectPathValue(
		path,
		value,
		injectInto,
		true, // replace
		maxTraverseDepth,
	)
}

func injectPathValue(
	path string,
	value *MappingNode,
	injectInto *MappingNode,
	replace bool,
	maxTraverseDepth int,
) error {
	parsedPath, err := parsePath(
		path,
		/* allowPatterns */ false,
	)
	if err != nil {
		return err
	}

	current := injectInto
	pathExists := true
	i := 0
	maxDepth := int(math.Min(float64(maxTraverseDepth), float64(len(parsedPath))))
	for pathExists && current != nil && i < maxDepth {
		pathItem := parsedPath[i]
		if pathItem.fieldName != "" && current.Fields != nil {
			injectIntoFields(current, pathItem, parsedPath, i, value, replace)
			current = current.Fields[pathItem.fieldName]
		} else if pathItem.arrayIndex != nil && current.Items != nil {
			injectIntoItems(current, pathItem, parsedPath, i, value)
			arrayIndex := math.Min(
				float64(*pathItem.arrayIndex),
				float64(len(current.Items)-1),
			)
			current = current.Items[int(arrayIndex)]
		} else {
			pathExists = false
		}

		i += 1
	}

	if !pathExists {
		return fmt.Errorf(
			"path %q could not be injected into the mapping node, "+
				"the structure of the mapping node does not match the path",
			path,
		)
	}

	if maxDepth < len(parsedPath) {
		return fmt.Errorf(
			"path %q could not be injected into the mapping node, "+
				"the path goes beyond the maximum depth of the node",
			path,
		)
	}

	return nil
}

func injectIntoFields(
	target *MappingNode,
	pathItem *pathItem,
	parsedPath []*pathItem,
	i int,
	valueToInject *MappingNode,
	replace bool,
) {
	_, hasValue := target.Fields[pathItem.fieldName]
	if replace || !hasValue {
		if i == len(parsedPath)-1 {
			target.Fields[pathItem.fieldName] = valueToInject
		} else {
			target.Fields[pathItem.fieldName] = createFieldsOrItems(parsedPath, i+1)
		}
	}
}

func injectIntoItems(
	target *MappingNode,
	pathItem *pathItem,
	parsedPath []*pathItem,
	i int,
	valueToInject *MappingNode,
) {
	if *pathItem.arrayIndex >= len(target.Items) {
		// When the array index exceeds the last index of the array,
		// the value will be injected at the end of the array.
		// This is to ensure that the array is contiguous instead of having
		// to create empty items in between.
		if i == len(parsedPath)-1 {
			target.Items = append(target.Items, valueToInject)
		} else {
			target.Items = append(target.Items, createFieldsOrItems(parsedPath, i+1))
		}
	}
}

func createFieldsOrItems(parsedPath []*pathItem, nextIndex int) *MappingNode {
	if nextIndex >= len(parsedPath) {
		return &MappingNode{}
	}

	nextPathItem := parsedPath[nextIndex]
	if nextPathItem.fieldName != "" {
		return &MappingNode{
			Fields: map[string]*MappingNode{},
		}
	}

	if nextPathItem.arrayIndex != nil {
		return &MappingNode{
			Items: []*MappingNode{},
		}
	}

	return &MappingNode{}
}

// PathMatchesPattern determines if a given path matches the provided pattern.
// This can be an exact path match or a partial match where a pattern is used
// to indicate wildcard matches for array indices or map keys.
//
// Equality of a path is not the same as equality of a string,
// for example, the path "$[\"cluster\"].config.endpoints[0]"
// is equal to the path "$.cluster.config.endpoints[0]".
//
// A pattern does NOT refer to a regular expression, instead, it refers to a
// specific pattern where placeholders can be used to indicate any array index
// or map key.
// Placeholders in patterns are represented by "[*]" for any array index
// and ".*" for any map key.
// For example, the pattern "$.cluster.config.endpoints[*]" matches
// "$.cluster.config.endpoints[0]", "$.cluster.config.endpoints[1]",
// "$.cluster.config.endpoints[2]", etc.
// The pattern "$.cluster.config.endpoints.*" matches
// "$.cluster.config.endpoints[\"key1\"]", "$.cluster.config.endpoints[\"key2\"]",
// "$.cluster.config.endpoints[\"key3\"]", etc.
func PathMatchesPattern(path, pattern string) (bool, error) {
	if path == pattern {
		// There is no need to parse the path and pattern
		// if there is an exact string match.
		return true, nil
	}

	parsedPatternPath, err := parsePath(
		pattern,
		/* allowPatterns */ true,
	)
	if err != nil {
		return false, err
	}

	parsedPath, err := parsePath(
		path,
		/* allowPatterns */ false,
	)
	if err != nil {
		return false, err
	}

	if len(parsedPatternPath) != len(parsedPath) {
		return false, nil
	}

	for i := range parsedPath {
		patternItem := parsedPatternPath[i]
		pathItem := parsedPath[i]

		matchesFieldName := patternItem.fieldName == pathItem.fieldName ||
			(patternItem.anyFieldName && pathItem.fieldName != "")

		matchesArrayIndex := checkArrayIndexMatch(
			patternItem,
			pathItem,
		)

		if !matchesFieldName || !matchesArrayIndex {
			return false, nil
		}
	}

	return true, nil
}

func checkArrayIndexMatch(patternItem, pathItem *pathItem) bool {
	if patternItem.arrayIndex == nil && pathItem.arrayIndex == nil {
		return true
	}

	if patternItem.arrayIndex != nil && pathItem.arrayIndex != nil {
		return *patternItem.arrayIndex == *pathItem.arrayIndex
	}

	if patternItem.anyIndex {
		return true
	}

	return false
}

// Represents a single item in a path used to access
// values in a MappingNode.
type pathItem struct {
	fieldName  string
	arrayIndex *int
	// Indicates that the path item can match any index in an array,
	// this should only be used for patterns, regular parsing of paths
	// should not set this field.
	anyIndex bool
	// Indicates that the path item can match any key in a map,
	// this should only be used for patterns, regular parsing of paths
	// should not set this field.
	anyFieldName bool
}

func parsePath(path string, allowPatterns bool) ([]*pathItem, error) {

	if len(path) == 0 || path[0] != '$' {
		return nil, errInvalidMappingPath(path, nil)
	}

	pathWithoutRoot := path[1:]
	if len(pathWithoutRoot) == 0 {
		// "$" is a valid path to the root of the node.
		return []*pathItem{}, nil
	}

	return parsePathItems(pathWithoutRoot, allowPatterns)
}

func parsePathItems(pathWithoutRoot string, allowPatterns bool) ([]*pathItem, error) {
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
			currentItemStr, err = takeCurrentItem(
				&pathItems,
				currentItemStr,
				inFieldNameAccessor,
				inArrayIndexAccessor,
				allowPatterns,
			)
			// After we've taken the current item before the ".",
			// we should move to a state of being in a field name accessor.
			inFieldNameAccessor = true
		} else if isAccessorOpenBracket(char, inStringLiteral) {
			inOpenBracket = true
			currentItemStr, err = takeCurrentItem(
				&pathItems,
				currentItemStr,
				inFieldNameAccessor,
				inArrayIndexAccessor,
				allowPatterns,
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
				allowPatterns,
			)
		} else if isStringLiteralDelimiter(char, prevChar, inOpenBracket) {
			inStringLiteral = !inStringLiteral
			inFieldNameAccessor, currentItemStr, err = tryTakeCurrentItemEndOfStringLiteral(
				&pathItems,
				currentItemStr,
				inFieldNameAccessor,
				inArrayIndexAccessor,
				inStringLiteral,
				allowPatterns,
			)
		} else if isFirstDigitOfArrayIndex(char, prevChar, inOpenBracket, inStringLiteral) ||
			isWildcardArrayIndex(char, prevChar, inOpenBracket, inStringLiteral, allowPatterns) {
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
			allowPatterns,
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

func isWildcardArrayIndex(
	char rune,
	prevChar rune,
	inOpenBracket bool,
	inStringLiteral bool,
	allowPatterns bool,
) bool {
	if !allowPatterns {
		return false
	}

	return char == '*' && prevChar == '[' &&
		inOpenBracket && !inStringLiteral
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
	allowPatterns bool,
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
		allowPatterns,
	)

	return false, currentItemStr, err
}

func takeCurrentItem(
	pathItems *[]*pathItem,
	currentItemStr string,
	inFieldNameAccessor bool,
	inArrayIndexAccessor bool,
	allowPatterns bool,
) (string, error) {
	if len(currentItemStr) == 0 {
		return currentItemStr, nil
	}

	if inFieldNameAccessor && allowPatterns && currentItemStr == "*" {
		// If the current item is a wildcard for field names,
		// we treat it as a special case where it matches any field name.
		*pathItems = append(*pathItems, &pathItem{
			anyFieldName: true,
		})
		// Reset the current item string.
		return "", nil
	}

	if inFieldNameAccessor {
		*pathItems = append(*pathItems, &pathItem{
			// Unescape quotes in the field name.
			fieldName: strings.ReplaceAll(currentItemStr, "\\\"", "\""),
		})
		// Reset the current item string.
		return "", nil
	}

	if inArrayIndexAccessor && allowPatterns && currentItemStr == "*" {
		// If the current item is a wildcard for array indices,
		// we treat it as a special case where it matches any index.
		*pathItems = append(*pathItems, &pathItem{
			anyIndex: true,
		})
		// Reset the current item string.
		return "", nil
	}

	if inArrayIndexAccessor {
		index, err := strconv.Atoi(currentItemStr)
		if err != nil {
			return currentItemStr, err
		}
		*pathItems = append(*pathItems, &pathItem{
			arrayIndex: &index,
		})
		// Reset the current item string.
		return "", nil
	}

	return currentItemStr, nil
}
