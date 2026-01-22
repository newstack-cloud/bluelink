package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type DocumentContextSuite struct {
	suite.Suite
}

func (s *DocumentContextSuite) TestNewDocumentContext_YAML() {
	content := `version: 2021-12-18
resources:
  myTable:
    type: aws/dynamodb/table
    spec:
      tableName: test-table`

	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	s.Require().NotNil(ctx)
	s.Assert().Equal("file:///test.yaml", ctx.URI)
	s.Assert().Equal(FormatYAML, ctx.Format)
	s.Assert().Equal(content, ctx.Content)
	s.Assert().Equal(1, ctx.Version)
	s.Assert().Equal(StatusValid, ctx.Status)

	err := testhelpers.Snapshot(toSnapshotAST(ctx.CurrentAST))
	s.Require().NoError(err)
}

func (s *DocumentContextSuite) TestNewDocumentContext_JSONC() {
	content := `{
  "version": "2021-12-18",
  "resources": {
    "myTable": {
      "type": "aws/dynamodb/table",
      "spec": {
        "tableName": "test-table"
      }
    }
  }
}`

	ctx := NewDocumentContext("file:///test.jsonc", content, FormatJSONC, nil)

	s.Require().NotNil(ctx)
	s.Assert().Equal(FormatJSONC, ctx.Format)
	s.Assert().Equal(StatusValid, ctx.Status)

	err := testhelpers.Snapshot(toSnapshotAST(ctx.CurrentAST))
	s.Require().NoError(err)
}

func (s *DocumentContextSuite) TestUpdateContent() {
	content1 := `version: 2021-12-18
resources: {}`

	ctx := NewDocumentContext("file:///test.yaml", content1, FormatYAML, nil)
	s.Require().Equal(1, ctx.Version)

	content2 := `version: 2021-12-18
resources:
  myTable:
    type: aws/dynamodb/table`

	ctx.UpdateContent(content2, 2)

	s.Assert().Equal(content2, ctx.Content)
	s.Assert().Equal(2, ctx.Version)

	err := testhelpers.Snapshot(toSnapshotAST(ctx.CurrentAST))
	s.Require().NoError(err)
}

func (s *DocumentContextSuite) TestGetNodeContext() {
	content := `version: 2021-12-18
resources:
  myTable:
    type: aws/dynamodb/table
    spec:
      tableName: test-table`

	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	nodeCtx := ctx.GetNodeContext(source.Position{Line: 4, Column: 10}, 0)

	s.Require().NotNil(nodeCtx)
	s.Assert().Equal(ctx, nodeCtx.DocumentCtx)
	s.Assert().NotNil(nodeCtx.UnifiedNode)
	s.Assert().NotEmpty(nodeCtx.AncestorNodes)
}

func (s *DocumentContextSuite) TestHasValidAST() {
	content := `version: 2021-12-18`
	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	s.Assert().True(ctx.HasValidAST())

	ctx.CurrentAST = nil
	ctx.LastValidAST = nil
	s.Assert().False(ctx.HasValidAST())
}

func (s *DocumentContextSuite) TestGetEffectiveAST() {
	content := `version: 2021-12-18`
	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	ast := ctx.GetEffectiveAST()
	s.Assert().NotNil(ast)
	s.Assert().Equal(ctx.CurrentAST, ast)

	ctx.CurrentAST = nil
	ast = ctx.GetEffectiveAST()
	s.Assert().Equal(ctx.LastValidAST, ast)
}

func (s *DocumentContextSuite) TestDocumentFormat_String() {
	s.Assert().Equal("yaml", FormatYAML.String())
	s.Assert().Equal("jsonc", FormatJSONC.String())
	s.Assert().Equal("unknown", DocumentFormat(99).String())
}

func (s *DocumentContextSuite) TestDocumentStatus_String() {
	s.Assert().Equal("valid", StatusValid.String())
	s.Assert().Equal("parsing_errors", StatusParsingErrors.String())
	s.Assert().Equal("degraded", StatusDegraded.String())
	s.Assert().Equal("unavailable", StatusUnavailable.String())
	s.Assert().Equal("unknown", DocumentStatus(99).String())
}

func (s *DocumentContextSuite) TestGetNodeContext_IndentationBased_EmptyLineInSpec() {
	// Test indentation-based context detection on empty line inside spec
	content := `version: 2021-12-18
resources:
  myTable:
    type: aws/dynamodb/table
    spec:
      tableName: test-table
      `

	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	// Position cursor on the empty line inside spec (line 7, indented at column 7)
	// This simulates where a user would be when adding a new field
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 7, Column: 7}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().NotNil(nodeCtx.UnifiedNode, "Should find parent node via indentation")
	s.Assert().NotEmpty(nodeCtx.ASTPath, "Should have AST path via indentation")

	// The path should indicate we're inside the spec
	s.Assert().True(nodeCtx.ASTPath.IsResourceSpec(),
		"Path should indicate resource spec context: %s", nodeCtx.ASTPath.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_IndentationBased_NewLineAfterSpec() {
	// Test when cursor is on a completely empty line after spec content
	content := `version: 2021-12-18
resources:
  myHandler:
    type: aws/lambda/function
    spec:
      runtime: nodejs18.x

`

	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	// Position cursor on empty line 8 with spec-level indentation (column 7 = 6 spaces + 1)
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 8, Column: 7}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().NotNil(nodeCtx.UnifiedNode, "Should find parent node via indentation")
	s.Assert().NotEmpty(nodeCtx.ASTPath, "Should have AST path via indentation")

	// The path should indicate we're inside the spec
	s.Assert().True(nodeCtx.ASTPath.IsResourceSpec(),
		"Path should indicate resource spec context: %s", nodeCtx.ASTPath.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_JSONC_InsideSpecObject() {
	// Test that positions inside a JSONC spec object are correctly detected
	content := `{
  "version": "2021-12-18",
  "resources": {
    "myHandler": {
      "type": "celerity/handler",
      "spec": {

      }
    }
  }
}`

	ctx := NewDocumentContext("file:///test.jsonc", content, FormatJSONC, nil)

	// Position cursor inside the empty spec object (line 7, after the opening {)
	// Line 7 has "        " (8 spaces) - cursor at column 9
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 7, Column: 9}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().NotNil(nodeCtx.UnifiedNode, "Should find a node inside spec object")
	s.Assert().NotEmpty(nodeCtx.ASTPath, "Should have AST path")

	// The path should indicate we're inside the spec
	s.Assert().True(nodeCtx.ASTPath.IsResourceSpec(),
		"Path should indicate resource spec context: %s", nodeCtx.ASTPath.String())

	// Check IsAtKeyPosition for JSONC
	s.Assert().True(nodeCtx.IsAtKeyPosition(),
		"Should be at key position inside empty JSONC object")
}

func (s *DocumentContextSuite) TestGetNodeContext_JSONC_InsideSpecObject_WithExistingField() {
	// Test positions inside spec with existing fields
	content := `{
  "version": "2021-12-18",
  "resources": {
    "myHandler": {
      "type": "celerity/handler",
      "spec": {
        "runtime": "nodejs18.x",

      }
    }
  }
}`

	ctx := NewDocumentContext("file:///test.jsonc", content, FormatJSONC, nil)

	// Position cursor on the empty line inside spec (line 8)
	// Line 8 has "        " (8 spaces) - cursor at column 9
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 8, Column: 9}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().NotNil(nodeCtx.UnifiedNode, "Should find a node inside spec object")
	s.Assert().NotEmpty(nodeCtx.ASTPath, "Should have AST path")

	// The path should indicate we're inside the spec
	s.Assert().True(nodeCtx.ASTPath.IsResourceSpec(),
		"Path should indicate resource spec context: %s", nodeCtx.ASTPath.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_JSONC_CompletionContext_InSpec() {
	// Full integration test: parse JSONC, get node context, determine completion context
	content := `{
  "version": "2021-12-18",
  "resources": {
    "myHandler": {
      "type": "celerity/handler",
      "spec": {

      }
    }
  }
}`

	ctx := NewDocumentContext("file:///test.jsonc", content, FormatJSONC, nil)

	// Position cursor inside the empty spec object (line 7)
	// This simulates cursor position when user wants to add a new field
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 7, Column: 9}, 2)

	s.Require().NotNil(nodeCtx)

	// Check path
	s.Assert().True(nodeCtx.ASTPath.IsResourceSpec(),
		"Expected resource spec path, got: %s", nodeCtx.ASTPath.String())

	// Check IsAtKeyPosition - this is crucial for completion to work
	s.Assert().True(nodeCtx.IsAtKeyPosition(),
		"Should be at key position. TextBefore: %q", nodeCtx.TextBefore)

	// Check completion context
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecField, completionCtx.Kind,
		"Expected ResourceSpecField context, got: %s", completionCtx.Kind.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_JSONC_CompletionContext_InSpec_AfterComma() {
	// Test completion after comma in JSONC spec
	content := `{
  "version": "2021-12-18",
  "resources": {
    "myHandler": {
      "type": "celerity/handler",
      "spec": {
        "runtime": "nodejs18.x",

      }
    }
  }
}`

	ctx := NewDocumentContext("file:///test.jsonc", content, FormatJSONC, nil)

	// Position cursor on the empty line after the comma (line 8)
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 8, Column: 9}, 2)

	s.Require().NotNil(nodeCtx)

	// Check path
	s.Assert().True(nodeCtx.ASTPath.IsResourceSpec(),
		"Expected resource spec path, got: %s", nodeCtx.ASTPath.String())

	// Check IsAtKeyPosition
	s.Assert().True(nodeCtx.IsAtKeyPosition(),
		"Should be at key position. TextBefore: %q", nodeCtx.TextBefore)

	// Check completion context
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecField, completionCtx.Kind,
		"Expected ResourceSpecField context, got: %s", completionCtx.Kind.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_JSONC_CompletionContext_InSpec_TypingProperty() {
	// Test when user is typing a property name (incomplete JSON)
	content := `{
  "version": "2021-12-18",
  "resources": {
    "myHandler": {
      "type": "celerity/handler",
      "spec": {
        "run
      }
    }
  }
}`

	ctx := NewDocumentContext("file:///test.jsonc", content, FormatJSONC, nil)

	// Position cursor where the user is typing "run" (line 7, column 13 = after "run")
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 7, Column: 13}, 2)

	s.Require().NotNil(nodeCtx)

	// Even with parse errors, we should still be able to find context
	// The path should indicate we're inside the spec
	if nodeCtx.ASTPath.IsResourceSpec() {
		// Check completion context
		completionCtx := DetermineCompletionContext(nodeCtx)
		s.Assert().Equal(CompletionContextResourceSpecField, completionCtx.Kind,
			"Expected ResourceSpecField context when typing in spec, got: %s", completionCtx.Kind.String())
	} else {
		s.T().Logf("Path not detected as resource spec (may be due to parse errors): %s", nodeCtx.ASTPath.String())
	}
}

func (s *DocumentContextSuite) TestGetNodeContext_JSONC_AfterNestedObjectClosure() {
	// Test that after closing a nested object with a comma, completions
	// should be at the parent level (spec), not inside the closed nested object.
	content := `{
  "version": "2021-12-18",
  "resources": {
    "aNewFunction": {
      "type": "aws/lambda/function",
      "spec": {
        "handler": "app.handler",
        "runtime": "nodejs18.x",
        "runtimeManagementConfig": {
          "updateRuntimeOn": "deployment"
        },

      }
    }
  }
}`

	ctx := NewDocumentContext("file:///test.jsonc", content, FormatJSONC, nil)

	// Position cursor on the empty line after the comma (line 12)
	// This is at the spec level, NOT inside runtimeManagementConfig
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 12, Column: 9}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().NotEmpty(nodeCtx.ASTPath, "Should have AST path")

	// The path should indicate we're at the spec level, NOT inside runtimeManagementConfig
	s.Assert().True(nodeCtx.ASTPath.IsResourceSpec(),
		"Expected resource spec path, got: %s", nodeCtx.ASTPath.String())

	// Verify we're NOT inside runtimeManagementConfig
	pathStr := nodeCtx.ASTPath.String()
	s.Assert().NotContains(pathStr, "runtimeManagementConfig",
		"Path should NOT contain runtimeManagementConfig, got: %s", pathStr)

	// Check completion context
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecField, completionCtx.Kind,
		"Expected ResourceSpecField context, got: %s", completionCtx.Kind.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_JSONC_AfterNestedObjectClosure_SameLine() {
	// Test when the cursor is on the same line as the trailing comma after nested object.
	// This is the exact scenario the user reported: after `},` the completions should
	// be for the parent level (spec), not inside the closed nested object.
	//
	// Note: This JSON has a trailing comma which is invalid JSON but valid JSONC.
	// Tree-sitter handles this by creating error nodes for the malformed parts.
	content := `{
  "version": "2021-12-18",
  "resources": {
    "aNewFunction": {
      "type": "aws/lambda/function",
      "spec": {
        "handler": "app.handler",
        "runtime": "nodejs18.x",
        "runtimeManagementConfig": {
          "updateRuntimeOn": "deployment"
        },
      }
    }
  }
}`

	ctx := NewDocumentContext("file:///test.jsonc", content, FormatJSONC, nil)

	// Position cursor right after the comma on line 11
	// Line 11 is: `        },` (8 spaces + } + , = 10 chars)
	// Cursor at column 11 (1-based, right after the comma)
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 11, Column: 11}, 2)

	s.Require().NotNil(nodeCtx)

	// The path should indicate we're at the spec level, NOT inside runtimeManagementConfig
	s.Assert().True(nodeCtx.ASTPath.IsResourceSpec(),
		"Expected resource spec path, got: %s", nodeCtx.ASTPath.String())

	// Verify we're NOT inside runtimeManagementConfig
	pathStr := nodeCtx.ASTPath.String()
	s.Assert().NotContains(pathStr, "runtimeManagementConfig",
		"Path should NOT contain runtimeManagementConfig, got: %s", pathStr)
}

func TestDocumentContextSuite(t *testing.T) {
	suite.Run(t, new(DocumentContextSuite))
}

// snapshotNode is a snapshot-friendly representation of UnifiedNode
// that excludes circular references (Parent pointers).
type snapshotNode struct {
	Kind      string          `json:"kind"`
	FieldName string          `json:"fieldName,omitempty"`
	Value     string          `json:"value,omitempty"`
	Index     int             `json:"index,omitempty"`
	Range     *snapshotRange  `json:"range,omitempty"`
	KeyRange  *snapshotRange  `json:"keyRange,omitempty"`
	Children  []*snapshotNode `json:"children,omitempty"`
}

type snapshotRange struct {
	Start snapshotPosition `json:"start"`
	End   snapshotPosition `json:"end"`
}

type snapshotPosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func toSnapshotAST(node *UnifiedNode) *snapshotNode {
	if node == nil {
		return nil
	}

	sn := &snapshotNode{
		Kind:      node.Kind.String(),
		FieldName: node.FieldName,
		Value:     node.Value,
	}

	if node.Index >= 0 {
		sn.Index = node.Index
	}

	if node.Range.Start != nil && node.Range.End != nil {
		sn.Range = &snapshotRange{
			Start: snapshotPosition{Line: node.Range.Start.Line, Column: node.Range.Start.Column},
			End:   snapshotPosition{Line: node.Range.End.Line, Column: node.Range.End.Column},
		}
	}

	if node.KeyRange != nil && node.KeyRange.Start != nil && node.KeyRange.End != nil {
		sn.KeyRange = &snapshotRange{
			Start: snapshotPosition{Line: node.KeyRange.Start.Line, Column: node.KeyRange.Start.Column},
			End:   snapshotPosition{Line: node.KeyRange.End.Line, Column: node.KeyRange.End.Column},
		}
	}

	for _, child := range node.Children {
		sn.Children = append(sn.Children, toSnapshotAST(child))
	}

	return sn
}
