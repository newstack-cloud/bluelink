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

type HoverServiceSuite struct {
	suite.Suite
	service *HoverService
	docCtx  *docmodel.DocumentContext
}

func (s *HoverServiceSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	funcRegistry := &testutils.FunctionRegistryMock{
		Functions: map[string]provider.Function{},
	}
	resourceRegistry := &testutils.ResourceRegistryMock{}
	dataSourceRegistry := &testutils.DataSourceRegistryMock{}
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

// -- Substitution-level hover tests (updated line numbers for expanded fixture) --

func (s *HoverServiceSuite) Test_hover_on_resource_metadata_annotations_key() {
	lspCtx := &common.LSPContext{}
	// Line 71 (0-indexed: 70): annotationRef: "${resources.ordersTable.metadata.annotations['environment.v1']}"
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 70, Character: 65},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "metadata.annotations")
	s.Assert().Contains(hoverContent.Value, "environment.v1")
}

func (s *HoverServiceSuite) Test_hover_on_resource_metadata_annotations() {
	lspCtx := &common.LSPContext{}
	// Line 73 (0-indexed: 72): allAnnotations: "${resources.ordersTable.metadata.annotations}"
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 72, Character: 53},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "metadata.annotations")
	s.Assert().Contains(hoverContent.Value, "map[string]string")
}

func (s *HoverServiceSuite) Test_hover_on_resource_metadata() {
	lspCtx := &common.LSPContext{}
	// Line 72 (0-indexed: 71): metadataRef: "${resources.ordersTable.metadata}"
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 71, Character: 44},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "metadata")
	s.Assert().Contains(hoverContent.Value, "object")
}

func (s *HoverServiceSuite) Test_hover_on_resource_metadata_labels() {
	lspCtx := &common.LSPContext{}
	// Line 74 (0-indexed: 73): allLabels: "${resources.ordersTable.metadata.labels}"
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 73, Character: 49},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "metadata.labels")
	s.Assert().Contains(hoverContent.Value, "map[string]string")
}

// -- Named element hover tests --
// These hover on map keys where FieldsSourceMeta tracks the key position.

func (s *HoverServiceSuite) Test_hover_on_resource_name() {
	lspCtx := &common.LSPContext{}
	// "  ordersTable:" at line 16 (0-indexed: 15)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 15, Character: 4},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Resource")
	s.Assert().Contains(hoverContent.Value, "ordersTable")
	s.Assert().Contains(hoverContent.Value, "aws/dynamodb/table")
}

func (s *HoverServiceSuite) Test_hover_on_variable_name() {
	lspCtx := &common.LSPContext{}
	// "  environment:" at line 4 (0-indexed: 3)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 3, Character: 4},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Variable")
	s.Assert().Contains(hoverContent.Value, "environment")
	s.Assert().Contains(hoverContent.Value, "string")
}

func (s *HoverServiceSuite) Test_hover_on_value_name() {
	lspCtx := &common.LSPContext{}
	// "  tableName:" at line 10 (0-indexed: 9)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 9, Character: 4},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Value")
	s.Assert().Contains(hoverContent.Value, "tableName")
	s.Assert().Contains(hoverContent.Value, "string")
}

func (s *HoverServiceSuite) Test_hover_on_datasource_name() {
	lspCtx := &common.LSPContext{}
	// "  network:" at line 44 (0-indexed: 43)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 43, Character: 4},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Data Source")
	s.Assert().Contains(hoverContent.Value, "network")
	s.Assert().Contains(hoverContent.Value, "aws/vpc")
}

func (s *HoverServiceSuite) Test_hover_on_export_name_produces_no_hover() {
	lspCtx := &common.LSPContext{}
	// "  tableArn:" at line 65 (0-indexed: 64)
	// Neither SchemaElementExport nor SchemaElementExports support hover,
	// so hovering anywhere in the exports section produces no popup.
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 64, Character: 4},
	})
	s.Require().NoError(err)
	s.Assert().Empty(hoverContent.Value)
}

func (s *HoverServiceSuite) Test_hover_on_include_name() {
	lspCtx := &common.LSPContext{}
	// "  auth:" at line 60 (0-indexed: 59)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 59, Character: 4},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Include")
	s.Assert().Contains(hoverContent.Value, "auth")
	s.Assert().Contains(hoverContent.Value, "./auth-blueprint.yaml")
}

func (s *HoverServiceSuite) Test_hover_on_datasource_export_field() {
	lspCtx := &common.LSPContext{}
	// "      vpcId:" at line 52 (0-indexed: 51)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 51, Character: 6},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Export Field")
	s.Assert().Contains(hoverContent.Value, "vpcId")
	s.Assert().Contains(hoverContent.Value, "string")
}

func (s *HoverServiceSuite) Test_hover_on_metadata_displayName_key() {
	lspCtx := &common.LSPContext{}
	// "      displayName: Orders Table" at line 20 (0-indexed: 19)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 19, Character: 8},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "displayName")
	s.Assert().Contains(hoverContent.Value, "display name")
}

func (s *HoverServiceSuite) Test_hover_on_filter_field_key() {
	lspCtx := &common.LSPContext{}
	// "      field: environment" at line 48 (0-indexed: 47)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 47, Character: 7},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "field")
	s.Assert().Contains(hoverContent.Value, "environment")
	s.Assert().Contains(hoverContent.Value, "filter")
}

func (s *HoverServiceSuite) Test_hover_on_filter_operator_key() {
	lspCtx := &common.LSPContext{}
	// "      operator: "="" at line 49 (0-indexed: 48)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 48, Character: 8},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "operator")
	s.Assert().Contains(hoverContent.Value, "=")
	s.Assert().Contains(hoverContent.Value, "valid operators")
}

func (s *HoverServiceSuite) Test_hover_on_filter_search_key() {
	lspCtx := &common.LSPContext{}
	// "      search: production" at line 50 (0-indexed: 49)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 49, Character: 7},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "search")
	s.Assert().Contains(hoverContent.Value, "match")
}

func TestHoverServiceSuite(t *testing.T) {
	suite.Run(t, new(HoverServiceSuite))
}
