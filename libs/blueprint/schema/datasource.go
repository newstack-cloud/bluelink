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

// DataSource represents a blueprint
// data source in the specification.
// Data sources are accessible to all resources in a blueprint via the
// ${datasources.{dataSourceName}.{exportedField}} reference.
// For example, you would access a data source called network with an exported
// vpc field via ${datasources.network.vpc}.
type DataSource struct {
	Type               *DataSourceTypeWrapper               `yaml:"type" json:"type"`
	DataSourceMetadata *DataSourceMetadata                  `yaml:"metadata" json:"metadata"`
	Filter             *DataSourceFilters                   `yaml:"filter" json:"filter"`
	Exports            *DataSourceFieldExportMap            `yaml:"exports" json:"exports"`
	Description        *substitutions.StringOrSubstitutions `yaml:"description,omitempty" json:"description,omitempty"`
	SourceMeta         *source.Meta                         `yaml:"-" json:"-"`
	FieldsSourceMeta   map[string]*source.Meta              `yaml:"-" json:"-"`
}

func (s *DataSource) UnmarshalYAML(value *yaml.Node) error {
	s.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}
	s.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromYAMLNode(value)

	type dataSourceAlias DataSource
	var alias dataSourceAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	s.Type = alias.Type
	s.DataSourceMetadata = alias.DataSourceMetadata
	s.Filter = alias.Filter
	s.Exports = alias.Exports
	s.Description = alias.Description

	return nil
}

func (s *DataSource) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	s.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	s.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromJSONNodeMap(nodeMap, linePositions)

	s.Type = &DataSourceTypeWrapper{}
	err := bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"type",
		s.Type,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
		bpcore.WithParentNode(node),
	)
	if err != nil {
		return err
	}

	s.DataSourceMetadata = &DataSourceMetadata{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"metadata",
		s.DataSourceMetadata,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
		bpcore.WithParentNode(node),
	)
	if err != nil {
		return err
	}

	s.Filter = &DataSourceFilters{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"filter",
		s.Filter,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	s.Exports = &DataSourceFieldExportMap{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"exports",
		s.Exports,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
		bpcore.WithParentNode(node),
	)
	if err != nil {
		return err
	}

	s.Description = &substitutions.StringOrSubstitutions{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"description",
		s.Description,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	return nil
}

// DataSourceTypeWrapper provides a struct that holds a data source type
// value.
type DataSourceTypeWrapper struct {
	Value      string
	SourceMeta *source.Meta
}

func (t *DataSourceTypeWrapper) MarshalYAML() (interface{}, error) {
	return t.Value, nil
}

func (t *DataSourceTypeWrapper) UnmarshalYAML(value *yaml.Node) error {
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

func (t *DataSourceTypeWrapper) MarshalJSON() ([]byte, error) {
	escaped := jsonutils.EscapeJSONString(string(t.Value))
	return []byte(fmt.Sprintf("\"%s\"", escaped)), nil
}

func (t *DataSourceTypeWrapper) UnmarshalJSON(data []byte) error {
	var typeVal string
	err := json.Unmarshal(data, &typeVal)
	if err != nil {
		return err
	}

	t.Value = typeVal

	return nil
}

func (t *DataSourceTypeWrapper) FromJSONNode(
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

// DataSourceFieldExportMap provides a mapping of names to
// data source field exports.
// This includes extra information about the locations of
// the keys in the original source being unmarshalled.
// This information will not always be present, it is populated
// when unmarshalling from YAML and JWCC source documents.
type DataSourceFieldExportMap struct {
	Values map[string]*DataSourceFieldExport
	// Indicates if all fields should be exported,
	// this is set to true if the `export` field is set to `*`
	// in the blueprint.
	ExportAll bool
	// Mapping of exported field names to their source locations.
	SourceMeta map[string]*source.Meta
}

func (m *DataSourceFieldExportMap) MarshalYAML() (any, error) {
	if m.ExportAll {
		// If export all is set, we return a single value
		// that indicates that all fields should be exported.
		return "*", nil
	}
	return m.Values, nil
}

func (m *DataSourceFieldExportMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Value == "*" {
		m.ExportAll = true
	}

	if value.Kind != yaml.MappingNode && !m.ExportAll {
		return errInvalidMap(bpcore.YAMLNodeToPosInfo(value), "exports")
	}

	m.Values = make(map[string]*DataSourceFieldExport)
	m.SourceMeta = make(map[string]*source.Meta)
	if !m.ExportAll {
		// Only collect values if we are not exporting all fields.
		// location-based information will be scoped to the data source
		// element definition for errors with a string exports value.
		for i := 0; i < len(value.Content); i += 2 {
			key := value.Content[i]
			val := value.Content[i+1]

			m.SourceMeta[key.Value] = &source.Meta{
				Position: source.Position{
					Line:   key.Line,
					Column: key.Column,
				},
			}

			var export DataSourceFieldExport
			err := val.Decode(&export)
			if err != nil {
				return err
			}

			m.Values[key.Value] = &export
		}
	}

	return nil
}

func (m *DataSourceFieldExportMap) MarshalJSON() ([]byte, error) {
	if m.ExportAll {
		return []byte("\"*\""), nil
	}

	return json.Marshal(m.Values)
}

func (m *DataSourceFieldExportMap) UnmarshalJSON(data []byte) error {
	if string(data) == "\"*\"" {
		m.ExportAll = true
		m.Values = make(map[string]*DataSourceFieldExport)
		return nil
	}

	values := make(map[string]*DataSourceFieldExport)
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	m.Values = values
	return nil
}

func (m *DataSourceFieldExportMap) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		nodeStr, ok := node.Value.(string)
		if ok && nodeStr == "*" {
			m.ExportAll = true
			m.Values = make(map[string]*DataSourceFieldExport)
			m.SourceMeta = make(map[string]*source.Meta)
			return nil
		}
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	m.Values = make(map[string]*DataSourceFieldExport)
	m.SourceMeta = make(map[string]*source.Meta)
	for key, node := range nodeMap {
		export := &DataSourceFieldExport{}
		fieldPath := bpcore.CreateJSONNodePath(key, parentPath, false)
		err := export.FromJSONNode(&node, linePositions, fieldPath)
		if err != nil {
			return err
		}
		m.Values[key] = export
		m.SourceMeta[key] = source.ExtractSourcePositionFromJSONNode(
			&node,
			linePositions,
		)
	}

	return nil
}

// DataSourceFilters provides a slice of one or more filters
// parsed from a mapping for a single filter or a sequence of filters.
type DataSourceFilters struct {
	Filters []*DataSourceFilter `yaml:"filters" json:"filters"`
}

func (s *DataSourceFilters) MarshalYAML() (interface{}, error) {
	// Export as a single value if there is only one value,
	// in a similar way to how the user would define a single filter.
	if len(s.Filters) == 1 {
		return s.Filters[0], nil
	}

	return s.Filters, nil
}

func (s *DataSourceFilters) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.SequenceNode {
		filters := []*DataSourceFilter{}
		for _, node := range value.Content {
			filter := &DataSourceFilter{}
			err := filter.UnmarshalYAML(node)
			if err != nil {
				return wrapErrorWithLineInfo(err, node)
			}
			filters = append(filters, filter)
		}
		s.Filters = filters
	}

	singleFilter := &DataSourceFilter{}
	err := singleFilter.UnmarshalYAML(value)
	if err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	s.Filters = []*DataSourceFilter{singleFilter}
	return nil
}

func (s *DataSourceFilters) MarshalJSON() ([]byte, error) {
	if len(s.Filters) == 1 {
		return json.Marshal(s.Filters[0])
	}

	return json.Marshal(s.Filters)
}

func (s *DataSourceFilters) UnmarshalJSON(data []byte) error {
	singleFilter := DataSourceFilter{}
	err := json.Unmarshal(data, &singleFilter)
	if err == nil {
		s.Filters = []*DataSourceFilter{&singleFilter}
		return nil
	}

	multipleFilters := []*DataSourceFilter{}
	err = json.Unmarshal(data, &multipleFilters)
	if err != nil {
		return err
	}

	s.Filters = multipleFilters
	return nil
}

func (f *DataSourceFilters) FromJSONNode(node *json.Node, linePositions []int, parentPath string) error {
	nodeList, err := f.deriveJSONNodeList(node, linePositions, parentPath)
	if err != nil {
		return err
	}

	for i, node := range nodeList {
		filter := &DataSourceFilter{}
		key := fmt.Sprintf("%d", i)
		itemPath := bpcore.CreateJSONNodePath(key, parentPath, false)
		err := filter.FromJSONNode(&node, linePositions, itemPath)
		if err != nil {
			return err
		}
		f.Filters = append(f.Filters, filter)
	}

	return nil
}

func (f *DataSourceFilters) deriveJSONNodeList(
	node *json.Node,
	linePositions []int,
	parentPath string,
) ([]json.Node, error) {
	// Filter can take a single filter definition or an array of filter
	// definitions.
	_, ok := node.Value.(map[string]json.Node)
	if ok {
		return []json.Node{*node}, nil
	}

	nodeList, ok := node.Value.([]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return nil, errInvalidArrayOrString(&position, parentPath)
	}

	return nodeList, nil
}

// DataSourceFilter provides the definition of a filter
// used to select a specific data source instance from a provider.
type DataSourceFilter struct {
	Field      *bpcore.ScalarValue              `yaml:"field" json:"field"`
	Operator   *DataSourceFilterOperatorWrapper `yaml:"operator" json:"operator"`
	Search     *DataSourceFilterSearch          `yaml:"search" json:"search"`
	SourceMeta *source.Meta                     `yaml:"-" json:"-"`
}

func (f *DataSourceFilter) UnmarshalYAML(value *yaml.Node) error {
	f.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}

	type dataSourceFilterAlias DataSourceFilter
	var alias dataSourceFilterAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	f.Field = alias.Field
	f.Operator = alias.Operator
	f.Search = alias.Search

	return nil
}

func (f *DataSourceFilter) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	f.Field = &bpcore.ScalarValue{}
	err := bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"field",
		f.Field,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	f.Operator = &DataSourceFilterOperatorWrapper{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"operator",
		f.Operator,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	f.Search = &DataSourceFilterSearch{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"search",
		f.Search,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	f.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	return nil
}

// DataSourceFilterSearch provides the definition of one or more
// search values for a data source filter.
type DataSourceFilterSearch struct {
	Values     []*substitutions.StringOrSubstitutions
	SourceMeta *source.Meta
}

func (s *DataSourceFilterSearch) MarshalYAML() (interface{}, error) {
	// Export as a single value if there is only one value,
	// in a similar way to how the user would define a single search value
	// for a filter.
	if len(s.Values) == 1 {
		return s.Values[0], nil
	}

	return s.Values, nil
}

func (s *DataSourceFilterSearch) UnmarshalYAML(value *yaml.Node) error {
	s.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}

	if value.Kind == yaml.SequenceNode {
		values := []*substitutions.StringOrSubstitutions{}
		for _, node := range value.Content {
			value := &substitutions.StringOrSubstitutions{}
			err := value.UnmarshalYAML(node)
			if err != nil {
				return wrapErrorWithLineInfo(err, node)
			}
			values = append(values, value)
		}
		s.Values = values
	}

	singleValue := &substitutions.StringOrSubstitutions{}
	err := singleValue.UnmarshalYAML(value)
	if err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	s.Values = []*substitutions.StringOrSubstitutions{singleValue}
	return nil
}

func (s *DataSourceFilterSearch) MarshalJSON() ([]byte, error) {
	if len(s.Values) == 1 {
		return json.Marshal(s.Values[0])
	}

	return json.Marshal(s.Values)
}

func (s *DataSourceFilterSearch) UnmarshalJSON(data []byte) error {
	singleValue := substitutions.StringOrSubstitutions{}
	err := json.Unmarshal(data, &singleValue)
	if err == nil {
		s.Values = []*substitutions.StringOrSubstitutions{&singleValue}
		return nil
	}

	multipleValues := []*substitutions.StringOrSubstitutions{}
	err = json.Unmarshal(data, &multipleValues)
	if err != nil {
		return err
	}

	s.Values = multipleValues
	return nil
}

func (s *DataSourceFilterSearch) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeList, err := s.deriveJSONNodeList(node, linePositions, parentPath)
	if err != nil {
		return err
	}

	s.Values = make([]*substitutions.StringOrSubstitutions, len(nodeList))
	for i, node := range nodeList {
		stringOrSubs := &substitutions.StringOrSubstitutions{}
		key := fmt.Sprintf("%d", i)
		itemPath := bpcore.CreateJSONNodePath(key, parentPath, false)
		err := stringOrSubs.FromJSONNode(&node, linePositions, itemPath)
		if err != nil {
			return err
		}
		s.Values[i] = stringOrSubs
	}

	s.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	return nil
}

func (s *DataSourceFilterSearch) deriveJSONNodeList(
	node *json.Node,
	linePositions []int,
	parentPath string,
) ([]json.Node, error) {
	// Search can take a single string or an array of strings.
	_, ok := node.Value.(string)
	if ok {
		return []json.Node{*node}, nil
	}

	nodeList, ok := node.Value.([]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return nil, errInvalidArrayOrString(&position, parentPath)
	}

	return nodeList, nil
}

// DataSourceFilterOperatorWrapper provides a struct that holds a data source filter operator
// value.
// The reason that this exists is to allow more fine-grained control
// when serialising and deserialising data source filter operators in a blueprint
// so we can check precise values.
type DataSourceFilterOperatorWrapper struct {
	Value      DataSourceFilterOperator
	SourceMeta *source.Meta
}

func (w *DataSourceFilterOperatorWrapper) MarshalYAML() (interface{}, error) {
	return w.Value, nil
}

func (w *DataSourceFilterOperatorWrapper) UnmarshalYAML(value *yaml.Node) error {
	w.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
		EndPosition: source.EndSourcePositionFromYAMLScalarNode(value),
	}
	valueFilterOperator := DataSourceFilterOperator(value.Value)

	w.Value = valueFilterOperator
	return nil
}

func (w *DataSourceFilterOperatorWrapper) MarshalJSON() ([]byte, error) {
	escaped := jsonutils.EscapeJSONString(string(w.Value))
	return []byte(fmt.Sprintf("\"%s\"", escaped)), nil
}

func (w *DataSourceFilterOperatorWrapper) UnmarshalJSON(data []byte) error {
	var typeVal string
	err := json.Unmarshal(data, &typeVal)
	if err != nil {
		return err
	}

	typeValDataSourceFilterOperator := DataSourceFilterOperator(typeVal)
	w.Value = typeValDataSourceFilterOperator

	return nil
}

func (w *DataSourceFilterOperatorWrapper) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	w.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)
	stringVal := node.Value.(string)
	w.Value = DataSourceFilterOperator(stringVal)
	return nil
}

// DataSourceFilterOperator represents a filter operator
// for a data source defined in a blueprint.
// Can be one of "=", "!=", "in", "not in", "has key", "not has key",
// "contains", "not contains", "starts with", "not starts with", "ends with",
// "not ends with", ">", "<", ">=" or "<=".
type DataSourceFilterOperator string

func (t DataSourceFilterOperator) Equal(compareWith DataSourceFilterOperator) bool {
	return t == compareWith
}

const (
	// DataSourceFilterOperatorEquals represents the "=" filter operator.
	DataSourceFilterOperatorEquals DataSourceFilterOperator = "="
	// DataSourceFilterOperatorNotEquals represents the "!=" filter operator.
	DataSourceFilterOperatorNotEquals DataSourceFilterOperator = "!="
	// DataSourceFilterOperatorIn represents the "in" filter operator.
	DataSourceFilterOperatorIn DataSourceFilterOperator = "in"
	// DataSourceFilterOperatorNotIn represents the "not in" filter operator.
	DataSourceFilterOperatorNotIn DataSourceFilterOperator = "not in"
	// DataSourceFilterOperatorHasKey represents the "has key" filter operator.
	DataSourceFilterOperatorHasKey DataSourceFilterOperator = "has key"
	// DataSourceFilterOperatorNotHasKey represents the "not has key" filter operator.
	DataSourceFilterOperatorNotHasKey DataSourceFilterOperator = "not has key"
	// DataSourceFilterOperatorContains represents the "contains" filter operator.
	DataSourceFilterOperatorContains DataSourceFilterOperator = "contains"
	// DataSourceFilterOperatorNotContains represents the "not contains" filter operator.
	DataSourceFilterOperatorNotContains DataSourceFilterOperator = "not contains"
	// DataSourceFilterOperatorStartsWith represents the "starts with" filter operator.
	DataSourceFilterOperatorStartsWith DataSourceFilterOperator = "starts with"
	// DataSourceFilterOperatorNotStartsWith represents the "not starts with" filter operator.
	DataSourceFilterOperatorNotStartsWith DataSourceFilterOperator = "not starts with"
	// DataSourceFilterOperatorEndsWith represents the "ends with" filter operator.
	DataSourceFilterOperatorEndsWith DataSourceFilterOperator = "ends with"
	// DataSourceFilterOperatorNotEndsWith represents the "not ends with" filter operator.
	DataSourceFilterOperatorNotEndsWith DataSourceFilterOperator = "not ends with"
	// DataSourceFilterOperatorGreaterThan represents the ">" filter operator.
	DataSourceFilterOperatorGreaterThan DataSourceFilterOperator = ">"
	// DataSourceFilterOperatorLessThan represents the "<" filter operator.
	DataSourceFilterOperatorLessThan DataSourceFilterOperator = "<"
	// DataSourceFilterOperatorGreaterThanOrEqual represents the ">=" filter operator.
	DataSourceFilterOperatorGreaterThanOrEqual DataSourceFilterOperator = ">="
	// DataSourceFilterOperatorLessThanOrEqual represents the "<=" filter operator.
	DataSourceFilterOperatorLessThanOrEqual DataSourceFilterOperator = "<="
)

var (
	// DataSourceFilterOperators provides a slice of all the supported
	// data source filter operators.
	DataSourceFilterOperators = []DataSourceFilterOperator{
		DataSourceFilterOperatorEquals,
		DataSourceFilterOperatorNotEquals,
		DataSourceFilterOperatorIn,
		DataSourceFilterOperatorNotIn,
		DataSourceFilterOperatorHasKey,
		DataSourceFilterOperatorNotHasKey,
		DataSourceFilterOperatorContains,
		DataSourceFilterOperatorNotContains,
		DataSourceFilterOperatorStartsWith,
		DataSourceFilterOperatorNotStartsWith,
		DataSourceFilterOperatorEndsWith,
		DataSourceFilterOperatorNotEndsWith,
		DataSourceFilterOperatorGreaterThan,
		DataSourceFilterOperatorLessThan,
		DataSourceFilterOperatorGreaterThanOrEqual,
		DataSourceFilterOperatorLessThanOrEqual,
	}
)

// DataSourceFieldExport provides the definition of an exported field
// from a data source in a blueprint.
type DataSourceFieldExport struct {
	Type        *DataSourceFieldTypeWrapper          `yaml:"type" json:"type"`
	AliasFor    *bpcore.ScalarValue                  `yaml:"aliasFor" json:"aliasFor"`
	Description *substitutions.StringOrSubstitutions `yaml:"description,omitempty" json:"description,omitempty"`
	SourceMeta  *source.Meta                         `yaml:"-" json:"-"`
}

func (e *DataSourceFieldExport) UnmarshalYAML(value *yaml.Node) error {
	e.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}

	type dataSourceFieldExportAlias DataSourceFieldExport
	var alias dataSourceFieldExportAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	e.Type = alias.Type
	e.AliasFor = alias.AliasFor
	e.Description = alias.Description

	return nil
}

func (e *DataSourceFieldExport) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	nodeMap, ok := node.Value.(map[string]json.Node)
	if !ok {
		position := source.PositionFromJSONNode(node, linePositions)
		return errInvalidMap(&position, parentPath)
	}

	e.Type = &DataSourceFieldTypeWrapper{}
	err := bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"type",
		e.Type,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ true,
	)
	if err != nil {
		return err
	}

	e.AliasFor = &bpcore.ScalarValue{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"aliasFor",
		e.AliasFor,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	e.Description = &substitutions.StringOrSubstitutions{}
	err = bpcore.UnpackValueFromJSONMapNode(
		nodeMap,
		"description",
		e.Description,
		linePositions,
		parentPath,
		/* parentIsRoot */ false,
		/* required */ false,
	)
	if err != nil {
		return err
	}

	e.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)

	return nil
}

// DataSourceMetadata represents the metadata associated
// with a blueprint data source that can be used to provide
// annotations that are used to configure data sources when fetching data
// from the data source provider.
type DataSourceMetadata struct {
	DisplayName      *substitutions.StringOrSubstitutions `yaml:"displayName" json:"displayName"`
	Annotations      *StringOrSubstitutionsMap            `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Custom           *bpcore.MappingNode                  `yaml:"custom,omitempty" json:"custom,omitempty"`
	SourceMeta       *source.Meta                         `yaml:"-" json:"-"`
	FieldsSourceMeta map[string]*source.Meta              `yaml:"-" json:"-"`
}

func (m *DataSourceMetadata) UnmarshalYAML(value *yaml.Node) error {
	m.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
	}
	m.FieldsSourceMeta = source.ExtractFieldsSourceMetaFromYAMLNode(value)

	type dataSourceMetadataAlias DataSourceMetadata
	var alias dataSourceMetadataAlias
	if err := value.Decode(&alias); err != nil {
		return wrapErrorWithLineInfo(err, value)
	}

	m.DisplayName = alias.DisplayName
	m.Annotations = alias.Annotations
	m.Custom = alias.Custom

	return nil
}

func (m *DataSourceMetadata) FromJSONNode(
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
	err := bpcore.UnpackValueFromJSONMapNode(
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
	err = bpcore.UnpackValueFromJSONMapNode(
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

	m.Custom = &bpcore.MappingNode{}
	err = bpcore.UnpackValueFromJSONMapNode(
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

// DataSourceFieldTypeWrapper provides a struct that holds a data source field type
// value.
// The reason that this exists is to allow more fine-grained control
// when serialising and deserialising data source field exports in a blueprint
// so we can check precise values.
type DataSourceFieldTypeWrapper struct {
	Value      DataSourceFieldType
	SourceMeta *source.Meta
}

func (t *DataSourceFieldTypeWrapper) MarshalYAML() (interface{}, error) {
	return t.Value, nil
}

func (t *DataSourceFieldTypeWrapper) UnmarshalYAML(value *yaml.Node) error {
	t.SourceMeta = &source.Meta{
		Position: source.Position{
			Line:   value.Line,
			Column: value.Column,
		},
		EndPosition: source.EndSourcePositionFromYAMLScalarNode(value),
	}
	valueDataSourceFieldType := DataSourceFieldType(value.Value)

	t.Value = valueDataSourceFieldType
	return nil
}

func (t *DataSourceFieldTypeWrapper) MarshalJSON() ([]byte, error) {
	escaped := jsonutils.EscapeJSONString(string(t.Value))
	return []byte(fmt.Sprintf("\"%s\"", escaped)), nil
}

func (t *DataSourceFieldTypeWrapper) UnmarshalJSON(data []byte) error {
	var typeVal string
	err := json.Unmarshal(data, &typeVal)
	if err != nil {
		return err
	}

	t.Value = DataSourceFieldType(typeVal)

	return nil
}

func (t *DataSourceFieldTypeWrapper) FromJSONNode(
	node *json.Node,
	linePositions []int,
	parentPath string,
) error {
	t.SourceMeta = source.ExtractSourcePositionFromJSONNode(
		node,
		linePositions,
	)
	stringVal := node.Value.(string)
	t.Value = DataSourceFieldType(stringVal)
	return nil
}

// DataSourceFieldType represents a type of exported field
// for a data source defined in a blueprint.
// Can be one of "string", "integer", "float", "boolean", "array" or "object".
type DataSourceFieldType string

func (t DataSourceFieldType) Equal(compareWith DataSourceFieldType) bool {
	return t == compareWith
}

const (
	// DataSourceFieldTypeString is for an exported
	// string field from a data source in a blueprint.
	DataSourceFieldTypeString DataSourceFieldType = "string"
	// DataSourceFieldTypeInteger is for an exported
	// integer field from a data source in a blueprint.
	DataSourceFieldTypeInteger DataSourceFieldType = "integer"
	// DataSourceFieldTypeFloat is for an exported
	// float field from a data source in a blueprint.
	DataSourceFieldTypeFloat DataSourceFieldType = "float"
	// DataSourceFieldTypeBoolean is for an exported
	// boolean field from a data source in a blueprint.
	DataSourceFieldTypeBoolean DataSourceFieldType = "boolean"
	// DataSourceFieldTypeArray is for an exported
	// array field from a data source in a blueprint.
	DataSourceFieldTypeArray DataSourceFieldType = "array"
)

var (
	// DataSourceFieldTypes provides a slice of all the supported
	// data source field types to be used for clean validation of fields
	// with a field with DataSourceFieldType.
	DataSourceFieldTypes = []DataSourceFieldType{
		DataSourceFieldTypeString,
		DataSourceFieldTypeInteger,
		DataSourceFieldTypeFloat,
		DataSourceFieldTypeBoolean,
		DataSourceFieldTypeArray,
	}
)
