package schema

import (
	"fmt"

	json "github.com/coreos/go-json"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/jsonutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"gopkg.in/yaml.v3"
)

// Value provides the definition of a value
// that can be used in a blueprint.
type Value struct {
	Type        *ValueTypeWrapper                    `yaml:"type" json:"type"`
	Value       *bpcore.MappingNode                  `yaml:"value" json:"value"`
	Description *substitutions.StringOrSubstitutions `yaml:"description,omitempty" json:"description,omitempty"`
	Secret      *bpcore.ScalarValue                  `yaml:"secret" json:"secret"`
	SourceMeta  *source.Meta                         `yaml:"-" json:"-"`
}

func (t *Value) UnmarshalYAML(value *yaml.Node) error {
	t.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}

	type valueAlias Value
	var alias valueAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	t.Type = alias.Type
	t.Description = alias.Description
	t.Secret = alias.Secret
	t.Value = alias.Value

	return nil
}

func (v *Value) FromJSONNode(node *json.Node, linePositions []int, parentPath string) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	v.Type = &ValueTypeWrapper{}
	err := bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"type",
		v.Type,
		linePositions,
		/* parentPath */ parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	v.Value = &bpcore.MappingNode{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"value",
		v.Value,
		linePositions,
		/* parentPath */ parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	v.Description = &substitutions.StringOrSubstitutions{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"description",
		v.Description,
		linePositions,
		/* parentPath */ parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	v.Secret = &bpcore.ScalarValue{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"secret",
		v.Secret,
		linePositions,
		/* parentPath */ parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	v.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	return nil
}

// ValueTypeWrapper provides a struct that holds a value type.
// The reason that this exists is to allow more fine-grained control
// when serialising and deserialising values in a blueprint
// so we can check precise value types.
type ValueTypeWrapper struct {
	Value      ValueType
	SourceMeta *source.Meta
}

func (t *ValueTypeWrapper) MarshalYAML() (any, error) {
	return t.Value, nil
}

func (t *ValueTypeWrapper) UnmarshalYAML(value *yaml.Node) error {
	t.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
		EndPosition: source.EndSourcePositionFromYAMLScalarNode(value),
	}
	valueType := ValueType(value.Value)
	t.Value = valueType
	return nil
}

func (t *ValueTypeWrapper) MarshalJSON() ([]byte, error) {
	escaped := jsonutils.EscapeJSONString(string(t.Value))
	return []byte(fmt.Sprintf("\"%s\"", escaped)), nil
}

func (t *ValueTypeWrapper) UnmarshalJSON(data []byte) error {
	var typeVal string
	err := json.Unmarshal(data, &typeVal)
	if err != nil {
		return err
	}

	valueType := ValueType(typeVal)
	t.Value = valueType

	return nil
}

func (t *ValueTypeWrapper) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	t.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)
	stringVal := node.Value.(string)
	t.Value = ValueType(stringVal)
	return nil
}

// ValueType represents a type of value
// defined in a blueprint.
// Can be one of "string", "integer", "float", "boolean", "array" or "object".
type ValueType string

func (t ValueType) Equal(compareWith ValueType) bool {
	return t == compareWith
}

const (
	// ValueTypeString is for a string value
	// in a blueprint.
	ValueTypeString ValueType = "string"
	// ValueTypeInteger is for an integer value
	// in a blueprint.
	ValueTypeInteger ValueType = "integer"
	// ValueTypeFloat is for a float value
	// in a blueprint.
	ValueTypeFloat ValueType = "float"
	// ValueTypeBoolean is for a boolean value
	// in a blueprint.
	ValueTypeBoolean ValueType = "boolean"
	// ValueTypeArray is for an array value
	// in a blueprint.
	ValueTypeArray ValueType = "array"
	// ValueTypeObject is for an object value
	// in a blueprint.
	ValueTypeObject ValueType = "object"
)

var (
	// ValueTypes provides a slice of all the supported
	// value types to be used for validation of
	// local value types in a blueprint.
	ValueTypes = []ValueType{
		ValueTypeString,
		ValueTypeInteger,
		ValueTypeFloat,
		ValueTypeBoolean,
		ValueTypeArray,
		ValueTypeObject,
	}
)
