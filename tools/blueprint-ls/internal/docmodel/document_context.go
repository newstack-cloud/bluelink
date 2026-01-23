package docmodel

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"go.uber.org/zap"
)

// DocumentFormat identifies the document format.
type DocumentFormat int

const (
	FormatYAML DocumentFormat = iota
	FormatJSONC
)

// DocumentStatus indicates the validity state of a document.
type DocumentStatus int

const (
	StatusValid         DocumentStatus = iota // Document parses and validates successfully
	StatusParsingErrors                       // Has parse errors but some features work
	StatusDegraded                            // Using stale data from last valid parse
	StatusUnavailable                         // No AST available
)

// DocumentContext provides unified access to document information for language features.
// It maintains both current and last-known-good state for robustness during editing.
type DocumentContext struct {
	URI     string
	Format  DocumentFormat
	Content string
	Version int

	// Current state (may be invalid)
	CurrentAST *UnifiedNode
	ParseError error
	Status     DocumentStatus

	// Last-known-good state (always valid when available)
	LastValidAST     *UnifiedNode
	LastValidSchema  *schema.Blueprint
	LastValidTree    *schema.TreeNode
	LastValidVersion int

	// Position index for efficient lookups
	PositionIndex *PositionIndex

	// Schema information (only available when document validates)
	Blueprint  *schema.Blueprint
	SchemaTree *schema.TreeNode

	logger *zap.Logger
}

// NewDocumentContext creates a new document context from content.
func NewDocumentContext(
	uri string,
	content string,
	format DocumentFormat,
	logger *zap.Logger,
) *DocumentContext {
	ctx := &DocumentContext{
		URI:     uri,
		Format:  format,
		Content: content,
		Version: 1,
		Status:  StatusUnavailable,
		logger:  logger,
	}

	ctx.parseContent()
	return ctx
}

// parseContent parses the document content using tree-sitter.
func (ctx *DocumentContext) parseContent() {
	var ast *UnifiedNode
	var err error

	switch ctx.Format {
	case FormatYAML:
		ast, err = ParseYAMLToUnified(ctx.Content)
	case FormatJSONC:
		ast, err = ParseJSONCToUnified(ctx.Content)
	}

	ctx.ParseError = err
	ctx.CurrentAST = ast

	if ast != nil {
		ctx.PositionIndex = NewPositionIndex(ast)

		// Only update last-known-good state if the AST is not an error node.
		// Tree-sitter produces partial ASTs even on parse errors, but these
		// shouldn't replace our last-known-good state.
		if !ast.IsError {
			ctx.Status = StatusValid
			ctx.LastValidAST = ast
			ctx.LastValidVersion = ctx.Version
		} else if ctx.LastValidAST != nil {
			ctx.Status = StatusParsingErrors
		} else {
			ctx.Status = StatusUnavailable
		}
	} else if ctx.LastValidAST != nil {
		ctx.Status = StatusDegraded
	} else {
		ctx.Status = StatusUnavailable
	}
}

// UpdateContent updates the document content and re-parses.
func (ctx *DocumentContext) UpdateContent(content string, version int) {
	ctx.Content = content
	ctx.Version = version
	ctx.parseContent()
}

// UpdateSchema sets the schema information after successful validation.
func (ctx *DocumentContext) UpdateSchema(blueprint *schema.Blueprint, tree *schema.TreeNode) {
	ctx.Blueprint = blueprint
	ctx.SchemaTree = tree

	if blueprint != nil {
		ctx.LastValidSchema = blueprint
		ctx.LastValidTree = tree
		ctx.LastValidVersion = ctx.Version
	}
}

// GetNodeContext returns context information for a position.
func (ctx *DocumentContext) GetNodeContext(pos source.Position, leeway int) *NodeContext {
	nodeCtx := &NodeContext{
		Position:      pos,
		DocumentCtx:   ctx,
		AncestorNodes: make([]*UnifiedNode, 0),
	}

	// Try current AST first
	if ctx.CurrentAST != nil && ctx.PositionIndex != nil {
		nodes := ctx.PositionIndex.NodesAtPosition(pos, leeway)
		if len(nodes) > 0 {
			nodeCtx.UnifiedNode = nodes[len(nodes)-1]
			nodeCtx.AncestorNodes = nodes
			nodeCtx.ASTPath = StructuredPath(nodeCtx.UnifiedNode.AncestorPath())
		}
	}

	// Fall back to last valid AST if current is unavailable
	if nodeCtx.UnifiedNode == nil && ctx.LastValidAST != nil {
		lastValidIndex := NewPositionIndex(ctx.LastValidAST)
		nodes := lastValidIndex.NodesAtPosition(pos, leeway)
		if len(nodes) > 0 {
			nodeCtx.UnifiedNode = nodes[len(nodes)-1]
			nodeCtx.AncestorNodes = nodes
			nodeCtx.ASTPath = StructuredPath(nodeCtx.UnifiedNode.AncestorPath())
			nodeCtx.IsStale = true
		}
	}

	// Correlate with schema tree if available
	if nodeCtx.UnifiedNode != nil && ctx.SchemaTree != nil {
		nodeCtx.SchemaNode = ctx.findCorrespondingSchemaNode(nodeCtx.ASTPath)
		if nodeCtx.SchemaNode != nil {
			nodeCtx.SchemaElement = nodeCtx.SchemaNode.SchemaElement
		}
	}

	// If no AST is available, fall back to schema tree directly
	if nodeCtx.UnifiedNode == nil && ctx.SchemaTree != nil {
		collected := ctx.CollectSchemaNodesAtPosition(pos, leeway)
		if len(collected) > 0 {
			nodeCtx.SchemaNode = collected[len(collected)-1]
			nodeCtx.SchemaElement = nodeCtx.SchemaNode.SchemaElement
			nodeCtx.ASTPath = schemaPathToStructuredPath(nodeCtx.SchemaNode.Path)
		}
	}

	// Extract text context for completion
	nodeCtx.extractTextContext(ctx.Content, pos)

	// For YAML documents, also try indentation-based detection.
	// This handles cases where:
	// 1. No path was found via position index
	// 2. A path was found, but there might be a deeper parent based on indentation
	//    (e.g., cursor is at child indent level after "spec:" with no value)
	if ctx.Format == FormatYAML {
		ctx.tryIndentationBasedContext(nodeCtx, pos)
	} else if len(nodeCtx.ASTPath) == 0 {
		// For non-YAML formats, only try if no path found
		ctx.tryIndentationBasedContext(nodeCtx, pos)
	}

	return nodeCtx
}

// tryIndentationBasedContext attempts to determine context from indentation.
// This handles cases where:
// 1. No AST node is found at the position (empty lines inside YAML blocks)
// 2. A node was found but a deeper parent exists based on indentation
//    (e.g., cursor at child indent after "spec:" with no value - spec node's
//    range doesn't extend to the cursor line but it's the correct parent)
func (ctx *DocumentContext) tryIndentationBasedContext(nodeCtx *NodeContext, pos source.Position) {
	if ctx.Content == "" {
		return
	}

	// Get the indentation level of the current line
	lines := strings.Split(ctx.Content, "\n")
	lineIndex := pos.Line - 1
	if lineIndex < 0 || lineIndex >= len(lines) {
		return
	}

	lineContent := lines[lineIndex]
	leadingSpaces := countLeadingSpaces(lineContent)

	// Determine the effective indentation level for parent lookup.
	// - If the line is empty or only whitespace, use the cursor column position
	//   (this handles the case where user presses Enter+Tab to add a child field)
	// - If the line has content, use the line's leading whitespace
	//   (this handles the case where user is typing a new sibling field)
	var currentIndent int
	lineHasContent := len(strings.TrimSpace(lineContent)) > 0
	if lineHasContent {
		// Line has content - use the leading whitespace to determine indent level
		// This ensures typing "export" at indent 4 finds orderTable (indent 2) as parent,
		// not metadata (which is a sibling at indent 4)
		currentIndent = leadingSpaces
	} else {
		// Empty/whitespace-only line - use cursor position
		// This handles Enter+Tab scenarios where cursor column indicates desired child level
		currentIndent = pos.Column - 1
		if currentIndent < 0 {
			currentIndent = 0
		}
	}

	// Search backwards to find the parent context based on indentation.
	// Prefer LastValidAST if the current AST has parse errors (is an error node),
	// because the last valid AST has the correct structure while the error AST
	// may have a broken/flattened structure.
	ast := ctx.CurrentAST
	if ast == nil || ast.IsError {
		ast = ctx.LastValidAST
	}
	if ast == nil {
		return
	}

	// Find the deepest node on a previous line that could contain this indentation
	parentNode := ctx.findParentByIndentation(ast, pos.Line, currentIndent)
	if parentNode == nil {
		return
	}

	newPath := StructuredPath(parentNode.AncestorPath())

	// Only update if the indentation-based result is deeper (longer path) than
	// what we already have. This ensures we find the most specific parent.
	if len(newPath) > len(nodeCtx.ASTPath) {
		nodeCtx.ASTPath = newPath
		nodeCtx.UnifiedNode = parentNode
		nodeCtx.IsStale = true

		// Correlate with schema tree
		if ctx.SchemaTree != nil {
			nodeCtx.SchemaNode = ctx.findCorrespondingSchemaNode(nodeCtx.ASTPath)
			if nodeCtx.SchemaNode != nil {
				nodeCtx.SchemaElement = nodeCtx.SchemaNode.SchemaElement
			}
		}
	}
}

// findParentByIndentation finds the deepest AST node on a previous line
// that could logically contain content at the given indentation level.
func (ctx *DocumentContext) findParentByIndentation(root *UnifiedNode, currentLine int, indent int) *UnifiedNode {
	if root == nil {
		return nil
	}

	var bestMatch *UnifiedNode
	ctx.walkNodesForIndent(root, currentLine, indent, &bestMatch)
	return bestMatch
}

// walkNodesForIndent recursively searches for nodes that could contain the indentation.
// It finds the deepest mapping node that could be the parent of a new field at the given indent.
func (ctx *DocumentContext) walkNodesForIndent(node *UnifiedNode, currentLine int, indent int, bestMatch **UnifiedNode) {
	if node == nil || node.Range.Start == nil {
		return
	}

	nodeStartLine := node.Range.Start.Line
	nodeStartCol := node.Range.Start.Column

	// Node must start before the current line
	if nodeStartLine >= currentLine {
		return
	}

	// For a node to be a valid parent for adding a new child field:
	// - The cursor's indent must be strictly greater than the node's start column.
	// - This ensures we find the PARENT mapping, not sibling fields at the same level.
	//
	// Note: nodeStartCol is 1-based (from the AST), while indent is 0-based
	// (calculated as pos.Column - 1). We need to convert nodeStartCol to 0-based
	// for an accurate comparison.
	//
	// Example: When adding "path:" at indent 4 (cursor at column 5) in:
	//   include:           (col 1, 0-based: 0)
	//     myInclude:       (col 3, 0-based: 2)
	//       metadata:      (col 5, 0-based: 4) <- existing sibling
	//
	// We want to find "myInclude" (0-based col 2) as the parent, not "metadata" (0-based col 4).
	// Since 4 > 2, we match myInclude. Since 4 is not > 4, we don't match metadata.
	nodeStartColZeroBased := nodeStartCol - 1
	if indent > nodeStartColZeroBased {
		// This node could contain our position - check if it's better than current match
		if *bestMatch == nil || nodeStartLine > (*bestMatch).Range.Start.Line {
			// Only consider mapping nodes (objects) as valid parents for field completion
			if node.Kind == NodeKindMapping {
				*bestMatch = node
			}
		}
	}

	// Recurse into children
	for _, child := range node.Children {
		ctx.walkNodesForIndent(child, currentLine, indent, bestMatch)
	}
}

// countLeadingSpaces returns the number of leading space characters in a string.
func countLeadingSpaces(s string) int {
	count := 0
	for _, c := range s {
		if c == ' ' {
			count += 1
		} else if c == '\t' {
			count += 2 // Treat tabs as 2 spaces
		} else {
			break
		}
	}
	return count
}

// schemaPathToStructuredPath converts a schema tree path string to a StructuredPath.
// Schema paths have the format "/section/name/field" (e.g., "/datasources/network/exports/vpc/type")
func schemaPathToStructuredPath(schemaPath string) StructuredPath {
	if schemaPath == "" || schemaPath == "/" {
		return StructuredPath{}
	}

	// Remove leading slash and split
	parts := strings.Split(strings.TrimPrefix(schemaPath, "/"), "/")
	segments := make([]PathSegment, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}
		segments = append(segments, PathSegment{
			Kind:      PathSegmentField,
			FieldName: part,
		})
	}

	return StructuredPath(segments)
}

// findCorrespondingSchemaNode finds the schema tree node that corresponds to a path.
func (ctx *DocumentContext) findCorrespondingSchemaNode(path StructuredPath) *schema.TreeNode {
	if ctx.SchemaTree == nil || len(path) == 0 {
		return ctx.SchemaTree
	}

	return findSchemaNodeByPath(ctx.SchemaTree, path.String())
}

// findSchemaNodeByPath searches for a schema node matching the path.
func findSchemaNodeByPath(node *schema.TreeNode, targetPath string) *schema.TreeNode {
	if node == nil {
		return nil
	}

	if node.Path == targetPath {
		return node
	}

	for _, child := range node.Children {
		if found := findSchemaNodeByPath(child, targetPath); found != nil {
			return found
		}
	}

	return nil
}

// CollectSchemaNodesAtPosition collects all schema tree nodes containing the position.
// This is used for hover to find the hoverable element in the ancestor chain.
func (ctx *DocumentContext) CollectSchemaNodesAtPosition(pos source.Position, leeway int) []*schema.TreeNode {
	if ctx.SchemaTree == nil {
		return nil
	}

	var collected []*schema.TreeNode
	collectSchemaNodes(ctx.SchemaTree, pos, leeway, &collected)
	return collected
}

// collectSchemaNodes recursively collects schema nodes containing the position.
func collectSchemaNodes(node *schema.TreeNode, pos source.Position, leeway int, collected *[]*schema.TreeNode) {
	if node == nil || node.Range == nil {
		return
	}

	if containsSchemaPosition(node.Range, pos, leeway) {
		*collected = append(*collected, node)
		for _, child := range node.Children {
			collectSchemaNodes(child, pos, leeway, collected)
		}
	}
}

// containsSchemaPosition checks if a schema range contains the position.
func containsSchemaPosition(r *source.Range, pos source.Position, leeway int) bool {
	if r == nil || r.Start == nil {
		return false
	}

	// Handle ranges without end position (open-ended)
	if r.End == nil {
		return pos.Line > r.Start.Line ||
			(pos.Line == r.Start.Line && pos.Column >= r.Start.Column-leeway)
	}

	// Single line range
	if pos.Line == r.Start.Line && pos.Line == r.End.Line {
		return pos.Column >= r.Start.Column-leeway &&
			pos.Column <= r.End.Column+leeway
	}

	// Position on start line
	if pos.Line == r.Start.Line {
		return pos.Column >= r.Start.Column-leeway
	}

	// Position on end line
	if pos.Line == r.End.Line {
		return pos.Column <= r.End.Column+leeway
	}

	// Position on middle lines
	return pos.Line > r.Start.Line && pos.Line < r.End.Line
}

// HasValidAST returns true if a valid AST is available (current or last-known-good).
func (ctx *DocumentContext) HasValidAST() bool {
	return ctx.CurrentAST != nil || ctx.LastValidAST != nil
}

// FindFunctionAtPosition finds the deepest function expression at the given position.
// This traverses the schema tree to find nested function calls.
func (ctx *DocumentContext) FindFunctionAtPosition(pos source.Position) *substitutions.SubstitutionFunctionExpr {
	if ctx.SchemaTree == nil {
		return nil
	}
	return findFunctionInTree(ctx.SchemaTree, pos)
}

// findFunctionInTree recursively searches for the deepest function at a position.
func findFunctionInTree(node *schema.TreeNode, pos source.Position) *substitutions.SubstitutionFunctionExpr {
	if node == nil {
		return nil
	}

	if !containsSchemaPosition(node.Range, pos, 0) {
		return nil
	}

	// Check if this node is a function
	subFunc, isFunc := node.SchemaElement.(*substitutions.SubstitutionFunctionExpr)

	// If this is a function with no children, return it
	if isFunc && len(node.Children) == 0 {
		return subFunc
	}

	// Search children for a deeper function
	for _, child := range node.Children {
		if childFunc := findFunctionInTree(child, pos); childFunc != nil {
			return childFunc
		}
	}

	// If we found a function at this level but no deeper one, return it
	if isFunc {
		return subFunc
	}

	return nil
}

// HasSchema returns true if schema information is available.
func (ctx *DocumentContext) HasSchema() bool {
	return ctx.Blueprint != nil
}

// GetEffectiveAST returns the current AST or falls back to last-known-good.
func (ctx *DocumentContext) GetEffectiveAST() *UnifiedNode {
	if ctx.CurrentAST != nil {
		return ctx.CurrentAST
	}
	return ctx.LastValidAST
}

// GetEffectiveSchema returns the current schema or falls back to last-known-good.
func (ctx *DocumentContext) GetEffectiveSchema() *schema.Blueprint {
	if ctx.Blueprint != nil {
		return ctx.Blueprint
	}
	return ctx.LastValidSchema
}

// String returns a string representation of DocumentFormat.
func (f DocumentFormat) String() string {
	switch f {
	case FormatYAML:
		return "yaml"
	case FormatJSONC:
		return "jsonc"
	default:
		return "unknown"
	}
}

// String returns a string representation of DocumentStatus.
func (s DocumentStatus) String() string {
	switch s {
	case StatusValid:
		return "valid"
	case StatusParsingErrors:
		return "parsing_errors"
	case StatusDegraded:
		return "degraded"
	case StatusUnavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

// NewDocumentContextFromSchema creates a DocumentContext from existing schema
// information. This is useful for transitioning from the old state management
// approach where schema and tree are already available.
func NewDocumentContextFromSchema(
	uri string,
	blueprint *schema.Blueprint,
	tree *schema.TreeNode,
) *DocumentContext {
	ctx := &DocumentContext{
		URI:        uri,
		Blueprint:  blueprint,
		SchemaTree: tree,
		Status:     StatusValid,
	}

	if blueprint != nil {
		ctx.LastValidSchema = blueprint
		ctx.LastValidTree = tree
	}

	return ctx
}
