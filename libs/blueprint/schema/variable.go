package schema

import (
	"fmt"

	json "github.com/coreos/go-json"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/jsonutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"gopkg.in/yaml.v3"
)

// Variable provides the definition of a variable
// that can be used in a blueprint.
type Variable struct {
	Type          *VariableTypeWrapper  `yaml:"type" json:"type"`
	Description   *bpcore.ScalarValue   `yaml:"description,omitempty" json:"description,omitempty"`
	Secret        *bpcore.ScalarValue   `yaml:"secret" json:"secret"`
	Default       *bpcore.ScalarValue   `yaml:"default,omitempty" json:"default,omitempty"`
	AllowedValues []*bpcore.ScalarValue `yaml:"allowedValues,omitempty" json:"allowedValues,omitempty"`
	SourceMeta    *source.Meta          `yaml:"-" json:"-"`
}

func (v *Variable) UnmarshalYAML(value *yaml.Node) error {
	v.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}

	type variableAlias Variable
	var alias variableAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	v.Type = alias.Type
	v.Description = alias.Description
	v.Secret = alias.Secret
	v.Default = alias.Default
	v.AllowedValues = alias.AllowedValues

	return nil
}

func (v *Variable) FromJSONNode(node *json.Node, linePositions []int, parentPath string) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	v.Type = &VariableTypeWrapper{}
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

	v.Description = &bpcore.ScalarValue{}
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

	v.Default = &bpcore.ScalarValue{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"default",
		v.Default,
		linePositions,
		/* parentPath */ parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	v.AllowedValues = []*bpcore.ScalarValue{}
	err = bpcore.UnpackValuesFromJSONMapNode(
		nodeMap,
		"allowedValues",
		&v.AllowedValues,
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

// VariableTypeWrapper provides a struct that holds a variable type
// value.
type VariableTypeWrapper struct {
	Value      VariableType
	SourceMeta *source.Meta
}

func (t *VariableTypeWrapper) MarshalYAML() (interface{}, error) {
	return t.Value, nil
}

func (t *VariableTypeWrapper) UnmarshalYAML(value *yaml.Node) error {
	t.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
		EndPosition: source.EndSourcePositionFromYAMLScalarNode(value),
	}

	t.Value = VariableType(value.Value)
	return nil
}

func (t *VariableTypeWrapper) MarshalJSON() ([]byte, error) {
	escaped := jsonutils.EscapeJSONString(string(t.Value))
	return []byte(fmt.Sprintf("\"%s\"", escaped)), nil
}

func (t *VariableTypeWrapper) UnmarshalJSON(data []byte) error {
	var typeVal string
	err := json.Unmarshal(data, &typeVal)
	if err != nil {
		return err
	}

	t.Value = VariableType(typeVal)

	return nil
}

func (t *VariableTypeWrapper) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	t.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)
	stringVal := node.Value.(string)
	t.Value = VariableType(stringVal)
	return nil
}

// VariableType represents a type of variable
// defined in a blueprint.
// Can be one of "string", "integer", "float" or "boolean" or a custom type
// defined by a resource provider.
type VariableType string

func (t VariableType) Equal(compareWith VariableType) bool {
	return t == compareWith
}

const (
	// VariableTypeString is for a string variable
	// in a blueprint.
	VariableTypeString VariableType = "string"
	// VariableTypeInteger is for an integer value
	// in a blueprint.
	VariableTypeInteger VariableType = "integer"
	// VariableTypeFloat is for a float value
	// in a blueprint.
	VariableTypeFloat VariableType = "float"
	// VariableTypeBoolean is for a boolean value
	// in a blueprint.
	VariableTypeBoolean VariableType = "boolean"
)

var (
	// CoreVariableTypes provides a slice of all the core supported
	// variable types to be used for clean validation of fields
	// with a field with VariableType.
	// This does not represent all possible variable types,
	// as provider custom variable types are also supported.
	CoreVariableTypes = []VariableType{
		VariableTypeString,
		VariableTypeInteger,
		VariableTypeFloat,
		VariableTypeBoolean,
	}
)
