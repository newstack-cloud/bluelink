package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type CompletionContextSuite struct {
	suite.Suite
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceType_ExistingPath() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myTable"},
			{Kind: PathSegmentField, FieldName: "type"},
		},
		TextBefore: "type: ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceType, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceType_NewField() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myTable"},
		},
		TextBefore: "    type: ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceType, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceType() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "type"},
		},
		TextBefore: "type: ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceType, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_VariableType() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "variables"},
			{Kind: PathSegmentField, FieldName: "myVar"},
			{Kind: PathSegmentField, FieldName: "type"},
		},
		TextBefore: "",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextVariableType, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_ValueType() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "values"},
			{Kind: PathSegmentField, FieldName: "myValue"},
			{Kind: PathSegmentField, FieldName: "type"},
		},
		TextBefore: "",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextValueType, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_ExportType() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "exports"},
			{Kind: PathSegmentField, FieldName: "myExport"},
			{Kind: PathSegmentField, FieldName: "type"},
		},
		TextBefore: "",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextExportType, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceFilterField() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "filter"},
			{Kind: PathSegmentIndex, Index: 0},
			{Kind: PathSegmentField, FieldName: "field"},
		},
		TextBefore: "",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceFilterField, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceFilterOperator() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "filter"},
			{Kind: PathSegmentIndex, Index: 0},
			{Kind: PathSegmentField, FieldName: "operator"},
		},
		TextBefore: "",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceFilterOperator, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_NewFilterField() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "filters"},
			{Kind: PathSegmentIndex, Index: 0},
		},
		TextBefore: "      field: ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceFilterField, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_NewFilterOperator() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "filters"},
			{Kind: PathSegmentIndex, Index: 0},
		},
		TextBefore: "      operator: ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceFilterOperator, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubVariableRef() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${variables.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubVariableRef, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceRef() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${resources.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceRef, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceProperty() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${resources.myTable.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceProperty, ctx.Kind)
	s.Assert().Equal("myTable", ctx.ResourceName)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceProperty_WithSchemaElement() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${myTable.spec.",
		SchemaElement: &substitutions.SubstitutionResourceProperty{
			ResourceName: "myTable",
			Path: []*substitutions.SubstitutionPathItem{
				{FieldName: "spec"},
			},
		},
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceProperty, ctx.Kind)
	s.Assert().Equal("myTable", ctx.ResourceName)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubDataSourceRef() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${datasources.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubDataSourceRef, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubDataSourceProperty() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${datasources.myDS.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubDataSourceProperty, ctx.Kind)
	s.Assert().Equal("myDS", ctx.DataSourceName)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubValueRef() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${values.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubValueRef, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubChildRef() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${children.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubChildRef, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubElemRef() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${elem.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubElemRef, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubOpen() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSub, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_Unknown() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "some random text",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextUnknown, ctx.Kind)
}

func (s *CompletionContextSuite) TestCompletionContextKind_String() {
	tests := []struct {
		kind     CompletionContextKind
		expected string
	}{
		{CompletionContextUnknown, "unknown"},
		{CompletionContextResourceType, "resourceType"},
		{CompletionContextDataSourceType, "dataSourceType"},
		{CompletionContextVariableType, "variableType"},
		{CompletionContextValueType, "valueType"},
		{CompletionContextExportType, "exportType"},
		{CompletionContextDataSourceFieldType, "dataSourceFieldType"},
		{CompletionContextDataSourceFilterField, "dataSourceFilterField"},
		{CompletionContextDataSourceFilterOperator, "dataSourceFilterOperator"},
		{CompletionContextStringSub, "stringSub"},
		{CompletionContextStringSubVariableRef, "stringSubVariableRef"},
		{CompletionContextStringSubResourceRef, "stringSubResourceRef"},
		{CompletionContextStringSubResourceProperty, "stringSubResourceProperty"},
		{CompletionContextStringSubDataSourceRef, "stringSubDataSourceRef"},
		{CompletionContextStringSubDataSourceProperty, "stringSubDataSourceProperty"},
		{CompletionContextStringSubValueRef, "stringSubValueRef"},
		{CompletionContextStringSubChildRef, "stringSubChildRef"},
		{CompletionContextStringSubElemRef, "stringSubElemRef"},
	}

	for _, tt := range tests {
		s.Run(tt.expected, func() {
			s.Assert().Equal(tt.expected, tt.kind.String())
		})
	}
}

func (s *CompletionContextSuite) TestCompletionContextKind_IsTypeField() {
	typeFields := []CompletionContextKind{
		CompletionContextResourceType,
		CompletionContextDataSourceType,
		CompletionContextVariableType,
		CompletionContextValueType,
		CompletionContextExportType,
		CompletionContextDataSourceFieldType,
	}

	for _, kind := range typeFields {
		s.Assert().True(kind.IsTypeField(), "expected %s to be a type field", kind)
	}

	s.Assert().False(CompletionContextStringSub.IsTypeField())
	s.Assert().False(CompletionContextUnknown.IsTypeField())
}

func (s *CompletionContextSuite) TestCompletionContextKind_IsSubstitution() {
	subKinds := []CompletionContextKind{
		CompletionContextStringSub,
		CompletionContextStringSubVariableRef,
		CompletionContextStringSubResourceRef,
		CompletionContextStringSubResourceProperty,
		CompletionContextStringSubDataSourceRef,
		CompletionContextStringSubDataSourceProperty,
		CompletionContextStringSubValueRef,
		CompletionContextStringSubChildRef,
		CompletionContextStringSubElemRef,
	}

	for _, kind := range subKinds {
		s.Assert().True(kind.IsSubstitution(), "expected %s to be a substitution", kind)
	}

	s.Assert().False(CompletionContextResourceType.IsSubstitution())
	s.Assert().False(CompletionContextUnknown.IsSubstitution())
}

func (s *CompletionContextSuite) TestCompletionContextKind_IsDataSourceFilter() {
	s.Assert().True(CompletionContextDataSourceFilterField.IsDataSourceFilter())
	s.Assert().True(CompletionContextDataSourceFilterOperator.IsDataSourceFilter())
	s.Assert().False(CompletionContextDataSourceType.IsDataSourceFilter())
	s.Assert().False(CompletionContextUnknown.IsDataSourceFilter())
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_Integration_WithDocumentContext() {
	content := `version: 2021-12-18
resources:
  myTable:
    type: aws/dynamodb/table`

	docCtx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	// Position at "type: aws/dynamodb/table" - on the value
	pos := source.Position{Line: 4, Column: 15}
	nodeCtx := docCtx.GetNodeContext(pos, 2)

	// Verify the context was created
	s.Require().NotNil(nodeCtx)
	s.Assert().NotNil(nodeCtx.DocumentCtx)

	// The text context should be extracted
	s.Assert().Contains(nodeCtx.TextBefore, "type:")

	// The CompletionContext detection should work
	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().NotNil(ctx)
	s.Assert().NotNil(ctx.NodeCtx)

	// Note: The integration between tree-sitter AST paths and schema-style paths
	// requires additional work in Phase 5/6 to fully map YAML structure to
	// blueprint paths. For now, text-based detection is the fallback.
	// TODO: Come back and expand this test when schema mapping is implemented.
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceType_JSON() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myTable"},
		},
		TextBefore: `      "type": `,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceType, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceFilterField_JSON() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "filters"},
			{Kind: PathSegmentIndex, Index: 0},
		},
		TextBefore: `        "field": `,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceFilterField, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceFilterOperator_JSON() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "filters"},
			{Kind: PathSegmentIndex, Index: 0},
		},
		TextBefore: `        "operator": `,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceFilterOperator, ctx.Kind)
}

func TestCompletionContextSuite(t *testing.T) {
	suite.Run(t, new(CompletionContextSuite))
}
