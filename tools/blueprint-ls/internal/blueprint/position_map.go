package blueprint

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

// CreatePositionMap creates a map of positions to tree nodes
// for a given tree.
// This allows for efficient lookup of elements in a blueprint
// based on their starting location.
//
// This produces a list of tree nodes for each start position key,
// with the element with the smallest range being the last element
// in the list (Due to order of traversal).
//
// This uses the `PositionKey` function to encode the start position
// as a string key.
// Positions are expected to be 1-indexed, further conversion is needed
// to produce 0-indexed positions compatible with the LSP.
func CreatePositionMap(tree *schema.TreeNode) map[string][]*schema.TreeNode {
	positionMap := map[string][]*schema.TreeNode{}
	populatePositionMap(tree, positionMap)
	return positionMap
}

func populatePositionMap(
	tree *schema.TreeNode,
	positionMap map[string][]*schema.TreeNode,
) {
	if tree == nil || tree.Range == nil || tree.Range.Start == nil {
		return
	}

	existingNodes, ok := positionMap[PositionKey(tree.Range.Start)]
	if !ok {
		positionMap[PositionKey(tree.Range.Start)] = []*schema.TreeNode{tree}
	} else {
		positionMap[PositionKey(tree.Range.Start)] = append(existingNodes, tree)
	}

	for _, child := range tree.Children {
		populatePositionMap(child, positionMap)
	}
}

// PositionKey encodes a position as a string key.
// This produces a key of the form `line:column`.
// If the provided source meta struct is nil, `1:1`
// is returned.
func PositionKey(pos *source.Position) string {
	if pos == nil {
		return "1:1"
	}

	return fmt.Sprintf("%d:%d", pos.Line, pos.Column)
}
