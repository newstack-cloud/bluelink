package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
)

type SymbolsSuite struct {
	suite.Suite
}

func (s *SymbolsSuite) Test_BuildDocumentSymbols_nil_root_returns_empty() {
	symbols := BuildDocumentSymbols(nil, 100)
	s.Assert().NotNil(symbols)
	s.Assert().Empty(symbols)
}

func (s *SymbolsSuite) Test_BuildDocumentSymbols_document_with_mapping() {
	// Realistic structure: Document → Mapping → scalar children
	root := &UnifiedNode{
		Kind: NodeKindDocument,
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 5, Column: 1},
		},
	}

	topMapping := &UnifiedNode{
		Kind:   NodeKindMapping,
		Parent: root,
		Index:  -1,
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 5, Column: 1},
		},
	}

	version := &UnifiedNode{
		Kind:      NodeKindScalar,
		FieldName: "version",
		Value:     "2025-11-02",
		Tag:       "string",
		Parent:    topMapping,
		Index:     -1,
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 1, Column: 20},
		},
	}

	topMapping.Children = []*UnifiedNode{version}
	root.Children = []*UnifiedNode{topMapping}

	symbols := BuildDocumentSymbols(root, 5)
	s.Require().Len(symbols, 1)
	s.Assert().Equal("document", symbols[0].Name)
	s.Assert().Equal(lsp.SymbolKindFile, symbols[0].Kind)

	// Document child is the top-level mapping, named "content" by collectDocumentNode
	s.Require().Len(symbols[0].Children, 1)
	contentSymbol := symbols[0].Children[0]
	s.Assert().Equal("content", contentSymbol.Name)
	s.Assert().Equal(lsp.SymbolKindObject, contentSymbol.Kind)

	// Mapping children use their FieldName
	s.Require().Len(contentSymbol.Children, 1)
	s.Assert().Equal("version", contentSymbol.Children[0].Name)
	s.Assert().Equal(lsp.SymbolKindString, contentSymbol.Children[0].Kind)
}

func (s *SymbolsSuite) Test_BuildDocumentSymbols_nested_mapping() {
	root := &UnifiedNode{
		Kind: NodeKindDocument,
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 10, Column: 1},
		},
	}

	topMapping := &UnifiedNode{
		Kind:   NodeKindMapping,
		Parent: root,
		Index:  -1,
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 10, Column: 1},
		},
	}

	resources := &UnifiedNode{
		Kind:      NodeKindMapping,
		FieldName: "resources",
		Parent:    topMapping,
		Index:     -1,
		Range: source.Range{
			Start: &source.Position{Line: 2, Column: 1},
			End:   &source.Position{Line: 8, Column: 1},
		},
	}

	handler := &UnifiedNode{
		Kind:      NodeKindScalar,
		FieldName: "handler",
		Value:     "aws/lambda/function",
		Tag:       "string",
		Parent:    resources,
		Index:     -1,
		Range: source.Range{
			Start: &source.Position{Line: 3, Column: 5},
			End:   &source.Position{Line: 3, Column: 25},
		},
	}

	table := &UnifiedNode{
		Kind:      NodeKindScalar,
		FieldName: "table",
		Value:     "aws/dynamodb/table",
		Tag:       "string",
		Parent:    resources,
		Index:     -1,
		Range: source.Range{
			Start: &source.Position{Line: 4, Column: 5},
			End:   &source.Position{Line: 4, Column: 24},
		},
	}

	resources.Children = []*UnifiedNode{handler, table}
	topMapping.Children = []*UnifiedNode{resources}
	root.Children = []*UnifiedNode{topMapping}

	symbols := BuildDocumentSymbols(root, 10)
	s.Require().Len(symbols, 1)
	docSymbol := symbols[0]
	s.Require().Len(docSymbol.Children, 1)

	contentSymbol := docSymbol.Children[0]
	s.Assert().Equal("content", contentSymbol.Name)
	s.Require().Len(contentSymbol.Children, 1)

	resourcesSymbol := contentSymbol.Children[0]
	s.Assert().Equal("resources", resourcesSymbol.Name)
	s.Assert().Equal(lsp.SymbolKindObject, resourcesSymbol.Kind)
	s.Require().Len(resourcesSymbol.Children, 2)
	s.Assert().Equal("handler", resourcesSymbol.Children[0].Name)
	s.Assert().Equal("table", resourcesSymbol.Children[1].Name)
}

func (s *SymbolsSuite) Test_BuildDocumentSymbols_sequence_with_items() {
	root := &UnifiedNode{
		Kind: NodeKindDocument,
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 10, Column: 1},
		},
	}

	topMapping := &UnifiedNode{
		Kind:   NodeKindMapping,
		Parent: root,
		Index:  -1,
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 10, Column: 1},
		},
	}

	seq := &UnifiedNode{
		Kind:      NodeKindSequence,
		FieldName: "dependsOn",
		Parent:    topMapping,
		Index:     -1,
		Range: source.Range{
			Start: &source.Position{Line: 2, Column: 1},
			End:   &source.Position{Line: 5, Column: 1},
		},
	}

	item0 := &UnifiedNode{
		Kind:   NodeKindScalar,
		Value:  "handler",
		Tag:    "string",
		Parent: seq,
		Index:  0,
		Range: source.Range{
			Start: &source.Position{Line: 3, Column: 5},
			End:   &source.Position{Line: 3, Column: 12},
		},
	}

	item1 := &UnifiedNode{
		Kind:   NodeKindScalar,
		Value:  "table",
		Tag:    "string",
		Parent: seq,
		Index:  1,
		Range: source.Range{
			Start: &source.Position{Line: 4, Column: 5},
			End:   &source.Position{Line: 4, Column: 10},
		},
	}

	seq.Children = []*UnifiedNode{item0, item1}
	topMapping.Children = []*UnifiedNode{seq}
	root.Children = []*UnifiedNode{topMapping}

	symbols := BuildDocumentSymbols(root, 10)
	s.Require().Len(symbols, 1)
	docSymbol := symbols[0]
	s.Require().Len(docSymbol.Children, 1)

	contentSymbol := docSymbol.Children[0]
	s.Require().Len(contentSymbol.Children, 1)

	seqSymbol := contentSymbol.Children[0]
	s.Assert().Equal("dependsOn", seqSymbol.Name)
	s.Assert().Equal(lsp.SymbolKindArray, seqSymbol.Kind)
	s.Require().Len(seqSymbol.Children, 2)
	s.Assert().Equal("[0]", seqSymbol.Children[0].Name)
	s.Assert().Equal("[1]", seqSymbol.Children[1].Name)
}

func (s *SymbolsSuite) Test_BuildDocumentSymbols_scalar_tag_to_symbol_kind() {
	tests := []struct {
		tag          string
		expectedKind lsp.SymbolKind
	}{
		{"string", lsp.SymbolKindString},
		{"integer", lsp.SymbolKindNumber},
		{"float", lsp.SymbolKindNumber},
		{"!!int", lsp.SymbolKindNumber},
		{"!!float", lsp.SymbolKindNumber},
		{"boolean", lsp.SymbolKindBoolean},
		{"!!bool", lsp.SymbolKindBoolean},
		{"null", lsp.SymbolKindNull},
		{"!!null", lsp.SymbolKindNull},
		{"", lsp.SymbolKindString},
	}

	for _, tt := range tests {
		s.Run("tag_"+tt.tag, func() {
			root := &UnifiedNode{
				Kind: NodeKindDocument,
				Range: source.Range{
					Start: &source.Position{Line: 1, Column: 1},
					End:   &source.Position{Line: 2, Column: 1},
				},
			}
			mapping := &UnifiedNode{
				Kind:   NodeKindMapping,
				Parent: root,
				Index:  -1,
				Range: source.Range{
					Start: &source.Position{Line: 1, Column: 1},
					End:   &source.Position{Line: 2, Column: 1},
				},
			}
			scalar := &UnifiedNode{
				Kind:      NodeKindScalar,
				FieldName: "field",
				Tag:       tt.tag,
				Parent:    mapping,
				Index:     -1,
				Range: source.Range{
					Start: &source.Position{Line: 1, Column: 1},
					End:   &source.Position{Line: 1, Column: 10},
				},
			}
			mapping.Children = []*UnifiedNode{scalar}
			root.Children = []*UnifiedNode{mapping}

			symbols := BuildDocumentSymbols(root, 2)
			s.Require().Len(symbols, 1)
			s.Require().Len(symbols[0].Children, 1)
			s.Require().Len(symbols[0].Children[0].Children, 1)
			s.Assert().Equal(tt.expectedKind, symbols[0].Children[0].Children[0].Kind)
		})
	}
}

func (s *SymbolsSuite) Test_BuildDocumentSymbols_mapping_child_without_fieldname() {
	// When a mapping child has no FieldName, it gets named "[index]"
	root := &UnifiedNode{
		Kind: NodeKindDocument,
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 5, Column: 1},
		},
	}

	topMapping := &UnifiedNode{
		Kind:   NodeKindMapping,
		Parent: root,
		Index:  -1,
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 5, Column: 1},
		},
	}

	// Child with empty FieldName
	child := &UnifiedNode{
		Kind:   NodeKindScalar,
		Value:  "orphan",
		Tag:    "string",
		Parent: topMapping,
		Index:  -1,
		Range: source.Range{
			Start: &source.Position{Line: 2, Column: 1},
			End:   &source.Position{Line: 2, Column: 10},
		},
	}

	topMapping.Children = []*UnifiedNode{child}
	root.Children = []*UnifiedNode{topMapping}

	symbols := BuildDocumentSymbols(root, 5)
	s.Require().Len(symbols, 1)
	s.Require().Len(symbols[0].Children, 1)
	s.Require().Len(symbols[0].Children[0].Children, 1)
	// Empty FieldName gets formatted as "[0]"
	s.Assert().Equal("[0]", symbols[0].Children[0].Children[0].Name)
}

func TestSymbolsSuite(t *testing.T) {
	suite.Run(t, new(SymbolsSuite))
}
