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

func (s *PathSuite) TestStructuredPath_SegmentBasics() {
	empty := StructuredPath{}
	threeSegs := StructuredPath{
		{Kind: PathSegmentField, FieldName: "resources"},
		{Kind: PathSegmentField, FieldName: "myResource"},
		{Kind: PathSegmentField, FieldName: "type"},
	}

	s.Run("Len_empty", func() {
		s.Assert().Equal(0, empty.Len())
	})
	s.Run("Len_three", func() {
		s.Assert().Equal(3, threeSegs.Len())
	})
	s.Run("IsEmpty_true", func() {
		s.Assert().True(empty.IsEmpty())
	})
	s.Run("IsEmpty_false", func() {
		s.Assert().False(threeSegs.IsEmpty())
	})
	s.Run("Last_empty", func() {
		seg := empty.Last()
		s.Assert().Equal(PathSegment{}, seg)
	})
	s.Run("Last_populated", func() {
		seg := threeSegs.Last()
		s.Assert().Equal("type", seg.FieldName)
	})
	s.Run("At_valid_index", func() {
		seg := threeSegs.At(1)
		s.Assert().Equal("myResource", seg.FieldName)
	})
	s.Run("At_out_of_bounds", func() {
		seg := threeSegs.At(5)
		s.Assert().Equal(PathSegment{}, seg)
	})
	s.Run("At_negative_index", func() {
		seg := threeSegs.At(-1)
		s.Assert().Equal(PathSegment{}, seg)
	})
}

func (s *PathSuite) TestStructuredPath_SectionCheckers() {
	tests := []struct {
		name     string
		path     StructuredPath
		inRes    bool
		inDS     bool
		inVars   bool
		inVals   bool
		inExps   bool
		inIncl   bool
	}{
		{
			name: "resources path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myResource"},
			},
			inRes: true,
		},
		{
			name: "datasources path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
			},
			inDS: true,
		},
		{
			name: "variables path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
				{Kind: PathSegmentField, FieldName: "myVar"},
			},
			inVars: true,
		},
		{
			name: "values path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "values"},
				{Kind: PathSegmentField, FieldName: "myVal"},
			},
			inVals: true,
		},
		{
			name: "exports path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "myExport"},
			},
			inExps: true,
		},
		{
			name: "include path",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "include"},
				{Kind: PathSegmentField, FieldName: "myInclude"},
			},
			inIncl: true,
		},
		{
			name:  "empty path",
			path:  StructuredPath{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.inRes, tt.path.IsInResources())
			s.Assert().Equal(tt.inDS, tt.path.IsInDataSources())
			s.Assert().Equal(tt.inVars, tt.path.IsInVariables())
			s.Assert().Equal(tt.inVals, tt.path.IsInValues())
			s.Assert().Equal(tt.inExps, tt.path.IsInExports())
			s.Assert().Equal(tt.inIncl, tt.path.IsInIncludes())
		})
	}
}

func (s *PathSuite) TestStructuredPath_GetterMethods() {
	tests := []struct {
		name         string
		path         StructuredPath
		method       string
		expectedName string
		expectedOk   bool
	}{
		{
			name: "GetVariableName valid",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
				{Kind: PathSegmentField, FieldName: "environment"},
			},
			method:       "GetVariableName",
			expectedName: "environment",
			expectedOk:   true,
		},
		{
			name:         "GetVariableName too short",
			path:         StructuredPath{{Kind: PathSegmentField, FieldName: "variables"}},
			method:       "GetVariableName",
			expectedName: "",
			expectedOk:   false,
		},
		{
			name: "GetValueName valid",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "values"},
				{Kind: PathSegmentField, FieldName: "tableName"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			method:       "GetValueName",
			expectedName: "tableName",
			expectedOk:   true,
		},
		{
			name:         "GetValueName too short",
			path:         StructuredPath{{Kind: PathSegmentField, FieldName: "values"}},
			method:       "GetValueName",
			expectedName: "",
			expectedOk:   false,
		},
		{
			name: "GetDataSourceName valid",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
			},
			method:       "GetDataSourceName",
			expectedName: "network",
			expectedOk:   true,
		},
		{
			name:         "GetDataSourceName wrong section",
			path:         StructuredPath{{Kind: PathSegmentField, FieldName: "resources"}, {Kind: PathSegmentField, FieldName: "x"}},
			method:       "GetDataSourceName",
			expectedName: "",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var name string
			var ok bool
			switch tt.method {
			case "GetVariableName":
				name, ok = tt.path.GetVariableName()
			case "GetValueName":
				name, ok = tt.path.GetValueName()
			case "GetDataSourceName":
				name, ok = tt.path.GetDataSourceName()
			}
			s.Assert().Equal(tt.expectedName, name)
			s.Assert().Equal(tt.expectedOk, ok)
		})
	}
}

func (s *PathSuite) TestStructuredPath_AnnotationAndExportGetters() {
	tests := []struct {
		name         string
		path         StructuredPath
		method       string
		expectedName string
		expectedOk   bool
	}{
		{
			name: "GetAnnotationKey valid",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "annotations"},
				{Kind: PathSegmentField, FieldName: "aws.lambda.runtime"},
			},
			method:       "GetAnnotationKey",
			expectedName: "aws.lambda.runtime",
			expectedOk:   true,
		},
		{
			name: "GetAnnotationKey too short",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "annotations"},
			},
			method:       "GetAnnotationKey",
			expectedName: "",
			expectedOk:   false,
		},
		{
			name: "GetDataSourceAnnotationKey valid",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "annotations"},
				{Kind: PathSegmentField, FieldName: "aws.vpc.id"},
			},
			method:       "GetDataSourceAnnotationKey",
			expectedName: "aws.vpc.id",
			expectedOk:   true,
		},
		{
			name: "GetDataSourceAnnotationKey wrong section",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "annotations"},
				{Kind: PathSegmentField, FieldName: "key"},
			},
			method:       "GetDataSourceAnnotationKey",
			expectedName: "",
			expectedOk:   false,
		},
		{
			name: "GetDataSourceExportName valid",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "vpcId"},
			},
			method:       "GetDataSourceExportName",
			expectedName: "vpcId",
			expectedOk:   true,
		},
		{
			name: "GetDataSourceExportName too short",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "exports"},
			},
			method:       "GetDataSourceExportName",
			expectedName: "",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var name string
			var ok bool
			switch tt.method {
			case "GetAnnotationKey":
				name, ok = tt.path.GetAnnotationKey()
			case "GetDataSourceAnnotationKey":
				name, ok = tt.path.GetDataSourceAnnotationKey()
			case "GetDataSourceExportName":
				name, ok = tt.path.GetDataSourceExportName()
			}
			s.Assert().Equal(tt.expectedName, name)
			s.Assert().Equal(tt.expectedOk, ok)
		})
	}
}

func (s *PathSuite) TestStructuredPath_TypeCheckers() {
	tests := []struct {
		name    string
		path    StructuredPath
		isVarT  bool
		isValT  bool
		isExpT  bool
	}{
		{
			name: "variable type",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
				{Kind: PathSegmentField, FieldName: "env"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			isVarT: true,
		},
		{
			name: "value type",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "values"},
				{Kind: PathSegmentField, FieldName: "tableName"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			isValT: true,
		},
		{
			name: "export type",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "apiUrl"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			isExpT: true,
		},
		{
			name: "resource type is not variable/value/export type",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
		},
		{
			name: "too short",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "variables"},
				{Kind: PathSegmentField, FieldName: "env"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.isVarT, tt.path.IsVariableType())
			s.Assert().Equal(tt.isValT, tt.path.IsValueType())
			s.Assert().Equal(tt.isExpT, tt.path.IsExportType())
		})
	}
}

func (s *PathSuite) TestStructuredPath_ResourceMetadataCheckers() {
	tests := []struct {
		name        string
		path        StructuredPath
		isMeta      bool
		isAnnot     bool
		isAnnotVal  bool
		isLabels    bool
		isLinkSel   bool
		isLinkExcl  bool
	}{
		{
			name: "resource metadata root",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "metadata"},
			},
			isMeta: true,
		},
		{
			name: "resource metadata annotations",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "annotations"},
			},
			isMeta: true, isAnnot: true,
		},
		{
			name: "resource metadata annotation value",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "annotations"},
				{Kind: PathSegmentField, FieldName: "aws.runtime"},
			},
			isMeta: true, isAnnot: true, isAnnotVal: true,
		},
		{
			name: "resource metadata labels",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "labels"},
			},
			isMeta: true, isLabels: true,
		},
		{
			name: "resource linkSelector",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "linkSelector"},
			},
			isLinkSel: true,
		},
		{
			name: "resource linkSelector exclude",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "linkSelector"},
				{Kind: PathSegmentField, FieldName: "exclude"},
			},
			isLinkExcl: true,
		},
		{
			name: "datasource metadata is not resource metadata",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "ds"},
				{Kind: PathSegmentField, FieldName: "metadata"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.isMeta, tt.path.IsResourceMetadata())
			s.Assert().Equal(tt.isAnnot, tt.path.IsResourceMetadataAnnotations())
			s.Assert().Equal(tt.isAnnotVal, tt.path.IsResourceMetadataAnnotationValue())
			s.Assert().Equal(tt.isLabels, tt.path.IsResourceMetadataLabels())
			s.Assert().Equal(tt.isLinkSel, tt.path.IsResourceLinkSelector())
			s.Assert().Equal(tt.isLinkExcl, tt.path.IsResourceLinkSelectorExclude())
		})
	}
}

func (s *PathSuite) TestStructuredPath_DataSourceAnnotationAndFilterCheckers() {
	tests := []struct {
		name         string
		path         StructuredPath
		isDSAnnot    bool
		isDSAnnotVal bool
		isDSAliasFor bool
		isDSFilterF  bool
		isDSFilterOp bool
	}{
		{
			name: "datasource annotations",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "annotations"},
			},
			isDSAnnot: true,
		},
		{
			name: "datasource annotation value",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "annotations"},
				{Kind: PathSegmentField, FieldName: "aws.vpc.id"},
			},
			isDSAnnot: true, isDSAnnotVal: true,
		},
		{
			name: "datasource export aliasFor",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "vpcId"},
				{Kind: PathSegmentField, FieldName: "aliasFor"},
			},
			isDSAliasFor: true,
		},
		{
			name: "datasource export aliasFor wrong field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "exports"},
				{Kind: PathSegmentField, FieldName: "vpcId"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
		},
		{
			name: "datasource filter field (singular pattern)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "filter"},
				{Kind: PathSegmentField, FieldName: "field"},
			},
			isDSFilterF: true,
		},
		{
			name: "datasource filter operator (singular pattern)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "filter"},
				{Kind: PathSegmentField, FieldName: "operator"},
			},
			isDSFilterOp: true,
		},
		{
			name: "datasource filter field (plural pattern)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "filters"},
				{Kind: PathSegmentIndex, Index: 0},
				{Kind: PathSegmentField, FieldName: "filter"},
				{Kind: PathSegmentField, FieldName: "field"},
			},
			isDSFilterF: true,
		},
		{
			name: "datasource filter operator (plural pattern)",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "network"},
				{Kind: PathSegmentField, FieldName: "filters"},
				{Kind: PathSegmentIndex, Index: 0},
				{Kind: PathSegmentField, FieldName: "filter"},
				{Kind: PathSegmentField, FieldName: "operator"},
			},
			isDSFilterOp: true,
		},
		{
			name: "resource annotations are not datasource annotations",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "handler"},
				{Kind: PathSegmentField, FieldName: "metadata"},
				{Kind: PathSegmentField, FieldName: "annotations"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.isDSAnnot, tt.path.IsDataSourceMetadataAnnotations())
			s.Assert().Equal(tt.isDSAnnotVal, tt.path.IsDataSourceMetadataAnnotationValue())
			s.Assert().Equal(tt.isDSAliasFor, tt.path.IsDataSourceExportAliasFor())
			s.Assert().Equal(tt.isDSFilterF, tt.path.IsDataSourceFilterField())
			s.Assert().Equal(tt.isDSFilterOp, tt.path.IsDataSourceFilterOperator())
		})
	}
}

func (s *PathSuite) TestPathSegment_String() {
	s.Assert().Equal("resources", PathSegment{Kind: PathSegmentField, FieldName: "resources"}.String())
	s.Assert().Equal("0", PathSegment{Kind: PathSegmentIndex, Index: 0}.String())
	s.Assert().Equal("42", PathSegment{Kind: PathSegmentIndex, Index: 42}.String())
}

func TestPathSuite(t *testing.T) {
	suite.Run(t, new(PathSuite))
}
