package docmodel

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
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
	StatusValid          DocumentStatus = iota // Document parses and validates successfully
	StatusParsingErrors                        // Has parse errors but some features work
	StatusDegraded                             // Using stale data from last valid parse
	StatusUnavailable                          // No AST available
)

// DocumentContext provides unified access to document information for language features.
// It maintains both current and last-known-good state for robustness during editing.
type DocumentContext struct {
	URI     string
	Format  DocumentFormat
	Content string
	Version int

	// Current state (may be invalid)
	CurrentAST   *UnifiedNode
	ParseError   error
	Status       DocumentStatus

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

	// Extract text context for completion
	nodeCtx.extractTextContext(ctx.Content, pos)

	return nodeCtx
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

// HasValidAST returns true if a valid AST is available (current or last-known-good).
func (ctx *DocumentContext) HasValidAST() bool {
	return ctx.CurrentAST != nil || ctx.LastValidAST != nil
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
