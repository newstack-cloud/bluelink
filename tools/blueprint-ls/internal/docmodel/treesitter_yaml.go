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
// This function flattens the tree-sitter structure to produce a cleaner tree
// where mappings directly contain their key-value entries with fieldNames set.
func convertYAMLTreeSitterNode(
	tsNode *tree_sitter.Node,
	parent *UnifiedNode,
	content []byte,
) *UnifiedNode {
	if tsNode == nil {
		return nil
	}

	kind := tsNode.Kind()

	// Handle structural passthrough nodes by unwrapping them
	if isPassthroughNode(kind) {
		return unwrapPassthroughNode(tsNode, parent, content)
	}

	startPoint := tsNode.StartPosition()
	endPoint := tsNode.EndPosition()

	unified := &UnifiedNode{
		Kind:    mapYAMLNodeKind(kind),
		IsError: tsNode.IsError() || tsNode.IsMissing(),
		TSKind:  kind,
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

	// Extract tag for scalar type detection
	if isScalarNode(kind) {
		unified.Tag = scalarKindToTag(kind)
		if content != nil {
			unified.Value = tsNode.Utf8Text(content)
		}
	}

	// Process children based on node type
	switch kind {
	case "block_mapping", "flow_mapping":
		processYAMLMappingChildren(tsNode, unified, content)
	case "block_sequence", "flow_sequence":
		processYAMLSequenceChildren(tsNode, unified, content)
	default:
		processYAMLDefaultChildren(tsNode, unified, content)
	}

	return unified
}

// isPassthroughNode returns true for nodes that should be unwrapped.
func isPassthroughNode(kind string) bool {
	switch kind {
	case "stream", "block_node", "flow_node":
		return true
	}
	return false
}

// isScalarNode returns true for scalar value nodes.
func isScalarNode(kind string) bool {
	switch kind {
	case "plain_scalar", "single_quote_scalar", "double_quote_scalar",
		"block_scalar", "string_scalar", "integer_scalar", "float_scalar",
		"boolean_scalar", "null_scalar":
		return true
	}
	return false
}

// scalarKindToTag maps tree-sitter scalar kinds to type tags.
func scalarKindToTag(kind string) string {
	switch kind {
	case "integer_scalar":
		return "!!int"
	case "float_scalar":
		return "!!float"
	case "boolean_scalar":
		return "!!bool"
	case "null_scalar":
		return "!!null"
	default:
		return "!!str"
	}
}

// unwrapPassthroughNode recursively unwraps passthrough nodes to find the
// meaningful child node.
func unwrapPassthroughNode(
	tsNode *tree_sitter.Node,
	parent *UnifiedNode,
	content []byte,
) *UnifiedNode {
	childCount := tsNode.ChildCount()
	for i := range childCount {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		childKind := child.Kind()
		// Skip tokens like colons, dashes
		if childKind == ":" || childKind == "-" {
			continue
		}
		return convertYAMLTreeSitterNode(child, parent, content)
	}
	return nil
}

// processYAMLMappingChildren processes children of a mapping node.
// Extracts key-value pairs and sets fieldNames on value nodes.
func processYAMLMappingChildren(
	tsNode *tree_sitter.Node,
	unified *UnifiedNode,
	content []byte,
) {
	childCount := tsNode.ChildCount()
	for i := range childCount {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		childKind := child.Kind()

		if childKind == "block_mapping_pair" || childKind == "flow_pair" {
			processMappingPair(child, unified, content)
		}
	}
}

// processMappingPair extracts key and value from a mapping pair and adds
// the value as a child of the parent mapping with the key as fieldName.
// If the mapping pair has a key but no value (e.g., "spec:"), an empty mapping
// node is created to allow indentation-based lookups to find it as a parent.
func processMappingPair(
	pairNode *tree_sitter.Node,
	parentMapping *UnifiedNode,
	content []byte,
) {
	var keyNode, valueNode *tree_sitter.Node

	childCount := pairNode.ChildCount()
	foundColon := false
	for i := range childCount {
		child := pairNode.Child(i)
		if child == nil {
			continue
		}
		childKind := child.Kind()

		if childKind == ":" {
			foundColon = true
			continue
		}

		if !foundColon {
			// Before colon = key
			if keyNode == nil {
				keyNode = child
			}
		} else {
			// After colon = value
			if valueNode == nil {
				valueNode = child
			}
		}
	}

	if keyNode == nil {
		return
	}

	// Extract key string
	keyStr := extractScalarText(keyNode, content)

	// Handle case where there's a key but no value (e.g., "spec:")
	// Create an empty mapping node to represent the key, allowing
	// indentation-based lookups to find it as a potential parent.
	if valueNode == nil {
		pairStart := pairNode.StartPosition()
		pairEnd := pairNode.EndPosition()
		keyStart := keyNode.StartPosition()
		keyEnd := keyNode.EndPosition()

		emptyMapping := &UnifiedNode{
			Kind:      NodeKindMapping,
			TSKind:    "empty_mapping",
			Parent:    parentMapping,
			Index:     -1,
			FieldName: keyStr,
			Range: source.Range{
				Start: &source.Position{
					Line:   int(pairStart.Row) + 1,
					Column: int(pairStart.Column) + 1,
				},
				End: &source.Position{
					Line:   int(pairEnd.Row) + 1,
					Column: int(pairEnd.Column) + 1,
				},
			},
			KeyRange: &source.Range{
				Start: &source.Position{
					Line:   int(keyStart.Row) + 1,
					Column: int(keyStart.Column) + 1,
				},
				End: &source.Position{
					Line:   int(keyEnd.Row) + 1,
					Column: int(keyEnd.Column) + 1,
				},
			},
			Children: []*UnifiedNode{},
		}

		parentMapping.Children = append(parentMapping.Children, emptyMapping)
		return
	}

	// Convert value node
	valueUnified := convertYAMLTreeSitterNode(valueNode, parentMapping, content)
	if valueUnified == nil {
		return
	}

	// Set field name and key range
	valueUnified.FieldName = keyStr
	keyStart := keyNode.StartPosition()
	keyEnd := keyNode.EndPosition()
	valueUnified.KeyRange = &source.Range{
		Start: &source.Position{
			Line:   int(keyStart.Row) + 1,
			Column: int(keyStart.Column) + 1,
		},
		End: &source.Position{
			Line:   int(keyEnd.Row) + 1,
			Column: int(keyEnd.Column) + 1,
		},
	}

	// Update range to include key
	if valueUnified.Range.Start != nil {
		valueUnified.Range.Start = &source.Position{
			Line:   int(keyStart.Row) + 1,
			Column: int(keyStart.Column) + 1,
		}
	}

	parentMapping.Children = append(parentMapping.Children, valueUnified)
}

// processYAMLSequenceChildren processes children of a sequence node.
func processYAMLSequenceChildren(
	tsNode *tree_sitter.Node,
	unified *UnifiedNode,
	content []byte,
) {
	childCount := tsNode.ChildCount()
	index := 0
	for i := range childCount {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		childKind := child.Kind()

		if childKind == "block_sequence_item" || childKind == "flow_sequence_item" {
			// Extract the value from the sequence item
			itemValue := extractSequenceItemValue(child)
			if itemValue != nil {
				childUnified := convertYAMLTreeSitterNode(itemValue, unified, content)
				if childUnified != nil {
					childUnified.Index = index
					unified.Children = append(unified.Children, childUnified)
					index++
				}
			}
		} else if childKind != "-" && childKind != "," && childKind != "[" && childKind != "]" {
			// Direct children in flow sequences
			childUnified := convertYAMLTreeSitterNode(child, unified, content)
			if childUnified != nil {
				childUnified.Index = index
				unified.Children = append(unified.Children, childUnified)
				index++
			}
		}
	}
}

// extractSequenceItemValue extracts the value node from a sequence item.
func extractSequenceItemValue(itemNode *tree_sitter.Node) *tree_sitter.Node {
	childCount := itemNode.ChildCount()
	for i := range childCount {
		child := itemNode.Child(i)
		if child == nil {
			continue
		}
		childKind := child.Kind()
		if childKind != "-" && childKind != "," {
			return child
		}
	}
	return nil
}

// processYAMLDefaultChildren processes children for other node types.
func processYAMLDefaultChildren(
	tsNode *tree_sitter.Node,
	unified *UnifiedNode,
	content []byte,
) {
	childCount := tsNode.ChildCount()
	for i := range childCount {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		childKind := child.Kind()
		// Skip tokens
		if childKind == ":" || childKind == "-" || childKind == "," ||
			childKind == "[" || childKind == "]" || childKind == "{" || childKind == "}" {
			continue
		}

		childUnified := convertYAMLTreeSitterNode(child, unified, content)
		if childUnified != nil {
			unified.Children = append(unified.Children, childUnified)
		}
	}
}

// extractScalarText extracts the text value from a scalar node.
func extractScalarText(node *tree_sitter.Node, content []byte) string {
	if node == nil {
		return ""
	}

	kind := node.Kind()
	if isScalarNode(kind) {
		return node.Utf8Text(content)
	}

	// For wrapper nodes, recurse into children
	childCount := node.ChildCount()
	for i := range childCount {
		child := node.Child(i)
		if child == nil {
			continue
		}
		text := extractScalarText(child, content)
		if text != "" {
			return text
		}
	}

	return ""
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
	case "plain_scalar", "single_quote_scalar", "double_quote_scalar",
		"block_scalar", "string_scalar", "integer_scalar", "float_scalar",
		"boolean_scalar", "null_scalar":
		return NodeKindScalar
	case "ERROR":
		return NodeKindError
	default:
		// Most structural nodes (flow_node, block_node, block_mapping_pair, etc.)
		// are passthrough nodes that should be flattened during conversion
		return NodeKindScalar
	}
}

