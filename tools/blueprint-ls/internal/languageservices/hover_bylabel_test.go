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

func createByLabelHoverService(
	t *testing.T,
	linkRegistry provider.LinkRegistry,
) (*HoverService, *docmodel.DocumentContext) {
	t.Helper()

	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}

	funcRegistry := &testutils.FunctionRegistryMock{
		Functions: map[string]provider.Function{},
	}
	signatureService := NewSignatureService(funcRegistry, logger)
	service := NewHoverService(
		funcRegistry,
		&testutils.ResourceRegistryMock{},
		&testutils.DataSourceRegistryMock{},
		linkRegistry,
		signatureService,
		nil,
		logger,
	)

	content, err := loadTestBlueprintContent("blueprint-hover-annotation.yaml")
	if err != nil {
		t.Fatal(err)
	}

	blueprint, err := schema.LoadString(content, schema.YAMLSpecFormat)
	if err != nil {
		t.Fatal(err)
	}

	tree := schema.SchemaToTree(blueprint)
	docCtx := docmodel.NewDocumentContextFromSchema(string(blueprintURI), blueprint, tree)

	return service, docCtx
}

type HoverByLabelSuite struct {
	suite.Suite
}

func (s *HoverByLabelSuite) Test_hover_on_byLabel_with_link_registry_shows_linked_status() {
	linkRegistry := &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{
			"aws/dynamodb/table::aws/lambda/function": &testutils.MockLink{},
		},
	}
	service, docCtx := createByLabelHoverService(s.T(), linkRegistry)

	lspCtx := &common.LSPContext{}
	// Line 12 (0-indexed: 11): "        subsystem: eventProcessing"
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 11, Character: 10},
	})
	s.Require().NoError(err)
	s.Assert().Contains(hoverContent.Value, "processOrders")
	s.Assert().Contains(hoverContent.Value, "aws/lambda/function")
	s.Assert().Contains(hoverContent.Value, "linked")
}

func (s *HoverByLabelSuite) Test_hover_on_byLabel_without_link_registry_shows_no_linked_status() {
	service, docCtx := createByLabelHoverService(s.T(), nil)

	lspCtx := &common.LSPContext{}
	// Line 12 (0-indexed: 11): "        subsystem: eventProcessing"
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 11, Character: 10},
	})
	s.Require().NoError(err)
	s.Assert().Contains(hoverContent.Value, "processOrders")
	s.Assert().NotContains(hoverContent.Value, "linked")
}

func (s *HoverByLabelSuite) Test_hover_on_byLabel_with_reversed_link_direction() {
	linkRegistry := &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{
			"aws/lambda/function::aws/dynamodb/table": &testutils.MockLink{},
		},
	}
	service, docCtx := createByLabelHoverService(s.T(), linkRegistry)

	lspCtx := &common.LSPContext{}
	// Line 12 (0-indexed: 11): "        subsystem: eventProcessing"
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 11, Character: 10},
	})
	s.Require().NoError(err)
	s.Assert().Contains(hoverContent.Value, "processOrders")
	s.Assert().Contains(hoverContent.Value, "linked")
}

func TestHoverByLabelSuite(t *testing.T) {
	suite.Run(t, new(HoverByLabelSuite))
}
