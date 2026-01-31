package schema

import (
	"fmt"

	json "github.com/coreos/go-json"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/jsonutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"gopkg.in/yaml.v3"
)

// Resource represents a blueprint
// resource in the specification.
type Resource struct {
	Type             *ResourceTypeWrapper                 `yaml:"type" json:"type"`
	Description      *substitutions.StringOrSubstitutions `yaml:"description,omitempty" json:"description,omitempty"`
	Metadata         *Metadata                            `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	DependsOn        *DependsOnList                       `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
	Condition        *Condition                           `yaml:"condition,omitempty" json:"condition,omitempty"`
	Each             *substitutions.StringOrSubstitutions `yaml:"each,omitempty" json:"each,omitempty"`
	LinkSelector     *LinkSelector                        `yaml:"linkSelector,omitempty" json:"linkSelector,omitempty"`
	Spec             *core.MappingNode                    `yaml:"spec" json:"spec"`
	SourceMeta       *source.Meta                         `yaml:"-" json:"-"`
	FieldsSourceMeta map[string]*source.Meta              `yaml:"-" json:"-"`
}

func (r *Resource) UnmarshalYAML(value *yaml.Node) error {
	r.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}
	r.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromYAMLNode(value)

	type resourceAlias Resource
	var alias resourceAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	r.Type = alias.Type
	r.Description = alias.Description
	r.Metadata = alias.Metadata
	r.DependsOn = alias.DependsOn
	r.Condition = alias.Condition
	r.Each = alias.Each
	r.LinkSelector = alias.LinkSelector
	r.Spec = alias.Spec

	return nil
}

func (r *Resource) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	r.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromJSONNodeMap(nodeMap, linePositions)

	r.Type = &ResourceTypeWrapper{}
	err := core.UnpackValueFromJSONMapNode(
		nodeMap,
		"type",
		r.Type,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	r.Description = &substitutions.StringOrSubstitutions{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"description",
		r.Description,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	r.Metadata = &Metadata{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"metadata",
		r.Metadata,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	r.DependsOn = &DependsOnList{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"dependsOn",
		r.DependsOn,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	r.Condition = &Condition{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"condition",
		r.Condition,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	r.Each = &substitutions.StringOrSubstitutions{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"each",
		r.Each,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	r.LinkSelector = &LinkSelector{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"linkSelector",
		r.LinkSelector,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	r.Spec = &core.MappingNode{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"spec",
		r.Spec,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	r.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	return nil
}

// DependsOnList provides a list of resource names
// that a resource depends on.
// This can include extra information about the locations of
// elements in the list in the original source,
// depending on the source format.
type DependsOnList struct {
	StringList
}

func (t *DependsOnList) MarshalYAML() (any, error) {
	return t.StringList.MarshalYAML()
}

func (t *DependsOnList) UnmarshalYAML(value *yaml.Node) error {
	return t.StringList.unmarshalYAML(value, errInvalidDependencyType, "dependency")
}

func (t *DependsOnList) MarshalJSON() ([]byte, error) {
	return t.StringList.MarshalJSON()
}

func (t *DependsOnList) UnmarshalJSON(data []byte) error {
	return t.unmarshalJSON(data, errInvalidDependencyType, "dependency")
}

func (t *DependsOnList) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	return t.StringList.FromJSONNode(node, linePositions, parentPath)
}

// ResourceTypeWrapper provides a struct that holds a resource type
// value.
type ResourceTypeWrapper struct {
	Value      string
	SourceMeta *source.Meta
}

func (t *ResourceTypeWrapper) MarshalYAML() (any, error) {
	return t.Value, nil
}

func (t *ResourceTypeWrapper) UnmarshalYAML(value *yaml.Node) error {
	t.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
		EndPosition: source.EndSourcePositionFromYAMLScalarNode(value),
	}

	t.Value = value.Value
	return nil
}

func (t *ResourceTypeWrapper) MarshalJSON() ([]byte, error) {
	escaped := jsonutils.EscapeJSONString(string(t.Value))
	return []byte(fmt.Sprintf("\"%s\"", escaped)), nil
}

func (t *ResourceTypeWrapper) UnmarshalJSON(data []byte) error {
	var typeVal string
	err := json.Unmarshal(data, &typeVal)
	if err != nil {
		return err
	}

	t.Value = typeVal

	return nil
}

func (t *ResourceTypeWrapper) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	t.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)
	stringVal := node.Value.(string)
	t.Value = stringVal
	return nil
}

// Metadata represents the metadata associated
// with a blueprint resource that can be used to provide labels
// and annotations that can be used to configure
// instances and used for link selections.
type Metadata struct {
	DisplayName      *substitutions.StringOrSubstitutions `yaml:"displayName" json:"displayName"`
	Annotations      *StringOrSubstitutionsMap            `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Labels           *StringMap                           `yaml:"labels,omitempty" json:"labels,omitempty"`
	Custom           *core.MappingNode                    `yaml:"custom,omitempty" json:"custom,omitempty"`
	SourceMeta       *source.Meta                         `yaml:"-" json:"-"`
	FieldsSourceMeta map[string]*source.Meta              `yaml:"-" json:"-"`
}

func (m *Metadata) UnmarshalYAML(value *yaml.Node) error {
	m.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}
	m.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromYAMLNode(value)

	type metadataAlias Metadata
	var alias metadataAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	m.DisplayName = alias.DisplayName
	m.Annotations = alias.Annotations
	m.Labels = alias.Labels
	m.Custom = alias.Custom

	return nil
}

func (m *Metadata) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromJSONNodeMap(nodeMap, linePositions)

	m.DisplayName = &substitutions.StringOrSubstitutions{}
	err := core.UnpackValueFromJSONMapNode(
		nodeMap,
		"displayName",
		m.DisplayName,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	m.Annotations = &StringOrSubstitutionsMap{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"annotations",
		m.Annotations,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	m.Labels = &StringMap{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"labels",
		m.Labels,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	m.Custom = &core.MappingNode{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"custom",
		m.Custom,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	m.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	return nil
}

// LinkSelector allows a resource to select other resources
// to link to by label.
type LinkSelector struct {
	ByLabel          *StringMap              `yaml:"byLabel" json:"byLabel"`
	Exclude          *StringList             `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	SourceMeta       *source.Meta            `yaml:"-" json:"-"`
	FieldsSourceMeta map[string]*source.Meta `yaml:"-" json:"-"`
}

func (s *LinkSelector) UnmarshalYAML(value *yaml.Node) error {
	s.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}
	s.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromYAMLNode(value)

	type linkSelectorAlias LinkSelector
	var alias linkSelectorAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	s.ByLabel = alias.ByLabel
	s.Exclude = alias.Exclude

	return nil
}

func (s *LinkSelector) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	s.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromJSONNodeMap(nodeMap, linePositions)

	s.ByLabel = &StringMap{}
	err := core.UnpackValueFromJSONMapNode(
		nodeMap,
		"byLabel",
		s.ByLabel,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	s.Exclude = &StringList{}
	err = core.UnpackValueFromJSONMapNode(
		nodeMap,
		"exclude",
		s.Exclude,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	s.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	return nil
}
