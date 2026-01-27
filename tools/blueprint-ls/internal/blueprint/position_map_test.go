package blueprint

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/stretchr/testify/suite"
)

type PositionMapSuite struct {
	suite.Suite
}

func (s *PositionMapSuite) TestCreatePositionMap_NilTree() {
	result := CreatePositionMap(nil)
	s.Empty(result)
}

func (s *PositionMapSuite) TestCreatePositionMap_TreeWithNilRange() {
	tree := &schema.TreeNode{
		Range: nil,
	}
	result := CreatePositionMap(tree)
	s.Empty(result)
}

func (s *PositionMapSuite) TestCreatePositionMap_TreeWithNilRangeStart() {
	tree := &schema.TreeNode{
		Range: &source.Range{
			Start: nil,
		},
	}
	result := CreatePositionMap(tree)
	s.Empty(result)
}

func (s *PositionMapSuite) TestCreatePositionMap_SingleNode() {
	tree := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 1, Column: 10},
		},
	}
	result := CreatePositionMap(tree)
	s.Len(result, 1)
	s.Contains(result, "1:1")
	s.Len(result["1:1"], 1)
	s.Equal(tree, result["1:1"][0])
}

func (s *PositionMapSuite) TestCreatePositionMap_NestedNodes() {
	child := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 2, Column: 3},
			End:   &source.Position{Line: 2, Column: 10},
		},
	}
	tree := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 3, Column: 1},
		},
		Children: []*schema.TreeNode{child},
	}
	result := CreatePositionMap(tree)
	s.Len(result, 2)
	s.Contains(result, "1:1")
	s.Contains(result, "2:3")
}

func (s *PositionMapSuite) TestCreatePositionMap_MultipleNodesAtSamePosition() {
	// When two nodes start at the same position
	child := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 1, Column: 5},
		},
	}
	tree := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 3, Column: 1},
		},
		Children: []*schema.TreeNode{child},
	}
	result := CreatePositionMap(tree)
	s.Len(result, 1)
	s.Contains(result, "1:1")
	s.Len(result["1:1"], 2)
	// The child with smaller range should be last (due to traversal order)
	s.Equal(tree, result["1:1"][0])
	s.Equal(child, result["1:1"][1])
}

func (s *PositionMapSuite) TestCreatePositionMap_ComplexTree() {
	grandchild := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 3, Column: 5},
			End:   &source.Position{Line: 3, Column: 15},
		},
	}
	child1 := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 2, Column: 3},
			End:   &source.Position{Line: 4, Column: 3},
		},
		Children: []*schema.TreeNode{grandchild},
	}
	child2 := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 5, Column: 3},
			End:   &source.Position{Line: 5, Column: 10},
		},
	}
	tree := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 6, Column: 1},
		},
		Children: []*schema.TreeNode{child1, child2},
	}
	result := CreatePositionMap(tree)
	s.Len(result, 4)
	s.Contains(result, "1:1")
	s.Contains(result, "2:3")
	s.Contains(result, "3:5")
	s.Contains(result, "5:3")
}

func (s *PositionMapSuite) TestCreatePositionMap_ChildWithNilRange() {
	child := &schema.TreeNode{
		Range: nil,
	}
	tree := &schema.TreeNode{
		Range: &source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 3, Column: 1},
		},
		Children: []*schema.TreeNode{child},
	}
	result := CreatePositionMap(tree)
	s.Len(result, 1)
	s.Contains(result, "1:1")
}

func (s *PositionMapSuite) TestPositionKey_NilPosition() {
	result := PositionKey(nil)
	s.Equal("1:1", result)
}

func (s *PositionMapSuite) TestPositionKey_ValidPosition() {
	pos := &source.Position{Line: 5, Column: 10}
	result := PositionKey(pos)
	s.Equal("5:10", result)
}

func (s *PositionMapSuite) TestPositionKey_ZeroPosition() {
	pos := &source.Position{Line: 0, Column: 0}
	result := PositionKey(pos)
	s.Equal("0:0", result)
}

func (s *PositionMapSuite) TestPositionKey_LargeNumbers() {
	pos := &source.Position{Line: 1000, Column: 500}
	result := PositionKey(pos)
	s.Equal("1000:500", result)
}

func TestPositionMapSuite(t *testing.T) {
	suite.Run(t, new(PositionMapSuite))
}
