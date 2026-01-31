package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

var annotationDefs = map[string]*provider.LinkAnnotationDefinition{
	"aws/lambda/function::aws.dynamodb.lambda.stream.startingPosition": {
		Name:        "aws.dynamodb.lambda.stream.startingPosition",
		Label:       "Starting Position",
		Type:        core.ScalarTypeString,
		Description: "The position in the DynamoDB stream from which to start reading.",
		AllowedValues: []*core.ScalarValue{
			core.ScalarFromString("TRIM_HORIZON"),
			core.ScalarFromString("LATEST"),
		},
	},
}

func createAnnotationHoverService(
	t *testing.T,
	linkRegistry provider.LinkRegistry,
	treeSitter bool,
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

	var docCtx *docmodel.DocumentContext
	if treeSitter {
		docCtx = docmodel.NewDocumentContext(blueprintURI, content, docmodel.FormatYAML, nil)
		docCtx.UpdateSchema(blueprint, tree)
	} else {
		docCtx = docmodel.NewDocumentContextFromSchema(string(blueprintURI), blueprint, tree)
	}

	return service, docCtx
}

// HoverAnnotationSuite tests annotation hover for link annotations,
// covering both link directions and tree-sitter/schema-only contexts.
type HoverAnnotationSuite struct {
	suite.Suite
}

// Test_hover_on_link_annotation_key_target_side verifies that hovering on a link
// annotation key on the target (B) side of a link resolves the annotation definition.
// The DynamoDB table (A) has a linkSelector selecting the Lambda function (B).
func (s *HoverAnnotationSuite) Test_hover_on_link_annotation_key_target_side() {
	linkRegistry := &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{
			"aws/dynamodb/table::aws/lambda/function": &testutils.MockLink{
				AnnotationDefs: annotationDefs,
			},
		},
	}
	service, docCtx := createAnnotationHoverService(s.T(), linkRegistry, false)

	lspCtx := &common.LSPContext{}
	// Line 21 (0-indexed: 20): "        aws.dynamodb.lambda.stream.startingPosition: LATEST"
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 20, Character: 15},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Link Annotation")
	s.Assert().Contains(hoverContent.Value, "aws.dynamodb.lambda.stream.startingPosition")
	s.Assert().Contains(hoverContent.Value, "string")
}

// Test_hover_on_link_annotation_key_reversed_link_direction verifies annotation hover
// works when the link registry direction (Lambda -> DynamoDB) differs from the selector
// direction (DynamoDB selects Lambda via linkSelector).
func (s *HoverAnnotationSuite) Test_hover_on_link_annotation_key_reversed_link_direction() {
	linkRegistry := &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{
			"aws/lambda/function::aws/dynamodb/table": &testutils.MockLink{
				AnnotationDefs: annotationDefs,
			},
		},
	}
	service, docCtx := createAnnotationHoverService(s.T(), linkRegistry, false)

	lspCtx := &common.LSPContext{}
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 20, Character: 15},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Link Annotation")
	s.Assert().Contains(hoverContent.Value, "aws.dynamodb.lambda.stream.startingPosition")
}

// Test_hover_on_link_annotation_key_with_tree_sitter verifies annotation hover
// works with tree-sitter document context, matching real language server usage.
func (s *HoverAnnotationSuite) Test_hover_on_link_annotation_key_with_tree_sitter() {
	linkRegistry := &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{
			"aws/dynamodb/table::aws/lambda/function": &testutils.MockLink{
				AnnotationDefs: annotationDefs,
			},
		},
	}
	service, docCtx := createAnnotationHoverService(s.T(), linkRegistry, true)

	lspCtx := &common.LSPContext{}
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 20, Character: 15},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Link Annotation")
	s.Assert().Contains(hoverContent.Value, "aws.dynamodb.lambda.stream.startingPosition")
}

// Test_hover_on_annotation_key_fallback verifies that hovering on an annotation key
// shows basic annotation info when no link annotation definition is found.
func (s *HoverAnnotationSuite) Test_hover_on_annotation_key_fallback() {
	// Empty link registry - no definitions available
	linkRegistry := &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{},
	}
	service, docCtx := createAnnotationHoverService(s.T(), linkRegistry, false)

	lspCtx := &common.LSPContext{}
	hoverContent, err := service.GetHoverContent(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
		Position:     lsp.Position{Line: 20, Character: 15},
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(hoverContent.Value)
	s.Assert().Contains(hoverContent.Value, "Annotation")
	s.Assert().Contains(hoverContent.Value, "aws.dynamodb.lambda.stream.startingPosition")
	s.Assert().Contains(hoverContent.Value, "LATEST")
}

func TestHoverAnnotationSuite(t *testing.T) {
	suite.Run(t, new(HoverAnnotationSuite))
}
