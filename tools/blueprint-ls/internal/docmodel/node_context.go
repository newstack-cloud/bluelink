package docmodel

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// QuoteType represents the enclosing string quote style.
type QuoteType int

const (
	QuoteTypeNone   QuoteType = iota // Plain scalar or not in a string
	QuoteTypeSingle                  // Single-quoted string
	QuoteTypeDouble                  // Double-quoted string
)

// NodeContext provides rich context information for a specific position.
// It combines AST-level and schema-level information for language features.
type NodeContext struct {
	// Position information
	Position source.Position

	// Document reference
	DocumentCtx *DocumentContext

	// AST-level context (always available when document parses)
	UnifiedNode   *UnifiedNode
	ASTPath       StructuredPath
	AncestorNodes []*UnifiedNode

	// Schema-level context (available if document validates)
	SchemaNode    *schema.TreeNode
	SchemaElement any

	// Text context for completion
	TextBefore  string // Text on line before cursor
	TextAfter   string // Text on line after cursor
	CurrentWord string // Word at cursor position
	CurrentLine string // Full current line

	// Extracted context (set during completion context determination)
	ExtractedFieldName string // Field name extracted from TextBefore when AST path is incomplete

	// State flags
	IsStale bool // True if using last-known-good data
}

// ElementKind returns the kind of schema element at this context.
func (ctx *NodeContext) ElementKind() SchemaElementKind {
	if ctx.SchemaElement != nil {
		return KindFromSchemaElement(ctx.SchemaElement)
	}
	return SchemaElementUnknown
}

// InSubstitution returns true if the position is inside a ${...} substitution.
func (ctx *NodeContext) InSubstitution() bool {
	if ctx.TextBefore == "" {
		return false
	}

	// Check for unclosed ${ before cursor
	openCount := strings.Count(ctx.TextBefore, "${")
	closeCount := strings.Count(ctx.TextBefore, "}")

	return openCount > closeCount
}

// GetSubstitutionText returns the text inside the current substitution if any.
func (ctx *NodeContext) GetSubstitutionText() string {
	if !ctx.InSubstitution() {
		return ""
	}

	// Find the last ${ before cursor
	lastOpen := strings.LastIndex(ctx.TextBefore, "${")
	if lastOpen == -1 {
		return ""
	}

	return ctx.TextBefore[lastOpen+2:]
}

// extractTextContext extracts text context from the document content.
func (ctx *NodeContext) extractTextContext(content string, pos source.Position) {
	lines := strings.Split(content, "\n")
	lineIndex := pos.Line - 1 // Convert 1-based to 0-based

	if lineIndex < 0 || lineIndex >= len(lines) {
		return
	}

	ctx.CurrentLine = lines[lineIndex]
	colIndex := max(
		// Convert 1-based to 0-based
		pos.Column-1,
		0,
	)
	colIndex = min(colIndex, len(ctx.CurrentLine))

	ctx.TextBefore = ctx.CurrentLine[:colIndex]
	ctx.TextAfter = ctx.CurrentLine[colIndex:]
	ctx.CurrentWord = extractWordAtPosition(ctx.CurrentLine, colIndex)
}

// extractWordAtPosition extracts the word at the given column position.
func extractWordAtPosition(line string, col int) string {
	if col < 0 || col > len(line) {
		return ""
	}

	// Find word start
	start := col
	for start > 0 && isWordChar(line[start-1]) {
		start -= 1
	}

	// Find word end
	end := col
	for end < len(line) && isWordChar(line[end]) {
		end += 1
	}

	return line[start:end]
}

// isWordChar returns true if the character is part of a word.
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '-' || c == '.'
}

// IsAtTypeField returns true if the position is at a type field.
func (ctx *NodeContext) IsAtTypeField() bool {
	return ctx.ASTPath.IsResourceType() ||
		ctx.ASTPath.IsDataSourceType() ||
		ctx.ASTPath.IsVariableType() ||
		ctx.ASTPath.IsValueType() ||
		ctx.ASTPath.IsExportType()
}

// IsAtResourceSpec returns true if the position is in a resource spec.
func (ctx *NodeContext) IsAtResourceSpec() bool {
	return ctx.ASTPath.IsResourceSpec()
}

// IsAtDataSourceFilter returns true if the position is in a data source filter.
func (ctx *NodeContext) IsAtDataSourceFilter() bool {
	return ctx.ASTPath.IsDataSourceFilter()
}

// GetResourceName returns the resource name if the position is under a resource.
func (ctx *NodeContext) GetResourceName() (string, bool) {
	return ctx.ASTPath.GetResourceName()
}

// GetDataSourceName returns the data source name if the position is under a data source.
func (ctx *NodeContext) GetDataSourceName() (string, bool) {
	return ctx.ASTPath.GetDataSourceName()
}

// GetVariableName returns the variable name if the position is under a variable.
func (ctx *NodeContext) GetVariableName() (string, bool) {
	return ctx.ASTPath.GetVariableName()
}

// GetValueName returns the value name if the position is under a value.
func (ctx *NodeContext) GetValueName() (string, bool) {
	return ctx.ASTPath.GetValueName()
}

// HasError returns true if the position is in an error region.
func (ctx *NodeContext) HasError() bool {
	if ctx.UnifiedNode == nil {
		return false
	}
	return ctx.UnifiedNode.IsError
}

// IsAtKeyPosition returns true if the cursor is at a position where a new
// mapping key (field name) should be entered, rather than a value.
// For YAML: detects if we're at whitespace-only start of line or after a completed key-value pair
// For JSONC: detects if we're after { or , expecting a new property name
func (ctx *NodeContext) IsAtKeyPosition() bool {
	// Only consider the current line (text after the last newline)
	// This handles multi-line cases where previous lines have colons
	currentLineText := ctx.TextBefore
	if lastNewline := strings.LastIndex(ctx.TextBefore, "\n"); lastNewline >= 0 {
		currentLineText = ctx.TextBefore[lastNewline+1:]
	}

	trimmed := strings.TrimLeft(currentLineText, " \t")

	// Empty or whitespace-only before cursor on current line - potential key position
	if trimmed == "" {
		return true
	}

	// JSONC/JSON: After { or , we're at a key position
	// e.g., `"spec": { ` -> key position (after opening brace)
	// e.g., `"field": "value", ` -> key position (after comma)
	trimmedEnd := strings.TrimRight(trimmed, " \t")
	if strings.HasSuffix(trimmedEnd, "{") || strings.HasSuffix(trimmedEnd, ",") {
		return true
	}

	// If there's a colon followed only by whitespace, we're at a value position
	// e.g., "fieldName: " -> value position
	if strings.Contains(trimmed, ":") {
		afterColon := strings.TrimSpace(trimmed[strings.LastIndex(trimmed, ":")+1:])
		if afterColon == "" {
			return false // Value position after colon
		}
	}

	// If text before doesn't contain a colon and is just a word being typed,
	// we're typing a key name
	// e.g., "  run" -> typing key "run"
	if !strings.Contains(trimmed, ":") {
		return true
	}

	return false
}

// IsAtValuePosition returns true if the cursor is at a position where a
// value should be entered (after a colon in YAML, after : in JSON).
func (ctx *NodeContext) IsAtValuePosition() bool {
	// Only consider the current line (text after the last newline)
	currentLineText := ctx.TextBefore
	if lastNewline := strings.LastIndex(ctx.TextBefore, "\n"); lastNewline >= 0 {
		currentLineText = ctx.TextBefore[lastNewline+1:]
	}

	trimmed := strings.TrimLeft(currentLineText, " \t")

	// Check for "key: " pattern (value position)
	if strings.Contains(trimmed, ":") {
		afterColon := strings.TrimSpace(trimmed[strings.LastIndex(trimmed, ":")+1:])
		// If nothing or just typing after colon, it's a value position
		return afterColon == "" || !strings.Contains(afterColon, ":")
	}

	return false
}

// GetTypedPrefix returns the text being typed at the current position.
// This is used for filtering completion suggestions.
func (ctx *NodeContext) GetTypedPrefix() string {
	// Only consider the current line (text after the last newline)
	currentLineText := ctx.TextBefore
	if lastNewline := strings.LastIndex(ctx.TextBefore, "\n"); lastNewline >= 0 {
		currentLineText = ctx.TextBefore[lastNewline+1:]
	}

	// For key positions, return the current word being typed
	if ctx.IsAtKeyPosition() {
		trimmed := strings.TrimLeft(currentLineText, " \t")
		// Extract just the identifier being typed (no colon or other punctuation)
		prefix := ""
		for _, c := range trimmed {
			if isWordChar(byte(c)) {
				prefix += string(c)
			} else {
				prefix = "" // Reset on non-word chars, keep only trailing word
			}
		}
		return prefix
	}

	// For value positions, return what's after the colon on the current line
	if ctx.IsAtValuePosition() {
		if idx := strings.LastIndex(currentLineText, ":"); idx >= 0 {
			return strings.TrimSpace(currentLineText[idx+1:])
		}
	}

	return ctx.CurrentWord
}

// IsEmpty returns true if no node was found at the position.
func (ctx *NodeContext) IsEmpty() bool {
	return ctx.UnifiedNode == nil
}

// GetRange returns the range of the current node.
func (ctx *NodeContext) GetRange() *source.Range {
	if ctx.UnifiedNode == nil {
		return nil
	}
	return &ctx.UnifiedNode.Range
}

// GetKeyRange returns the key range if this is a map value.
func (ctx *NodeContext) GetKeyRange() *source.Range {
	if ctx.UnifiedNode == nil {
		return nil
	}
	return ctx.UnifiedNode.KeyRange
}

// GetValue returns the scalar value if this is a scalar node.
func (ctx *NodeContext) GetValue() string {
	if ctx.UnifiedNode == nil {
		return ""
	}
	return ctx.UnifiedNode.Value
}

// GetFieldName returns the field name if this is a map value.
func (ctx *NodeContext) GetFieldName() string {
	if ctx.UnifiedNode == nil {
		return ""
	}
	return ctx.UnifiedNode.FieldName
}

// IsPrecededByOperatorField returns true if the cursor is right after "operator:" or "operator":
// This is used for determining text edit ranges in completion.
func (ctx *NodeContext) IsPrecededByOperatorField() bool {
	return operatorFieldPattern.MatchString(ctx.TextBefore)
}

// GetEnclosingQuoteType returns the quote type of the enclosing string node.
// This is used to determine the appropriate quote style for bracket notation completions.
func (ctx *NodeContext) GetEnclosingQuoteType() QuoteType {
	for _, ancestor := range ctx.AncestorNodes {
		switch ancestor.TSKind {
		case "double_quote_scalar": // YAML
			return QuoteTypeDouble
		case "single_quote_scalar": // YAML
			return QuoteTypeSingle
		case "string_scalar", "string_content", "string": // JSON/JSONC - always double quoted
			return QuoteTypeDouble
		case "plain_scalar", "block_scalar":
			return QuoteTypeNone
		}
	}
	return QuoteTypeNone
}
