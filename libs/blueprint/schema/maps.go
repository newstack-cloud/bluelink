package schema

import (
	json "github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"gopkg.in/yaml.v3"
)

// StringOrSubstitutionsMap provides a mapping of names to expanded
// strings that may contain substitutions.
// This includes extra information about the locations of
// the keys in the original source being unmarshalled.
// This information will not always be present, it is populated
// when unmarshalling from YAML and JWCC source documents.
type StringOrSubstitutionsMap struct {
	Values map[string]*substitutions.StringOrSubstitutions
	// Mapping of field names to their source locations.
	SourceMeta map[string]*source.Meta
}

func (m *StringOrSubstitutionsMap) MarshalYAML() (interface{}, error) {
	return m.Values, nil
}

func (m *StringOrSubstitutionsMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errInvalidGeneralMap(value)
	}

	m.Values = make(map[string]*substitutions.StringOrSubstitutions)
	m.SourceMeta = make(map[string]*source.Meta)
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i]
		val := value.Content[i+1]

		m.SourceMeta[key.Value] = &source.Meta{
			Position: source.Position{
				Line:   key.Line,
				Column: key.Column,
			},
		}

		var stringOrSubs substitutions.StringOrSubstitutions
		err := val.Decode(&stringOrSubs)
		if err != nil {
			return err
		}

		m.Values[key.Value] = &stringOrSubs
	}

	return nil
}

func (m *StringOrSubstitutionsMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Values)
}

func (m *StringOrSubstitutionsMap) UnmarshalJSON(data []byte) error {
	values := make(map[string]*substitutions.StringOrSubstitutions)
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	m.Values = values
	return nil
}

func (m *StringOrSubstitutionsMap) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.Values = make(map[string]*substitutions.StringOrSubstitutions)
	m.SourceMeta = make(map[string]*source.Meta)

	for key, valNode := range nodeMap {
		m.SourceMeta[key] = source.ExtractSourcePositionFromJSONNode(
			&valNode,
			linePositions,
		)

		stringOrSubs := &substitutions.StringOrSubstitutions{}
		fieldPath := core.CreateJSONNodePath(key, parentPath, false /* parentIsRoot */)
		err := stringOrSubs.FromJSONNode(
			&valNode,
			linePositions,
			/* parentPath */ fieldPath,
		)
		if err != nil {
			return err
		}

		m.Values[key] = stringOrSubs
	}

	return nil
}

// StringMap provides a mapping of names to string literals.
// This includes extra information about the locations of
// the keys in the original source being unmarshalled.
// This information will not always be present, it is populated
// when unmarshalling from YAML and JWCC source documents.
type StringMap struct {
	Values map[string]string
	// Mapping of field names to their source locations.
	SourceMeta map[string]*source.Meta
}

func (m *StringMap) MarshalYAML() (interface{}, error) {
	return m.Values, nil
}

func (m *StringMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errInvalidGeneralMap(value)
	}

	m.Values = make(map[string]string)
	m.SourceMeta = make(map[string]*source.Meta)
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i]
		val := value.Content[i+1]

		m.SourceMeta[key.Value] = &source.Meta{
			Position: source.Position{
				Line:   key.Line,
				Column: key.Column,
			},
		}

		var str string
		err := val.Decode(&str)
		if err != nil {
			return err
		}

		m.Values[key.Value] = str
	}

	return nil
}

func (m *StringMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Values)
}

func (m *StringMap) UnmarshalJSON(data []byte) error {
	values := make(map[string]string)
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	m.Values = values
	return nil
}

func (m *StringMap) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.Values = make(map[string]string)
	m.SourceMeta = make(map[string]*source.Meta)

	for key, valNode := range nodeMap {
		m.SourceMeta[key] = source.ExtractSourcePositionFromJSONNode(
			&valNode,
			linePositions,
		)

		str := valNode.Value.(string)
		m.Values[key] = str
	}

	return nil
}
