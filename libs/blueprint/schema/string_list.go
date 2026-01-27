package schema

import (
	"fmt"
	"strings"

	json "github.com/coreos/go-json"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/common/core"
	"gopkg.in/yaml.v3"
)

// StringList provides a list of strings with source meta information
// that is populated for certain source formats.
// This should always be embedded in a struct that provides context
// such as a transform list or a depends on list so more specific errors
// can be returned.
type StringList struct {
	Values []string
	// A list of source meta information for each string value
	// that if populated, will be in the same order as the values.
	SourceMeta []*source.Meta
}

func (t *StringList) MarshalYAML() (any, error) {
	// Always marshal as a slice.
	return t.Values, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// This allows StringList to be used directly in struct fields
// (e.g., LinkSelector.Exclude) without requiring a wrapper type.
func (t *StringList) UnmarshalYAML(value *yaml.Node) error {
	return t.unmarshalYAML(value, errInvalidStringListType, "string list")
}

func errInvalidStringListType(underlyingError error, line *int, column *int) error {
	return &Error{
		ReasonCode: ErrorSchemaReasonCodeInvalidStringList,
		Err: fmt.Errorf(
			"invalid string list value, must be a string or a list of strings: %s",
			underlyingError.Error(),
		),
		SourceLine:   line,
		SourceColumn: column,
	}
}

func (t *StringList) unmarshalYAML(
	value *yaml.Node,
	errFunc func(error, *int, *int) error,
	elementType string,
) error {
	if value.Kind == yaml.ScalarNode {
		t.Values = []string{value.Value}
		t.SourceMeta = []*source.Meta{
			{
				Position: source.Position{
					Line:   value.Line,
					Column: value.Column,
				},
				EndPosition: source.EndSourcePositionFromYAMLScalarNode(value),
			},
		}
		return nil
	}

	if value.Kind == yaml.SequenceNode {
		values, positions, err := collectStringNodeValues(value.Content, errFunc, elementType)
		if err != nil {
			return err
		}
		t.Values = values
		t.SourceMeta = positions
		return nil
	}

	return errFunc(
		fmt.Errorf("unexpected yaml node for %s: %s", elementType, yamlKindMappings[value.Kind]),
		&value.Line,
		&value.Column,
	)
}

func (t *StringList) MarshalJSON() ([]byte, error) {
	// Always marshal as a slice.
	return json.Marshal(t.Values)
}

func (t *StringList) unmarshalJSON(
	data []byte,
	errFunc func(error, *int, *int) error,
	elementType string,
) error {
	transformValues := []string{}
	// Try to parse a slice, then fall back to a single string.
	// There is no better way to know with the built-in JSON library,
	// yes there are more efficient checks you can do by simply looking
	// at the characters in the string but they will not be as reliable
	// as unmarshalling.
	err := json.Unmarshal(data, &transformValues)
	if err == nil {
		t.Values = transformValues
		return nil
	}

	var transformValue string
	err = json.Unmarshal(data, &transformValue)
	if err != nil {
		return errFunc(
			fmt.Errorf("unexpected value provided for %s in json: %s", elementType, err.Error()),
			nil,
			nil,
		)
	}
	t.Values = []string{transformValue}

	return nil
}

func (t *StringList) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	transformStringNodes, ok := node.Value.([]json.Node)
	if !ok {
		transformStringNodes = []json.Node{*node}
	}

	t.Values = make([]string, len(transformStringNodes))
	t.SourceMeta = make([]*source.Meta, len(transformStringNodes))
	for i, stringNode := range transformStringNodes {
		t.SourceMeta[i] = source.ExtractSourcePositionFromJSONNode(
			&stringNode,
			linePositions,
		)
		t.Values[i] = stringNode.Value.(string)
	}

	return nil
}

func collectStringNodeValues(
	nodes []*yaml.Node,
	errFunc func(error, *int, *int) error,
	elementType string,
) ([]string, []*source.Meta, error) {
	values := []string{}
	sourceMeta := []*source.Meta{}
	// For at least 99% of the cases it will be trivial to go through
	// the entire list of transform value nodes and identify any invalid
	// values. This is much better for users of the spec too!
	nonScalarNodeKinds := []yaml.Kind{}
	firstNonScalarIndex := -1
	for i, node := range nodes {
		if node.Kind != yaml.ScalarNode {
			nonScalarNodeKinds = append(nonScalarNodeKinds, node.Kind)
			if firstNonScalarIndex == -1 {
				firstNonScalarIndex = i
			}
		} else {
			values = append(values, node.Value)
			sourceMeta = append(sourceMeta, &source.Meta{
				Position: source.Position{
					Line:   node.Line,
					Column: node.Column,
				},
				EndPosition: source.EndSourcePositionFromYAMLScalarNode(node),
			})
		}
	}

	if len(nonScalarNodeKinds) > 0 {
		return nil, nil, errFunc(
			fmt.Errorf(
				"unexpected yaml nodes in %s list, only scalars are supported: %s",
				elementType,
				formatYamlNodeKindsForError(nonScalarNodeKinds),
			),
			// Take the position of the first non-scalar node,
			// the error message will be detailed enough for the user to figure out
			// which values in the list are invalid.
			&nodes[firstNonScalarIndex].Line,
			&nodes[firstNonScalarIndex].Column,
		)
	}

	return values, sourceMeta, nil
}

func formatYamlNodeKindsForError(nodeKinds []yaml.Kind) string {
	return strings.Join(
		core.Map(nodeKinds, func(kind yaml.Kind, index int) string {
			return fmt.Sprintf("%d:%s", index, yamlKindMappings[kind])
		}),
		",",
	)
}

var yamlKindMappings = map[yaml.Kind]string{
	yaml.AliasNode:    "alias",
	yaml.DocumentNode: "document",
	yaml.ScalarNode:   "scalar",
	yaml.MappingNode:  "mapping",
	yaml.SequenceNode: "sequence",
}
