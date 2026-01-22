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
		signatureService,
		logger,
	)
	content, err := loadTestBlueprintContent("blueprint-hover.yaml")
	s.Require().NoError(err)
	blueprint, err := schema.LoadString(content, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)
	s.docCtx = docmodel.NewDocumentContextFromSchema(string(blueprintURI), blueprint, tree)
}

func (s *HoverServiceSuite) Test_hover_on_resource_metadata_annotations_key() {
	lspCtx := &common.LSPContext{}
	// Line 23 (0-indexed: 22): annotationRef: "${resources.ordersTable.metadata.annotations['environment.v1']}"
	// Position on 'environment.v1' (character ~65)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      22,
			Character: 65,
		},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "metadata.annotations")
	s.Assert().Contains(hoverContent.Value, "environment.v1")
}

func (s *HoverServiceSuite) Test_hover_on_resource_metadata_annotations() {
	lspCtx := &common.LSPContext{}
	// Line 25 (0-indexed: 24): allAnnotations: "${resources.ordersTable.metadata.annotations}"
	// Position on 'annotations' (character ~53)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      24,
			Character: 53,
		},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "metadata.annotations")
	s.Assert().Contains(hoverContent.Value, "map[string]string")
}

func (s *HoverServiceSuite) Test_hover_on_resource_metadata() {
	lspCtx := &common.LSPContext{}
	// Line 24 (0-indexed: 23): metadataRef: "${resources.ordersTable.metadata}"
	// Position on 'metadata' (character ~44)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      23,
			Character: 44,
		},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "metadata")
	s.Assert().Contains(hoverContent.Value, "object")
}

func (s *HoverServiceSuite) Test_hover_on_resource_metadata_labels() {
	lspCtx := &common.LSPContext{}
	// Line 26 (0-indexed: 25): allLabels: "${resources.ordersTable.metadata.labels}"
	// Position on 'labels' (character ~49)
	hoverContent, err := s.service.GetHoverContent(lspCtx, s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      25,
			Character: 49,
		},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "metadata.labels")
	s.Assert().Contains(hoverContent.Value, "map[string]string")
}

func TestHoverServiceSuite(t *testing.T) {
	suite.Run(t, new(HoverServiceSuite))
}
