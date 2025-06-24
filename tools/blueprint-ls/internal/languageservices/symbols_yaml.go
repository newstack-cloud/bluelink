package languageservices

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func (s *SymbolService) getYAMLDocumentSymbols(
	docURI lsp.URI,
	content string,
) ([]lsp.DocumentSymbol, error) {

	symbols := []lsp.DocumentSymbol{}

	yamlNode := s.state.GetDocumentYAMLNode(docURI)
	if yamlNode == nil {
		var err error
		yamlNode, err = blueprint.ParseYAMLNode(content)
		if err != nil {
			return nil, err
		}
		s.state.SetDocumentYAMLNode(docURI, yamlNode)
	}

	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	numberOfLines := len(lines)
	s.collectYAMLDocumentSymbols(
		"document",
		yamlNode,
		&symbols,
		/* next */ nil,
		/* mappingKeyNode */ nil,
		numberOfLines,
	)

	return symbols, nil
}

func (s *SymbolService) collectYAMLDocumentSymbols(
	name string,
	node *yaml.Node,
	symbols *[]lsp.DocumentSymbol,
	next *yaml.Node,
	mappingKeyNode *yaml.Node,
	numberOfLinesInDocument int,
) {
	if node.Kind == yaml.DocumentNode {
		symbolRange := yamlNodeToLSPRange(node, nil, nil)
		symbolRange.End.Line = lsp.UInteger(numberOfLinesInDocument)
		symbol := lsp.DocumentSymbol{
			Name:           name,
			Kind:           lsp.SymbolKindFile,
			Range:          symbolRange,
			SelectionRange: symbolRange,
			Children:       []lsp.DocumentSymbol{},
		}
		for i, child := range node.Content {
			childNext := next
			if i+1 < len(node.Content) {
				childNext = node.Content[i+1]
			}
			s.collectYAMLDocumentSymbols(
				"content",
				child,
				&symbol.Children,
				childNext,
				nil,
				numberOfLinesInDocument,
			)
		}
		*symbols = append(*symbols, symbol)
		return
	}

	if node.Kind == yaml.MappingNode {
		symbolRange := yamlNodeToLSPRange(node, next, mappingKeyNode)

		symbol := lsp.DocumentSymbol{
			Name:     name,
			Kind:     lsp.SymbolKindObject,
			Children: []lsp.DocumentSymbol{},
		}

		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			childNext := next
			if i+2 < len(node.Content) {
				childNext = node.Content[i+2]
			}

			s.collectYAMLDocumentSymbols(
				key.Value,
				value,
				&symbol.Children,
				childNext,
				key,
				numberOfLinesInDocument,
			)
		}

		if len(symbol.Children) > 0 {
			lastChild := symbol.Children[len(symbol.Children)-1]
			symbolRange.End = lastChild.Range.End
		} else if next == nil {
			symbolRange.End.Line = lsp.UInteger(numberOfLinesInDocument)
		}

		symbol.Range = symbolRange
		symbol.SelectionRange = symbolRange

		*symbols = append(*symbols, symbol)
		return
	}

	if node.Kind == yaml.SequenceNode {
		symbolRange := yamlNodeToLSPRange(node, next, mappingKeyNode)

		symbol := lsp.DocumentSymbol{
			Name:     name,
			Kind:     lsp.SymbolKindArray,
			Children: []lsp.DocumentSymbol{},
		}

		for i := 0; i < len(node.Content); i += 1 {
			item := node.Content[i]

			childNext := next
			if i+1 < len(node.Content) {
				childNext = node.Content[i+1]
			}

			s.collectYAMLDocumentSymbols(
				fmt.Sprintf("[%d]", i),
				item,
				&symbol.Children,
				childNext,
				nil,
				numberOfLinesInDocument,
			)
		}

		if len(symbol.Children) > 0 {
			lastChild := symbol.Children[len(symbol.Children)-1]
			symbolRange.End = lastChild.Range.End
		} else if next == nil {
			symbolRange.End.Line = lsp.UInteger(numberOfLinesInDocument)
		}

		symbol.Range = symbolRange
		symbol.SelectionRange = symbolRange

		*symbols = append(*symbols, symbol)
		return
	}

	if node.Kind == yaml.ScalarNode {
		s.logger.Debug("Adding scalar symbol", zap.String("name", name), zap.String("value", node.Value), zap.Int("style", int(node.Style)))
		symbolRange := yamlNodeToLSPRange(node, next, mappingKeyNode)

		symbolKind := determineYAMLScalarSymbolKind(node)

		symbol := lsp.DocumentSymbol{
			Name:           name,
			Kind:           symbolKind,
			Range:          symbolRange,
			SelectionRange: symbolRange,
		}
		*symbols = append(*symbols, symbol)
	}
}

func yamlNodeToLSPRange(node *yaml.Node, next *yaml.Node, mappingKeyNode *yaml.Node) lsp.Range {
	start := lsp.Position{
		// yaml.v3 package uses 1-based line and column numbers,
		// LSP uses 0-based line and column numbers.
		Line:      lsp.UInteger(node.Line - 1),
		Character: lsp.UInteger(node.Column - 1),
	}
	if mappingKeyNode != nil {
		start.Line = lsp.UInteger(mappingKeyNode.Line - 1)
		start.Character = lsp.UInteger(mappingKeyNode.Column - 1)
	}

	end := lsp.Position{}
	if node.Kind == yaml.ScalarNode {
		end = scalarYamlNodeEndPosition(node)
	} else if next != nil {
		end.Line = lsp.UInteger(next.Line - 1)
		end.Character = lsp.UInteger(next.Column - 1)
	}

	return lsp.Range{
		Start: start,
		End:   end,
	}
}

func scalarYamlNodeEndPosition(node *yaml.Node) lsp.Position {

	if node.Style == yaml.DoubleQuotedStyle || node.Style == yaml.SingleQuotedStyle {
		charCount := utf8.RuneCountInString(node.Value) + 2
		return lsp.Position{
			Line:      lsp.UInteger(node.Line - 1),
			Character: lsp.UInteger(node.Column + charCount - 1),
		}
	}

	// 0 indicates plain style
	if node.Style == 0 {
		return lsp.Position{
			Line:      lsp.UInteger(node.Line - 1),
			Character: lsp.UInteger(node.Column + utf8.RuneCountInString(node.Value) - 1),
		}
	}

	lines := strings.Split(strings.ReplaceAll(node.Value, "\r\n", "\n"), "\n")
	lineCountInBlock := len(lines) - 1
	columnOnLastLine := node.Column

	if lineCountInBlock > 0 {
		columnOnLastLine += utf8.RuneCountInString(lines[lineCountInBlock-1])
	}

	return lsp.Position{
		Line:      lsp.UInteger(node.Line + lineCountInBlock - 1),
		Character: lsp.UInteger(columnOnLastLine - 1),
	}
}

func determineYAMLScalarSymbolKind(node *yaml.Node) lsp.SymbolKind {
	switch node.Tag {
	case "!!int":
		return lsp.SymbolKindNumber
	case "!!bool":
		return lsp.SymbolKindBoolean
	case "!!null":
		return lsp.SymbolKindNull
	case "!!float":
		return lsp.SymbolKindNumber
	default:
		return lsp.SymbolKindString
	}
}
