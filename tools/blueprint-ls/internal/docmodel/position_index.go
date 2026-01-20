package docmodel

import "github.com/newstack-cloud/bluelink/libs/blueprint/source"

// PositionIndex provides efficient lookup of nodes at a given position.
// It indexes nodes by line number for O(1) line lookup, then filters by column.
type PositionIndex struct {
	root     *UnifiedNode
	lineMap  map[int][]*UnifiedNode // Nodes indexed by start line
	allNodes []*UnifiedNode         // Flat list of all nodes
	maxLine  int
}

// NewPositionIndex builds an index from a unified node tree.
func NewPositionIndex(root *UnifiedNode) *PositionIndex {
	idx := &PositionIndex{
		root:    root,
		lineMap: make(map[int][]*UnifiedNode),
	}

	if root != nil {
		idx.indexNode(root)
	}

	return idx
}

// indexNode recursively indexes a node and its children.
func (idx *PositionIndex) indexNode(node *UnifiedNode) {
	if node == nil || node.Range.Start == nil {
		return
	}

	startLine := node.Range.Start.Line
	idx.lineMap[startLine] = append(idx.lineMap[startLine], node)
	idx.allNodes = append(idx.allNodes, node)

	if startLine > idx.maxLine {
		idx.maxLine = startLine
	}

	// Also index by end line if it spans multiple lines
	if node.Range.End != nil && node.Range.End.Line > startLine {
		for line := startLine + 1; line <= node.Range.End.Line; line++ {
			idx.lineMap[line] = append(idx.lineMap[line], node)
			if line > idx.maxLine {
				idx.maxLine = line
			}
		}
	}

	for _, child := range node.Children {
		idx.indexNode(child)
	}
}

// NodesAtPosition returns all nodes containing the given position,
// ordered from root to most specific (deepest).
func (idx *PositionIndex) NodesAtPosition(pos source.Position, leeway int) []*UnifiedNode {
	if idx.root == nil {
		return nil
	}

	// Get candidate nodes from line map
	candidates := idx.lineMap[pos.Line]
	if len(candidates) == 0 {
		return nil
	}

	// Filter candidates that actually contain the position
	var matching []*UnifiedNode
	for _, node := range candidates {
		if node.ContainsPositionWithLeeway(pos, leeway) {
			matching = append(matching, node)
		}
	}

	// Sort by specificity (depth) - deeper nodes should come last
	sortByDepth(matching)

	return matching
}

// DeepestNodeAtPosition returns the most specific node at the position.
func (idx *PositionIndex) DeepestNodeAtPosition(pos source.Position, leeway int) *UnifiedNode {
	nodes := idx.NodesAtPosition(pos, leeway)
	if len(nodes) == 0 {
		return nil
	}
	return nodes[len(nodes)-1]
}

// NodesOnLine returns all nodes that intersect with the given line.
func (idx *PositionIndex) NodesOnLine(line int) []*UnifiedNode {
	return idx.lineMap[line]
}

// Root returns the root node of the indexed tree.
func (idx *PositionIndex) Root() *UnifiedNode {
	return idx.root
}

// AllNodes returns all indexed nodes.
func (idx *PositionIndex) AllNodes() []*UnifiedNode {
	return idx.allNodes
}

// MaxLine returns the maximum line number in the index.
func (idx *PositionIndex) MaxLine() int {
	return idx.maxLine
}

// sortByDepth sorts nodes by their depth (number of ancestors).
// Deeper nodes come last in the slice.
func sortByDepth(nodes []*UnifiedNode) {
	if len(nodes) <= 1 {
		return
	}

	// Calculate depths
	depths := make(map[*UnifiedNode]int)
	for _, node := range nodes {
		depths[node] = calculateDepth(node)
	}

	// Simple insertion sort (typically few nodes)
	for i := 1; i < len(nodes); i++ {
		j := i
		for j > 0 && depths[nodes[j-1]] > depths[nodes[j]] {
			nodes[j-1], nodes[j] = nodes[j], nodes[j-1]
			j -= 1
		}
	}
}

// calculateDepth returns the depth of a node (distance from root).
func calculateDepth(node *UnifiedNode) int {
	depth := 0
	current := node
	for current.Parent != nil {
		depth++
		current = current.Parent
	}
	return depth
}

// FindNodeByPath finds a node by its path string.
func (idx *PositionIndex) FindNodeByPath(path string) *UnifiedNode {
	if idx.root == nil || path == "" || path == "/" {
		return idx.root
	}

	return findNodeByPathRecursive(idx.root, path)
}

// findNodeByPathRecursive recursively searches for a node by path.
func findNodeByPathRecursive(node *UnifiedNode, targetPath string) *UnifiedNode {
	if node == nil {
		return nil
	}

	if node.Path() == targetPath {
		return node
	}

	for _, child := range node.Children {
		if found := findNodeByPathRecursive(child, targetPath); found != nil {
			return found
		}
	}

	return nil
}
