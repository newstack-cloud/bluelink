package docmodel

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PathSuite struct {
	suite.Suite
}

func (s *PathSuite) TestStructuredPath_IsResourceType() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "valid resource type path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: true,
		},
		{
			name: "resource spec path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "spec"},
			},
			expected: false,
		},
		{
			name: "datasource type path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "too short",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsResourceType())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsDataSourceType() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "valid datasource type path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: true,
		},
		{
			name: "resource type path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsDataSourceType())
		})
	}
}

func (s *PathSuite) TestStructuredPath_GetResourceName() {
	tests := []struct {
		name         string
		path         StructuredPath
		expectedName string
		expectedOk   bool
	}{
		{
			name: "valid resource path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expectedName: "myResource",
			expectedOk:   true,
		},
		{
			name: "not a resource path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
				{Kind: PathSegmentField, FieldName: "myVar"},
			},
			expectedName: "",
			expectedOk:   false,
		},
		{
			name:         "empty path",
			path:         StructuredPath{},
			expectedName: "",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			name, ok := tt.path.GetResourceName()
			s.Assert().Equal(tt.expectedName, name)
			s.Assert().Equal(tt.expectedOk, ok)
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsResourceSpec() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "in spec",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "spec"},
				{Kind: PathSegmentField, FieldName: "tableName"},
			},
			expected: true,
		},
		{
			name: "at spec level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "spec"},
			},
			expected: true,
		},
		{
			name: "not spec",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsResourceSpec())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsResourceDefinition() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at resource definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
			},
			expected: true,
		},
		{
			name: "in resource spec",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "spec"},
			},
			expected: false,
		},
		{
			name: "in resource type",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "at resources level only",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
			},
			expected: false,
		},
		{
			name: "empty path",
			path: StructuredPath{},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsResourceDefinition())
		})
	}
}

func (s *PathSuite) TestStructuredPath_GetSpecPath() {
	tests := []struct {
		name           string
		path           StructuredPath
		expectedLen    int
		expectedNil    bool
		expectedFields []string
	}{
		{
			name: "nested spec path returns segments after spec",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "spec"},
				{Kind: PathSegmentField, FieldName: "tableName"},
				{Kind: PathSegmentField, FieldName: "value"},
			},
			expectedLen:    2,
			expectedNil:    false,
			expectedFields: []string{"tableName", "value"},
		},
		{
			name: "at spec root returns empty slice not nil",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "spec"},
			},
			expectedLen:    0,
			expectedNil:    false,
			expectedFields: []string{},
		},
		{
			name: "one level deep in spec",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "spec"},
				{Kind: PathSegmentField, FieldName: "handler"},
			},
			expectedLen:    1,
			expectedNil:    false,
			expectedFields: []string{"handler"},
		},
		{
			name: "not a spec path returns nil",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expectedLen: 0,
			expectedNil: true,
		},
		{
			name: "variables path returns nil",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
				{Kind: PathSegmentField, FieldName: "myVar"},
			},
			expectedLen: 0,
			expectedNil: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			specPath := tt.path.GetSpecPath()
			if tt.expectedNil {
				s.Assert().Nil(specPath)
			} else {
				s.Assert().NotNil(specPath)
				s.Assert().Len(specPath, tt.expectedLen)
				for i, expectedField := range tt.expectedFields {
					s.Assert().Equal(expectedField, specPath[i].FieldName)
				}
			}
		})
	}
}

func (s *PathSuite) TestStructuredPath_String() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected string
	}{
		{
			name:     "empty path",
			path:     StructuredPath{},
			expected: "/",
		},
		{
			name: "simple path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
			},
			expected: "/resources/myResource",
		},
		{
			name: "path with index",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "dependsOn"},
				{Kind: PathSegmentIndex, Index: 0},
			},
			expected: "/resources/myResource/dependsOn/0",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.String())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsDataSourceFilter() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "in filter",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "filters"},
				{Kind: PathSegmentIndex, Index: 0},
			},
			expected: true,
		},
		{
			name: "not filter",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsDataSourceFilter())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsVariableDefinition() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at variable definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
				{Kind: PathSegmentField, FieldName: "myVar"},
			},
			expected: true,
		},
		{
			name: "in variable type field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
				{Kind: PathSegmentField, FieldName: "myVar"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "at variables section only",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
			},
			expected: false,
		},
		{
			name: "resource path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsVariableDefinition())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsValueDefinition() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at value definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "values"},
				{Kind: PathSegmentField, FieldName: "myValue"},
			},
			expected: true,
		},
		{
			name: "in value type field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "values"},
				{Kind: PathSegmentField, FieldName: "myValue"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "at values section only",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "values"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsValueDefinition())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsDataSourceDefinition() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at datasource definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
			},
			expected: true,
		},
		{
			name: "in datasource type field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "in datasource filters",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "filters"},
			},
			expected: false,
		},
		{
			name: "at datasources section only",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsDataSourceDefinition())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsIncludeDefinition() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at include definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "include"},
				{Kind: PathSegmentField, FieldName: "myInclude"},
			},
			expected: true,
		},
		{
			name: "in include path field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "include"},
				{Kind: PathSegmentField, FieldName: "myInclude"},
				{Kind: PathSegmentField, FieldName: "path"},
			},
			expected: false,
		},
		{
			name: "at include section only",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "include"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsIncludeDefinition())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsExportDefinition() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at export definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
			},
			expected: true,
		},
		{
			name: "in export type field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "at exports section only",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsExportDefinition())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsBlueprintTopLevel() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name:     "empty path (no parsed structure)",
			path:     StructuredPath{},
			expected: false,
		},
		{
			name: "at resources section",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
			},
			expected: true,
		},
		{
			name: "at variables section",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
			},
			expected: true,
		},
		{
			name: "at version field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "version"},
			},
			expected: true,
		},
		{
			name: "at metadata field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "metadata"},
			},
			expected: true,
		},
		{
			name: "inside resources",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
			},
			expected: false,
		},
		{
			name: "unknown top-level field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "unknownField"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsBlueprintTopLevel())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsBlueprintMetadata() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at metadata root",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "metadata"},
			},
			expected: true,
		},
		{
			name: "in metadata displayName",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "displayName"},
			},
			expected: true,
		},
		{
			name: "in metadata labels",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "labels"},
				{Kind: PathSegmentField, FieldName: "env"},
			},
			expected: true,
		},
		{
			name: "in resource metadata (not blueprint metadata)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "metadata"},
			},
			expected: false,
		},
		{
			name: "at resources section",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsBlueprintMetadata())
		})
	}
}

func (s *PathSuite) TestStructuredPath_GetIncludeName() {
	tests := []struct {
		name         string
		path         StructuredPath
		expectedName string
		expectedOk   bool
	}{
		{
			name: "valid include path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "include"},
				{Kind: PathSegmentField, FieldName: "myInclude"},
			},
			expectedName: "myInclude",
			expectedOk:   true,
		},
		{
			name: "include with nested field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "include"},
				{Kind: PathSegmentField, FieldName: "myInclude"},
				{Kind: PathSegmentField, FieldName: "path"},
			},
			expectedName: "myInclude",
			expectedOk:   true,
		},
		{
			name: "not an include path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
			},
			expectedName: "",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			name, ok := tt.path.GetIncludeName()
			s.Assert().Equal(tt.expectedName, name)
			s.Assert().Equal(tt.expectedOk, ok)
		})
	}
}

func (s *PathSuite) TestStructuredPath_GetExportName() {
	tests := []struct {
		name         string
		path         StructuredPath
		expectedName string
		expectedOk   bool
	}{
		{
			name: "valid export path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
			},
			expectedName: "myExport",
			expectedOk:   true,
		},
		{
			name: "export with nested field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expectedName: "myExport",
			expectedOk:   true,
		},
		{
			name: "not an export path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
			},
			expectedName: "",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			name, ok := tt.path.GetExportName()
			s.Assert().Equal(tt.expectedName, name)
			s.Assert().Equal(tt.expectedOk, ok)
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsDataSourceMetadata() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at datasource metadata root",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "metadata"},
			},
			expected: true,
		},
		{
			name: "in datasource metadata displayName",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "displayName"},
			},
			expected: true,
		},
		{
			name: "in datasource type field (not metadata)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "at datasource definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
			},
			expected: false,
		},
		{
			name: "at datasources section only",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
			},
			expected: false,
		},
		{
			name: "in resource metadata (not datasource metadata)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "metadata"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsDataSourceMetadata())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsDataSourceExports() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at datasource exports root",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "exports"},
			},
			expected: true,
		},
		{
			name: "at specific export definition",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "vpcId"},
			},
			expected: true,
		},
		{
			name: "in export type field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "vpcId"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: true,
		},
		{
			name: "in datasource type field (not exports)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "at datasource definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
			},
			expected: false,
		},
		{
			name: "blueprint-level exports (not datasource exports)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsDataSourceExports())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsDataSourceExportDefinition() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at specific export definition",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "vpcId"},
			},
			expected: true,
		},
		{
			name: "at datasource exports root (not definition)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "exports"},
			},
			expected: false,
		},
		{
			name: "in export type field (too deep)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "vpcId"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "blueprint-level export (not datasource export)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
			},
			expected: false,
		},
		{
			name: "at datasource definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsDataSourceExportDefinition())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsDataSourceFilterDefinition() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "at filter root (singular)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "filter"},
			},
			expected: true,
		},
		{
			name: "at filter root (plural)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "filters"},
			},
			expected: true,
		},
		{
			name: "at filter index",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "filter"},
				{Kind: PathSegmentIndex, Index: 0},
			},
			expected: true,
		},
		{
			name: "at filters index (plural)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "filters"},
				{Kind: PathSegmentIndex, Index: 1},
			},
			expected: true,
		},
		{
			name: "inside filter field (too deep)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "filter"},
				{Kind: PathSegmentIndex, Index: 0},
				{Kind: PathSegmentField, FieldName: "field"},
			},
			expected: false,
		},
		{
			name: "at datasource definition level",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
			},
			expected: false,
		},
		{
			name: "at datasource type field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "empty path",
			path: StructuredPath{},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsDataSourceFilterDefinition())
		})
	}
}

func (s *PathSuite) TestStructuredPath_IsExportField() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "valid export field path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
				{Kind: PathSegmentField, FieldName: "field"},
			},
			expected: true,
		},
		{
			name: "export type path (not field)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: false,
		},
		{
			name: "export definition level only",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
			},
			expected: false,
		},
		{
			name: "exports section only",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
			},
			expected: false,
		},
		{
			name: "resource spec field (not export field)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
				{Kind: PathSegmentField, FieldName: "field"},
			},
			expected: false,
		},
		{
			name: "empty path",
			path: StructuredPath{},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, tt.path.IsExportField())
		})
	}
}

func TestPathSuite(t *testing.T) {
	suite.Run(t, new(PathSuite))
}
