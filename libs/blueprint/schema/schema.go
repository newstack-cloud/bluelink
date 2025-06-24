package schema

import (
	json "github.com/coreos/go-json"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"gopkg.in/yaml.v3"
)

// Blueprint provides the type for a blueprint
// specification loaded into memory.
type Blueprint struct {
	Version     *core.ScalarValue      `yaml:"version" json:"version"`
	Transform   *TransformValueWrapper `yaml:"transform,omitempty" json:"transform,omitempty"`
	Variables   *VariableMap           `yaml:"variables,omitempty" json:"variables,omitempty"`
	Values      *ValueMap              `yaml:"values,omitempty" json:"values,omitempty"`
	Include     *IncludeMap            `yaml:"include,omitempty" json:"include,omitempty"`
	Resources   *ResourceMap           `yaml:"resources" json:"resources"`
	DataSources *DataSourceMap         `yaml:"datasources,omitempty" json:"datasources,omitempty"`
	Exports     *ExportMap             `yaml:"exports,omitempty" json:"exports,omitempty"`
	Metadata    *core.MappingNode      `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// VariableMap provides a mapping of names to variable values
// in a blueprint.
// This includes extra information about the locations of
// the keys in the original source being unmarshalled.
// This information will not always be present, it is populated
// when unmarshalling from YAML and JWCC source documents.
type VariableMap struct {
	Values map[string]*Variable
	// Mapping of variable names to their source locations.
	SourceMeta map[string]*source.Meta
}

func (m *VariableMap) MarshalYAML() (any, error) {
	return m.Values, nil
}

func (m *VariableMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errInvalidMap(core.YAMLNodeToPosInfo(value), "variables")
	}

	m.Values = make(map[string]*Variable)
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

		var variable Variable
		err := val.Decode(&variable)
		if err != nil {
			return err
		}

		m.Values[key.Value] = &variable
	}

	return nil
}

func (m *VariableMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Values)
}

func (m *VariableMap) UnmarshalJSON(data []byte) error {
	values := make(map[string]*Variable)
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	m.Values = values
	return nil
}

func (m *VariableMap) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	variableNodes, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.Values = map[string]*Variable{}
	m.SourceMeta = map[string]*source.Meta{}
	for key, variableNode := range variableNodes {
		m.SourceMeta[key] = source.ExtractSourcePositionFromJSONNode(
			&variableNode,
			linePositions,
		)
		variable := &Variable{}
		varPath := core.CreateJSONNodePath(key, parentPath, false /* parentIsRoot */)
		err := variable.FromJSONNode(&variableNode, linePositions, varPath)
		if err != nil {
			return err
		}
		m.Values[key] = variable
	}

	return nil
}

// ValueMap provides a mapping of names to value definitions
// in a blueprint.
// This includes extra information about the locations of
// the keys in the original source being unmarshalled.
// This information will not always be present, it is populated
// when unmarshalling from YAML and JWCC source documents.
type ValueMap struct {
	Values map[string]*Value
	// Mapping of value names to their source locations.
	SourceMeta map[string]*source.Meta
}

func (m *ValueMap) MarshalYAML() (any, error) {
	return m.Values, nil
}

func (m *ValueMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errInvalidMap(core.YAMLNodeToPosInfo(value), "values")
	}

	m.Values = make(map[string]*Value)
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

		var valDef Value
		err := val.Decode(&valDef)
		if err != nil {
			return err
		}

		m.Values[key.Value] = &valDef
	}

	return nil
}

func (m *ValueMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Values)
}

func (m *ValueMap) UnmarshalJSON(data []byte) error {
	values := make(map[string]*Value)
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	m.Values = values
	return nil
}

func (m *ValueMap) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	valueNodes, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.Values = map[string]*Value{}
	m.SourceMeta = map[string]*source.Meta{}
	for key, valueNode := range valueNodes {
		m.SourceMeta[key] = source.ExtractSourcePositionFromJSONNode(
			&valueNode,
			linePositions,
		)
		valueDef := &Value{}
		valPath := core.CreateJSONNodePath(key, parentPath, false /* parentIsRoot */)
		err := valueDef.FromJSONNode(&valueNode, linePositions, valPath)
		if err != nil {
			return err
		}
		m.Values[key] = valueDef
	}

	return nil
}

// IncludeMap provides a mapping of names to child
// blueprint includes.
// This includes extra information about the locations of
// the keys in the original source being unmarshalled.
// This information will not always be present, it is populated
// when unmarshalling from YAML and JWCC source documents.
type IncludeMap struct {
	Values map[string]*Include
	// Mapping of include names to their source locations.
	SourceMeta map[string]*source.Meta
}

func (m *IncludeMap) MarshalYAML() (any, error) {
	return m.Values, nil
}

func (m *IncludeMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errInvalidMap(core.YAMLNodeToPosInfo(value), "include")
	}

	m.Values = make(map[string]*Include)
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

		var include Include
		err := val.Decode(&include)
		if err != nil {
			return err
		}

		m.Values[key.Value] = &include
	}

	return nil
}

func (m *IncludeMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Values)
}

func (m *IncludeMap) UnmarshalJSON(data []byte) error {
	values := make(map[string]*Include)
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	m.Values = values
	return nil
}

func (m *IncludeMap) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	includeNodes, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.Values = map[string]*Include{}
	m.SourceMeta = map[string]*source.Meta{}
	for key, includeNode := range includeNodes {
		m.SourceMeta[key] = source.ExtractSourcePositionFromJSONNode(
			&includeNode,
			linePositions,
		)
		include := &Include{}
		includePath := core.CreateJSONNodePath(key, parentPath, false /* parentIsRoot */)
		err := include.FromJSONNode(&includeNode, linePositions, includePath)
		if err != nil {
			return err
		}
		m.Values[key] = include
	}

	return nil
}

// ResourceMap provides a mapping of names to resources.
// This includes extra information about the locations of
// the keys in the original source being unmarshalled.
// This information will not always be present, it is populated
// when unmarshalling from YAML and JWCC source documents.
type ResourceMap struct {
	Values map[string]*Resource
	// Mapping of resource names to their source locations.
	SourceMeta map[string]*source.Meta
}

func (m *ResourceMap) MarshalYAML() (any, error) {
	return m.Values, nil
}

func (m *ResourceMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errInvalidMap(core.YAMLNodeToPosInfo(value), "resources")
	}

	m.Values = make(map[string]*Resource)
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

		var resource Resource
		err := val.Decode(&resource)
		if err != nil {
			return err
		}

		m.Values[key.Value] = &resource
	}

	return nil
}

func (m *ResourceMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Values)
}

func (m *ResourceMap) UnmarshalJSON(data []byte) error {
	values := make(map[string]*Resource)
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	m.Values = values
	return nil
}

func (m *ResourceMap) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	resourceNodes, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.Values = map[string]*Resource{}
	m.SourceMeta = map[string]*source.Meta{}
	for key, resourceNode := range resourceNodes {
		m.SourceMeta[key] = source.ExtractSourcePositionFromJSONNode(
			&resourceNode,
			linePositions,
		)
		resource := &Resource{}
		resourcePath := core.CreateJSONNodePath(key, parentPath, false /* parentIsRoot */)
		err := resource.FromJSONNode(&resourceNode, linePositions, resourcePath)
		if err != nil {
			return err
		}
		m.Values[key] = resource
	}

	return nil
}

// DataSourceMap provides a mapping of names to data sources.
// This includes extra information about the locations of
// the keys in the original source being unmarshalled.
// This information will not always be present, it is populated
// when unmarshalling from YAML and JWCC source documents.
type DataSourceMap struct {
	Values map[string]*DataSource
	// Mapping of data source names to their source locations.
	SourceMeta map[string]*source.Meta
}

func (m *DataSourceMap) MarshalYAML() (any, error) {
	return m.Values, nil
}

func (m *DataSourceMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errInvalidMap(core.YAMLNodeToPosInfo(value), "datasources")
	}

	m.Values = make(map[string]*DataSource)
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

		var dataSource DataSource
		err := val.Decode(&dataSource)
		if err != nil {
			return err
		}

		m.Values[key.Value] = &dataSource
	}

	return nil
}

func (m *DataSourceMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Values)
}

func (m *DataSourceMap) UnmarshalJSON(data []byte) error {
	values := make(map[string]*DataSource)
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	m.Values = values
	return nil
}

func (m *DataSourceMap) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	dataSourceNodes, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.Values = map[string]*DataSource{}
	m.SourceMeta = map[string]*source.Meta{}
	for key, dataSourceNode := range dataSourceNodes {
		m.SourceMeta[key] = source.ExtractSourcePositionFromJSONNode(
			&dataSourceNode,
			linePositions,
		)
		dataSource := &DataSource{}
		dataSourcePath := core.CreateJSONNodePath(key, parentPath, false /* parentIsRoot */)
		err := dataSource.FromJSONNode(&dataSourceNode, linePositions, dataSourcePath)
		if err != nil {
			return err
		}
		m.Values[key] = dataSource
	}

	return nil
}

// ExportMap provides a mapping of names to exports.
// This includes extra information about the locations of
// the keys in the original source being unmarshalled.
// This information will not always be present, it is populated
// when unmarshalling from YAML and JWCC source documents.
type ExportMap struct {
	Values map[string]*Export
	// Mapping of export names to their source locations.
	SourceMeta map[string]*source.Meta
}

func (m *ExportMap) MarshalYAML() (any, error) {
	return m.Values, nil
}

func (m *ExportMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errInvalidMap(core.YAMLNodeToPosInfo(value), "exports")
	}

	m.Values = make(map[string]*Export)
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

		var export Export
		err := val.Decode(&export)
		if err != nil {
			return err
		}

		m.Values[key.Value] = &export
	}

	return nil
}

func (m *ExportMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Values)
}

func (m *ExportMap) UnmarshalJSON(data []byte) error {
	values := make(map[string]*Export)
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	m.Values = values
	return nil
}

func (m *ExportMap) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	exportNodes, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.Values = map[string]*Export{}
	m.SourceMeta = map[string]*source.Meta{}
	for key, exportNode := range exportNodes {
		m.SourceMeta[key] = source.ExtractSourcePositionFromJSONNode(
			&exportNode,
			linePositions,
		)
		export := &Export{}
		exportPath := core.CreateJSONNodePath(key, parentPath, false /* parentIsRoot */)
		err := export.FromJSONNode(&exportNode, linePositions, exportPath)
		if err != nil {
			return err
		}
		m.Values[key] = export
	}

	return nil
}
