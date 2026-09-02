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
// - "[@ = \"<value>\"]" To target a specific scalar item in an array by its own value
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
			current, pathExists = itemAtIndex(current.Items, *pathItem.arrayIndex)
		} else if pathItem.arrayItemSelector != nil && current.Items != nil {
			targetItemIndex := slices.IndexFunc(
				current.Items,
				objectHasPropertyWithValue(
					pathItem.arrayItemSelector,
				),
			)
			if targetItemIndex < 0 {
				pathExists = false
				current = nil
			} else {
				current = current.Items[targetItemIndex]
			}
		} else {
			// The path item cannot be applied to this node, for example a field
			// lookup against a scalar. Without this the node would be left
			// untouched and returned as though the path had resolved, which
			// silently yields the wrong value rather than reporting a miss.
			pathExists = false
			current = nil
		}

		i += 1
	}

	if maxDepth < len(parsedPath) {
		return nil, nil
	}

	return current, nil
}

// Safely indexes into array items, treating an out-of-range index
// as a path miss rather than panicking on a path a caller cannot validate up
// front.
func itemAtIndex(items []*MappingNode, index int) (*MappingNode, bool) {
	if index < 0 || index >= len(items) {
		return nil, false
	}

	return items[index], true
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
// - "[@ = \"<value>\"]" To target a specific scalar item in an array by its own value
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
// - "[@ = \"<value>\"]" To target a specific scalar item in an array by its own value
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
	selectorMismatch := ""
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
			injectedIndex, mismatch := injectIntoItemsWithSelector(
				current,
				pathItem,
				parsedPath,
				value,
				i,
			)
			if injectedIndex < 0 {
				pathExists = false
				selectorMismatch = mismatch
			} else {
				current = current.Items[injectedIndex]
			}
		} else {
			pathExists = false
		}

		i += 1
	}

	if !pathExists {
		if selectorMismatch != "" {
			return fmt.Errorf(
				"path %q could not be injected into the mapping node, %s",
				path,
				selectorMismatch,
			)
		}

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

// Reports the index the value was injected at, or -1 with a reason describing why the
// selector could not be satisfied, for the cases where the reason is not simply that the
// node's structure does not match the path.
func injectIntoItemsWithSelector(
	target *MappingNode,
	pathItem *pathItem,
	parsedPath []*pathItem,
	valueToInject *MappingNode,
	i int,
) (int, string) {
	targetItemIndex := slices.IndexFunc(
		target.Items,
		objectHasPropertyWithValue(
			pathItem.arrayItemSelector,
		),
	)
	if targetItemIndex < 0 {
		// A selector in the final position fully describes the item to inject, whether it
		// matches a scalar item by its own value ("[@ = \"x\"]") or an object item by an
		// attribute ("[@.<key> = \"<value>\"]"), so a missing item can be appended.
		//
		// It appends only when the value satisfies the selector, which is what makes
		// injecting the same value twice a no-op. An item that does not match the
		// selector that created it could never be found again, so every later injection
		// would append another copy.
		if i == len(parsedPath)-1 {
			if !objectHasPropertyWithValue(pathItem.arrayItemSelector)(valueToInject) {
				return -1, selectorMismatchReason(pathItem.arrayItemSelector, valueToInject)
			}

			target.Items = append(target.Items, valueToInject)
			return len(target.Items) - 1, ""
		}

		// A selector that is traversed through rather than injected into describes the
		// item it is looking for completely enough to create it, in the same way a
		// missing field or array is created on the way to the value. Its conditions are
		// the item's attributes, so an item built from them is matched by the selector
		// that created it on every later injection.
		newItem := createItemForSelector(pathItem.arrayItemSelector)
		if newItem == nil {
			return -1, ""
		}

		target.Items = append(target.Items, newItem)
		return len(target.Items) - 1, ""
	}

	if i == len(parsedPath)-1 {
		target.Items[targetItemIndex] = valueToInject
	}
	return targetItemIndex, ""
}

// Explains a value that a final-position selector does not match, which is otherwise
// indistinguishable from a structural mismatch and reads as though the data is missing
// when the value is present but keyed differently.
//
// It names the conditions and what the value carries for them, without reporting any
// value the caller supplied beyond the keys, since a contribution can hold credentials
// or other data that must not reach an error message.
func selectorMismatchReason(selector *arrayItemSelector, valueToInject *MappingNode) string {
	if selector.isScalar() {
		return fmt.Sprintf(
			"the value to inject does not match the selector's expected item value %q",
			selector.conditions[0].value,
		)
	}

	missing := []string{}
	differing := []string{}
	for _, condition := range selector.conditions {
		value, exists := fieldsOf(valueToInject)[condition.key]
		if !exists {
			missing = append(missing, condition.key)
			continue
		}
		if StringValue(value) != condition.value {
			differing = append(differing, condition.key)
		}
	}

	reason := "the value to inject does not match the selector that would locate it"
	if len(missing) > 0 {
		reason += fmt.Sprintf(", it has no %s field", strings.Join(missing, ", "))
	}
	if len(differing) > 0 {
		reason += fmt.Sprintf(", its %s field holds a different value", strings.Join(differing, ", "))
	}

	return reason + ". A contribution has to carry the fields its own path selects on"
}

func fieldsOf(node *MappingNode) map[string]*MappingNode {
	if node == nil || node.Fields == nil {
		return map[string]*MappingNode{}
	}
	return node.Fields
}

func objectHasPropertyWithValue(selector *arrayItemSelector) func(*MappingNode) bool {
	return func(item *MappingNode) bool {
		// A scalar selector matches the array item by its own value
		// (e.g. "[@ = \"x\"]" against an array of strings).
		if selector.isScalar() {
			return StringValue(item) == selector.conditions[0].value
		}

		if item.Fields == nil {
			return false
		}

		// Every condition must hold (AND semantics) for the item to match.
		for _, condition := range selector.conditions {
			value, exists := item.Fields[condition.key]
			if !exists {
				return false
			}
			// Only string matching is supported for now, the path parser will need to be
			// updated to support other types of values if needed in the future.
			if StringValue(value) != condition.value {
				return false
			}
		}

		return true
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

// Builds an array item that the given selector matches, for a selector that is traversed
// through on the way to the value being injected.
//
// The selector's conditions are attribute equalities, so they double as a constructor.
// An item carrying each condition's key and value is matched by that same selector, which
// keeps a later injection finding the item rather than creating a second one.
//
// A scalar selector ("[@ = \"x\"]") describes an item that has no attributes to traverse
// into, so there is nothing sensible to create for one in this position and it reports
// that the path cannot be injected.
func createItemForSelector(selector *arrayItemSelector) *MappingNode {
	if selector.isScalar() {
		return nil
	}

	newItem := &MappingNode{
		Fields: map[string]*MappingNode{},
	}
	for _, condition := range selector.conditions {
		newItem.Fields[condition.key] = MappingNodeFromString(condition.value)
	}

	return newItem
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

	if nextPathItem.arrayIndex != nil || nextPathItem.arrayItemSelector != nil {
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
	// conditions are matched against an array item with AND semantics: an item is
	// selected only when every condition holds. A single condition with an empty key is
	// a scalar selector that matches an array item by its own value ("[@ = \"<value>\"]");
	// otherwise each condition matches an object attribute ("[@.<key> = \"<value>\"]"),
	// and multiple conditions are joined with "&&" to target an item by a composite key
	// ("[@.<k1> = \"<v1>\" && @.<k2> = \"<v2>\"]").
	conditions []arrayItemSelectorCondition
}

type arrayItemSelectorCondition struct {
	key   string
	value string
}

// Reports whether the selector matches a scalar array item by its own value,
// i.e. a single condition with an empty key ("[@ = \"<value>\"]").
func (s *arrayItemSelector) isScalar() bool {
	return len(s.conditions) == 1 && s.conditions[0].key == ""
}

func parsePath(path string, allowPatterns bool) ([]*pathItem, error) {
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

// selector = "[" , "@" , [ "." , name ] , "=" , stringLiteral , "]" ;
// The optional ".name" targets an item in an array of objects by a unique
// attribute (e.g. "[@.id = \"x\"]"); omitting it targets a scalar array item by its
// own value (e.g. "[@ = \"x\"]").
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

	condition := p.selectorCondition()
	if condition == nil {
		p.backtrack()
		return nil
	}
	conditions := []arrayItemSelectorCondition{*condition}

	// Additional conditions are joined with "&&" to form a composite selector that
	// targets an item by more than one attribute, e.g.
	// "[@.<k1> = \"<v1>\" && @.<k2> = \"<v2>\"]".
	for {
		p.consumeWhiteSpace()
		if !p.matchDoubleAmpersand() {
			break
		}
		next := p.selectorCondition()
		if next == nil {
			p.backtrack()
			return nil
		}
		conditions = append(conditions, *next)
	}

	// There can be white space before the closing bracket in a selector.
	p.consumeWhiteSpace()

	if !p.match(']') {
		p.backtrack()
		return nil
	}

	p.popPos()
	return &pathItem{
		arrayItemSelector: &arrayItemSelector{conditions: conditions},
	}
}

// Parses a single "@.<name> = \"<value>\"" condition (or the scalar
// "@ = \"<value>\"" form, where the key stays empty). It does not manage backtracking; the
// caller owns the saved position and reverts on a nil result.
func (p *pathParser) selectorCondition() *arrayItemSelectorCondition {
	// There can be white space before the "@" character in a condition.
	p.consumeWhiteSpace()

	if !p.match('@') {
		return nil
	}

	// There can be white space after the "@" character (before "." or "=").
	p.consumeWhiteSpace()

	// An optional ".name" makes this an object-attribute condition; without it the
	// condition matches a scalar array item by its own value (key stays empty).
	name := ""
	if p.match('.') {
		parsedName := p.name()
		if parsedName == nil {
			return nil
		}
		name = *parsedName

		// There can be white space before the "=" character in a condition.
		p.consumeWhiteSpace()
	}

	if !p.match('=') {
		return nil
	}

	// There can be white space before the string literal in a condition.
	p.consumeWhiteSpace()

	stringLiteral := p.stringLiteral()
	if stringLiteral == nil {
		return nil
	}

	return &arrayItemSelectorCondition{key: name, value: *stringLiteral}
}

func (p *pathParser) matchDoubleAmpersand() bool {
	if p.peek() != '&' {
		return false
	}
	save := p.pos
	p.advance()
	if p.peek() != '&' {
		p.pos = save
		return false
	}
	p.advance()
	return true
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
