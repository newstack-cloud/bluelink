package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/linkinfo"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func createLinkSelectorHoverService(
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
		linkinfo.NewProviderSource(linkRegistry),
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

type HoverLinkSelectorSuite struct {
	suite.Suite
}

// linkSelector sits on line 10 (0-indexed: 9); the keyword starts at column 4.
const linkSelectorLine = 9
const linkSelectorCharacter = 10

func (s *HoverLinkSelectorSuite) Test_hover_on_linkSelector_without_source_lists_targets_without_cardinality() {
	// With no link source, selector-matched targets still surface but no link
	// cardinality is attached.
	service, docCtx := createLinkSelectorHoverService(s.T(), nil)

	lspCtx := &common.LSPContext{}
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: linkSelectorLine, Character: linkSelectorCharacter},
	})
	s.Require().NoError(err)
	s.Assert().Contains(hoverContent.Value, "linkSelector")
	s.Assert().Contains(hoverContent.Value, "processOrders")
	s.Assert().NotContains(hoverContent.Value, "at most")
	s.Assert().NotContains(hoverContent.Value, "exactly")
}

func (s *HoverLinkSelectorSuite) Test_hover_on_linkSelector_lists_matched_target_with_cardinality() {
	linkRegistry := &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{
			"aws/dynamodb/table::aws/lambda/function": &testutils.MockLink{
				CardinalityA: provider.LinkCardinality{Min: 0, Max: 5},
			},
		},
	}
	service, docCtx := createLinkSelectorHoverService(s.T(), linkRegistry)

	lspCtx := &common.LSPContext{}
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: linkSelectorLine, Character: linkSelectorCharacter},
	})
	s.Require().NoError(err)
	s.Assert().Contains(hoverContent.Value, "Will link to")
	s.Assert().Contains(hoverContent.Value, "processOrders")
	s.Assert().Contains(hoverContent.Value, "aws/lambda/function")
	s.Assert().Contains(hoverContent.Value, "at most 5")
}

func (s *HoverLinkSelectorSuite) Test_hover_on_linkSelector_lists_matched_target_without_cardinality_when_link_missing() {
	service, docCtx := createLinkSelectorHoverService(s.T(), &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{},
	})

	lspCtx := &common.LSPContext{}
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: linkSelectorLine, Character: linkSelectorCharacter},
	})
	s.Require().NoError(err)
	s.Assert().Contains(hoverContent.Value, "Will link to")
	s.Assert().Contains(hoverContent.Value, "processOrders")
	s.Assert().NotContains(hoverContent.Value, "at most")
	s.Assert().NotContains(hoverContent.Value, "exactly")
}

func TestHoverLinkSelectorSuite(t *testing.T) {
	suite.Run(t, new(HoverLinkSelectorSuite))
}
