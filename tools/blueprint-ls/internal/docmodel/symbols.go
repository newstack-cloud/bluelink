package docmodel

import (
	"fmt"

	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// BuildDocumentSymbols creates LSP document symbols from a UnifiedNode tree.
// This provides a format-agnostic symbol building implementation that works
// for both YAML and JSON documents.
func BuildDocumentSymbols(root *UnifiedNode, totalLines int) []lsp.DocumentSymbol {
	if root == nil {
		return []lsp.DocumentSymbol{}
	}

	var symbols []lsp.DocumentSymbol
	collectDocumentSymbols("document", root, &symbols, nil, totalLines)
	return symbols
}

func collectDocumentSymbols(
	name string,
	node *UnifiedNode,
	symbols *[]lsp.DocumentSymbol,
	nextSibling *UnifiedNode,
	totalLines int,
) {
	if node == nil {
		return
	}

	switch node.Kind {
	case NodeKindDocument:
		collectDocumentNode(name, node, symbols, totalLines)
	case NodeKindMapping:
		collectMappingNode(name, node, symbols, nextSibling, totalLines)
	case NodeKindSequence:
		collectSequenceNode(name, node, symbols, nextSibling, totalLines)
	case NodeKindScalar:
		collectScalarNode(name, node, symbols, nextSibling)
	}
}

func collectDocumentNode(
	name string,
	node *UnifiedNode,
	symbols *[]lsp.DocumentSymbol,
	totalLines int,
) {
	symbolRange := unifiedNodeToLSPRange(node, nil)
	symbolRange.End.Line = lsp.UInteger(totalLines)

	symbol := lsp.DocumentSymbol{
		Name:           name,
		Kind:           lsp.SymbolKindFile,
		Range:          symbolRange,
		SelectionRange: symbolRange,
		Children:       []lsp.DocumentSymbol{},
	}

	for i, child := range node.Children {
		var nextSibling *UnifiedNode
		if i+1 < len(node.Children) {
			nextSibling = node.Children[i+1]
		}
		collectDocumentSymbols(
			"content",
			child,
			&symbol.Children,
			nextSibling,
			totalLines,
		)
	}

	*symbols = append(*symbols, symbol)
}

func collectMappingNode(
	name string,
	node *UnifiedNode,
	symbols *[]lsp.DocumentSymbol,
	nextSibling *UnifiedNode,
	totalLines int,
) {
	symbolRange := unifiedNodeToLSPRange(node, nextSibling)

	symbol := lsp.DocumentSymbol{
		Name:     name,
		Kind:     lsp.SymbolKindObject,
		Children: []lsp.DocumentSymbol{},
	}

	for i, child := range node.Children {
		var childNext *UnifiedNode
		if i+1 < len(node.Children) {
			childNext = node.Children[i+1]
		} else {
			childNext = nextSibling
		}

		childName := child.FieldName
		if childName == "" {
			childName = fmt.Sprintf("[%d]", i)
		}

		collectDocumentSymbols(
			childName,
			child,
			&symbol.Children,
			childNext,
			totalLines,
		)
	}

	if len(symbol.Children) > 0 {
		lastChild := symbol.Children[len(symbol.Children)-1]
		symbolRange.End = lastChild.Range.End
	} else if nextSibling == nil {
		symbolRange.End.Line = lsp.UInteger(totalLines)
	}

	symbol.Range = symbolRange
	symbol.SelectionRange = symbolRange

	*symbols = append(*symbols, symbol)
}

func collectSequenceNode(
	name string,
	node *UnifiedNode,
	symbols *[]lsp.DocumentSymbol,
	nextSibling *UnifiedNode,
	totalLines int,
) {
	symbolRange := unifiedNodeToLSPRange(node, nextSibling)

	symbol := lsp.DocumentSymbol{
		Name:     name,
		Kind:     lsp.SymbolKindArray,
		Children: []lsp.DocumentSymbol{},
	}

	for i, child := range node.Children {
		var childNext *UnifiedNode
		if i+1 < len(node.Children) {
			childNext = node.Children[i+1]
		} else {
			childNext = nextSibling
		}

		collectDocumentSymbols(
			fmt.Sprintf("[%d]", i),
			child,
			&symbol.Children,
			childNext,
			totalLines,
		)
	}

	if len(symbol.Children) > 0 {
		lastChild := symbol.Children[len(symbol.Children)-1]
		symbolRange.End = lastChild.Range.End
	} else if nextSibling == nil {
		symbolRange.End.Line = lsp.UInteger(totalLines)
	}

	symbol.Range = symbolRange
	symbol.SelectionRange = symbolRange

	*symbols = append(*symbols, symbol)
}

func collectScalarNode(
	name string,
	node *UnifiedNode,
	symbols *[]lsp.DocumentSymbol,
	nextSibling *UnifiedNode,
) {
	symbolRange := unifiedNodeToLSPRange(node, nextSibling)
	symbolKind := determineScalarSymbolKind(node.Tag)

	symbol := lsp.DocumentSymbol{
		Name:           name,
		Kind:           symbolKind,
		Range:          symbolRange,
		SelectionRange: symbolRange,
	}
	*symbols = append(*symbols, symbol)
}

func unifiedNodeToLSPRange(node *UnifiedNode, nextSibling *UnifiedNode) lsp.Range {
	start := lsp.Position{
		Line:      0,
		Character: 0,
	}

	if node.KeyRange != nil && node.KeyRange.Start != nil {
		// Use key range start if available (for mapping entries)
		start.Line = lsp.UInteger(node.KeyRange.Start.Line - 1)
		start.Character = lsp.UInteger(node.KeyRange.Start.Column - 1)
	} else if node.Range.Start != nil {
		// Fall back to node range start
		start.Line = lsp.UInteger(node.Range.Start.Line - 1)
		start.Character = lsp.UInteger(node.Range.Start.Column - 1)
	}

	end := lsp.Position{}
	if node.Range.End != nil {
		end.Line = lsp.UInteger(node.Range.End.Line - 1)
		end.Character = lsp.UInteger(node.Range.End.Column - 1)
	} else if nextSibling != nil && nextSibling.Range.Start != nil {
		end.Line = lsp.UInteger(nextSibling.Range.Start.Line - 1)
		end.Character = lsp.UInteger(nextSibling.Range.Start.Column - 1)
	}

	return lsp.Range{
		Start: start,
		End:   end,
	}
}

func determineScalarSymbolKind(tag string) lsp.SymbolKind {
	switch tag {
	case "integer", "float", "!!int", "!!float":
		return lsp.SymbolKindNumber
	case "boolean", "!!bool":
		return lsp.SymbolKindBoolean
	case "null", "!!null":
		return lsp.SymbolKindNull
	default:
		return lsp.SymbolKindString
	}
}
