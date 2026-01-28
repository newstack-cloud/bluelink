package docmodel

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// CursorContext provides unified context information for a cursor position.
// It composes structural path information (format-agnostic) with syntactic
// editing context (format-aware but with a unified interface).
//
// This is the preferred context type for new code. It provides cleaner
// separation between structural position ("where am I in the document tree")
// and syntactic position ("what kind of edit position is this").
type CursorContext struct {
	// Position is the cursor position in the document.
	Position source.Position

	// StructuralPath is the format-agnostic path from root to the current node.
	// Example: /resources/myResource/spec/tableName
	StructuralPath StructuredPath

	// UnifiedNode is the AST node at the cursor position.
	UnifiedNode *UnifiedNode

	// AncestorNodes contains nodes from root to the deepest node at the position.
	AncestorNodes []*UnifiedNode

	// Syntactic contains the syntactic editing context at the cursor.
	// This is format-aware but provides a unified interface.
	Syntactic *SyntacticContext

	// SchemaNode is the corresponding schema node if available.
	SchemaNode *schema.TreeNode

	// SchemaElement is the schema element at this position if available.
	SchemaElement any

	// DocumentCtx is a reference to the containing document context.
	DocumentCtx *DocumentContext

	// CurrentLine is the full current line of text.
	CurrentLine string

	// TextBefore is the text on the line before the cursor.
	TextBefore string

	// TextAfter is the text on the line after the cursor.
	TextAfter string

	// CurrentWord is the word at the cursor position.
	CurrentWord string

	// ExtractedFieldName is set during completion context determination
	// when the field name needs to be extracted from TextBefore.
	ExtractedFieldName string

	// IsStale is true if using last-known-good data due to parse errors.
	IsStale bool
}

// NewCursorContext creates a CursorContext for a given position in a document.
// It resolves the structural path, detects the syntactic context, and correlates
// with schema information if available.
func (ctx *DocumentContext) NewCursorContext(pos source.Position, leeway int) *CursorContext {
	cursorCtx := &CursorContext{
		Position:      pos,
		DocumentCtx:   ctx,
		AncestorNodes: make([]*UnifiedNode, 0),
	}

	// 1. Resolve structural position via AST/position index
	cursorCtx.resolveStructuralContext(ctx, pos, leeway)

	// 2. Extract text context
	cursorCtx.extractTextContext(ctx.Content, pos)

	// 3. Detect syntactic context using format-aware detection
	cursorCtx.Syntactic = cursorCtx.detectSyntacticContext(ctx.Format)

	// 4. Correlate with schema if available
	cursorCtx.resolveSchemaContext(ctx)

	return cursorCtx
}

// resolveStructuralContext resolves the structural path and node using
// the position index, with fallback to last-known-good AST.
func (cursorCtx *CursorContext) resolveStructuralContext(
	ctx *DocumentContext,
	pos source.Position,
	leeway int,
) {
	// Try current AST first
	if ctx.CurrentAST != nil && ctx.PositionIndex != nil {
		nodes := ctx.PositionIndex.NodesAtPosition(pos, leeway)
		if len(nodes) > 0 {
			cursorCtx.UnifiedNode = nodes[len(nodes)-1]
			cursorCtx.AncestorNodes = nodes
			cursorCtx.StructuralPath = StructuredPath(cursorCtx.UnifiedNode.AncestorPath())
		}
	}

	// Fall back to last valid AST if current is unavailable
	if cursorCtx.UnifiedNode == nil && ctx.LastValidAST != nil {
		lastValidIndex := NewPositionIndex(ctx.LastValidAST)
		nodes := lastValidIndex.NodesAtPosition(pos, leeway)
		if len(nodes) > 0 {
			cursorCtx.UnifiedNode = nodes[len(nodes)-1]
			cursorCtx.AncestorNodes = nodes
			cursorCtx.StructuralPath = StructuredPath(cursorCtx.UnifiedNode.AncestorPath())
			cursorCtx.IsStale = true
		}
	}

	// For YAML, also try indentation-based context enhancement
	if ctx.Format == FormatYAML {
		cursorCtx.tryIndentationBasedEnhancement(ctx, pos)
	}
}

// tryIndentationBasedEnhancement attempts to enhance context using indentation
// analysis for YAML documents. This handles cases where the cursor is at an
// indented position that logically belongs to a parent node whose range
// doesn't extend to the cursor line (e.g., empty lines, new lines being typed).
func (cursorCtx *CursorContext) tryIndentationBasedEnhancement(
	ctx *DocumentContext,
	pos source.Position,
) {
	if ctx.Content == "" {
		return
	}

	lines := strings.Split(ctx.Content, "\n")
	lineIndex := pos.Line - 1
	if lineIndex < 0 || lineIndex >= len(lines) {
		return
	}

	currentLine := lines[lineIndex]

	// Determine the effective indentation for parent lookup.
	//
	// For whitespace-only lines (empty, spaces, or tabs), we consider two signals:
	// 1. The visual indent calculated from the line content (tabs to tab stops)
	// 2. The cursor column (which may reflect the editor's knowledge of position)
	//
	// We use the MAX of these because:
	// - Visual indent from content handles mixed spaces/tabs correctly
	// - Cursor column handles cases where the editor positions the cursor
	//   beyond the content (e.g., after auto-indentation or virtual whitespace)
	//
	// For non-whitespace lines, we only use the content-based calculation.
	currentIndent := 0
	if isWhitespaceOnly(currentLine) {
		visualIndent := calculateVisualIndent(currentLine)
		cursorIndent := 0
		if pos.Column > 1 {
			cursorIndent = int(pos.Column) - 1
		}
		// Use whichever is larger
		if cursorIndent > visualIndent {
			currentIndent = cursorIndent
		} else {
			currentIndent = visualIndent
		}
	} else {
		currentIndent = calculateVisualIndent(currentLine)
		// For lines that START with a tab character (user pressed Tab then started
		// typing), add 1 to the indent. This handles the case where pressing Tab
		// puts the cursor at the same visual position as the parent (e.g., tab width 4
		// and parent at indent 4) - we want to consider them "inside" the parent,
		// not as a sibling.
		//
		// We only apply this for TAB, not SPACE, because:
		// - Tab is typically used for "quick indent to child level"
		// - Space indentation is normal YAML formatting where the user has
		//   deliberately set their indent level
		if len(currentLine) > 0 && currentLine[0] == '\t' {
			currentIndent += 1
		}
	}

	// Look for parent based on indentation.
	// We prefer the current AST if it's valid (not an error node), otherwise
	// fall back to the last-known-good AST. This is important because when
	// the user is typing (e.g., with tab indentation that creates invalid YAML),
	// the current AST may be a degraded error tree with no proper structure.
	ast := ctx.CurrentAST
	if ast == nil || ast.IsError {
		if ctx.LastValidAST != nil {
			ast = ctx.LastValidAST
		}
	}
	if ast == nil {
		return
	}

	// Find the best parent at or before this line with indent < currentIndent
	bestParent := cursorCtx.findBetterParentByIndentation(ast, pos.Line, currentIndent, lines)

	// Use the best parent if:
	// 1. We found one AND
	// 2. Either:
	//    a. We have no current node, OR
	//    b. The current node is an error node (invalid YAML, e.g., mixed tabs/spaces), OR
	//    c. The best parent's path is deeper (more specific context)
	if bestParent != nil {
		shouldUpdate := cursorCtx.UnifiedNode == nil ||
			cursorCtx.UnifiedNode.IsError ||
			len(bestParent.AncestorPath()) > len(cursorCtx.StructuralPath)

		if shouldUpdate {
			cursorCtx.UnifiedNode = bestParent
			cursorCtx.StructuralPath = StructuredPath(bestParent.AncestorPath())
			// Rebuild ancestors
			cursorCtx.AncestorNodes = cursorCtx.buildAncestorChain(bestParent)
		}
	}
}

// findBetterParentByIndentation searches for a parent node that better matches
// the cursor's indentation level. It finds the closest ancestor (highest line number
// before target) with indent less than the target indent.
//
// For YAML key-value pairs where the cursor is between the key line and the first
// content line (e.g., typing a new field in an empty mapping), we also check the
// node's KeyRange to determine if we're logically inside that node.
func (cursorCtx *CursorContext) findBetterParentByIndentation(
	root *UnifiedNode,
	targetLine int,
	targetIndent int,
	lines []string,
) *UnifiedNode {
	var best *UnifiedNode
	var bestLine int
	var bestDepth int

	var visit func(node *UnifiedNode)
	visit = func(node *UnifiedNode) {
		if node == nil {
			return
		}

		// Check if this node is before the target line
		// We consider a node "before" if either:
		// 1. Its content starts before the target line, OR
		// 2. Its key (for map values) is before the target line
		nodeBefore := false
		nodeEffectiveLine := 0

		if node.Range.Start != nil && node.Range.Start.Line < targetLine {
			nodeBefore = true
			nodeEffectiveLine = node.Range.Start.Line
		}

		// Also check KeyRange - if this node is a value in a key-value pair,
		// the key might be before the cursor even if the value content isn't.
		// This handles the case where cursor is on a new line inside an empty mapping:
		//   spec:
		//     |  <- cursor here (new empty line)
		//     functionName: ...  <- content starts here
		if node.KeyRange != nil && node.KeyRange.Start != nil && node.KeyRange.Start.Line < targetLine {
			nodeBefore = true
			// Use the key's line as effective line if it's closer to target
			if node.KeyRange.Start.Line > nodeEffectiveLine {
				nodeEffectiveLine = node.KeyRange.Start.Line
			}
		}

		if nodeBefore {
			// For indent comparison, prefer using the key's indent if this is a map value.
			// This correctly identifies the logical nesting level (e.g., "spec:" at indent 4
			// means we're inside the spec mapping when typing at indent 6).
			nodeIndent := 0
			if node.KeyRange != nil && node.KeyRange.Start != nil {
				keyLineIndex := node.KeyRange.Start.Line - 1
				if keyLineIndex >= 0 && keyLineIndex < len(lines) {
					nodeIndent = countLeadingSpaces(lines[keyLineIndex])
				}
			} else {
				nodeLine := node.Range.Start.Line - 1
				if nodeLine >= 0 && nodeLine < len(lines) {
					nodeIndent = countLeadingSpaces(lines[nodeLine])
				}
			}

			// If node's indent is less than target's indent, it could be a parent
			if nodeIndent < targetIndent {
				if node.Kind == NodeKindMapping || node.Kind == NodeKindSequence {
					nodeDepth := len(node.AncestorPath())

					// Prefer: 1) deeper paths, 2) higher line numbers (closer to target)
					isBetter := best == nil ||
						nodeDepth > bestDepth ||
						(nodeDepth == bestDepth && nodeEffectiveLine > bestLine)

					if isBetter {
						best = node
						bestLine = nodeEffectiveLine
						bestDepth = nodeDepth
					}
				}
			}
		}

		for _, child := range node.Children {
			visit(child)
		}
	}

	visit(root)
	return best
}

// buildAncestorChain builds the ancestor chain for a node.
func (cursorCtx *CursorContext) buildAncestorChain(node *UnifiedNode) []*UnifiedNode {
	var chain []*UnifiedNode
	current := node
	for current != nil {
		chain = append([]*UnifiedNode{current}, chain...)
		current = current.Parent
	}
	return chain
}

// extractTextContext extracts text context from the document content.
func (cursorCtx *CursorContext) extractTextContext(content string, pos source.Position) {
	lines := strings.Split(content, "\n")
	lineIndex := pos.Line - 1

	if lineIndex < 0 || lineIndex >= len(lines) {
		return
	}

	cursorCtx.CurrentLine = lines[lineIndex]
	colIndex := max(pos.Column-1, 0)
	colIndex = min(colIndex, len(cursorCtx.CurrentLine))

	cursorCtx.TextBefore = cursorCtx.CurrentLine[:colIndex]
	cursorCtx.TextAfter = cursorCtx.CurrentLine[colIndex:]
	cursorCtx.CurrentWord = extractWordAtPosition(cursorCtx.CurrentLine, colIndex)
}

// detectSyntacticContext determines the syntactic editing context.
func (cursorCtx *CursorContext) detectSyntacticContext(format DocumentFormat) *SyntacticContext {
	style := DetectSyntacticStyle(cursorCtx.AncestorNodes, format)
	position := DetectSyntacticPosition(
		style,
		cursorCtx.UnifiedNode,
		cursorCtx.AncestorNodes,
		cursorCtx.Position,
		cursorCtx.TextBefore,
	)

	return &SyntacticContext{
		Position:         position,
		Style:            style,
		TypedPrefix:      ExtractTypedPrefix(position, style, cursorCtx.TextBefore),
		QuoteType:        cursorCtx.detectQuoteType(),
		InSubstitution:   cursorCtx.detectInSubstitution(),
		SubstitutionText: cursorCtx.extractSubstitutionText(),
	}
}

// resolveSchemaContext correlates with schema tree if available.
func (cursorCtx *CursorContext) resolveSchemaContext(ctx *DocumentContext) {
	if cursorCtx.UnifiedNode != nil && ctx.SchemaTree != nil {
		cursorCtx.SchemaNode = ctx.findCorrespondingSchemaNode(cursorCtx.StructuralPath)
		if cursorCtx.SchemaNode != nil {
			cursorCtx.SchemaElement = cursorCtx.SchemaNode.SchemaElement
		}
	}

	// If no AST is available, fall back to schema tree directly
	if cursorCtx.UnifiedNode == nil && ctx.SchemaTree != nil {
		collected := ctx.CollectSchemaNodesAtPosition(cursorCtx.Position, 0)
		if len(collected) > 0 {
			cursorCtx.SchemaNode = collected[len(collected)-1]
			cursorCtx.SchemaElement = cursorCtx.SchemaNode.SchemaElement
			cursorCtx.StructuralPath = schemaPathToStructuredPath(cursorCtx.SchemaNode.Path)
		}
	}
}

// detectQuoteType determines the enclosing quote type.
func (cursorCtx *CursorContext) detectQuoteType() QuoteType {
	for _, ancestor := range cursorCtx.AncestorNodes {
		switch ancestor.TSKind {
		case "double_quote_scalar": // YAML
			return QuoteTypeDouble
		case "single_quote_scalar": // YAML
			return QuoteTypeSingle
		case "string_scalar", "string_content", "string": // JSON/JSONC
			return QuoteTypeDouble
		case "plain_scalar", "block_scalar":
			return QuoteTypeNone
		}
	}
	return QuoteTypeNone
}

// detectInSubstitution checks if the cursor is inside a ${...} substitution.
func (cursorCtx *CursorContext) detectInSubstitution() bool {
	if cursorCtx.TextBefore == "" {
		return false
	}
	openCount := strings.Count(cursorCtx.TextBefore, "${")
	closeCount := strings.Count(cursorCtx.TextBefore, "}")
	return openCount > closeCount
}

// extractSubstitutionText returns the text inside the current substitution.
func (cursorCtx *CursorContext) extractSubstitutionText() string {
	if !cursorCtx.detectInSubstitution() {
		return ""
	}
	lastOpen := strings.LastIndex(cursorCtx.TextBefore, "${")
	if lastOpen == -1 {
		return ""
	}
	return cursorCtx.TextBefore[lastOpen+2:]
}

// --- Convenience methods that delegate to Syntactic with text-based fallbacks ---

// IsAtKeyPosition returns true if the cursor is at a key/field name position.
func (ctx *CursorContext) IsAtKeyPosition() bool {
	if ctx.Syntactic != nil {
		return ctx.Syntactic.IsAtKeyPosition()
	}
	return ctx.isAtKeyPositionFallback()
}

// isAtKeyPositionFallback provides text-based key position detection.
func (ctx *CursorContext) isAtKeyPositionFallback() bool {
	// Only consider the current line (text after the last newline)
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
	trimmedEnd := strings.TrimRight(trimmed, " \t")
	if strings.HasSuffix(trimmedEnd, "{") || strings.HasSuffix(trimmedEnd, ",") {
		return true
	}

	// If there's a colon followed only by whitespace, we're at a value position
	if strings.Contains(trimmed, ":") {
		afterColon := strings.TrimSpace(trimmed[strings.LastIndex(trimmed, ":")+1:])
		if afterColon == "" {
			return false // Value position after colon
		}
	}

	// If text before doesn't contain a colon and is just a word being typed, we're typing a key name
	if !strings.Contains(trimmed, ":") {
		return true
	}

	return false
}

// IsAtValuePosition returns true if the cursor is at a value position.
func (ctx *CursorContext) IsAtValuePosition() bool {
	if ctx.Syntactic != nil {
		return ctx.Syntactic.IsAtValuePosition()
	}
	return ctx.isAtValuePositionFallback()
}

// isAtValuePositionFallback provides text-based value position detection.
func (ctx *CursorContext) isAtValuePositionFallback() bool {
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

// IsAtSequenceItemPosition returns true if the cursor is at a sequence item position.
func (ctx *CursorContext) IsAtSequenceItemPosition() bool {
	if ctx.Syntactic != nil {
		return ctx.Syntactic.IsAtSequenceItemPosition()
	}
	return ctx.isAtSequenceItemPositionFallback()
}

// isAtSequenceItemPositionFallback provides text-based sequence item detection.
func (ctx *CursorContext) isAtSequenceItemPositionFallback() bool {
	// Only consider the current line (text after the last newline)
	currentLineText := ctx.TextBefore
	if lastNewline := strings.LastIndex(ctx.TextBefore, "\n"); lastNewline >= 0 {
		currentLineText = ctx.TextBefore[lastNewline+1:]
	}

	trimmed := strings.TrimLeft(currentLineText, " \t")

	// Check for YAML sequence item: starts with "- " or just "-"
	if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
		return true
	}

	return false
}

// GetTypedPrefix returns the text being typed at the current position.
func (ctx *CursorContext) GetTypedPrefix() string {
	if ctx.Syntactic != nil {
		return ctx.Syntactic.TypedPrefix
	}
	return ctx.getTypedPrefixFallback()
}

// getTypedPrefixFallback provides text-based typed prefix extraction.
func (ctx *CursorContext) getTypedPrefixFallback() string {
	// Only consider the current line (text after the last newline)
	currentLineText := ctx.TextBefore
	if lastNewline := strings.LastIndex(ctx.TextBefore, "\n"); lastNewline >= 0 {
		currentLineText = ctx.TextBefore[lastNewline+1:]
	}

	// For YAML sequence item positions, return what's after "- "
	if ctx.isAtSequenceItemPositionFallback() {
		trimmed := strings.TrimLeft(currentLineText, " \t")
		if strings.HasPrefix(trimmed, "- ") {
			return strings.TrimLeft(trimmed[2:], " ")
		}
		return ""
	}

	// For JSONC empty array values, return empty string
	if ctx.isInsideJSONCEmptyArray(currentLineText) {
		return ""
	}

	// For JSONC array values (inside a string within an array)
	if ctx.isInsideJSONCArrayString(currentLineText) {
		lastBracket := strings.LastIndex(currentLineText, "[")
		if lastBracket >= 0 {
			afterBracket := currentLineText[lastBracket+1:]
			lastQuote := strings.LastIndex(afterBracket, "\"")
			if lastQuote >= 0 {
				return afterBracket[lastQuote+1:]
			}
		}
		return ""
	}

	// For key positions, return the current word being typed
	if ctx.isAtKeyPositionFallback() {
		trimmed := strings.TrimLeft(currentLineText, " \t")
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
	if ctx.isAtValuePositionFallback() {
		if idx := strings.LastIndex(currentLineText, ":"); idx >= 0 {
			return strings.TrimSpace(currentLineText[idx+1:])
		}
	}

	return ctx.CurrentWord
}

// isInsideJSONCArrayString returns true if inside a string within a JSON array.
func (ctx *CursorContext) isInsideJSONCArrayString(currentLineText string) bool {
	lastBracket := strings.LastIndex(currentLineText, "[")
	if lastBracket < 0 {
		return false
	}

	afterBracket := currentLineText[lastBracket+1:]
	quoteCount := strings.Count(afterBracket, "\"")

	lastCloseBracket := strings.LastIndex(afterBracket, "]")
	if lastCloseBracket >= 0 {
		lastQuote := strings.LastIndex(afterBracket, "\"")
		if lastCloseBracket > lastQuote {
			return false
		}
	}

	return quoteCount%2 == 1
}

// isInsideJSONCEmptyArray returns true if inside an empty JSON array.
func (ctx *CursorContext) isInsideJSONCEmptyArray(currentLineText string) bool {
	lastBracket := strings.LastIndex(currentLineText, "[")
	if lastBracket < 0 {
		return false
	}

	afterBracket := currentLineText[lastBracket+1:]
	trimmed := strings.TrimSpace(afterBracket)
	return trimmed == ""
}

// InSubstitution returns true if the cursor is inside a ${...} substitution.
func (ctx *CursorContext) InSubstitution() bool {
	// If Syntactic is available, use it
	if ctx.Syntactic != nil {
		return ctx.Syntactic.InSubstitution
	}

	// Fall back to text-based detection (used in tests and when Syntactic is not set)
	if ctx.TextBefore == "" {
		return false
	}

	// Check for unclosed ${ before cursor
	openCount := strings.Count(ctx.TextBefore, "${")
	closeCount := strings.Count(ctx.TextBefore, "}")

	return openCount > closeCount
}

// GetSubstitutionText returns the text inside the current substitution.
func (ctx *CursorContext) GetSubstitutionText() string {
	// If Syntactic is available, use it
	if ctx.Syntactic != nil {
		return ctx.Syntactic.SubstitutionText
	}

	// Fall back to text-based detection
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

// IsBlockStyle returns true if the context is in block-style YAML.
func (ctx *CursorContext) IsBlockStyle() bool {
	return ctx.Syntactic != nil && ctx.Syntactic.Style.IsBlockStyle()
}

// IsFlowStyle returns true if the context is in flow-style YAML or JSONC.
func (ctx *CursorContext) IsFlowStyle() bool {
	return ctx.Syntactic != nil && ctx.Syntactic.Style.IsFlowStyle()
}

// GetEnclosingQuoteType returns the quote type of the enclosing string.
func (ctx *CursorContext) GetEnclosingQuoteType() QuoteType {
	if ctx.Syntactic == nil {
		return QuoteTypeNone
	}
	return ctx.Syntactic.QuoteType
}

// --- Structural path convenience methods (delegate to StructuralPath) ---

// IsInResources returns true if the position is under the resources section.
func (ctx *CursorContext) IsInResources() bool {
	return ctx.StructuralPath.IsInResources()
}

// IsInDataSources returns true if the position is under the datasources section.
func (ctx *CursorContext) IsInDataSources() bool {
	return ctx.StructuralPath.IsInDataSources()
}

// GetResourceName returns the resource name if under a resource.
func (ctx *CursorContext) GetResourceName() (string, bool) {
	return ctx.StructuralPath.GetResourceName()
}

// GetDataSourceName returns the data source name if under a data source.
func (ctx *CursorContext) GetDataSourceName() (string, bool) {
	return ctx.StructuralPath.GetDataSourceName()
}

// IsAtResourceSpec returns true if the position is in a resource spec.
func (ctx *CursorContext) IsAtResourceSpec() bool {
	return ctx.StructuralPath.IsResourceSpec()
}

// IsAtResourceType returns true if the position is at a resource type field.
func (ctx *CursorContext) IsAtResourceType() bool {
	return ctx.StructuralPath.IsResourceType()
}

// --- Schema-related methods ---

// ElementKind returns the kind of schema element at this context.
func (ctx *CursorContext) ElementKind() SchemaElementKind {
	if ctx.SchemaElement != nil {
		return KindFromSchemaElement(ctx.SchemaElement)
	}
	return SchemaElementUnknown
}

// IsAtTypeField returns true if the position is at a type field.
func (ctx *CursorContext) IsAtTypeField() bool {
	return ctx.StructuralPath.IsResourceType() ||
		ctx.StructuralPath.IsDataSourceType() ||
		ctx.StructuralPath.IsVariableType() ||
		ctx.StructuralPath.IsValueType() ||
		ctx.StructuralPath.IsExportType()
}

// IsAtDataSourceFilter returns true if the position is in a data source filter.
func (ctx *CursorContext) IsAtDataSourceFilter() bool {
	return ctx.StructuralPath.IsDataSourceFilter()
}

// GetVariableName returns the variable name if under a variable.
func (ctx *CursorContext) GetVariableName() (string, bool) {
	return ctx.StructuralPath.GetVariableName()
}

// GetValueName returns the value name if under a value.
func (ctx *CursorContext) GetValueName() (string, bool) {
	return ctx.StructuralPath.GetValueName()
}

// HasError returns true if the position is in an error region.
func (ctx *CursorContext) HasError() bool {
	if ctx.UnifiedNode == nil {
		return false
	}
	return ctx.UnifiedNode.IsError
}

// IsEmpty returns true if no node was found at the position.
func (ctx *CursorContext) IsEmpty() bool {
	return ctx.UnifiedNode == nil
}

// GetRange returns the range of the current node.
func (ctx *CursorContext) GetRange() *source.Range {
	if ctx.UnifiedNode == nil {
		return nil
	}
	return &ctx.UnifiedNode.Range
}

// GetKeyRange returns the key range if this is a map value.
func (ctx *CursorContext) GetKeyRange() *source.Range {
	if ctx.UnifiedNode == nil {
		return nil
	}
	return ctx.UnifiedNode.KeyRange
}

// GetValue returns the scalar value if this is a scalar node.
func (ctx *CursorContext) GetValue() string {
	if ctx.UnifiedNode == nil {
		return ""
	}
	return ctx.UnifiedNode.Value
}

// GetFieldName returns the field name if this is a map value.
func (ctx *CursorContext) GetFieldName() string {
	if ctx.UnifiedNode == nil {
		return ""
	}
	return ctx.UnifiedNode.FieldName
}

// IsPrecededByOperatorField returns true if the cursor is right after "operator:".
func (ctx *CursorContext) IsPrecededByOperatorField() bool {
	return operatorFieldPattern.MatchString(ctx.TextBefore)
}

// GetASTPath returns the structural path.
// Deprecated: Use StructuralPath field directly.
func (ctx *CursorContext) GetASTPath() StructuredPath {
	return ctx.StructuralPath
}

// --- Helper functions ---

// extractWordAtPosition extracts the word at the given column position.
func extractWordAtPosition(line string, column int) string {
	if len(line) == 0 || column < 0 || column > len(line) {
		return ""
	}

	// Find word start
	start := column
	for start > 0 && isWordChar(line[start-1]) {
		start -= 1
	}

	// Find word end
	end := column
	for end < len(line) && isWordChar(line[end]) {
		end += 1
	}

	if start == end {
		return ""
	}
	return line[start:end]
}

// isWordChar returns true if c is a word character (alphanumeric or underscore).
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// isWhitespaceOnly returns true if the string contains only whitespace (spaces and tabs)
// or is empty.
func isWhitespaceOnly(s string) bool {
	for _, c := range s {
		if c != ' ' && c != '\t' {
			return false
		}
	}
	return true
}

// defaultTabWidth is the assumed tab width for visual indent calculation.
// This is used when we need to convert tabs to visual columns but don't
// know the editor's actual tab width setting.
const defaultTabWidth = 4

// calculateVisualIndent calculates the visual indentation of a line,
// treating tabs as moving to the next tab stop (multiples of defaultTabWidth).
// This provides a reasonable approximation when we don't know the editor's
// actual tab width setting.
func calculateVisualIndent(line string) int {
	visualCol := 0
L:
	for _, c := range line {
		switch c {
		case ' ':
			visualCol += 1
		case '\t':
			// Move to next tab stop
			visualCol = ((visualCol / defaultTabWidth) + 1) * defaultTabWidth
		default:
			break L
		}
	}
	return visualCol
}
