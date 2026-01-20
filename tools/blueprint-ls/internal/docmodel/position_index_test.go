package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/stretchr/testify/suite"
)

type PositionIndexSuite struct {
	suite.Suite
}

func (s *PositionIndexSuite) TestNewPositionIndex_NilRoot() {
	idx := NewPositionIndex(nil)
	s.Require().NotNil(idx)
	s.Assert().Nil(idx.Root())
	s.Assert().Empty(idx.AllNodes())
}

func (s *PositionIndexSuite) TestNodesAtPosition() {
	root := s.createTestTree()
	idx := NewPositionIndex(root)

	tests := []struct {
		name          string
		pos           source.Position
		leeway        int
		expectedCount int
		expectedLast  string
	}{
		{
			name:          "position in deepest node",
			pos:           source.Position{Line: 5, Column: 10},
			leeway:        0,
			expectedCount: 4,
			expectedLast:  "type",
		},
		{
			name:          "position in middle node",
			pos:           source.Position{Line: 3, Column: 5},
			leeway:        0,
			expectedCount: 3,
			expectedLast:  "myResource",
		},
		{
			name:          "position outside tree",
			pos:           source.Position{Line: 20, Column: 1},
			leeway:        0,
			expectedCount: 0,
			expectedLast:  "",
		},
		{
			name:          "position with leeway",
			pos:           source.Position{Line: 5, Column: 3},
			leeway:        5,
			expectedCount: 4,
			expectedLast:  "type",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			nodes := idx.NodesAtPosition(tt.pos, tt.leeway)
			s.Assert().Len(nodes, tt.expectedCount)
			if tt.expectedCount > 0 {
				s.Assert().Equal(tt.expectedLast, nodes[len(nodes)-1].FieldName)
			}
		})
	}
}

func (s *PositionIndexSuite) TestDeepestNodeAtPosition() {
	root := s.createTestTree()
	idx := NewPositionIndex(root)

	node := idx.DeepestNodeAtPosition(source.Position{Line: 5, Column: 10}, 0)
	s.Require().NotNil(node)
	s.Assert().Equal("type", node.FieldName)

	node = idx.DeepestNodeAtPosition(source.Position{Line: 20, Column: 1}, 0)
	s.Assert().Nil(node)
}

func (s *PositionIndexSuite) TestNodesOnLine() {
	root := s.createTestTree()
	idx := NewPositionIndex(root)

	nodes := idx.NodesOnLine(5)
	s.Assert().NotEmpty(nodes)

	nodes = idx.NodesOnLine(100)
	s.Assert().Empty(nodes)
}

func (s *PositionIndexSuite) TestFindNodeByPath() {
	root := s.createTestTree()
	idx := NewPositionIndex(root)

	tests := []struct {
		name         string
		path         string
		shouldFind   bool
		expectedName string
	}{
		{
			name:         "root path",
			path:         "/",
			shouldFind:   true,
			expectedName: "",
		},
		{
			name:         "resources path",
			path:         "/resources",
			shouldFind:   true,
			expectedName: "resources",
		},
		{
			name:         "nested path",
			path:         "/resources/myResource/type",
			shouldFind:   true,
			expectedName: "type",
		},
		{
			name:       "non-existent path",
			path:       "/nonexistent",
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			node := idx.FindNodeByPath(tt.path)
			if tt.shouldFind {
				s.Require().NotNil(node)
				s.Assert().Equal(tt.expectedName, node.FieldName)
			} else {
				s.Assert().Nil(node)
			}
		})
	}
}

func (s *PositionIndexSuite) TestMaxLine() {
	root := s.createTestTree()
	idx := NewPositionIndex(root)

	s.Assert().Equal(10, idx.MaxLine())
}

func (s *PositionIndexSuite) TestAllNodes() {
	root := s.createTestTree()
	idx := NewPositionIndex(root)

	allNodes := idx.AllNodes()
	s.Assert().Len(allNodes, 4)
}

func (s *PositionIndexSuite) TestSortByDepth() {
	root := &UnifiedNode{Kind: NodeKindDocument}
	child := &UnifiedNode{Kind: NodeKindMapping, Parent: root}
	grandchild := &UnifiedNode{Kind: NodeKindScalar, Parent: child}

	nodes := []*UnifiedNode{grandchild, root, child}
	sortByDepth(nodes)

	s.Assert().Equal(root, nodes[0])
	s.Assert().Equal(child, nodes[1])
	s.Assert().Equal(grandchild, nodes[2])
}

func (s *PositionIndexSuite) TestCalculateDepth() {
	root := &UnifiedNode{Kind: NodeKindDocument}
	child := &UnifiedNode{Kind: NodeKindMapping, Parent: root}
	grandchild := &UnifiedNode{Kind: NodeKindScalar, Parent: child}

	s.Assert().Equal(0, calculateDepth(root))
	s.Assert().Equal(1, calculateDepth(child))
	s.Assert().Equal(2, calculateDepth(grandchild))
}

func (s *PositionIndexSuite) createTestTree() *UnifiedNode {
	root := &UnifiedNode{
		Kind:  NodeKindDocument,
		Range: source.Range{Start: &source.Position{Line: 1, Column: 1}, End: &source.Position{Line: 10, Column: 1}},
		Index: -1,
	}
	resources := &UnifiedNode{
		Kind:      NodeKindMapping,
		FieldName: "resources",
		Range:     source.Range{Start: &source.Position{Line: 2, Column: 1}, End: &source.Position{Line: 8, Column: 1}},
		Parent:    root,
		Index:     -1,
	}
	myResource := &UnifiedNode{
		Kind:      NodeKindMapping,
		FieldName: "myResource",
		Range:     source.Range{Start: &source.Position{Line: 3, Column: 3}, End: &source.Position{Line: 7, Column: 1}},
		Parent:    resources,
		Index:     -1,
	}
	typeNode := &UnifiedNode{
		Kind:      NodeKindScalar,
		FieldName: "type",
		Value:     "aws/dynamodb/table",
		Range:     source.Range{Start: &source.Position{Line: 5, Column: 5}, End: &source.Position{Line: 5, Column: 20}},
		Parent:    myResource,
		Index:     -1,
	}

	root.Children = []*UnifiedNode{resources}
	resources.Children = []*UnifiedNode{myResource}
	myResource.Children = []*UnifiedNode{typeNode}

	return root
}

func TestPositionIndexSuite(t *testing.T) {
	suite.Run(t, new(PositionIndexSuite))
}
