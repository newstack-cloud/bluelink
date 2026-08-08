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

const openEndedRangeBlueprint = `version: "2025-11-02"
variables:
  environment:
    type: string
    description: The environment to deploy to
    default: dev
values:
  tableLabel:
    type: string
    value: "${variables.environment}-orders"
resources:
  ordersTable:
    type: aws/dynamodb/table
    metadata:
      displayName: Orders
      labels:
        app: orders
    condition:
      and:
        - stringValue: "true"
        - not:
            stringValue: "false"
    linkSelector:
      byLabel:
        app: orders
    spec:
      tableName: orders
exports:
  tableName:
    type: string
    field: resources.ordersTable.spec.tableName
`

// OpenEndedRangeSuite documents which blueprint tree nodes carry a source range
// with no end position.
//
// The root is open ended by design, because the last leaf's range runs to the
// end of the document, and mapping node sections such as metadata and
// linkSelector turn out to be open ended too. Position conversion has to
// support all of them. No hover, definition or reference path is currently
// known to convert one of these ranges, so this suite records the shapes that
// conversion must keep handling rather than reproducing a specific failure.
type OpenEndedRangeSuite struct {
	suite.Suite
}

func (s *OpenEndedRangeSuite) tree() *schema.TreeNode {
	blueprint, err := schema.LoadString(openEndedRangeBlueprint, schema.YAMLSpecFormat)
	s.Require().NoError(err)

	tree := schema.SchemaToTree(blueprint)
	s.Require().NotNil(tree)

	return tree
}

func (s *OpenEndedRangeSuite) Test_root_node_range_is_open_ended() {
	tree := s.tree()

	s.Require().NotNil(tree.Range)
	s.Require().NotNil(tree.Range.Start)
	s.Nil(tree.Range.End, "root node range is open ended by design")
}

// Mapping node sections such as metadata and linkSelector keep an open ended
// range, so open endedness is not confined to the root and reaches nodes users
// routinely put their cursor on.
func (s *OpenEndedRangeSuite) Test_mapping_node_sections_have_open_ended_ranges() {
	tree := s.tree()

	openEnded := map[string]bool{}
	walkTreeNodes(tree, func(node *schema.TreeNode) {
		if node.Range != nil && node.Range.End == nil {
			openEnded[node.Path] = true
		}
	})

	s.True(openEnded["/resources/ordersTable/metadata"])
	s.True(openEnded["/resources/ordersTable/metadata/labels"])
	s.True(openEnded["/resources/ordersTable/linkSelector"])
}

// Every node carrying a range must carry a start, since position conversion
// cannot place a node without one.
func (s *OpenEndedRangeSuite) Test_every_range_has_a_start_position() {
	tree := s.tree()

	missingStart := []string{}
	walkTreeNodes(tree, func(node *schema.TreeNode) {
		if node.Range != nil && node.Range.Start == nil {
			missingStart = append(missingStart, node.Path)
		}
	})

	s.Empty(missingStart)
}

// A smoke test over the sections with open ended ranges. It passes with or
// without nil end handling in position conversion today, and exists to catch a
// future hover path that starts converting these nodes' own ranges.
func (s *OpenEndedRangeSuite) Test_hover_over_open_ended_sections() {
	logger := zap.NewNop()
	funcRegistry := &testutils.FunctionRegistryMock{Functions: map[string]provider.Function{}}
	service := NewHoverService(
		funcRegistry,
		&testutils.ResourceRegistryMock{},
		&testutils.DataSourceRegistryMock{},
		nil, // linkRegistry
		NewSignatureService(funcRegistry, logger),
		nil, // childResolver
		logger,
	)

	blueprint, err := schema.LoadString(openEndedRangeBlueprint, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	docCtx := docmodel.NewDocumentContextFromSchema(
		string(blueprintURI),
		blueprint,
		schema.SchemaToTree(blueprint),
	)

	positions := map[string]lsp.Position{
		"metadata key":     {Line: 13, Character: 4},
		"labels key":       {Line: 15, Character: 6},
		"label entry":      {Line: 16, Character: 8},
		"linkSelector key": {Line: 22, Character: 4},
		"byLabel entry":    {Line: 24, Character: 8},
	}

	for name, position := range positions {
		s.Run(name, func() {
			_, err := service.GetHoverContent(
				&common.LSPContext{},
				docCtx,
				&lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
					Position:     position,
				},
			)
			s.Require().NoError(err)
		})
	}
}

func walkTreeNodes(node *schema.TreeNode, visit func(*schema.TreeNode)) {
	if node == nil {
		return
	}

	visit(node)
	for _, child := range node.Children {
		walkTreeNodes(child, visit)
	}
}

func TestOpenEndedRangeSuite(t *testing.T) {
	suite.Run(t, new(OpenEndedRangeSuite))
}
