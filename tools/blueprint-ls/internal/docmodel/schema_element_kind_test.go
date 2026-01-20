package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type SchemaElementKindSuite struct {
	suite.Suite
}

func (s *SchemaElementKindSuite) TestKindFromSchemaElement() {
	tests := []struct {
		name     string
		element  interface{}
		expected SchemaElementKind
	}{
		{
			name:     "nil element",
			element:  nil,
			expected: SchemaElementUnknown,
		},
		{
			name:     "resource type wrapper",
			element:  &schema.ResourceTypeWrapper{},
			expected: SchemaElementResourceType,
		},
		{
			name:     "datasource type wrapper",
			element:  &schema.DataSourceTypeWrapper{},
			expected: SchemaElementDataSourceType,
		},
		{
			name:     "function expression",
			element:  &substitutions.SubstitutionFunctionExpr{},
			expected: SchemaElementFunctionCall,
		},
		{
			name:     "variable reference",
			element:  &substitutions.SubstitutionVariable{},
			expected: SchemaElementVariableRef,
		},
		{
			name:     "value reference",
			element:  &substitutions.SubstitutionValueReference{},
			expected: SchemaElementValueRef,
		},
		{
			name:     "resource property reference",
			element:  &substitutions.SubstitutionResourceProperty{},
			expected: SchemaElementResourceRef,
		},
		{
			name:     "datasource property reference",
			element:  &substitutions.SubstitutionDataSourceProperty{},
			expected: SchemaElementDataSourceRef,
		},
		{
			name:     "child reference",
			element:  &substitutions.SubstitutionChild{},
			expected: SchemaElementChildRef,
		},
		{
			name:     "elem reference",
			element:  &substitutions.SubstitutionElemReference{},
			expected: SchemaElementElemRef,
		},
		{
			name:     "elem index reference",
			element:  &substitutions.SubstitutionElemIndexReference{},
			expected: SchemaElementElemIndexRef,
		},
		{
			name:     "resource",
			element:  &schema.Resource{},
			expected: SchemaElementResource,
		},
		{
			name:     "datasource",
			element:  &schema.DataSource{},
			expected: SchemaElementDataSource,
		},
		{
			name:     "variable",
			element:  &schema.Variable{},
			expected: SchemaElementVariable,
		},
		{
			name:     "value",
			element:  &schema.Value{},
			expected: SchemaElementValue,
		},
		{
			name:     "include",
			element:  &schema.Include{},
			expected: SchemaElementInclude,
		},
		{
			name:     "export",
			element:  &schema.Export{},
			expected: SchemaElementExport,
		},
		{
			name:     "resource map",
			element:  &schema.ResourceMap{},
			expected: SchemaElementResources,
		},
		{
			name:     "datasource map",
			element:  &schema.DataSourceMap{},
			expected: SchemaElementDataSources,
		},
		{
			name:     "variable map",
			element:  &schema.VariableMap{},
			expected: SchemaElementVariables,
		},
		{
			name:     "value map",
			element:  &schema.ValueMap{},
			expected: SchemaElementValues,
		},
		{
			name:     "include map",
			element:  &schema.IncludeMap{},
			expected: SchemaElementIncludes,
		},
		{
			name:     "export map",
			element:  &schema.ExportMap{},
			expected: SchemaElementExports,
		},
		{
			name:     "blueprint returns unknown",
			element:  &schema.Blueprint{},
			expected: SchemaElementUnknown,
		},
		{
			name:     "unknown type",
			element:  "random string",
			expected: SchemaElementUnknown,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := KindFromSchemaElement(tt.element)
			s.Assert().Equal(tt.expected, result)
		})
	}
}

func (s *SchemaElementKindSuite) TestString() {
	tests := []struct {
		kind     SchemaElementKind
		expected string
	}{
		{SchemaElementUnknown, "unknown"},
		{SchemaElementVariables, "variables"},
		{SchemaElementValues, "values"},
		{SchemaElementResources, "resources"},
		{SchemaElementDataSources, "datasources"},
		{SchemaElementIncludes, "includes"},
		{SchemaElementExports, "exports"},
		{SchemaElementVariable, "variable"},
		{SchemaElementValue, "value"},
		{SchemaElementResource, "resource"},
		{SchemaElementDataSource, "datasource"},
		{SchemaElementInclude, "include"},
		{SchemaElementExport, "export"},
		{SchemaElementResourceType, "resource_type"},
		{SchemaElementDataSourceType, "datasource_type"},
		{SchemaElementVariableType, "variable_type"},
		{SchemaElementValueType, "value_type"},
		{SchemaElementExportType, "export_type"},
		{SchemaElementDataSourceFieldType, "datasource_field_type"},
		{SchemaElementDataSourceFilterField, "datasource_filter_field"},
		{SchemaElementDataSourceFilterOperator, "datasource_filter_operator"},
		{SchemaElementSubstitution, "substitution"},
		{SchemaElementFunctionCall, "function_call"},
		{SchemaElementVariableRef, "variable_ref"},
		{SchemaElementValueRef, "value_ref"},
		{SchemaElementResourceRef, "resource_ref"},
		{SchemaElementDataSourceRef, "datasource_ref"},
		{SchemaElementChildRef, "child_ref"},
		{SchemaElementElemRef, "elem_ref"},
		{SchemaElementElemIndexRef, "elem_index_ref"},
		{SchemaElementScalar, "scalar"},
		{SchemaElementMapping, "mapping"},
		{SchemaElementSequence, "sequence"},
		{SchemaElementKind(999), "unknown"},
	}

	for _, tt := range tests {
		s.Run(tt.expected, func() {
			s.Assert().Equal(tt.expected, tt.kind.String())
		})
	}
}

func (s *SchemaElementKindSuite) TestIsTypeField() {
	typeFields := []SchemaElementKind{
		SchemaElementResourceType,
		SchemaElementDataSourceType,
		SchemaElementVariableType,
		SchemaElementValueType,
		SchemaElementExportType,
		SchemaElementDataSourceFieldType,
	}

	for _, kind := range typeFields {
		s.Assert().True(kind.IsTypeField(), "%s should be a type field", kind)
	}

	nonTypeFields := []SchemaElementKind{
		SchemaElementUnknown,
		SchemaElementResource,
		SchemaElementFunctionCall,
		SchemaElementScalar,
	}

	for _, kind := range nonTypeFields {
		s.Assert().False(kind.IsTypeField(), "%s should not be a type field", kind)
	}
}

func (s *SchemaElementKindSuite) TestIsSubstitution() {
	substitutions := []SchemaElementKind{
		SchemaElementSubstitution,
		SchemaElementFunctionCall,
		SchemaElementVariableRef,
		SchemaElementValueRef,
		SchemaElementResourceRef,
		SchemaElementDataSourceRef,
		SchemaElementChildRef,
		SchemaElementElemRef,
		SchemaElementElemIndexRef,
	}

	for _, kind := range substitutions {
		s.Assert().True(kind.IsSubstitution(), "%s should be a substitution", kind)
	}

	nonSubstitutions := []SchemaElementKind{
		SchemaElementUnknown,
		SchemaElementResource,
		SchemaElementResourceType,
		SchemaElementScalar,
	}

	for _, kind := range nonSubstitutions {
		s.Assert().False(kind.IsSubstitution(), "%s should not be a substitution", kind)
	}
}

func (s *SchemaElementKindSuite) TestIsReference() {
	references := []SchemaElementKind{
		SchemaElementVariableRef,
		SchemaElementValueRef,
		SchemaElementResourceRef,
		SchemaElementDataSourceRef,
		SchemaElementChildRef,
		SchemaElementElemRef,
		SchemaElementElemIndexRef,
	}

	for _, kind := range references {
		s.Assert().True(kind.IsReference(), "%s should be a reference", kind)
	}

	nonReferences := []SchemaElementKind{
		SchemaElementUnknown,
		SchemaElementFunctionCall,
		SchemaElementSubstitution,
		SchemaElementResource,
	}

	for _, kind := range nonReferences {
		s.Assert().False(kind.IsReference(), "%s should not be a reference", kind)
	}
}

func TestSchemaElementKindSuite(t *testing.T) {
	suite.Run(t, new(SchemaElementKindSuite))
}
