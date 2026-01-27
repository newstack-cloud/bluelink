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
			{Kind: PathSegmentField, FieldName: "filters"},
			{Kind: PathSegmentIndex, Index: 0},
			{Kind: PathSegmentField, FieldName: "filter"},
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
			{Kind: PathSegmentField, FieldName: "filters"},
			{Kind: PathSegmentIndex, Index: 0},
			{Kind: PathSegmentField, FieldName: "filter"},
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

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceProperty_WithSchemaElement_BracketNotation() {
	// Standalone resource name with bracket notation: ${myTable.metadata.annotations[
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${myTable.metadata.annotations[",
		SchemaElement: &substitutions.SubstitutionResourceProperty{
			ResourceName: "myTable",
			Path: []*substitutions.SubstitutionPathItem{
				{FieldName: "metadata"},
				{FieldName: "annotations"},
			},
		},
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceProperty, ctx.Kind)
	s.Assert().Equal("myTable", ctx.ResourceName)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceProperty_BracketNotation() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${resources.myTable.metadata.annotations[",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceProperty, ctx.Kind)
	s.Assert().Equal("myTable", ctx.ResourceName)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceProperty_BracketOnResource() {
	// resources.myTable[ - bracket directly on resource name (for array access)
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${resources.myTable[",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceProperty, ctx.Kind)
	s.Assert().Equal("myTable", ctx.ResourceName)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceRef_BracketNotation() {
	// resources[ - bracket on resources namespace
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${resources[",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceRef, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubDataSourceProperty_BracketNotation() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${datasources.myDS.exports[",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubDataSourceProperty, ctx.Kind)
	s.Assert().Equal("myDS", ctx.DataSourceName)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubDataSourceRef_BracketNotation() {
	// datasources[ - bracket on datasources namespace
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${datasources[",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubDataSourceRef, ctx.Kind)
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
	// Test case: at a value position (after colon) with no specific context
	// This should return Unknown because we're not at a key position
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "someKey: someValue",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextUnknown, ctx.Kind)
}

func (s *CompletionContextSuite) TestDetermineCompletionContext_BlueprintTopLevel_EmptyPath() {
	// Test case: at document root level with empty path (typing new top-level field)
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "res",
		Position:   source.Position{Line: 1, Column: 4},
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextBlueprintTopLevelField, ctx.Kind)
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

// TestDetermineCompletionContext_StringSubResourceProperty_JSONC tests resource property
// completion in JSONC format where text is wrapped in quotes.
func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceProperty_JSONC() {
	// Simulates cursor after typing . in "${resources.myTable.spec.}"
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: `"${resources.myTable.spec.`,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceProperty, ctx.Kind)
	s.Assert().Equal("myTable", ctx.ResourceName)
}

// TestDetermineCompletionContext_StringSubResourceProperty_JSONC_TopLevel tests resource
// property completion for top-level properties in JSONC format.
func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceProperty_JSONC_TopLevel() {
	// Simulates cursor after typing . in "${resources.myTable.}"
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: `"${resources.myTable.`,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceProperty, ctx.Kind)
	s.Assert().Equal("myTable", ctx.ResourceName)
}

// TestDetermineCompletionContext_StringSubOpen_JSONC tests completion trigger at ${
// in JSONC format.
func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubOpen_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: `"${`,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSub, ctx.Kind)
}

// TestDetermineCompletionContext_StringSubResourceRef_JSONC tests resource reference
// completion in JSONC format.
func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubResourceRef_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: `"${resources.`,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubResourceRef, ctx.Kind)
}

// TestDetermineCompletionContext_StringSubPartialPath tests that typing a partial property
// name without trailing dot returns no completions.
func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubPartialPath() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: `${resources.myTable.metadata`,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPartialPath, ctx.Kind)
}

// TestDetermineCompletionContext_StringSubPartialPath_JSONC tests partial path in JSONC.
func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubPartialPath_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: `"${resources.myTable.spec`,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPartialPath, ctx.Kind)
}

// TestDetermineCompletionContext_StringSubPartialPath_Deeper tests partial path deeper in path.
func (s *CompletionContextSuite) TestDetermineCompletionContext_StringSubPartialPath_Deeper() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: `${resources.myTable.metadata.annotations`,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPartialPath, ctx.Kind)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_DotNotation tests detection of
// potential standalone resource property patterns like ${myResource.
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_DotNotation() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${myResource.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPotentialResourceProp, ctx.Kind)
	s.Assert().Equal("myResource", ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_BracketNotation tests detection of
// potential standalone resource property patterns like ${myResource[
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_BracketNotation() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${myResource[",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPotentialResourceProp, ctx.Kind)
	s.Assert().Equal("myResource", ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_NestedPath tests detection of
// potential standalone resource property patterns with nested paths like ${myResource.metadata.
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_NestedPath() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${myResource.metadata.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPotentialResourceProp, ctx.Kind)
	s.Assert().Equal("myResource", ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_NestedBracket tests detection of
// potential standalone resource property patterns with nested bracket notation like ${myResource.metadata[
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_NestedBracket() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${myResource.metadata.annotations[",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPotentialResourceProp, ctx.Kind)
	s.Assert().Equal("myResource", ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_NotReserved_Resources tests that
// reserved namespaces like 'resources' are not detected as potential standalone resources.
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_NotReserved_Resources() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${resources.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	// Should be detected as StringSubResourceRef, not PotentialResourceProp
	s.Assert().Equal(CompletionContextStringSubResourceRef, ctx.Kind)
	s.Assert().Empty(ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_NotReserved_Variables tests that
// reserved namespaces like 'variables' are not detected as potential standalone resources.
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_NotReserved_Variables() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${variables.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	// Should be detected as StringSubVariableRef, not PotentialResourceProp
	s.Assert().Equal(CompletionContextStringSubVariableRef, ctx.Kind)
	s.Assert().Empty(ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_NotReserved_Values tests that
// reserved namespaces like 'values' are not detected as potential standalone resources.
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_NotReserved_Values() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${values.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	// Should be detected as StringSubValueRef, not PotentialResourceProp
	s.Assert().Equal(CompletionContextStringSubValueRef, ctx.Kind)
	s.Assert().Empty(ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_JSONC tests detection in JSONC format.
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: `"${myTable.`,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPotentialResourceProp, ctx.Kind)
	s.Assert().Equal("myTable", ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_WithHyphen tests resource names with hyphens.
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_WithHyphen() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${my-resource.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPotentialResourceProp, ctx.Kind)
	s.Assert().Equal("my-resource", ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_PotentialStandaloneResourceProp_WithUnderscore tests resource names with underscores.
func (s *CompletionContextSuite) TestDetermineCompletionContext_PotentialStandaloneResourceProp_WithUnderscore() {
	nodeCtx := &NodeContext{
		ASTPath:    StructuredPath{},
		TextBefore: "${my_resource.",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextStringSubPotentialResourceProp, ctx.Kind)
	s.Assert().Equal("my_resource", ctx.PotentialResourceName)
}

// TestDetermineCompletionContext_ResourceSpecField_TopLevel tests detection of resource spec
// field completion at the top level of spec (e.g., /resources/{name}/spec/).
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceSpecField_TopLevel() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myHandler"},
			{Kind: PathSegmentField, FieldName: "spec"},
		},
		TextBefore: "  spec:\n    ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecField, ctx.Kind)
	s.Assert().Equal("myHandler", ctx.ResourceName)
}

// TestDetermineCompletionContext_ResourceSpecField_Nested tests detection of resource spec
// field completion in nested spec paths (e.g., /resources/{name}/spec/nested/).
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceSpecField_Nested() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myHandler"},
			{Kind: PathSegmentField, FieldName: "spec"},
			{Kind: PathSegmentField, FieldName: "runtime"},
		},
		TextBefore: "      runtime:\n        ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecField, ctx.Kind)
	s.Assert().Equal("myHandler", ctx.ResourceName)
}

// TestDetermineCompletionContext_ResourceSpecField_NotInSubstitution tests that resource spec
// field detection is skipped when inside a substitution.
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceSpecField_NotInSubstitution() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myHandler"},
			{Kind: PathSegmentField, FieldName: "spec"},
		},
		TextBefore: `"${variables.`,
		SchemaElement: &substitutions.SubstitutionVariable{
			VariableName: "myVar",
		},
	}

	ctx := DetermineCompletionContext(nodeCtx)
	// Should detect the substitution context, not the resource spec field
	s.Assert().Equal(CompletionContextStringSubVariableRef, ctx.Kind)
}

// TestDetermineCompletionContext_ResourceSpecField_JSONC tests resource spec field detection in JSONC.
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceSpecField_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myHandler"},
			{Kind: PathSegmentField, FieldName: "spec"},
		},
		TextBefore: `"spec": { `,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecField, ctx.Kind)
	s.Assert().Equal("myHandler", ctx.ResourceName)
}

// TestDetermineCompletionContext_ResourceDefinitionField_YAML tests detection of resource
// definition field completion at the resource level (for fields like type, description, spec, etc.).
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceDefinitionField_YAML() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myHandler"},
		},
		TextBefore: "  myHandler:\n    ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceDefinitionField, ctx.Kind)
	s.Assert().Equal("myHandler", ctx.ResourceName)
}

// TestDetermineCompletionContext_ResourceDefinitionField_JSONC tests resource definition
// field completion in JSONC.
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceDefinitionField_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myHandler"},
		},
		TextBefore: `"myHandler": { `,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceDefinitionField, ctx.Kind)
	s.Assert().Equal("myHandler", ctx.ResourceName)
}

// TestDetermineCompletionContext_ResourceDefinitionField_WithPrefix tests resource definition
// field completion when user is typing a field name.
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceDefinitionField_WithPrefix() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myHandler"},
		},
		TextBefore: "  myHandler:\n    typ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceDefinitionField, ctx.Kind)
	s.Assert().Equal("myHandler", ctx.ResourceName)
}

// TestDetermineCompletionContext_DataSourceFilterDefinitionField tests detection of
// data source filter definition field completion (for fields like field, operator, search).
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceFilterDefinitionField_YAML() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "filter"},
		},
		TextBefore: "    filter:\n      ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceFilterDefinitionField, ctx.Kind)
}

// TestDetermineCompletionContext_DataSourceFilterDefinitionField_AtIndex tests detection
// at a specific filter array index.
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceFilterDefinitionField_AtIndex() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "filter"},
			{Kind: PathSegmentIndex, Index: 0},
		},
		TextBefore: "      - ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceFilterDefinitionField, ctx.Kind)
}

// TestDetermineCompletionContext_DataSourceFilterDefinitionField_Plural tests detection
// with plural "filters" field name (from validation paths).
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceFilterDefinitionField_Plural() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "filters"},
		},
		TextBefore: "    filters:\n      ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceFilterDefinitionField, ctx.Kind)
}

// TestDetermineCompletionContext_DataSourceExportDefinitionField tests detection of
// data source export definition field completion (for fields like type, aliasFor, description).
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceExportDefinitionField_YAML() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "exports"},
			{Kind: PathSegmentField, FieldName: "vpcId"},
		},
		TextBefore: "      vpcId:\n        ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceExportDefinitionField, ctx.Kind)
}

// TestDetermineCompletionContext_DataSourceExportDefinitionField_JSONC tests export definition
// field completion in JSONC format.
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceExportDefinitionField_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "exports"},
			{Kind: PathSegmentField, FieldName: "vpcId"},
		},
		TextBefore: `        "vpcId": { `,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceExportDefinitionField, ctx.Kind)
}

// TestDetermineCompletionContext_DataSourceExportName_AtSiblingIndent tests that
// when the cursor is at export name indent level (same level as existing exports),
// we get export NAME completions, not export definition field completions.
// This handles the case where user wants to add a new export after existing ones.
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceExportName_AtSiblingIndent() {
	// Path is at export definition level (4 segments: /datasources/myDS/exports/vpcId)
	// But cursor indent is at the export name level (same as vpcId), indicating
	// user wants to add a new sibling export, not a field inside vpcId.
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "exports"},
			{Kind: PathSegmentField, FieldName: "vpcId"},
		},
		TextBefore: "      ", // 6 spaces - same indent as export names
		UnifiedNode: &UnifiedNode{
			FieldName: "vpcId",
			Range: source.Range{
				Start: &source.Position{Line: 5, Column: 7}, // 1-based, so 0-based indent is 6
			},
		},
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceExportName, ctx.Kind,
		"Should return export name context when at sibling indent level")
}

// TestDetermineCompletionContext_DataSourceExportDefinitionField_AtChildIndent tests that
// when the cursor is at export definition field indent level (deeper than export name),
// we get export definition FIELD completions.
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceExportDefinitionField_AtChildIndent() {
	// Path is at export definition level (4 segments: /datasources/myDS/exports/vpcId)
	// Cursor indent is deeper than the export name level, indicating user wants
	// to add a field inside vpcId (like type, aliasFor, description).
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "exports"},
			{Kind: PathSegmentField, FieldName: "vpcId"},
		},
		TextBefore: "        ", // 8 spaces - deeper than export names (6 spaces)
		UnifiedNode: &UnifiedNode{
			FieldName: "vpcId",
			Range: source.Range{
				Start: &source.Position{Line: 5, Column: 7}, // 1-based, so 0-based indent is 6
			},
		},
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceExportDefinitionField, ctx.Kind,
		"Should return export definition field context when at child indent level")
}

// TestDetermineCompletionContext_CustomVariableTypeValue_TextBased tests that
// custom variable type value completions work when cursor is right after "default:"
// using text-based detection (when path doesn't include "default" segment yet).
func (s *CompletionContextSuite) TestDetermineCompletionContext_CustomVariableTypeValue_TextBased() {
	// Path is at variable definition level (2 segments: /variables/instanceType)
	// But TextBefore ends with "default:", indicating user wants default value completions
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "variables"},
			{Kind: PathSegmentField, FieldName: "instanceType"},
		},
		TextBefore: "    default: ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextCustomVariableTypeValue, ctx.Kind,
		"Should return custom variable type value context when text ends with 'default:'")
}

// TestDetermineCompletionContext_CustomVariableTypeValue_TextBased_JSONC tests the
// text-based fallback for JSONC format.
func (s *CompletionContextSuite) TestDetermineCompletionContext_CustomVariableTypeValue_TextBased_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "variables"},
			{Kind: PathSegmentField, FieldName: "instanceType"},
		},
		TextBefore: `      "default": `,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextCustomVariableTypeValue, ctx.Kind,
		"Should return custom variable type value context for JSONC 'default':  pattern")
}

// TestDetermineCompletionContext_DataSourceMetadataField tests detection of
// data source metadata field completion (for fields like displayName, annotations, custom).
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceMetadataField_YAML() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "metadata"},
		},
		TextBefore: "    metadata:\n      ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceMetadataField, ctx.Kind)
}

// TestDetermineCompletionContext_DataSourceMetadataField_JSONC tests metadata field
// completion in JSONC format.
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceMetadataField_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "metadata"},
		},
		TextBefore: `      "metadata": { `,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceMetadataField, ctx.Kind)
}

// TestDetermineCompletionContext_DataSourceMetadataField_Nested tests detection when
// inside nested metadata like annotations.
func (s *CompletionContextSuite) TestDetermineCompletionContext_DataSourceMetadataField_Nested() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "datasources"},
			{Kind: PathSegmentField, FieldName: "myDS"},
			{Kind: PathSegmentField, FieldName: "metadata"},
			{Kind: PathSegmentField, FieldName: "annotations"},
		},
		TextBefore: "      annotations:\n        ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceMetadataField, ctx.Kind)
}

// TestCompletionContextKind_String_NewDataSourceContexts tests the string representation
// of the new data source completion context kinds.
func (s *CompletionContextSuite) TestCompletionContextKind_String_NewDataSourceContexts() {
	tests := []struct {
		kind     CompletionContextKind
		expected string
	}{
		{CompletionContextDataSourceFilterDefinitionField, "dataSourceFilterDefinitionField"},
		{CompletionContextDataSourceExportDefinitionField, "dataSourceExportDefinitionField"},
		{CompletionContextDataSourceMetadataField, "dataSourceMetadataField"},
	}

	for _, tt := range tests {
		s.Run(tt.expected, func() {
			s.Assert().Equal(tt.expected, tt.kind.String())
		})
	}
}

// TestDetermineCompletionContext_ResourceSpecFieldValue_FullPath tests detection of resource spec
// field value completion when the full path includes the field name.
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceSpecFieldValue_FullPath() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myFunction"},
			{Kind: PathSegmentField, FieldName: "spec"},
			{Kind: PathSegmentField, FieldName: "architecture"},
		},
		TextBefore: "      architecture: ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecFieldValue, ctx.Kind)
	s.Assert().Equal("myFunction", ctx.ResourceName)
}

// TestDetermineCompletionContext_ResourceSpecFieldValue_Fallback tests detection of resource spec
// field value completion when the path is only at spec level but we can extract the field name
// from TextBefore. This handles the case where the cursor is positioned after "fieldName: "
// but outside the AST node's range.
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceSpecFieldValue_Fallback() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myFunction"},
			{Kind: PathSegmentField, FieldName: "spec"},
		},
		TextBefore: "      architecture: ",
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecFieldValue, ctx.Kind)
	s.Assert().Equal("myFunction", ctx.ResourceName)
	s.Assert().Equal("architecture", ctx.NodeCtx.ExtractedFieldName)
}

// TestDetermineCompletionContext_ResourceSpecFieldValue_Fallback_JSONC tests the fallback
// with JSONC syntax (quoted field names).
func (s *CompletionContextSuite) TestDetermineCompletionContext_ResourceSpecFieldValue_Fallback_JSONC() {
	nodeCtx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myFunction"},
			{Kind: PathSegmentField, FieldName: "spec"},
		},
		TextBefore: `      "architecture": `,
	}

	ctx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecFieldValue, ctx.Kind)
	s.Assert().Equal("myFunction", ctx.ResourceName)
	s.Assert().Equal("architecture", ctx.NodeCtx.ExtractedFieldName)
}

// TestExtractFieldNameFromTextBefore tests the helper function for extracting field names.
func (s *CompletionContextSuite) TestExtractFieldNameFromTextBefore() {
	tests := []struct {
		name       string
		textBefore string
		expected   string
	}{
		{"YAML with space", "      architecture: ", "architecture"},
		{"YAML without space", "      architecture:", "architecture"},
		{"JSONC with space", `      "architecture": `, "architecture"},
		{"JSONC without space", `      "architecture":`, "architecture"},
		{"field with underscore", "      my_field: ", "my_field"},
		{"field with hyphen", "      my-field: ", "my-field"},
		{"field with numbers", "      field123: ", "field123"},
		{"no match - key position", "      arch", ""},
		{"no match - in value", "      architecture: someValue", ""},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := extractFieldNameFromTextBefore(tt.textBefore)
			s.Assert().Equal(tt.expected, result)
		})
	}
}

func TestCompletionContextSuite(t *testing.T) {
	suite.Run(t, new(CompletionContextSuite))
}
