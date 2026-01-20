package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/stretchr/testify/suite"
)

type UnifiedNodeSuite struct {
	suite.Suite
}

func (s *UnifiedNodeSuite) TestContainsPosition_SingleLine() {
	r := source.Range{
		Start: &source.Position{Line: 5, Column: 10},
		End:   &source.Position{Line: 5, Column: 20},
	}

	tests := []struct {
		name     string
		pos      source.Position
		leeway   int
		expected bool
	}{
		{"before range", source.Position{Line: 5, Column: 5}, 0, false},
		{"at start", source.Position{Line: 5, Column: 10}, 0, true},
		{"in middle", source.Position{Line: 5, Column: 15}, 0, true},
		{"at end", source.Position{Line: 5, Column: 20}, 0, true},
		{"after range", source.Position{Line: 5, Column: 25}, 0, false},
		{"before with leeway", source.Position{Line: 5, Column: 8}, 2, true},
		{"after with leeway", source.Position{Line: 5, Column: 22}, 2, true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := containsPosition(r, tt.pos, tt.leeway)
			s.Assert().Equal(tt.expected, result)
		})
	}
}

func (s *UnifiedNodeSuite) TestContainsPosition_MultiLine() {
	r := source.Range{
		Start: &source.Position{Line: 5, Column: 10},
		End:   &source.Position{Line: 10, Column: 15},
	}

	tests := []struct {
		name     string
		pos      source.Position
		expected bool
	}{
		{"line before", source.Position{Line: 4, Column: 10}, false},
		{"start line before column", source.Position{Line: 5, Column: 5}, false},
		{"start line at column", source.Position{Line: 5, Column: 10}, true},
		{"middle line", source.Position{Line: 7, Column: 1}, true},
		{"end line in column", source.Position{Line: 10, Column: 10}, true},
		{"end line at end", source.Position{Line: 10, Column: 15}, true},
		{"end line after column", source.Position{Line: 10, Column: 20}, false},
		{"line after", source.Position{Line: 11, Column: 1}, false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := containsPosition(r, tt.pos, 0)
			s.Assert().Equal(tt.expected, result)
		})
	}
}

func (s *UnifiedNodeSuite) TestUnifiedNode_Path() {
	// Build a simple tree
	root := &UnifiedNode{Kind: NodeKindDocument}
	resources := &UnifiedNode{Kind: NodeKindMapping, FieldName: "resources", Parent: root, Index: -1}
	myResource := &UnifiedNode{Kind: NodeKindMapping, FieldName: "myResource", Parent: resources, Index: -1}
	typeNode := &UnifiedNode{Kind: NodeKindScalar, FieldName: "type", Parent: myResource, Index: -1}

	root.Children = []*UnifiedNode{resources}
	resources.Children = []*UnifiedNode{myResource}
	myResource.Children = []*UnifiedNode{typeNode}

	s.Assert().Equal("/", root.Path())
	s.Assert().Equal("/resources", resources.Path())
	s.Assert().Equal("/resources/myResource", myResource.Path())
	s.Assert().Equal("/resources/myResource/type", typeNode.Path())
}

func (s *UnifiedNodeSuite) TestUnifiedNode_AncestorPath() {
	root := &UnifiedNode{Kind: NodeKindDocument}
	resources := &UnifiedNode{Kind: NodeKindMapping, FieldName: "resources", Parent: root, Index: -1}
	myResource := &UnifiedNode{Kind: NodeKindMapping, FieldName: "myResource", Parent: resources, Index: -1}

	root.Children = []*UnifiedNode{resources}
	resources.Children = []*UnifiedNode{myResource}

	path := myResource.AncestorPath()
	s.Assert().Len(path, 2)
	s.Assert().Equal("resources", path[0].FieldName)
	s.Assert().Equal("myResource", path[1].FieldName)
}

func (s *UnifiedNodeSuite) TestUnifiedNode_DeepestChildAt() {
	root := &UnifiedNode{
		Kind:  NodeKindDocument,
		Range: source.Range{Start: &source.Position{Line: 1, Column: 1}, End: &source.Position{Line: 10, Column: 1}},
	}
	child1 := &UnifiedNode{
		Kind:   NodeKindMapping,
		Range:  source.Range{Start: &source.Position{Line: 2, Column: 1}, End: &source.Position{Line: 5, Column: 1}},
		Parent: root,
		Index:  -1,
	}
	child2 := &UnifiedNode{
		Kind:   NodeKindScalar,
		Range:  source.Range{Start: &source.Position{Line: 3, Column: 5}, End: &source.Position{Line: 3, Column: 15}},
		Parent: child1,
		Index:  -1,
	}
	root.Children = []*UnifiedNode{child1}
	child1.Children = []*UnifiedNode{child2}

	// Position in deepest child
	deepest := root.DeepestChildAt(source.Position{Line: 3, Column: 10}, 0)
	s.Assert().Equal(child2, deepest)

	// Position in middle child
	deepest = root.DeepestChildAt(source.Position{Line: 4, Column: 1}, 0)
	s.Assert().Equal(child1, deepest)

	// Position outside tree
	deepest = root.DeepestChildAt(source.Position{Line: 15, Column: 1}, 0)
	s.Assert().Nil(deepest)
}

func (s *UnifiedNodeSuite) TestNodeKind_String() {
	s.Assert().Equal("document", NodeKindDocument.String())
	s.Assert().Equal("mapping", NodeKindMapping.String())
	s.Assert().Equal("sequence", NodeKindSequence.String())
	s.Assert().Equal("scalar", NodeKindScalar.String())
	s.Assert().Equal("error", NodeKindError.String())
}

func TestUnifiedNodeSuite(t *testing.T) {
	suite.Run(t, new(UnifiedNodeSuite))
}
