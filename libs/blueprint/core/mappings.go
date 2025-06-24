package core

import (
	"fmt"
	"strconv"
	"strings"

	json "github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"gopkg.in/yaml.v3"
)

// MappingNode provides a tree structure for user-defined
// resource specs or metadata mappings.
//
// This is used to allow creators of resource providers to define
// custom specifications while supporting ${..} placeholder substitution
// as a first class member of the primary representation of the spec.
// The initial intention was to allow the definition of custom native structs
// for resource specs, but this was abandoned in favour of a structure that will
// make placeholder substitution easier to deal with for the framework.
//
// This is also used to provide a tree structure for metadata mappings
// to facilitate substitutions at all levels of nesting in user-provided
// metadata.
//
// A mapping node can be used to store data for resources, links and data sources
// along with storing the output of a ${..} substitution.
type MappingNode struct {
	// Scalar represents a scalar value in a mapping node.
	// This could be an integer, string, boolean or a floating point number.
	Scalar *ScalarValue
	// Fields represents a map of field names to child mapping nodes.
	Fields map[string]*MappingNode
	// Items represents a slice of child mapping nodes.
	Items []*MappingNode
	// StringWithSubstitutions is a slice of strings and substitutions
	// where substitutions are a representation of placeholders for variables,
	// resource properties, data source properties and child blueprint properties
	// or function calls contained within ${..}.
	StringWithSubstitutions *substitutions.StringOrSubstitutions
	// SourceMeta is the source metadata for the mapping node,
	// this is optional and may or may not be set depending on the context
	// and the source blueprint.
	SourceMeta *source.Meta
	// FieldsSourceMeta is a map of field names to source metadata
	// used to store the source location of fields in the original source.
	// This is optional and may or may not be set depending on the context
	// and the source blueprint.
	FieldsSourceMeta map[string]*source.Meta
}

// MarshalYAML fulfils the yaml.Marshaler interface
// to marshal a mapping node into a YAML representation.
func (m *MappingNode) MarshalYAML() (any, error) {
	if m.Scalar != nil {
		return m.Scalar, nil
	}

	if m.StringWithSubstitutions != nil {
		return m.StringWithSubstitutions, nil
	}

	if m.Fields != nil {
		return m.Fields, nil
	}

	if m.Items != nil {
		return m.Items, nil
	}

	return nil, errMissingMappingNode(nil)
}

// UnmarshalYAML fulfils the yaml.Unmarshaler interface
// to unmarshal a YAML representation into a mapping node.
func (m *MappingNode) UnmarshalYAML(node *yaml.Node) error {
	m.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   node.Line,
			Column: node.Column,
		},
	}

	if node.Kind == yaml.ScalarNode {
		return m.parseYAMLSubstitutionsOrScalar(node)
	}

	if node.Kind == yaml.SequenceNode {
		m.Items = make([]*MappingNode, len(node.Content))
		for i, item := range node.Content {
			m.Items[i] = &MappingNode{}
			if err := m.Items[i].UnmarshalYAML(item); err != nil {
				return err
			}
		}
		return nil
	}

	if node.Kind == yaml.MappingNode {
		m.Fields = make(map[string]*MappingNode)
		m.FieldsSourceMeta = make(map[string]*source.Meta)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			m.Fields[key.Value] = &MappingNode{}
			if err := m.Fields[key.Value].UnmarshalYAML(value); err != nil {
				return err
			}
			m.FieldsSourceMeta[key.Value] = &source.Meta{
				Position: source.Position{
					Line:   key.Line,
					Column: key.Column,
				},
			}
		}
		return nil
	}

	return errInvalidMappingNode(YAMLNodeToPosInfo(node))
}

type yamlNodePosInfoWrapper struct {
	node *yaml.Node
}

func (w *yamlNodePosInfoWrapper) GetLine() int {
	return w.node.Line
}

func (w *yamlNodePosInfoWrapper) GetColumn() int {
	return w.node.Column
}

// YAMLNodeToPosInfo returns a source.PositionInfo
// for a YAML node. This is used to provide position information
// for errors that are shared between YAML and JWCC parsers.
func YAMLNodeToPosInfo(node *yaml.Node) source.PositionInfo {
	return &yamlNodePosInfoWrapper{
		node,
	}
}

func (m *MappingNode) parseYAMLSubstitutionsOrScalar(node *yaml.Node) error {
	sourceMeta := &source.Meta{
		Position: source.Position{
			Line:   node.Line,
			Column: node.Column,
		},
		EndPosition: source.EndSourcePositionFromYAMLScalarNode(node),
	}

	isBlockStyle := node.Style == yaml.LiteralStyle || node.Style == yaml.FoldedStyle
	precedingCharCount := substitutions.GetYAMLNodePrecedingCharCount(node)
	sourceStartMeta := substitutions.DetermineYAMLSourceStartMeta(node, sourceMeta)
	strSubs, err := substitutions.ParseSubstitutionValues(
		"", // substitutionContext
		node.Value,
		sourceStartMeta,
		true, // outputLineInfo
		// Due to the difficulty involved in getting the precise starting column
		// of a "folded" or "literal" style block in a mapping or sequence,
		// the column number should be ignored until the difficulty of doing so changes.
		isBlockStyle, // ignoreParentColumn
		precedingCharCount,
	)
	if err != nil {
		// When substitutions are present but invalid, we must return an error to provide
		// the best possible user experience when debugging issues with a blueprint,
		// silently ignoring invalid substitutions and falling back to string literals
		// would make it harder to debug.
		return err
	} else if len(strSubs) == 0 || (len(strSubs) == 1 && strSubs[0].StringValue != nil) {
		// Parse scalar value if there are no substitutions.
		m.Scalar = &ScalarValue{}
		return m.Scalar.UnmarshalYAML(node)
	}

	m.StringWithSubstitutions = &substitutions.StringOrSubstitutions{
		Values:     strSubs,
		SourceMeta: sourceMeta,
	}
	return nil
}

// MarshalJSON fulfils the json.Marshaler interface
// to marshal a blueprint mapping node into a JSON representation.
func (m *MappingNode) MarshalJSON() ([]byte, error) {
	if m.Scalar != nil {
		return json.Marshal(m.Scalar)
	}

	if m.StringWithSubstitutions != nil {
		return json.Marshal(m.StringWithSubstitutions)
	}

	if m.Fields != nil {
		return json.Marshal(m.Fields)
	}

	if m.Items != nil {
		return json.Marshal(m.Items)
	}

	return nil, errMissingMappingNode(nil)
}

// UnmarshalJSON fulfils the json.Unmarshaler interface
// to unmarshal a serialised blueprint mapping node.
func (m *MappingNode) UnmarshalJSON(data []byte) error {
	var items []*MappingNode
	if err := json.Unmarshal(data, &items); err == nil {
		m.Items = items
		return nil
	}

	var fields map[string]*MappingNode
	if err := json.Unmarshal(data, &fields); err == nil {
		m.Fields = fields
		return nil
	}

	err := m.parseJSONSubstitutionsOrScalar(data)
	if err == nil {
		return nil
	}

	return errInvalidMappingNode(nil)
}

func (m *MappingNode) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	m.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	nodeMap, isMap := node.Value.(map[string]json.Node)
	if isMap {
		err := m.fieldsFromJSONNode(nodeMap, linePositions, parentPath)
		if err != nil {
			return err
		}

		return nil
	}

	nodeSlice, isSlice := node.Value.([]json.Node)
	if isSlice {
		err := m.itemsFromJSONNode(nodeSlice, linePositions, parentPath)
		if err != nil {
			return err
		}

		return nil
	}

	nodeStringVal, isString := node.Value.(string)
	if isString {
		err := m.substitutionsOrScalarFromJSONNode(
			node,
			nodeStringVal,
			linePositions,
			parentPath,
		)
		if err != nil {
			return err
		}

		return nil
	}

	m.Scalar = &ScalarValue{}
	err := m.Scalar.FromJSONNode(
		node,
		linePositions,
		parentPath,
	)
	if err == nil {
		return nil
	}

	return errInvalidMappingNode(&m.SourceMeta.Position)
}

func (m *MappingNode) fieldsFromJSONNode(
	nodeMap map[string]json.Node,
	linePositions []int,
	parentPath string,
) error {
	m.Fields = make(map[string]*MappingNode)
	m.FieldsSourceMeta = make(map[string]*source.Meta)

	for k, v := range nodeMap {
		m.Fields[k] = &MappingNode{}
		fieldPath := CreateJSONNodePath(k, parentPath, false)
		m.FieldsSourceMeta[k] = source.ExtractSourcePositionForJSONNodeMapField(
			&v,
			linePositions,
		)
		err := m.Fields[k].FromJSONNode(&v, linePositions, fieldPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MappingNode) itemsFromJSONNode(
	nodeSlice []json.Node,
	linePositions []int,
	parentPath string,
) error {
	m.Items = make([]*MappingNode, len(nodeSlice))
	for i, item := range nodeSlice {
		m.Items[i] = &MappingNode{}
		key := fmt.Sprintf("%d", i)
		fieldPath := CreateJSONNodePath(key, parentPath, false)
		err := m.Items[i].FromJSONNode(&item, linePositions, fieldPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MappingNode) substitutionsOrScalarFromJSONNode(
	node *json.Node,
	stringVal string,
	linePositions []int,
	parentPath string,
) error {
	m.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	precedingCharCount := substitutions.JSONNodePrecedingCharCount
	sourceStartMeta := substitutions.DetermineJSONSourceStartMeta(
		node,
		stringVal,
		linePositions,
	)
	parsedValues, err := substitutions.ParseSubstitutionValues(
		parentPath, // substitutionContext
		stringVal,
		sourceStartMeta,
		true, // outputLineInfo
		// For JSON with Commas and Comments, the column number will be reliable
		// so we can use it to get the precise starting column of the string.
		false,              // ignoreParentColumn
		precedingCharCount, // parentContextPrecedingCharCount
	)
	if err != nil {
		// When substitutions are present but invalid, we must return an error to provide
		// the best possible user experience when debugging issues with a blueprint,
		// silently ignoring invalid substitutions and falling back to string literals
		// would make it harder to debug.
		return err
	} else if len(parsedValues) == 0 || (len(parsedValues) == 1 && parsedValues[0].StringValue != nil) {
		// Parse scalar value if there are no substitutions.
		m.Scalar = &ScalarValue{}
		return m.Scalar.FromJSONNode(
			node,
			linePositions,
			parentPath,
		)
	}

	m.StringWithSubstitutions = &substitutions.StringOrSubstitutions{
		Values:     parsedValues,
		SourceMeta: m.SourceMeta,
	}
	return nil
}

func (m *MappingNode) parseJSONSubstitutionsOrScalar(data []byte) error {
	dataStr := string(data)
	// Remove the quotes from the string
	normalised := dataStr
	if len(dataStr) >= 2 && dataStr[0] == '"' && dataStr[len(dataStr)-1] == '"' {
		withoutSurroundingQuotes := dataStr[1 : len(dataStr)-1]
		// Remove a single escape character for escaped quotes inside the string.
		normalised = strings.ReplaceAll(withoutSurroundingQuotes, `\"`, `"`)
	}
	strSubs, err := substitutions.ParseSubstitutionValues("", normalised, nil, false, true, 0)

	if err != nil {
		// When substitutions are present but invalid, we must return an error to provide
		// the best possible user experience when debugging issues with a blueprint,
		// silently ignoring invalid substitutions and falling back to string literals
		// would make it harder to debug.
		return err
	} else if len(strSubs) == 0 || (len(strSubs) == 1 && strSubs[0].StringValue != nil) {
		// Parse scalar value if there are no substitutions.
		m.Scalar = &ScalarValue{}
		return m.Scalar.UnmarshalJSON(data)
	}

	m.StringWithSubstitutions = &substitutions.StringOrSubstitutions{
		Values: strSubs,
	}
	return nil
}

// MergeMaps merges multiple mapping nodes that represent a map of fields
// to values into a single mapping node.
func MergeMaps(nodes ...*MappingNode) *MappingNode {
	merged := make(map[string]*MappingNode)
	for _, node := range nodes {
		if node != nil && node.Fields != nil {
			for k, v := range node.Fields {
				merged[k] = v
			}
		}
	}
	return &MappingNode{
		Fields: merged,
	}
}

// IsNilMappingNode returns true if the mapping node is nil or has no content.
func IsNilMappingNode(node *MappingNode) bool {
	return node == nil ||
		(node.Scalar == nil &&
			node.StringWithSubstitutions == nil &&
			node.Fields == nil &&
			node.Items == nil)
}

// IsObjectMappingNode returns true if the mapping node is an object or map of fields.
func IsObjectMappingNode(node *MappingNode) bool {
	return node != nil && node.Fields != nil
}

// IsArrayMappingNode returns true if the mapping node is an array/slice of items.
func IsArrayMappingNode(node *MappingNode) bool {
	return node != nil && node.Items != nil
}

// IsScalarMappingNode returns true if the mapping node is a scalar value.
func IsScalarMappingNode(node *MappingNode) bool {
	return node != nil && node.Scalar != nil
}

// IsStringWithSubsMappingNode returns true if the mapping node
// is a string with substitutions.
func IsStringWithSubsMappingNode(node *MappingNode) bool {
	return node != nil && node.StringWithSubstitutions != nil
}

// ScalarMappingNodeEqual returns true if the scalar values of two mapping nodes are equal.
func ScalarMappingNodeEqual(nodeA, nodeB *MappingNode) bool {
	if (nodeA == nil || nodeA.Scalar == nil) &&
		(nodeB == nil || nodeB.Scalar == nil) {
		return true
	}

	if nodeA == nil || nodeA.Scalar == nil ||
		nodeB == nil || nodeB.Scalar == nil {
		return false
	}

	return nodeA.Scalar.Equal(nodeB.Scalar)
}

// MappingNodeEqual returns true if the mapping nodes are equal.
// This will carry out a deep comparison for mapping nodes that represent
// maps/objects and arrays.
func MappingNodeEqual(nodeA, nodeB *MappingNode) bool {
	if IsScalarMappingNode(nodeA) && IsScalarMappingNode(nodeB) {
		return ScalarMappingNodeEqual(nodeA, nodeB)
	}

	if IsObjectMappingNode(nodeA) && IsObjectMappingNode(nodeB) {
		if len(nodeA.Fields) != len(nodeB.Fields) {
			return false
		}

		for k, v := range nodeA.Fields {
			if !MappingNodeEqual(v, nodeB.Fields[k]) {
				return false
			}
		}
		return true
	}

	if IsArrayMappingNode(nodeA) && IsArrayMappingNode(nodeB) {
		if len(nodeA.Items) != len(nodeB.Items) {
			return false
		}

		for i := range nodeA.Items {
			if !MappingNodeEqual(nodeA.Items[i], nodeB.Items[i]) {
				return false
			}
		}
		return true
	}

	return false
}

// ParseScalarMappingNode parses a scalar mapping node
// (string, boolean, integer, or float) from a source string.
func ParseScalarMappingNode(source string) *MappingNode {
	// Try float first if the string contains a period.
	if strings.Contains(source, ".") {
		floatVal, err := strconv.ParseFloat(source, 64)
		if err == nil {
			return MappingNodeFromFloat(floatVal)
		}
	}

	intVal, err := strconv.ParseInt(source, 10, 64)
	if err == nil {
		return MappingNodeFromInt(int(intVal))
	}

	boolVal, err := strconv.ParseBool(source)
	if err == nil {
		return MappingNodeFromBool(boolVal)
	}

	// Default to string if another scalar type couldn't be parsed.
	return MappingNodeFromString(source)
}
