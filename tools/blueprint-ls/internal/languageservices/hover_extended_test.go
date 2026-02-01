package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type HoverExtendedSuite struct {
	suite.Suite
	service *HoverService
	docCtx  *docmodel.DocumentContext
}

func (s *HoverExtendedSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	funcRegistry := &testutils.FunctionRegistryMock{
		Functions: map[string]provider.Function{},
	}
	resourceRegistry := &testutils.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"aws/dynamodb/table": &testutils.DynamoDBTableResource{},
		},
	}
	dataSourceRegistry := &testutils.DataSourceRegistryMock{
		DataSources: map[string]provider.DataSource{
			"aws/vpc": &testutils.VPCDataSource{},
		},
	}
	signatureService := NewSignatureService(funcRegistry, logger)
	s.service = NewHoverService(
		funcRegistry,
		resourceRegistry,
		dataSourceRegistry,
		nil, // linkRegistry
		signatureService,
		nil, // childResolver
		logger,
	)

	content, err := loadTestBlueprintContent("blueprint-hover.yaml")
	s.Require().NoError(err)
	blueprint, err := schema.LoadString(content, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)
	s.docCtx = docmodel.NewDocumentContextFromSchema(string(blueprintURI), blueprint, tree)
}

func (s *HoverExtendedSuite) hoverAt(line, character int) (*HoverContent, error) {
	return s.service.GetHoverContent(
		&common.LSPContext{},
		s.docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: lsp.UInteger(line), Character: lsp.UInteger(character)},
		},
	)
}

func (s *HoverExtendedSuite) hoverOnDocCtx(
	docCtx *docmodel.DocumentContext,
	line, character int,
) (*HoverContent, error) {
	return s.service.GetHoverContent(
		&common.LSPContext{},
		docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: lsp.UInteger(line), Character: lsp.UInteger(character)},
		},
	)
}

func (s *HoverExtendedSuite) loadInlineBlueprint(content string) *docmodel.DocumentContext {
	blueprint, err := schema.LoadString(content, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)
	return docmodel.NewDocumentContextFromSchema(string(blueprintURI), blueprint, tree)
}

// -- Export field value hover tests (fixture-based) --

func (s *HoverExtendedSuite) Test_hover_export_field_resource_ref() {
	// Line 66 (0-indexed): '    field: "resources.ordersTable.state.arn"'
	// Cursor on "ordersTable" (char 25)
	content, err := s.hoverAt(66, 25)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "ordersTable")
}

// -- Export field value hover tests --

func (s *HoverExtendedSuite) Test_hover_export_field_value_child() {
	bp := "version: 2025-11-02\n" +
		"include:\n" +
		"  auth:\n" +
		"    path: ./auth.yaml\n" +
		"    description: Auth blueprint\n" +
		"exports:\n" +
		"  authToken:\n" +
		"    type: string\n" +
		"    field: children.auth.token"
	docCtx := s.loadInlineBlueprint(bp)
	// Line 8: "    field: children.auth.token"
	// Cursor on "auth" (char ~19)
	content, err := s.hoverOnDocCtx(docCtx, 8, 20)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "auth")
}

func (s *HoverExtendedSuite) Test_hover_export_field_value_value() {
	bp := "version: 2025-11-02\n" +
		"values:\n" +
		"  tableName:\n" +
		"    type: string\n" +
		"    value: test\n" +
		"    description: The table name\n" +
		"exports:\n" +
		"  nameExport:\n" +
		"    type: string\n" +
		"    field: values.tableName"
	docCtx := s.loadInlineBlueprint(bp)
	// Line 9: "    field: values.tableName"
	// Cursor on "tableName" (char ~17)
	content, err := s.hoverOnDocCtx(docCtx, 9, 20)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "tableName")
}

func (s *HoverExtendedSuite) Test_hover_datasource_metadata_key() {
	// Line 45 (0-indexed): "    description: ..." is not metadata.
	// DataSource metadata hover requires a datasource with metadata field.
	bp := "version: 2025-11-02\n" +
		"datasources:\n" +
		"  network:\n" +
		"    type: aws/vpc\n" +
		"    metadata:\n" +
		"      displayName: Network\n" +
		"    filter:\n" +
		"      field: vpcId"
	docCtx := s.loadInlineBlueprint(bp)
	// Line 4: "    metadata:"
	content, err := s.hoverOnDocCtx(docCtx, 4, 6)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "metadata")
}

func (s *HoverExtendedSuite) Test_hover_datasource_filter_operator_value() {
	// Line 48 (0-indexed): '      operator: "="'
	// The DataSourceFilterOperatorWrapper schema element is at the value position
	content, err := s.hoverAt(48, 17)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "operator")
}

// -- Resource/datasource type with registry --

func (s *HoverExtendedSuite) Test_hover_resource_type_with_registry() {
	// Line 16 (0-indexed): "    type: aws/dynamodb/table"
	content, err := s.hoverAt(16, 12)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "DynamoDB Table")
}

func (s *HoverExtendedSuite) Test_hover_datasource_type_with_registry() {
	// Line 44 (0-indexed): "    type: aws/vpc"
	content, err := s.hoverAt(44, 12)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "VPC")
}

// -- Spec field mapping node hover (with registry) --

func (s *HoverExtendedSuite) Test_hover_spec_field_tableName() {
	// Line 30 (0-indexed): "      tableName: orders"
	content, err := s.hoverAt(30, 8)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "tableName")
}

// -- DataSource exports map hover --

func (s *HoverExtendedSuite) Test_hover_datasource_exports_map() {
	// Line 50 (0-indexed): "    exports:"
	content, err := s.hoverAt(50, 6)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "exports")
}

// -- DataSource export field with spec lookup --

func (s *HoverExtendedSuite) Test_hover_datasource_export_field_with_spec() {
	// Line 51 (0-indexed): "      vpcId:"
	content, err := s.hoverAt(51, 8)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "vpcId")
	s.Assert().Contains(content.Value, "string")
}

// -- Variable ref substitution hover --

func (s *HoverExtendedSuite) Test_hover_variable_ref_substitution() {
	// Line 11 (0-indexed): '    value: "orders-${variables.environment}"'
	// The substitution ${variables.environment} starts around char 19
	// Hovering at char 35 should be within "environment"
	content, err := s.hoverAt(11, 35)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "environment")
}

// -- DataSource filter node hover --

func (s *HoverExtendedSuite) Test_hover_datasource_filter_node() {
	// Line 46 (0-indexed): "    filter:"
	content, err := s.hoverAt(46, 6)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "filter")
}

// -- Inline blueprint hover tests for elements not in the fixture --

func (s *HoverExtendedSuite) Test_hover_export_field_datasource_ref() {
	// Line 70 (0-indexed): "    field: datasources.network.vpcId"
	// Cursor on "vpcId" (char 33)
	content, err := s.hoverAt(70, 33)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "vpcId")
}

func (s *HoverExtendedSuite) Test_hover_export_field_variable_ref() {
	// Line 74 (0-indexed): "    field: variables.environment"
	// Cursor on "environment" (char 25)
	content, err := s.hoverAt(74, 25)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "environment")
}

func (s *HoverExtendedSuite) Test_hover_datasource_ref_substitution() {
	bp := "version: 2025-11-02\n" +
		"datasources:\n" +
		"  network:\n" +
		"    type: aws/vpc\n" +
		"    filter:\n" +
		"      field: vpcId\n" +
		"    exports:\n" +
		"      vpcId:\n" +
		"        type: string\n" +
		"resources:\n" +
		"  ordersTable:\n" +
		"    type: aws/dynamodb/table\n" +
		"    spec:\n" +
		"      tableName: \"${datasources.network.vpcId}\""
	docCtx := s.loadInlineBlueprint(bp)
	// Line 13: '      tableName: "${datasources.network.vpcId}"'
	// ${datasources.network.vpcId} starts at char 18
	// Hovering at char 36 should be within "vpcId"
	content, err := s.hoverOnDocCtx(docCtx, 13, 42)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "vpcId")
}

func (s *HoverExtendedSuite) Test_hover_resource_ref_no_path() {
	bp := "version: 2025-11-02\n" +
		"resources:\n" +
		"  ordersTable:\n" +
		"    type: aws/dynamodb/table\n" +
		"    spec:\n" +
		"      tableName: orders\n" +
		"  otherResource:\n" +
		"    type: aws/dynamodb/table\n" +
		"    spec:\n" +
		"      tableName: \"${resources.ordersTable}\""
	docCtx := s.loadInlineBlueprint(bp)
	// Line 9: '      tableName: "${resources.ordersTable}"'
	// Hovering at char 33 should be within "ordersTable"
	content, err := s.hoverOnDocCtx(docCtx, 9, 33)
	s.Require().NoError(err)
	s.Assert().NotEmpty(content.Value)
	s.Assert().Contains(content.Value, "ordersTable")
	s.Assert().Contains(content.Value, "aws/dynamodb/table")
}

func TestHoverExtendedSuite(t *testing.T) {
	suite.Run(t, new(HoverExtendedSuite))
}
