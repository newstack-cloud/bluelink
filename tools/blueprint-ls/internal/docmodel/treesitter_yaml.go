package docmodel

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	tree_sitter_yaml "github.com/tree-sitter-grammars/tree-sitter-yaml/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// YAMLParser wraps tree-sitter for parsing YAML documents.
// Must call Close() when done to free C memory.
type YAMLParser struct {
	parser *tree_sitter.Parser
}

// NewYAMLParser creates a new YAML parser using tree-sitter.
func NewYAMLParser() (*YAMLParser, error) {
	parser := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(tree_sitter_yaml.Language())
	if err := parser.SetLanguage(lang); err != nil {
		return nil, err
	}
	return &YAMLParser{parser: parser}, nil
}

// Parse parses YAML content and returns the tree-sitter tree.
func (p *YAMLParser) Parse(content []byte) *tree_sitter.Tree {
	return p.parser.Parse(content, nil)
}

// ParseIncremental parses YAML content with an existing tree for incremental updates.
func (p *YAMLParser) ParseIncremental(content []byte, oldTree *tree_sitter.Tree) *tree_sitter.Tree {
	return p.parser.Parse(content, oldTree)
}

// Close releases the parser resources.
func (p *YAMLParser) Close() {
	p.parser.Close()
}

// ParseYAMLToUnified parses YAML content and converts to a UnifiedNode tree.
// This is a convenience function that handles parser lifecycle.
func ParseYAMLToUnified(content string) (*UnifiedNode, error) {
	parser, err := NewYAMLParser()
	if err != nil {
		return nil, err
	}
	defer parser.Close()

	tree := parser.Parse([]byte(content))
	if tree == nil {
		return nil, nil
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	if rootNode == nil {
		return nil, nil
	}

	return convertYAMLTreeSitterNode(rootNode, nil, []byte(content)), nil
}

// convertYAMLTreeSitterNode converts a tree-sitter YAML node to UnifiedNode.
func convertYAMLTreeSitterNode(
	tsNode *tree_sitter.Node,
	parent *UnifiedNode,
	content []byte,
) *UnifiedNode {
	if tsNode == nil {
		return nil
	}

	startPoint := tsNode.StartPosition()
	endPoint := tsNode.EndPosition()

	unified := &UnifiedNode{
		Kind:    mapYAMLNodeKind(tsNode.Kind()),
		IsError: tsNode.IsError() || tsNode.IsMissing(),
		TSKind:  tsNode.Kind(),
		Parent:  parent,
		Index:   -1,
		Range: source.Range{
			Start: &source.Position{
				Line:   int(startPoint.Row) + 1,
				Column: int(startPoint.Column) + 1,
			},
			End: &source.Position{
				Line:   int(endPoint.Row) + 1,
				Column: int(endPoint.Column) + 1,
			},
		},
	}

	// Extract scalar values
	if unified.Kind == NodeKindScalar && content != nil {
		unified.Value = tsNode.Utf8Text(content)
	}

	// Process children based on node type
	childCount := tsNode.ChildCount()
	for i := range childCount {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}

		childUnified := convertYAMLTreeSitterNode(child, unified, content)
		if childUnified != nil {
			unified.Children = append(unified.Children, childUnified)
		}
	}

	// Post-process to extract field names and indices from YAML structure
	postProcessYAMLNode(unified)

	return unified
}

// mapYAMLNodeKind maps tree-sitter YAML node kinds to UnifiedNode kinds.
func mapYAMLNodeKind(kind string) NodeKind {
	switch kind {
	case "stream", "document":
		return NodeKindDocument
	case "block_mapping", "flow_mapping":
		return NodeKindMapping
	case "block_sequence", "flow_sequence":
		return NodeKindSequence
	case "block_mapping_pair", "flow_pair":
		return NodeKindMapping
	case "plain_scalar", "single_quote_scalar", "double_quote_scalar",
		"block_scalar", "string_scalar", "integer_scalar", "float_scalar",
		"boolean_scalar", "null_scalar":
		return NodeKindScalar
	case "flow_node", "block_node":
		return NodeKindScalar
	case "ERROR":
		return NodeKindError
	default:
		return NodeKindScalar
	}
}

// postProcessYAMLNode extracts field names and indices from YAML AST structure.
func postProcessYAMLNode(node *UnifiedNode) {
	if node == nil {
		return
	}

	switch node.TSKind {
	case "block_mapping", "flow_mapping":
		processYAMLMapping(node)
	case "block_sequence", "flow_sequence":
		processYAMLSequence(node)
	case "block_mapping_pair", "flow_pair":
		processYAMLPair(node)
	}
}

// processYAMLMapping processes a mapping node to extract key-value pairs.
func processYAMLMapping(node *UnifiedNode) {
	for _, child := range node.Children {
		if child.TSKind == "block_mapping_pair" || child.TSKind == "flow_pair" {
			processYAMLPair(child)
		}
	}
}

// processYAMLSequence sets indices on sequence children.
func processYAMLSequence(node *UnifiedNode) {
	index := 0
	for _, child := range node.Children {
		if child.TSKind != "comment" && child.TSKind != "-" {
			child.Index = index
			index++
		}
	}
}

// processYAMLPair extracts the key and value from a mapping pair.
func processYAMLPair(node *UnifiedNode) {
	if len(node.Children) < 2 {
		return
	}

	// First child is typically the key, second is the value
	keyNode := findYAMLKeyNode(node)
	valueNode := findYAMLValueNode(node)

	if keyNode != nil && valueNode != nil {
		valueNode.FieldName = extractYAMLScalarValue(keyNode)
		valueNode.KeyRange = &source.Range{
			Start: keyNode.Range.Start,
			End:   keyNode.Range.End,
		}
	}
}

// findYAMLKeyNode finds the key node in a mapping pair.
func findYAMLKeyNode(pair *UnifiedNode) *UnifiedNode {
	for _, child := range pair.Children {
		if isYAMLKeyNode(child.TSKind) {
			return child
		}
	}
	return nil
}

// findYAMLValueNode finds the value node in a mapping pair.
func findYAMLValueNode(pair *UnifiedNode) *UnifiedNode {
	foundKey := false
	for _, child := range pair.Children {
		if isYAMLKeyNode(child.TSKind) {
			foundKey = true
			continue
		}
		if foundKey && child.TSKind != ":" {
			return child
		}
	}
	return nil
}

// isYAMLKeyNode returns true if the kind represents a key node.
func isYAMLKeyNode(kind string) bool {
	switch kind {
	case "plain_scalar", "single_quote_scalar", "double_quote_scalar",
		"flow_node", "block_node":
		return true
	}
	return false
}

// extractYAMLScalarValue extracts the string value from a scalar node.
func extractYAMLScalarValue(node *UnifiedNode) string {
	if node == nil {
		return ""
	}

	if node.Value != "" {
		return node.Value
	}

	// Look for scalar value in children
	for _, child := range node.Children {
		if child.Kind == NodeKindScalar && child.Value != "" {
			return child.Value
		}
		// Recursively check nested nodes
		val := extractYAMLScalarValue(child)
		if val != "" {
			return val
		}
	}

	return ""
}
