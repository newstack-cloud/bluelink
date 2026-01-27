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

// IsEmpty returns true if the path has no segments.
func (p StructuredPath) IsEmpty() bool {
	return len(p) == 0
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

// IsResourceDefinition returns true if path is directly inside a resource definition,
// but not in a nested field like spec or metadata.
// Pattern: /resources/{name} (exactly 2 segments)
func (p StructuredPath) IsResourceDefinition() bool {
	return len(p) == 2 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "resources" &&
		p[1].Kind == PathSegmentField
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
// Pattern: /datasources/{name}/filters/... (note: "filters" plural in schema tree)
func (p StructuredPath) IsDataSourceFilter() bool {
	return len(p) >= 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "filters"
}

// IsDataSourceFilterField returns true if path points to a filter field.
// Pattern 1: /datasources/{name}/filter/field (4 segments, singular filter)
// Pattern 2: /datasources/{name}/filters/{index}/filter/field (6 segments, plural filters with index)
func (p StructuredPath) IsDataSourceFilterField() bool {
	// Pattern 1: singular filter (common case)
	if len(p) == 4 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "filter" &&
		p[3].Kind == PathSegmentField && p[3].FieldName == "field" {
		return true
	}
	// Pattern 2: plural filters with index
	return len(p) == 6 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "filters" &&
		p[4].Kind == PathSegmentField && p[4].FieldName == "filter" &&
		p[5].Kind == PathSegmentField && p[5].FieldName == "field"
}

// IsDataSourceFilterOperator returns true if path points to a filter operator.
// Pattern 1: /datasources/{name}/filter/operator (4 segments, singular filter)
// Pattern 2: /datasources/{name}/filters/{index}/filter/operator (6 segments, plural filters with index)
func (p StructuredPath) IsDataSourceFilterOperator() bool {
	// Pattern 1: singular filter (common case)
	if len(p) == 4 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "filter" &&
		p[3].Kind == PathSegmentField && p[3].FieldName == "operator" {
		return true
	}
	// Pattern 2: plural filters with index
	return len(p) == 6 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "filters" &&
		p[4].Kind == PathSegmentField && p[4].FieldName == "filter" &&
		p[5].Kind == PathSegmentField && p[5].FieldName == "operator"
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
// Returns an empty slice when at the root of spec (path length == 3),
// or nil if not in a resource spec path.
func (p StructuredPath) GetSpecPath() []PathSegment {
	if !p.IsResourceSpec() {
		return nil
	}
	if len(p) == 3 {
		return []PathSegment{}
	}
	return p[3:]
}

// IsVariableDefinition returns true if path is directly inside a variable definition.
// Pattern: /variables/{name} (exactly 2 segments)
func (p StructuredPath) IsVariableDefinition() bool {
	return len(p) == 2 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "variables" &&
		p[1].Kind == PathSegmentField
}

// IsValueDefinition returns true if path is directly inside a value definition.
// Pattern: /values/{name} (exactly 2 segments)
func (p StructuredPath) IsValueDefinition() bool {
	return len(p) == 2 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "values" &&
		p[1].Kind == PathSegmentField
}

// IsDataSourceDefinition returns true if path is directly inside a data source definition,
// but not in a nested field like metadata, filter, or exports.
// Pattern: /datasources/{name} (exactly 2 segments)
func (p StructuredPath) IsDataSourceDefinition() bool {
	return len(p) == 2 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField
}

// IsDataSourceMetadata returns true if path is inside a data source metadata field.
// Pattern: /datasources/{name}/metadata/...
func (p StructuredPath) IsDataSourceMetadata() bool {
	return len(p) >= 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "metadata"
}

// IsDataSourceExports returns true if path is inside a data source exports field.
// Pattern: /datasources/{name}/exports/...
func (p StructuredPath) IsDataSourceExports() bool {
	return len(p) >= 3 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "exports"
}

// IsDataSourceExportDefinition returns true if path is at a specific export definition.
// Pattern: /datasources/{name}/exports/{exportName} (exactly 4 segments)
func (p StructuredPath) IsDataSourceExportDefinition() bool {
	return len(p) == 4 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "exports" &&
		p[3].Kind == PathSegmentField
}

// IsDataSourceExportAliasFor returns true if path points to a data source export aliasFor field.
// Pattern: /datasources/{name}/exports/{exportName}/aliasFor (5 segments)
func (p StructuredPath) IsDataSourceExportAliasFor() bool {
	return len(p) == 5 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "exports" &&
		p[3].Kind == PathSegmentField &&
		p[4].Kind == PathSegmentField && p[4].FieldName == "aliasFor"
}

// GetDataSourceExportName returns the export name if path is under a data source export.
// Pattern: /datasources/{name}/exports/{exportName}/...
func (p StructuredPath) GetDataSourceExportName() (string, bool) {
	if len(p) >= 4 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "datasources" &&
		p[1].Kind == PathSegmentField &&
		p[2].Kind == PathSegmentField && p[2].FieldName == "exports" &&
		p[3].Kind == PathSegmentField {
		return p[3].FieldName, true
	}
	return "", false
}

// IsDataSourceFilterDefinition returns true if path is at a filter definition level.
// Pattern: /datasources/{name}/filter or /datasources/{name}/filter/{index}
// Note: The singular "filter" is used in the schema tree.
func (p StructuredPath) IsDataSourceFilterDefinition() bool {
	if len(p) < 3 {
		return false
	}
	if p[0].Kind != PathSegmentField || p[0].FieldName != "datasources" {
		return false
	}
	if p[1].Kind != PathSegmentField {
		return false
	}
	// Check for either "filter" (singular, from schema) or "filters" (from validation)
	if p[2].Kind != PathSegmentField || (p[2].FieldName != "filter" && p[2].FieldName != "filters") {
		return false
	}
	// At /datasources/{name}/filter level OR /datasources/{name}/filter/{index}
	return len(p) == 3 || (len(p) == 4 && p[3].Kind == PathSegmentIndex)
}

// IsIncludeDefinition returns true if path is directly inside an include definition.
// Pattern: /include/{name} (exactly 2 segments)
func (p StructuredPath) IsIncludeDefinition() bool {
	return len(p) == 2 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "include" &&
		p[1].Kind == PathSegmentField
}

// IsExportDefinition returns true if path is directly inside an export definition.
// Pattern: /exports/{name} (exactly 2 segments)
func (p StructuredPath) IsExportDefinition() bool {
	return len(p) == 2 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "exports" &&
		p[1].Kind == PathSegmentField
}

// IsBlueprintTopLevel returns true if path is at the blueprint root level.
// Pattern: /{field} (single segment that is a known top-level field)
// Note: Empty paths are not considered top-level as they indicate no parsed structure.
func (p StructuredPath) IsBlueprintTopLevel() bool {
	if len(p) != 1 || p[0].Kind != PathSegmentField {
		return false
	}
	// Check if it's a known top-level section
	switch p[0].FieldName {
	case "version", "transform", "variables", "values", "include",
		"resources", "datasources", "exports", "metadata":
		return true
	}
	return false
}

// IsBlueprintMetadata returns true if path is in blueprint-level metadata.
// Pattern: /metadata or /metadata/...
func (p StructuredPath) IsBlueprintMetadata() bool {
	return len(p) >= 1 &&
		p[0].Kind == PathSegmentField && p[0].FieldName == "metadata"
}

// GetIncludeName returns the include name if path is under an include.
func (p StructuredPath) GetIncludeName() (string, bool) {
	if len(p) >= 2 && p[0].Kind == PathSegmentField && p[0].FieldName == "include" &&
		p[1].Kind == PathSegmentField {
		return p[1].FieldName, true
	}
	return "", false
}

// GetExportName returns the export name if path is under an export.
func (p StructuredPath) GetExportName() (string, bool) {
	if len(p) >= 2 && p[0].Kind == PathSegmentField && p[0].FieldName == "exports" &&
		p[1].Kind == PathSegmentField {
		return p[1].FieldName, true
	}
	return "", false
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
