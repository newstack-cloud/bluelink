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
		// Leeway only extends the START boundary, not the END.
		// Positions after the node's end are clearly outside the node.
		{"after with leeway", source.Position{Line: 5, Column: 22}, 2, false},
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

// TestInlineJSONCArrayPathDetection verifies that path detection works correctly
// for inline JSONC arrays (arrays on a single line like `"exclude": [""]`).
func (s *UnifiedNodeSuite) TestInlineJSONCArrayPathDetection() {
	content := `{
  "version": "2025-11-02",
  "resources": {
    "newOrderTable": {
      "type": "aws/dynamodb/table",
      "linkSelector": {
        "byLabel": {
          "subsystem": "eventProcessing"
        },
        "exclude": [""]
      },
      "spec": {
        "tableName": "NewOrdersTable2025"
      }
    }
  }
}`

	docCtx := NewDocumentContext(
		"file:///test.jsonc",
		content,
		FormatJSONC,
		nil,
	)

	// Line 10 (1-indexed), character 22 (inside the empty string)
	// Line 10 contains: `        "exclude": [""]`
	pos := source.Position{Line: 10, Column: 22}
	nodeCtx := docCtx.GetCursorContext(pos, 2)

	// Verify path is correctly detected
	s.Assert().True(nodeCtx.StructuralPath.IsResourceLinkSelectorExclude(), "Path should be detected as linkSelector.exclude")

	// Verify completion context is correct
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextLinkSelectorExcludeValue, completionCtx.Kind, "Completion context should be LinkSelectorExcludeValue")
	s.Assert().Equal("newOrderTable", completionCtx.ResourceName, "Resource name should be newOrderTable")

	// Verify GetTypedPrefix returns empty string for empty string in inline array
	s.Assert().Equal("", nodeCtx.GetTypedPrefix(), "Typed prefix should be empty for empty string in array")
}

// TestInlineJSONCEmptyArrayPathDetection verifies path detection for empty arrays `[]`.
// This tests the case where the cursor is inside an empty array without any quotes.
func (s *UnifiedNodeSuite) TestInlineJSONCEmptyArrayPathDetection() {
	content := `{
  "version": "2025-11-02",
  "resources": {
    "newOrderTable": {
      "type": "aws/dynamodb/table",
      "linkSelector": {
        "byLabel": {
          "subsystem": "eventProcessing"
        },
        "exclude": []
      },
      "spec": {
        "tableName": "NewOrdersTable2025"
      }
    }
  }
}`

	docCtx := NewDocumentContext(
		"file:///test.jsonc",
		content,
		FormatJSONC,
		nil,
	)

	// Line 10 (1-indexed), character 21 (between the empty brackets [])
	// Line 10 contains: `        "exclude": []`
	pos := source.Position{Line: 10, Column: 21}
	nodeCtx := docCtx.GetCursorContext(pos, 2)

	// Verify path is correctly detected
	s.Assert().True(nodeCtx.StructuralPath.IsResourceLinkSelectorExclude(), "Path should be detected as linkSelector.exclude")

	// Verify completion context is correct
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextLinkSelectorExcludeValue, completionCtx.Kind, "Completion context should be LinkSelectorExcludeValue")
	s.Assert().Equal("newOrderTable", completionCtx.ResourceName, "Resource name should be newOrderTable")

	// Verify GetTypedPrefix returns empty string for empty array
	s.Assert().Equal("", nodeCtx.GetTypedPrefix(), "Typed prefix should be empty for empty array")
}

// TestInlineJSONCArrayAfterCommaPathDetection verifies path detection for cursor after comma.
// This tests the case where the cursor is after a comma following an existing item: ["item", |]
func (s *UnifiedNodeSuite) TestInlineJSONCArrayAfterCommaPathDetection() {
	content := `{
  "version": "2025-11-02",
  "resources": {
    "aNewFunction": {
      "type": "aws/lambda/function",
      "linkSelector": {
        "byLabel": {
          "app": "orders"
        },
        "exclude": ["newOrderTable", ]
      },
      "spec": {
        "functionName": "ANewFunction"
      }
    },
    "newOrderTable": {
      "type": "aws/dynamodb/table",
      "spec": {
        "tableName": "NewOrdersTable"
      }
    }
  }
}`

	docCtx := NewDocumentContext(
		"file:///test.jsonc",
		content,
		FormatJSONC,
		nil,
	)

	// Line 10 (1-indexed), character 37 (after the comma and space, before the ])
	// Line 10 contains: `        "exclude": ["newOrderTable", ]`
	// The exclude array range is columns 20-39, so position 37 is inside the array
	pos := source.Position{Line: 10, Column: 37}
	nodeCtx := docCtx.GetCursorContext(pos, 2)

	// Verify path is correctly detected
	s.Assert().True(nodeCtx.StructuralPath.IsResourceLinkSelectorExclude(), "Path should be detected as linkSelector.exclude")

	// Verify completion context is correct
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextLinkSelectorExcludeValue, completionCtx.Kind, "Completion context should be LinkSelectorExcludeValue")
	s.Assert().Equal("aNewFunction", completionCtx.ResourceName, "Resource name should be aNewFunction")
}

func TestUnifiedNodeSuite(t *testing.T) {
	suite.Run(t, new(UnifiedNodeSuite))
}
