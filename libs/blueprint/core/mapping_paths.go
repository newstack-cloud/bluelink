package core

import (
	"errors"
	"fmt"
	"math"
	"slices"
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
// - "[@.<key> = \"<value>\"]" To target a specific item in an array of objects by a unique attribute
//
// "$" represents the root of the path and must always be the first character
// in the path.
// This path syntax is similar to JSONPath, but is not an implementation of the JSONPath
// specification. A very limited set of selection features are provided intended to meet the needs
// of the Blueprint framework and provider implementations.
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
		} else if pathItem.arrayItemSelector != nil && current.Items != nil {
			targetItemIndex := slices.IndexFunc(
				current.Items,
				objectHasPropertyWithValue(
					pathItem.arrayItemSelector,
				),
			)
			if targetItemIndex < 0 {
				pathExists = false
			} else {
				current = current.Items[targetItemIndex]
			}
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
// Structures such as arrays and field mappings will be created
// if they do not exist in the injectInto node and the path is valid.
// Existing fields in objects will not be replaced, use InjectPathValueReplaceFields
// to ensure existing object fields are replaced.
// For arrays, values will be injected to replace existing items
// at the specified index or appended to the end of the array
// if the index exceeds the last index of the array.
//
// A path supports the following acessors:
//
// - "." for fields
// - "[\"<field>\"]" for fields with special characters
// - "[<index>]" for array items
// - "[@.<key> = \"<value>\"]" To target a specific item in an array of objects by a unique attribute
//
// "$" represents the root of the path and must always be the first character
// in the path.
// This path syntax is similar to JSONPath, but is not an implementation of the JSONPath
// specification. A very limited set of selection features are provided intended to meet the needs
// of the Blueprint framework and provider implementations.
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

// InjectPathValueReplaceFields injects a value into a MappingNode using a path.
// This will return an error if the provided path is invalid
// or if the path is not reachable in the given node.
// Structures such as an arrays and field mappings will be created
// if they do not exist in the injectInto node and the path is valid.
//
// InjectPathValueReplaceFields is similar to InjectPathValue,
// where the difference is that it replaces the existing value
// at the path if it exists for an object field, instead of skipping the injection.
// Values are always injected into arrays, even if the index already exists in both
// InjectPathValue and InjectPathValueReplaceFields.
//
// A path supports the following acessors:
//
// - "." for fields
// - "[\"<field>\"]" for fields with special characters
// - "[<index>]" for array items
// - "[@.<key> = \"<value>\"]" To target a specific item in an array of objects by a unique attribute
//
// "$" represents the root of the path and must always be the first character
// in the path.
// This path syntax is similar to JSONPath, but is not an implementation of the JSONPath
// specification. A very limited set of selection features are provided intended to meet the needs
// of the Blueprint framework and provider implementations.
//
// Example:
//
//	core.InjectPathValueReplaceFields("$[\"cluster.v1\"].config.endpoints[0]", value, injectInto, 3)
func InjectPathValueReplaceFields(
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
		} else if pathItem.arrayItemSelector != nil && current.Items != nil {
			injectedIndex := injectIntoItemsWithSelector(
				current,
				pathItem,
				parsedPath,
				value,
				i,
			)
			if injectedIndex < 0 {
				pathExists = false
			} else {
				current = current.Items[injectedIndex]
			}
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

func injectIntoItemsWithSelector(
	target *MappingNode,
	pathItem *pathItem,
	parsedPath []*pathItem,
	valueToInject *MappingNode,
	i int,
) int {
	targetItemIndex := slices.IndexFunc(
		target.Items,
		objectHasPropertyWithValue(
			pathItem.arrayItemSelector,
		),
	)
	if targetItemIndex < 0 {
		// If there is no item in the array that matches the selector,
		// the value can not be injected.
		return -1
	}

	if i == len(parsedPath)-1 {
		target.Items[targetItemIndex] = valueToInject
	}
	return targetItemIndex
}

func objectHasPropertyWithValue(selector *arrayItemSelector) func(*MappingNode) bool {
	return func(item *MappingNode) bool {
		if item.Fields == nil {
			return false
		}

		value, exists := item.Fields[selector.key]
		if !exists {
			return false
		}
		// Only string matching is supported for now,
		// the path parser will need to be updated
		// to support other types of values if needed in the future.
		if StringValue(value) == selector.value {
			return true
		}

		return false
	}
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
		} else if !hasValue {
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
//
// This path syntax is similar to JSONPath, but is not an implementation of the JSONPath
// specification. Paths and patterns are restricted to a limited set of features
// intended to meet the needs of the Blueprint framework and provider implementations.
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
	// Indicates that the path item is a selector for an array item
	// based on a key-value pair, e.g. "[?(@.<key> = \"<value>\")]".
	// This is used to target a specific item in an array of objects
	// by a unique attribute.
	arrayItemSelector *arrayItemSelector
}

type arrayItemSelector struct {
	key   string
	value string
}

func parsePath(path string, allowPatterns bool) ([]*pathItem, error) {
	// if len(path) == 0 || path[0] != '$' {
	// 	return nil, errInvalidMappingPath(path, nil)
	// }

	// pathWithoutRoot := path[1:]
	// if len(pathWithoutRoot) == 0 {
	// 	// "$" is a valid path to the root of the node.
	// 	return []*pathItem{}, nil
	// }

	// return parsePathItems(pathWithoutRoot, allowPatterns)
	parser := newPathParser(path, allowPatterns)
	return parser.parse()
}

type pathParser struct {
	input         string
	allowPatterns bool
	pos           int
	// A stack of positions in the sequence where a path item
	// evaluation started, this allows for state.pos updates
	// to be reverted when a path item evaluation fails.
	startPosStack []int
}

// endChar is a marker rune used to indicate the end of the input,
// it is the null unicode character (U+0000).
const endChar = rune(0)

func newPathParser(input string, allowPatterns bool) *pathParser {
	return &pathParser{
		input:         strings.TrimSpace(input),
		allowPatterns: allowPatterns,
		pos:           0,
		startPosStack: []int{},
	}
}

func (p *pathParser) parse() ([]*pathItem, error) {
	return p.path()
}

// path = '$' | pathWithAccessors ;
func (p *pathParser) path() ([]*pathItem, error) {
	if p.input == "$" {
		// "$" is a valid path to the root of the node.
		// An empty path array indicates that the path is for the root node.
		return []*pathItem{}, nil
	}

	return p.pathWithAccessors()
}

// pathWithAccessors = '$' , { nameAccessor | indexAccessor | selector } ;
func (p *pathParser) pathWithAccessors() ([]*pathItem, error) {
	if p.peek() != '$' {
		return nil, errInvalidMappingPath(
			p.input,
			errors.New("path must start with '$'"),
		)
	}

	p.advance()
	return p.propertyPath()
}

// propertyPath = { nameAccessor | indexAccessor | selector } ;
func (p *pathParser) propertyPath() ([]*pathItem, error) {
	isValidPathItem := true
	path := []*pathItem{}
	for isValidPathItem && !p.isAtEnd() {
		namePathItem := p.nameAccessor()
		if namePathItem != nil {
			path = append(path, namePathItem)
			continue
		}

		indexPathItem := p.indexAccessor()
		if indexPathItem != nil {
			path = append(path, indexPathItem)
			continue
		}

		selectorPathItem := p.selector()
		if selectorPathItem != nil {
			path = append(path, selectorPathItem)
		} else {
			isValidPathItem = false
		}
	}

	if !isValidPathItem {
		return nil, errInvalidMappingPath(
			p.input,
			fmt.Errorf(
				"invalid path item at position %d near %q",
				p.pos,
				getNextChars(p.input, p.pos),
			),
		)
	}

	return path, nil
}

func getNextChars(input string, pos int) string {
	if pos >= utf8.RuneCountInString(input) {
		return ""
	}

	nextChars := ""
	for i := pos; i < utf8.RuneCountInString(input) && i <
		pos+10; i++ {
		char, _ := utf8.DecodeRuneInString(input[i:])
		nextChars += string(char)
	}
	if len(nextChars) > 10 {
		nextChars = nextChars[:10]
	}
	return nextChars
}

// nameAccessorWithPatterns = ( "." , ( name | "*" ) ) | ( "[" , nameStringLiteral , "]" ) ;
// nameAccessor = ( "." , name ) | ( "[" , nameStringLiteral , "]" ) ;
func (p *pathParser) nameAccessor() *pathItem {
	// As a name accessor is not the only rule that can start with a "[",
	// we need to save the current position in the sequence so that we can revert
	// back in the case that a "[" character is not followed by a name string literal.
	p.savePos()
	if p.match('.') {
		return p.namePathItem()
	}

	if !p.match('[') {
		return nil
	}

	namePathItem := p.nameStringLiteralPathItem()
	if namePathItem == nil {
		p.backtrack()
		return nil
	}

	p.popPos()

	if p.match(']') {
		return namePathItem
	}

	return nil
}

// nameWithPatterns = "*" | ( ? [A-Za-z_] ? , { ? [A-Za-z0-9_\-] ? } ) ;
// name =  ? [A-Za-z_] ? , { ? [A-Za-z0-9_\-] ? } ;
func (p *pathParser) namePathItem() *pathItem {
	if p.allowPatterns && p.match('*') {
		return &pathItem{
			anyFieldName: true,
		}
	}

	name := p.name()
	if name != nil {
		return &pathItem{
			fieldName: *name,
		}
	}

	p.backtrack()
	return nil
}

// name = [A-Za-z_] , { [A-Za-z0-9_\-] } ;
func (p *pathParser) name() *string {
	name := ""
	next := p.peek()
	if !(unicode.IsLetter(next) || next == '_') {
		return nil
	}

	p.advance()
	name += string(next)

	isValidNameChar := true
	for isValidNameChar && !p.isAtEnd() {
		char := p.peek()
		if unicode.IsLetter(char) ||
			char == '_' ||
			unicode.IsDigit(char) ||
			char == '-' {
			p.advance()
			name += string(char)
		} else {
			isValidNameChar = false
		}
	}

	return &name
}

// nameStringLiteral = '"' , { ? [A-Za-z0-9_\-\.] ? } , '"' ;
func (p *pathParser) nameStringLiteralPathItem() *pathItem {
	name := p.nameStringLiteral()
	if name != nil {
		return &pathItem{
			fieldName: *name,
		}
	}

	return nil
}

// nameStringLiteral = '"' , { ? [A-Za-z0-9_\-\.] ? } , '"' ;
func (p *pathParser) nameStringLiteral() *string {
	if !p.match('"') {
		return nil
	}

	name := ""
	inStringLiteral := true
	for inStringLiteral && !p.isAtEnd() {
		if p.check('"') {
			inStringLiteral = false
			p.advance()
		} else {
			name += string(p.advance())
		}
	}

	if inStringLiteral {
		// The name string literal was not closed properly.
		return nil
	}

	return &name
}

// indexAccessWithPatterns = "[" , ( intLiteral | "*" ) , "]" ;
// indexAccessor = "[" , intLiteral , "]" ;
func (p *pathParser) indexAccessor() *pathItem {
	// As an index accessor is not the only rule that can start with a "[",
	// we need to save the current position in the sequence so that we can revert
	// back in the case that a "[" token is not followed by an int literal.
	p.savePos()
	if p.match('[') {
		anyIndex := false
		if p.allowPatterns && p.match('*') {
			anyIndex = true
		}

		index := (*int)(nil)
		if !anyIndex {
			index = p.intLiteral()
		}

		if !p.match(']') {
			// The next token could be a name string literal or selector, so we can't return
			// an error here and we need to backtrack to allow another rule (e.g. name accessor)
			// to match on the opening bracket.
			p.backtrack()
			return nil
		}

		p.popPos()
		return &pathItem{
			arrayIndex: index,
			anyIndex:   anyIndex,
		}
	}

	p.popPos()
	return nil
}

// selector = "[" , "@" , "." , name , "=" , stringLiteral , "]" ;
func (p *pathParser) selector() *pathItem {
	// As a selector is not the only rule that can start with a "[",
	// we need to save the current position in the sequence so that we can revert
	// back in the case that a "[" token is not followed by a valid selector.
	p.savePos()
	if !p.check('[') {
		p.popPos()
		return nil
	}

	// Consume the opening bracket.
	p.advance()

	// There can be white space before the "@" character in a selector.
	p.consumeWhiteSpace()

	if !p.match('@') {
		p.backtrack()
		return nil
	}

	if !p.match('.') {
		p.backtrack()
		return nil
	}

	name := p.name()
	if name == nil {
		p.backtrack()
		return nil
	}

	// There can be white space before the "=" character in a selector.
	p.consumeWhiteSpace()

	if !p.match('=') {
		p.backtrack()
		return nil
	}

	// There can be white space before the string literal in a selector.
	p.consumeWhiteSpace()

	stringLiteral := p.stringLiteral()
	if stringLiteral == nil {
		p.backtrack()
		return nil
	}

	// There can be white space before the closing bracket in a selector.
	p.consumeWhiteSpace()

	if !p.match(']') {
		p.backtrack()
		return nil
	}

	p.popPos()
	return &pathItem{
		arrayItemSelector: &arrayItemSelector{
			key:   *name,
			value: *stringLiteral,
		},
	}
}

func (p *pathParser) consumeWhiteSpace() {
	for !p.isAtEnd() && unicode.IsSpace(p.peek()) {
		p.advance()
	}
}

func (p *pathParser) intLiteral() *int {
	if p.isAtEnd() || !unicode.IsDigit(p.peek()) {
		return nil
	}

	intStr := ""
	for !p.isAtEnd() && unicode.IsDigit(p.peek()) {
		intStr += string(p.advance())
	}

	index, err := strconv.ParseInt(intStr, 10, 64)
	if err != nil {
		return nil
	}

	if index < 0 {
		return nil
	}

	indexAsInt := int(index)
	return &indexAsInt
}

// stringLiteral = '"' , ( ? utf-8 char excluding quote ? | escaped quote ) , '"' ;
func (p *pathParser) stringLiteral() *string {
	if !p.match('"') {
		return nil
	}

	stringLiteral := ""
	inStringLiteral := true
	for inStringLiteral && !p.isAtEnd() {
		if p.check('"') {
			inStringLiteral = false
			p.advance()
		} else if p.check('\\') {
			// Skip the escape character.
			p.advance()
			if !p.isAtEnd() {
				// Add the escaped character.
				stringLiteral += string(p.advance())
			}
		} else {
			stringLiteral += string(p.advance())
		}
	}

	if inStringLiteral {
		// The string literal was not closed properly.
		return nil
	}

	return &stringLiteral
}

func (p *pathParser) match(chars ...rune) bool {
	if slices.ContainsFunc(chars, p.check) {
		p.advance()
		return true
	}

	return false
}

func (p *pathParser) check(char rune) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek() == char
}

func (p *pathParser) advance() rune {
	if !p.isAtEnd() {
		p.pos += 1
	}
	return p.previous()
}

func (p *pathParser) previous() rune {
	prevChar, _ := utf8.DecodeRuneInString(p.input[p.pos-1:])
	return prevChar
}

func (p *pathParser) peek() rune {
	if p.isAtEnd() {
		return endChar
	}
	char, _ := utf8.DecodeRuneInString(p.input[p.pos:])
	return char
}

func (p *pathParser) isAtEnd() bool {
	return p.pos >= utf8.RuneCountInString(p.input)
}

func (p *pathParser) savePos() {
	p.startPosStack = append(p.startPosStack, p.pos)
}

func (p *pathParser) backtrack() {
	if len(p.startPosStack) > 0 {
		p.pos = p.startPosStack[len(p.startPosStack)-1]
		p.startPosStack = p.startPosStack[:len(p.startPosStack)-1]
	}
}

func (p *pathParser) popPos() {
	if len(p.startPosStack) > 0 {
		p.startPosStack = p.startPosStack[:len(p.startPosStack)-1]
	}
}
