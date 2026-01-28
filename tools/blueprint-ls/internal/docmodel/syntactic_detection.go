package docmodel

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// DetectSyntacticStyle determines the syntactic style at a position based on
// the document format and the tree-sitter node types (TSKind) in the ancestry.
//
// For YAML documents, it examines ancestors to find the nearest flow or block
// container. For JSONC documents, it always returns JSONC style.
func DetectSyntacticStyle(ancestors []*UnifiedNode, format DocumentFormat) SyntacticStyle {
	if format == FormatJSONC {
		return SyntacticStyleJSONC
	}

	// For YAML, walk ancestors to find the nearest container type
	// that indicates flow vs block style.
	for i := len(ancestors) - 1; i >= 0; i-- {
		node := ancestors[i]
		switch node.TSKind {
		case "flow_mapping", "flow_sequence", "flow_pair", "flow_sequence_item":
			return SyntacticStyleFlowYAML
		case "block_mapping", "block_sequence", "block_mapping_pair", "block_sequence_item":
			return SyntacticStyleBlockYAML
		}
	}

	// Default to block YAML if no container found
	return SyntacticStyleBlockYAML
}

// DetectSyntacticPosition determines the syntactic position at the cursor
// based on the detected style and text context.
//
// This delegates to style-specific detection logic:
// - Block YAML uses indentation and "- " marker heuristics
// - Flow YAML and JSONC share delimiter-based detection ({, [, ,, :)
func DetectSyntacticPosition(
	style SyntacticStyle,
	node *UnifiedNode,
	ancestors []*UnifiedNode,
	pos source.Position,
	textBefore string,
) SyntacticPosition {
	switch style {
	case SyntacticStyleBlockYAML:
		return detectBlockYAMLPosition(node, ancestors, textBefore)
	case SyntacticStyleFlowYAML, SyntacticStyleJSONC:
		// Flow YAML and JSONC share the same delimiter-based syntax
		return detectFlowPosition(ancestors, textBefore)
	default:
		return SyntacticPositionUnknown
	}
}

// detectBlockYAMLPosition handles position detection for block-style YAML.
// Block YAML uses indentation for structure and "- " markers for sequence items.
func detectBlockYAMLPosition(
	node *UnifiedNode,
	ancestors []*UnifiedNode,
	textBefore string,
) SyntacticPosition {
	currentLineText := extractCurrentLine(textBefore)
	trimmed := strings.TrimLeft(currentLineText, " \t")

	// Empty line or whitespace-only - determine from context
	if trimmed == "" {
		return determineEmptyLinePosition(ancestors)
	}

	// Check for YAML sequence item marker "- " or "-"
	if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
		return SyntacticPositionSequenceItem
	}

	// Check for colon presence (key: value pattern)
	if colonIdx := strings.LastIndex(currentLineText, ":"); colonIdx >= 0 {
		afterColon := strings.TrimSpace(currentLineText[colonIdx+1:])
		if afterColon == "" {
			return SyntacticPositionValueField
		}
		// Something after colon but no additional colon - still value position
		if !strings.Contains(afterColon, ":") {
			return SyntacticPositionValueField
		}
	}

	// No colon on line, we're typing a key
	return SyntacticPositionKeyField
}

// detectFlowPosition handles position detection for flow-style YAML and JSONC.
// Both use the same delimiter syntax: {}, [], commas, colons.
func detectFlowPosition(
	ancestors []*UnifiedNode,
	textBefore string,
) SyntacticPosition {
	currentLineText := extractCurrentLine(textBefore)
	trimmedEnd := strings.TrimRight(currentLineText, " \t")

	// Find the nearest container kind in ancestry
	containerKind := findNearestContainerKind(ancestors)

	// Check for empty container: [] or {}
	// Return specific position based on container type for consistency
	if isInsideEmptyContainer(textBefore) {
		if containerKind == NodeKindSequence {
			return SyntacticPositionSequenceItem
		}
		// Empty mapping - would type a key
		return SyntacticPositionKeyField
	}

	// Check if inside a string in an array
	if containerKind == NodeKindSequence && isInsideArrayString(textBefore) {
		return SyntacticPositionStringContent
	}

	// Check delimiter patterns
	if strings.HasSuffix(trimmedEnd, "{") {
		return SyntacticPositionKeyField
	}
	if strings.HasSuffix(trimmedEnd, "[") {
		return SyntacticPositionSequenceItem
	}
	if strings.HasSuffix(trimmedEnd, ",") {
		if containerKind == NodeKindMapping {
			return SyntacticPositionKeyField
		}
		return SyntacticPositionSequenceItem
	}
	if strings.HasSuffix(trimmedEnd, ":") {
		return SyntacticPositionValueField
	}

	// Check for value position after colon
	if colonIdx := strings.LastIndex(currentLineText, ":"); colonIdx >= 0 {
		afterColon := strings.TrimSpace(currentLineText[colonIdx+1:])
		// Check if we're still in value position (no comma or closing brace after)
		if afterColon == "" || !containsDelimiterAfterColon(afterColon) {
			return SyntacticPositionValueField
		}
	}

	// Default based on container type
	if containerKind == NodeKindMapping {
		return detectMappingPosition(currentLineText)
	}
	return SyntacticPositionSequenceItem
}

// determineEmptyLinePosition determines the position for empty lines in block YAML
// based on the nearest container in the ancestry.
func determineEmptyLinePosition(ancestors []*UnifiedNode) SyntacticPosition {
	for i := len(ancestors) - 1; i >= 0; i-- {
		switch ancestors[i].Kind {
		case NodeKindMapping:
			return SyntacticPositionKeyField
		case NodeKindSequence:
			return SyntacticPositionSequenceItem
		}
	}
	return SyntacticPositionKeyField // Default
}

// findNearestContainerKind finds the nearest mapping or sequence in the ancestry.
func findNearestContainerKind(ancestors []*UnifiedNode) NodeKind {
	for i := len(ancestors) - 1; i >= 0; i-- {
		switch ancestors[i].Kind {
		case NodeKindMapping, NodeKindSequence:
			return ancestors[i].Kind
		}
	}
	return NodeKindMapping // Default
}

// extractCurrentLine extracts the current line from textBefore (text after the last newline).
func extractCurrentLine(textBefore string) string {
	if lastNewline := strings.LastIndex(textBefore, "\n"); lastNewline >= 0 {
		return textBefore[lastNewline+1:]
	}
	return textBefore
}

// isInsideEmptyContainer checks if the cursor is inside an empty container: [] or {}
func isInsideEmptyContainer(textBefore string) bool {
	currentLine := extractCurrentLine(textBefore)

	// Check for empty array []
	lastBracket := strings.LastIndex(currentLine, "[")
	if lastBracket >= 0 {
		afterBracket := currentLine[lastBracket+1:]
		trimmed := strings.TrimSpace(afterBracket)
		if trimmed == "" {
			return true
		}
	}

	// Check for empty object {}
	lastBrace := strings.LastIndex(currentLine, "{")
	if lastBrace >= 0 {
		afterBrace := currentLine[lastBrace+1:]
		trimmed := strings.TrimSpace(afterBrace)
		if trimmed == "" {
			return true
		}
	}

	return false
}

// isInsideArrayString checks if the cursor is inside a string within an array.
// Detects patterns like: ["text|"] or ["a", "b|"] where | is the cursor.
func isInsideArrayString(textBefore string) bool {
	currentLine := extractCurrentLine(textBefore)
	lastBracket := strings.LastIndex(currentLine, "[")
	if lastBracket < 0 {
		return false
	}

	afterBracket := currentLine[lastBracket+1:]

	// Count quotes - odd number means we're inside a string
	quoteCount := strings.Count(afterBracket, "\"")

	// Check we're not after a closing bracket
	lastCloseBracket := strings.LastIndex(afterBracket, "]")
	if lastCloseBracket >= 0 {
		lastQuote := strings.LastIndex(afterBracket, "\"")
		if lastCloseBracket > lastQuote {
			return false
		}
	}

	return quoteCount%2 == 1
}

// containsDelimiterAfterColon checks if text after a colon contains a delimiter
// that would indicate we're past the value position.
func containsDelimiterAfterColon(afterColon string) bool {
	return strings.ContainsAny(afterColon, ",}]")
}

// detectMappingPosition determines position within a mapping based on text analysis.
func detectMappingPosition(currentLine string) SyntacticPosition {
	trimmed := strings.TrimLeft(currentLine, " \t")

	// If there's a colon, check what's after it
	if colonIdx := strings.LastIndex(trimmed, ":"); colonIdx >= 0 {
		afterColon := strings.TrimSpace(trimmed[colonIdx+1:])
		if afterColon == "" {
			return SyntacticPositionValueField
		}
	}

	// No colon or typing after value - could be typing a key
	if !strings.Contains(trimmed, ":") {
		return SyntacticPositionKeyField
	}

	return SyntacticPositionValueField
}

// ExtractTypedPrefix extracts the text being typed at the current position
// based on the syntactic position and style.
func ExtractTypedPrefix(
	position SyntacticPosition,
	style SyntacticStyle,
	textBefore string,
) string {
	currentLine := extractCurrentLine(textBefore)

	switch position {
	case SyntacticPositionSequenceItem:
		return extractSequenceItemPrefix(style, currentLine)

	case SyntacticPositionStringContent:
		return extractStringContentPrefix(currentLine)

	case SyntacticPositionEmptyContainer:
		return ""

	case SyntacticPositionKeyField:
		return extractKeyFieldPrefix(currentLine)

	case SyntacticPositionValueField:
		return extractValueFieldPrefix(currentLine)

	default:
		return extractWordAtEnd(currentLine)
	}
}

// extractSequenceItemPrefix extracts prefix for sequence item positions.
func extractSequenceItemPrefix(style SyntacticStyle, currentLine string) string {
	trimmed := strings.TrimLeft(currentLine, " \t")

	// Block YAML: after "- "
	if style == SyntacticStyleBlockYAML {
		if strings.HasPrefix(trimmed, "- ") {
			return strings.TrimLeft(trimmed[2:], " ")
		}
		return ""
	}

	// Flow YAML / JSONC: after [ or ,
	lastBracket := strings.LastIndex(currentLine, "[")
	if lastBracket >= 0 {
		afterBracket := currentLine[lastBracket+1:]
		quoteCount := strings.Count(afterBracket, "\"")

		// Odd number of quotes means we're inside a string
		if quoteCount%2 == 1 {
			// Return text after the last opening quote
			lastQuote := strings.LastIndex(afterBracket, "\"")
			if lastQuote >= 0 {
				return afterBracket[lastQuote+1:]
			}
		}

		// Even number of quotes (or no quotes) - we're outside any string
		// Return text after the last comma, or after the bracket if no comma
		lastComma := strings.LastIndex(afterBracket, ",")
		if lastComma >= 0 {
			return strings.TrimSpace(afterBracket[lastComma+1:])
		}
		return strings.TrimSpace(afterBracket)
	}

	return ""
}

// extractStringContentPrefix extracts prefix when inside a string in an array.
func extractStringContentPrefix(currentLine string) string {
	lastBracket := strings.LastIndex(currentLine, "[")
	if lastBracket >= 0 {
		afterBracket := currentLine[lastBracket+1:]
		lastQuote := strings.LastIndex(afterBracket, "\"")
		if lastQuote >= 0 {
			return afterBracket[lastQuote+1:]
		}
	}
	return ""
}

// extractKeyFieldPrefix extracts the word being typed at a key position.
func extractKeyFieldPrefix(currentLine string) string {
	trimmed := strings.TrimLeft(currentLine, " \t")
	// Extract just the identifier being typed (no colon or punctuation)
	prefix := ""
	for _, c := range trimmed {
		if isWordCharacter(byte(c)) {
			prefix += string(c)
		} else {
			prefix = "" // Reset on non-word chars, keep only trailing word
		}
	}
	return prefix
}

// extractValueFieldPrefix extracts text after the colon.
func extractValueFieldPrefix(currentLine string) string {
	if idx := strings.LastIndex(currentLine, ":"); idx >= 0 {
		return strings.TrimSpace(currentLine[idx+1:])
	}
	return ""
}

// extractWordAtEnd extracts the word at the end of the current line.
func extractWordAtEnd(currentLine string) string {
	// Find the last word character sequence
	end := len(currentLine)
	start := end
	for start > 0 && isWordCharacter(currentLine[start-1]) {
		start--
	}
	return currentLine[start:end]
}

// isWordCharacter returns true if the character is part of a word.
func isWordCharacter(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '-' || c == '.'
}
