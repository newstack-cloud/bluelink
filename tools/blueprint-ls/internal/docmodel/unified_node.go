package docmodel

import "github.com/newstack-cloud/bluelink/libs/blueprint/source"

// NodeKind represents the structural type of a unified AST node.
type NodeKind int

const (
	NodeKindDocument NodeKind = iota
	NodeKindMapping
	NodeKindSequence
	NodeKindScalar
	NodeKindKey
	NodeKindError // Marks invalid/error regions from tree-sitter
)

// UnifiedNode provides a format-agnostic representation of an AST node
// with accurate position information. It abstracts over YAML and JSON
// AST differences to provide consistent position-based lookups.
type UnifiedNode struct {
	Kind      NodeKind
	Range     source.Range
	KeyRange  *source.Range   // Range of the key if this is a map value
	Value     string          // Scalar value if NodeKindScalar
	Tag       string          // Type tag (e.g., "string", "integer")
	Children  []*UnifiedNode  // Child nodes
	Parent    *UnifiedNode    // Parent reference for upward traversal
	FieldName string          // Field name if child of a mapping
	Index     int             // Index if child of a sequence (-1 if not applicable)
	IsError   bool            // True if this node represents an error region
	TSKind    string          // Original tree-sitter node kind for debugging
}

// ContainsPosition returns true if the node's range contains the given position.
func (n *UnifiedNode) ContainsPosition(pos source.Position) bool {
	return containsPosition(n.Range, pos, 0)
}

// ContainsPositionWithLeeway returns true if the node's range contains the
// given position with configurable column leeway for fuzzy matching.
func (n *UnifiedNode) ContainsPositionWithLeeway(pos source.Position, leeway int) bool {
	return containsPosition(n.Range, pos, leeway)
}

// Path returns the structural path to this node (e.g., "/resources/myResource/type").
// Uses "/" as separator since "." can appear in element names.
func (n *UnifiedNode) Path() string {
	segments := n.AncestorPath()
	if len(segments) == 0 {
		return "/"
	}

	path := ""
	for _, seg := range segments {
		path += "/" + seg.String()
	}
	return path
}

// AncestorPath returns the path segments from root to this node.
// Only segments with meaningful field names or indices are included.
func (n *UnifiedNode) AncestorPath() []PathSegment {
	var segments []PathSegment
	current := n

	for current != nil && current.Parent != nil {
		// Only add segments that have meaningful identifiers
		if current.Index >= 0 {
			seg := PathSegment{Kind: PathSegmentIndex, Index: current.Index}
			segments = append([]PathSegment{seg}, segments...)
		} else if current.FieldName != "" {
			seg := PathSegment{Kind: PathSegmentField, FieldName: current.FieldName}
			segments = append([]PathSegment{seg}, segments...)
		}
		// Skip nodes without field names or indices (structural nodes like block_node)
		current = current.Parent
	}

	return segments
}

// DeepestChildAt returns the deepest child node containing the position.
// Returns the node itself if no children contain the position.
func (n *UnifiedNode) DeepestChildAt(pos source.Position, leeway int) *UnifiedNode {
	if !n.ContainsPositionWithLeeway(pos, leeway) {
		return nil
	}

	for _, child := range n.Children {
		if child.ContainsPositionWithLeeway(pos, leeway) {
			return child.DeepestChildAt(pos, leeway)
		}
	}

	return n
}

// CollectAncestors returns all ancestor nodes from root to this node (inclusive).
func (n *UnifiedNode) CollectAncestors() []*UnifiedNode {
	var ancestors []*UnifiedNode
	current := n

	for current != nil {
		ancestors = append([]*UnifiedNode{current}, ancestors...)
		current = current.Parent
	}

	return ancestors
}

// IsLeaf returns true if this node has no children.
func (n *UnifiedNode) IsLeaf() bool {
	return len(n.Children) == 0
}

// containsPosition checks if a range contains a position with optional leeway.
// Leeway is applied asymmetrically: it extends the START boundary (to catch
// positions slightly before the node) but NOT the END boundary (positions after
// a node's end are clearly outside it).
func containsPosition(r source.Range, pos source.Position, leeway int) bool {
	if r.Start == nil {
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
			pos.Column <= r.End.Column
	}

	// Position on start line
	if pos.Line == r.Start.Line {
		return pos.Column >= r.Start.Column-leeway
	}

	// Position on end line - no leeway extension past the end
	if pos.Line == r.End.Line {
		return pos.Column <= r.End.Column
	}

	// Position on middle lines
	return pos.Line > r.Start.Line && pos.Line < r.End.Line
}

// String returns a string representation of NodeKind for debugging.
func (k NodeKind) String() string {
	switch k {
	case NodeKindDocument:
		return "document"
	case NodeKindMapping:
		return "mapping"
	case NodeKindSequence:
		return "sequence"
	case NodeKindScalar:
		return "scalar"
	case NodeKindKey:
		return "key"
	case NodeKindError:
		return "error"
	default:
		return "unknown"
	}
}
