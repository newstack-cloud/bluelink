package languageservices

import (
	"sort"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// navigateMappingNode walks into a MappingNode tree following a sequence of field names.
// Returns the node at the end of the path, or nil if the path is invalid.
func navigateMappingNode(node *core.MappingNode, fieldNames []string) *core.MappingNode {
	current := node
	for _, name := range fieldNames {
		if current == nil || current.Fields == nil {
			return nil
		}
		next, exists := current.Fields[name]
		if !exists {
			return nil
		}
		current = next
	}
	return current
}

// navigateMappingNodeByIndex walks into a MappingNode array by index.
// Returns the item at the given index, or nil if out of bounds.
func navigateMappingNodeByIndex(node *core.MappingNode, index int) *core.MappingNode {
	if node == nil || node.Items == nil || index < 0 || index >= len(node.Items) {
		return nil
	}
	return node.Items[index]
}

// getMappingNodeFieldKeys returns sorted field keys from a MappingNode with Fields.
func getMappingNodeFieldKeys(node *core.MappingNode) []string {
	if node == nil || node.Fields == nil {
		return nil
	}
	keys := make([]string, 0, len(node.Fields))
	for k := range node.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// getMappingNodeArrayLength returns the length of a MappingNode's Items array.
func getMappingNodeArrayLength(node *core.MappingNode) int {
	if node == nil || node.Items == nil {
		return 0
	}
	return len(node.Items)
}

// isMappingNodeTerminal returns true if the node has no further navigable children.
// A terminal node is a scalar value, a string with substitutions, or nil.
func isMappingNodeTerminal(node *core.MappingNode) bool {
	if node == nil {
		return true
	}
	return node.Scalar != nil || node.StringWithSubstitutions != nil
}

// isMappingNodeObject returns true if the node has a Fields map.
func isMappingNodeObject(node *core.MappingNode) bool {
	return node != nil && node.Fields != nil && len(node.Fields) > 0
}

// isMappingNodeArray returns true if the node has an Items slice.
func isMappingNodeArray(node *core.MappingNode) bool {
	return node != nil && node.Items != nil && len(node.Items) > 0
}
