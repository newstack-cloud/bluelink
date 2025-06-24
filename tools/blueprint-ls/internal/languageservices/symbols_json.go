package languageservices

import (
	"fmt"

	"github.com/coreos/go-json"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

func (s *SymbolService) getJSONDocumentSymbols(
	docURI lsp.URI,
	content string,
) ([]lsp.DocumentSymbol, error) {

	symbols := []lsp.DocumentSymbol{}

	jsonNode := s.state.GetDocumentJSONNode(docURI)
	if jsonNode == nil {
		var err error
		jsonNode, err = blueprint.ParseJWCCNode(content)
		if err != nil {
			return nil, err
		}
		s.state.SetDocumentJSONNode(docURI, jsonNode)
	}
	linePositions := bpcore.LinePositionsFromSource(content)
	s.collectJSONDocumentSymbols(
		"document",
		jsonNode,
		&symbols,
		linePositions,
	)

	return symbols, nil
}

func (s *SymbolService) collectJSONDocumentSymbols(
	name string,
	node *json.Node,
	symbols *[]lsp.DocumentSymbol,
	linePositions []int,
) {
	mapNode, isMapNode := node.Value.(map[string]json.Node)
	if isMapNode {
		symbolRange := jsonNodeToLSPRange(node, linePositions)

		symbol := lsp.DocumentSymbol{
			Name:           name,
			Kind:           lsp.SymbolKindObject,
			Children:       []lsp.DocumentSymbol{},
			Range:          symbolRange,
			SelectionRange: symbolRange,
		}

		for key, valueNode := range mapNode {
			s.collectJSONDocumentSymbols(
				key,
				&valueNode,
				&symbol.Children,
				linePositions,
			)
		}

		*symbols = append(*symbols, symbol)
		return
	}

	sliceNode, isSliceNode := node.Value.([]json.Node)
	if isSliceNode {
		symbolRange := jsonNodeToLSPRange(node, linePositions)

		symbol := lsp.DocumentSymbol{
			Name:           name,
			Kind:           lsp.SymbolKindArray,
			Children:       []lsp.DocumentSymbol{},
			Range:          symbolRange,
			SelectionRange: symbolRange,
		}

		for i, valueNode := range sliceNode {
			s.collectJSONDocumentSymbols(
				fmt.Sprintf("[%d]", i),
				&valueNode,
				&symbol.Children,
				linePositions,
			)
		}

		*symbols = append(*symbols, symbol)
		return
	}

	s.collectScalarDocumentSymbols(
		name,
		node,
		symbols,
		linePositions,
	)
}

func (s *SymbolService) collectScalarDocumentSymbols(
	name string,
	node *json.Node,
	symbols *[]lsp.DocumentSymbol,
	linePositions []int,
) {
	symbolRange := jsonNodeToLSPRange(node, linePositions)
	symbol := lsp.DocumentSymbol{
		Name:           name,
		Range:          symbolRange,
		SelectionRange: symbolRange,
	}

	_, isStringNode := node.Value.(string)
	if isStringNode {
		symbol.Kind = lsp.SymbolKindString
	}

	// All numeric nodes are float64 when deserialised
	// as a json.Node.
	_, isNumberNode := node.Value.(float64)
	if isNumberNode {
		symbol.Kind = lsp.SymbolKindNumber
	}

	_, isBoolNode := node.Value.(bool)
	if isBoolNode {
		symbol.Kind = lsp.SymbolKindBoolean
	}

	if node.Value == nil {
		symbol.Kind = lsp.SymbolKindNull
	}

	if symbol.Kind != lsp.SymbolKind(0) {
		*symbols = append(*symbols, symbol)
	}
}

func jsonNodeToLSPRange(
	node *json.Node,
	linePositions []int,
) lsp.Range {
	startPos := getJSONNodeStartPos(node, linePositions)
	// coreos/go-json counts the end offset as the index of the last
	// character in the node, so we need to add 1 to get the end position.
	endOffset := node.End + 1
	endPos := source.PositionFromOffset(endOffset, linePositions)

	return lsp.Range{
		Start: lsp.Position{
			// blueprint schema source positions use 1-based line and column numbers,
			// LSP uses 0-based line and column numbers.
			Line:      lsp.UInteger(startPos.Line - 1),
			Character: lsp.UInteger(startPos.Column - 1),
		},
		End: lsp.Position{
			Line:      lsp.UInteger(endPos.Line - 1),
			Character: lsp.UInteger(endPos.Column - 1),
		},
	}
}

func getJSONNodeStartPos(
	node *json.Node,
	linePositions []int,
) source.Position {
	if node.KeyStart == 0 {
		// Not a key -> value mapping entry.
		return source.PositionFromOffset(node.Start, linePositions)
	}

	return source.PositionFromOffset(node.KeyStart, linePositions)
}
