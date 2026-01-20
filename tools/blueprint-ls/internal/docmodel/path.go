package docmodel

import "strconv"

// PathSegmentKind identifies the type of path segment.
type PathSegmentKind int

const (
	PathSegmentField PathSegmentKind = iota
	PathSegmentIndex
)

// PathSegment represents a single segment in a node path.
type PathSegment struct {
	Kind      PathSegmentKind
	FieldName string // Set when Kind == PathSegmentField
	Index     int    // Set when Kind == PathSegmentIndex
}

// String returns a string representation of the segment.
func (s PathSegment) String() string {
	if s.Kind == PathSegmentIndex {
		return strconv.Itoa(s.Index)
	}
	return s.FieldName
}

// StructuredPath wraps a slice of path segments with helper methods
// for type-safe path matching without string parsing.
type StructuredPath []PathSegment

// Len returns the number of segments in the path.
func (p StructuredPath) Len() int {
	return len(p)
}

// At returns the segment at the given index, or an empty segment if out of bounds.
func (p StructuredPath) At(index int) PathSegment {
	if index < 0 || index >= len(p) {
		return PathSegment{}
	}
	return p[index]
}

// Last returns the last segment in the path, or an empty segment if empty.
func (p StructuredPath) Last() PathSegment {
	if len(p) == 0 {
		return PathSegment{}
	}
	return p[len(p)-1]
}

// IsInResources returns true if the path is under /resources.
func (p StructuredPath) IsInResources() bool {
	return len(p) >= 1 && p[0].Kind == PathSegmentField && p[0].FieldName == "resources"
}

// IsInDataSources returns true if the path is under /datasources.
func (p StructuredPath) IsInDataSources() bool {
	return len(p) >= 1 && p[0].Kind == PathSegmentField && p[0].FieldName == "datasources"
}

// IsInVariables returns true if the path is under /variables.
func (p StructuredPath) IsInVariables() bool {
	return len(p) >= 1 && p[0].Kind == PathSegmentField && p[0].FieldName == "variables"
}

// IsInValues returns true if the path is under /values.
func (p StructuredPath) IsInValues() bool {
	return len(p) >= 1 && p[0].Kind == PathSegmentField && p[0].FieldName == "values"
}

// IsInExports returns true if the path is under /exports.
func (p StructuredPath) IsInExports() bool {
	return len(p) >= 1 && p[0].Kind == PathSegmentField && p[0].FieldName == "exports"
}

// IsInIncludes returns true if the path is under /include.
func (p StructuredPath) IsInIncludes() bool {
	return len(p) >= 1 && p[0].Kind == PathSegmentField && p[0].FieldName == "include"
}

// IsResourceType returns true if path points to a resource type field.
// Pattern: /resources/{name}/type
func (p StructuredPath) IsResourceType() bool {
	return len(p) == 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "resources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "type"
}

// IsDataSourceType returns true if path points to a data source type field.
// Pattern: /datasources/{name}/type
func (p StructuredPath) IsDataSourceType() bool {
	return len(p) == 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "type"
}

// IsVariableType returns true if path points to a variable type field.
// Pattern: /variables/{name}/type
func (p StructuredPath) IsVariableType() bool {
	return len(p) == 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "variables" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "type"
}

// IsValueType returns true if path points to a value type field.
// Pattern: /values/{name}/type
func (p StructuredPath) IsValueType() bool {
	return len(p) == 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "values" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "type"
}

// IsExportType returns true if path points to an export type field.
// Pattern: /exports/{name}/type
func (p StructuredPath) IsExportType() bool {
	return len(p) == 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "exports" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "type"
}

// IsResourceSpec returns true if path points to a resource spec field.
// Pattern: /resources/{name}/spec/...
func (p StructuredPath) IsResourceSpec() bool {
	return len(p) >= 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "resources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "spec"
}

// IsResourceMetadata returns true if path points to a resource metadata field.
// Pattern: /resources/{name}/metadata/...
func (p StructuredPath) IsResourceMetadata() bool {
	return len(p) >= 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "resources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "metadata"
}

// IsDataSourceFilter returns true if path is in a data source filter.
// Pattern: /datasources/{name}/filter/...
func (p StructuredPath) IsDataSourceFilter() bool {
	return len(p) >= 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "filter"
}

// IsDataSourceFilterField returns true if path points to a filter field.
// Pattern: /datasources/{name}/filter/{index}/field
func (p StructuredPath) IsDataSourceFilterField() bool {
	return len(p) == 5 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "filter" &&
		p[3].Kind == PathSegmentIndex &&
		p[4].Kind == PathSegmentField && p[4].FieldName == "field"
}

// IsDataSourceFilterOperator returns true if path points to a filter operator.
// Pattern: /datasources/{name}/filter/{index}/operator
func (p StructuredPath) IsDataSourceFilterOperator() bool {
	return len(p) == 5 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "filter" &&
		p[3].Kind == PathSegmentIndex &&
		p[4].Kind == PathSegmentField && p[4].FieldName == "operator"
}

// GetResourceName returns the resource name if path is under a resource.
func (p StructuredPath) GetResourceName() (string, bool) {
	if len(p) >= 2 && p[0].Kind == PathSegmentField && p[0].FieldName == "resources" &&
		p[1].Kind == PathSegmentField {
		return p[1].FieldName, true
	}
	return "", false
}

// GetDataSourceName returns the data source name if path is under a data source.
func (p StructuredPath) GetDataSourceName() (string, bool) {
	if len(p) >= 2 && p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField {
		return p[1].FieldName, true
	}
	return "", false
}

// GetVariableName returns the variable name if path is under a variable.
func (p StructuredPath) GetVariableName() (string, bool) {
	if len(p) >= 2 && p[0].Kind == PathSegmentField && p[0].FieldName == "variables" &&
		p[1].Kind == PathSegmentField {
		return p[1].FieldName, true
	}
	return "", false
}

// GetValueName returns the value name if path is under a value.
func (p StructuredPath) GetValueName() (string, bool) {
	if len(p) >= 2 && p[0].Kind == PathSegmentField && p[0].FieldName == "values" &&
		p[1].Kind == PathSegmentField {
		return p[1].FieldName, true
	}
	return "", false
}

// GetSpecPath returns the path segments after /resources/{name}/spec/.
func (p StructuredPath) GetSpecPath() []PathSegment {
	if !p.IsResourceSpec() || len(p) <= 3 {
		return nil
	}
	return p[3:]
}

// String returns the full path as a string.
func (p StructuredPath) String() string {
	if len(p) == 0 {
		return "/"
	}

	result := ""
	for _, seg := range p {
		result += "/" + seg.String()
	}
	return result
}
