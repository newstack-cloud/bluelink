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
		ctx.Status = StatusValid

		// Update last-known-good state
		ctx.LastValidAST = ast
		ctx.LastValidVersion = ctx.Version
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

	// If still no path found and we have an AST, try position-based detection
	// For YAML: uses indentation to find parent context on empty lines
	if len(nodeCtx.ASTPath) == 0 {
		ctx.tryIndentationBasedContext(nodeCtx, pos)
	}

	return nodeCtx
}

// tryIndentationBasedContext attempts to determine context from indentation
// when no AST node is found at the position. This handles empty lines inside
// YAML blocks where the cursor position doesn't match any existing node.
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

	currentIndent := countLeadingSpaces(lines[lineIndex])

	// If cursor is at position 1 (before any indent), use the indent of where
	// the cursor would naturally be typing (the current line's indent or cursor column)
	if pos.Column > 1 {
		currentIndent = pos.Column - 1
	}

	// Search backwards to find the parent context based on indentation
	ast := ctx.CurrentAST
	if ast == nil {
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

	nodeCtx.ASTPath = StructuredPath(parentNode.AncestorPath())
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
func (ctx *DocumentContext) walkNodesForIndent(node *UnifiedNode, currentLine int, indent int, bestMatch **UnifiedNode) {
	if node == nil || node.Range.Start == nil {
		return
	}

	nodeStartLine := node.Range.Start.Line
	nodeStartCol := node.Range.Start.Column
	nodeEndLine := currentLine // Default: assume extends to current line if no end

	if node.Range.End != nil {
		nodeEndLine = node.Range.End.Line
	}

	// Node must start before the current line and extend to or past it
	if nodeStartLine >= currentLine {
		return
	}

	// For a node to be a valid parent, our indent must be greater than the node's start column
	// (meaning we're indented inside the node's content)
	if indent > nodeStartCol && nodeEndLine >= currentLine {
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
