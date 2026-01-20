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

func (s *PathSuite) TestStructuredPath_GetSpecPath() {
	path := StructuredPath{
		{Kind: PathSegmentField, FieldName: "resources"},
		{Kind: PathSegmentField, FieldName: "myResource"},
		{Kind: PathSegmentField, FieldName: "spec"},
		{Kind: PathSegmentField, FieldName: "tableName"},
		{Kind: PathSegmentField, FieldName: "value"},
	}

	specPath := path.GetSpecPath()
	s.Assert().Len(specPath, 2)
	s.Assert().Equal("tableName", specPath[0].FieldName)
	s.Assert().Equal("value", specPath[1].FieldName)
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
				{Kind: PathSegmentField, FieldName: "filter"},
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

func TestPathSuite(t *testing.T) {
	suite.Run(t, new(PathSuite))
}
