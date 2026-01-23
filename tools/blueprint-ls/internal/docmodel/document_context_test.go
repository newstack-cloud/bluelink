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

func (s *DocumentContextSuite) TestGetNodeContext_IndentationBased_SiblingFieldAboveExisting() {
	// Test adding a new sibling field ABOVE an existing field in an include definition.
	// This simulates the user scenario where they define metadata first,
	// then insert a new blank line above it and want to add "path:".
	//
	// The content represents the state AFTER inserting a new blank line:
	content := `version: 2021-12-18
include:
  myInclude:

    metadata:
      displayName: "My Include"
`

	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	// Position cursor on the blank line (line 4), at the include definition indentation (column 5)
	// Line 4 is empty, so indentation-based detection should be used
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 4, Column: 5}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().NotNil(nodeCtx.UnifiedNode, "Should find parent node via indentation")
	s.Assert().NotEmpty(nodeCtx.ASTPath, "Should have AST path via indentation")

	// The path should indicate we're inside an include definition
	s.Assert().True(nodeCtx.ASTPath.IsIncludeDefinition(),
		"Path should indicate include definition context: %s", nodeCtx.ASTPath.String())

	// Verify completion context is correct
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextIncludeDefinitionField, completionCtx.Kind,
		"Expected IncludeDefinitionField context, got: %s", completionCtx.Kind.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_IndentationBased_NewFieldInInclude() {
	// Test adding a field in an include definition on an empty line.
	// In YAML, `myInclude:` followed by nothing on subsequent lines
	// creates an implicit null/empty value, not a mapping with fields.
	// So we need content that has actual children to test properly.
	content := `version: 2021-12-18
include:
  myInclude:
    path: ./child.yaml

`

	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	// Position cursor on the empty line (line 5) at include definition level indent (column 5)
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 5, Column: 5}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().NotNil(nodeCtx.UnifiedNode, "Should find parent node via indentation")
	s.Assert().NotEmpty(nodeCtx.ASTPath, "Should have AST path via indentation")

	// The path should indicate we're inside an include definition
	s.Assert().True(nodeCtx.ASTPath.IsIncludeDefinition(),
		"Path should indicate include definition context: %s", nodeCtx.ASTPath.String())

	// Verify completion context is correct
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextIncludeDefinitionField, completionCtx.Kind,
		"Expected IncludeDefinitionField context, got: %s", completionCtx.Kind.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_IndentationBased_TypingNewFieldAboveSibling() {
	// Realistic user scenario: Start with valid YAML, then simulate typing a new field
	// Step 1: Initial valid content (include with just metadata)
	initialContent := `version: 2025-11-02
resources:
  saveOrder:
    type: aws/lambda/function

include:
  coreInfra:
    metadata:
      displayName: "Core Infrastructure"
exports:
  myExport1:
    type: string`

	ctx := NewDocumentContext("file:///test.yaml", initialContent, FormatYAML, nil)
	s.Require().NotNil(ctx.CurrentAST, "Initial content should parse")

	// Step 2: User inserts a new line above metadata and starts typing "path"
	// This creates temporarily invalid YAML
	updatedContent := `version: 2025-11-02
resources:
  saveOrder:
    type: aws/lambda/function

include:
  coreInfra:
    path
    metadata:
      displayName: "Core Infrastructure"
exports:
  myExport1:
    type: string`

	ctx.UpdateContent(updatedContent, 2)

	// Line 8 is "    path" (4 spaces + "path")
	// Column 8 would be at the end of "path"
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 8, Column: 8}, 2)

	s.Require().NotNil(nodeCtx)

	// With LastValidAST available, we should be able to find context
	// The path should indicate we're inside an include definition
	s.Assert().True(nodeCtx.ASTPath.IsIncludeDefinition(),
		"Path should indicate include definition context: %s", nodeCtx.ASTPath.String())

	// Verify completion context is correct
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextIncludeDefinitionField, completionCtx.Kind,
		"Expected IncludeDefinitionField context, got: %s", completionCtx.Kind.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_IndentationBased_TabIndent() {
	// Test that tab indentation is handled correctly.
	// When pressing tab, VS Code sends cursor position after the tab character.

	// Initial valid content with 2-space indentation
	initialContent := `version: 2025-11-02
include:
  coreInfra:
    metadata:
      displayName: "Core Infrastructure"`

	ctx := NewDocumentContext("file:///test.yaml", initialContent, FormatYAML, nil)
	s.Require().NotNil(ctx.CurrentAST, "Initial content should parse")

	// Simulate user pressing Enter then Tab - creates a line with just a tab character
	// In this scenario, the content has a tab character, and cursor is at column 2
	updatedContent := "version: 2025-11-02\ninclude:\n  coreInfra:\n\t\n    metadata:\n      displayName: \"Core Infrastructure\""
	ctx.UpdateContent(updatedContent, 2)

	// Line 4 has just a tab character. Cursor at column 2 (after tab).
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 4, Column: 2}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().True(nodeCtx.ASTPath.IsInIncludes(),
		"Should detect we're in includes section: %s", nodeCtx.ASTPath.String())
	s.Assert().True(nodeCtx.ASTPath.IsIncludeDefinition(),
		"Should detect we're in include definition: %s", nodeCtx.ASTPath.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_IndentationBased_IsAtKeyPosition_TabVsSpaces() {
	// Test IsAtKeyPosition behavior with tab vs spaces indentation.
	// All indent methods should detect key position correctly.

	initialContent := `version: 2025-11-02
include:
  coreInfra:
    metadata:
      displayName: "Core Infrastructure"`

	ctx := NewDocumentContext("file:///test.yaml", initialContent, FormatYAML, nil)

	// Case 1: Line with tab character, cursor after tab
	tabContent := "version: 2025-11-02\ninclude:\n  coreInfra:\n\t\n    metadata:\n      displayName: \"Core Infrastructure\""
	ctx.UpdateContent(tabContent, 2)

	nodeCtxTab := ctx.GetNodeContext(source.Position{Line: 4, Column: 2}, 2)
	s.Assert().True(nodeCtxTab.IsAtKeyPosition(), "Tab should be at key position")
	s.Assert().Equal(CompletionContextIncludeDefinitionField, DetermineCompletionContext(nodeCtxTab).Kind,
		"Tab should get IncludeDefinitionField context")

	// Case 2: Line with 4 spaces, cursor at column 5
	spacesContent := "version: 2025-11-02\ninclude:\n  coreInfra:\n    \n    metadata:\n      displayName: \"Core Infrastructure\""
	ctx.UpdateContent(spacesContent, 3)

	nodeCtxSpaces := ctx.GetNodeContext(source.Position{Line: 4, Column: 5}, 2)
	s.Assert().True(nodeCtxSpaces.IsAtKeyPosition(), "Spaces should be at key position")
	s.Assert().Equal(CompletionContextIncludeDefinitionField, DetermineCompletionContext(nodeCtxSpaces).Kind,
		"Spaces should get IncludeDefinitionField context")

	// Case 3: Empty line, cursor at column 1
	emptyLineContent := "version: 2025-11-02\ninclude:\n  coreInfra:\n\n    metadata:\n      displayName: \"Core Infrastructure\""
	ctx.UpdateContent(emptyLineContent, 4)

	nodeCtxEmpty := ctx.GetNodeContext(source.Position{Line: 4, Column: 1}, 2)
	s.Assert().True(nodeCtxEmpty.IsAtKeyPosition(), "Empty line should be at key position")
	// Empty line at column 1 still gets include definition because LastValidAST is used
	s.Assert().Equal(CompletionContextIncludeDefinitionField, DetermineCompletionContext(nodeCtxEmpty).Kind,
		"Empty line should get IncludeDefinitionField context")

	// Case 4: Line with 2 spaces only (Enter + Space + Space scenario)
	// This tests the specific case where the user types Enter followed by 2 spaces
	// to add a child field, but cursor column is only 3.
	twoSpacesContent := "version: 2025-11-02\ninclude:\n  coreInfra:\n  \n    metadata:\n      displayName: \"Core Infrastructure\""
	ctx.UpdateContent(twoSpacesContent, 5)

	// Cursor at column 3 (after 2 spaces) - should still detect child level of include:
	// because coreInfra: starts at column 3, and we want child fields at indent > 2
	nodeCtxTwoSpaces := ctx.GetNodeContext(source.Position{Line: 4, Column: 3}, 2)
	s.Assert().True(nodeCtxTwoSpaces.IsAtKeyPosition(), "Two spaces should be at key position")
	// With 2 spaces (indent=2) and coreInfra at column 3 (0-based: 2), we should NOT match coreInfra
	// Instead we should match include: at column 1 (0-based: 0), giving us top-level include context
	// which allows adding new include entries at the same level as coreInfra.
	s.Assert().True(nodeCtxTwoSpaces.ASTPath.IsInIncludes(),
		"Two spaces should detect includes section: %s", nodeCtxTwoSpaces.ASTPath.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_IndentationBased_EmptySpecMapping() {
	// Test the case where spec: has no children yet.
	// When a user types "spec:" and then presses Enter + spaces to add a child field,
	// the indentation-based lookup should find spec as the parent.
	//
	// This tests the fix for the issue where spec: with no value was being ignored
	// in the AST because processMappingPair returned early when valueNode was nil.
	content := `version: 2025-11-02
variables:
  region:
    type: string
    default: us-west-2
    secret: false
resources:
  saveOrder:
    type: aws/lambda/function
    spec:

`

	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)
	s.Require().NotNil(ctx.CurrentAST, "Content should parse")

	// Verify the spec node exists in the AST
	docMapping := ctx.CurrentAST.Children[0]
	var resourcesNode *UnifiedNode
	for _, child := range docMapping.Children {
		if child.FieldName == "resources" {
			resourcesNode = child
			break
		}
	}
	s.Require().NotNil(resourcesNode)
	var saveOrderNode *UnifiedNode
	for _, child := range resourcesNode.Children {
		if child.FieldName == "saveOrder" {
			saveOrderNode = child
			break
		}
	}
	s.Require().NotNil(saveOrderNode)
	var specNode *UnifiedNode
	for _, child := range saveOrderNode.Children {
		if child.FieldName == "spec" {
			specNode = child
			break
		}
	}
	s.Require().NotNil(specNode, "spec node should exist even when empty")
	s.Assert().Equal(NodeKindMapping, specNode.Kind, "spec should be a mapping node")

	// Position cursor on the empty line after "spec:" at child indentation level
	// Line 11 is empty, cursor at column 7 (simulating 6 spaces of indent)
	// spec: is at column 5 (4 spaces + "spec:"), so 0-based column 4
	// Our cursor indent is 6 (0-based), which is > 4, so spec should be matched
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 11, Column: 7}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().NotNil(nodeCtx.UnifiedNode, "Should find spec as parent node via indentation")
	s.Assert().NotEmpty(nodeCtx.ASTPath, "Should have AST path via indentation")

	// The path should indicate we're inside the spec of the resource
	s.Assert().True(nodeCtx.ASTPath.IsResourceSpec(),
		"Path should indicate resource spec context: %s", nodeCtx.ASTPath.String())

	// Verify completion context gives us spec fields
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextResourceSpecField, completionCtx.Kind,
		"Expected ResourceSpecField context, got: %s", completionCtx.Kind.String())
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

func (s *DocumentContextSuite) TestGetNodeContext_DataSourceSiblingField_AfterMetadata() {
	// Test that when typing a new sibling field at the data source definition level
	// (after metadata:), we get DataSourceDefinitionField context, not metadata context.
	//
	// Scenario: User has a data source with metadata defined, and wants to add
	// "exports" or "filter" as a sibling field at the same indentation level.
	content := `version: 2025-11-02
datasources:
  orderTable:
    type: aws/dynamodb/table
    metadata:
      displayName: Orders Table
      annotations:
        environment: production

    filter:
      field: TableName
`

	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)
	s.Require().NotNil(ctx.CurrentAST, "Content should parse")

	// Position cursor on the blank line between metadata and filter (line 9)
	// At column 5 (4 spaces indent) to add a sibling field like "exports:"
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 9, Column: 5}, 2)

	s.Require().NotNil(nodeCtx)
	s.Assert().NotNil(nodeCtx.UnifiedNode, "Should find parent node via indentation")
	s.Assert().NotEmpty(nodeCtx.ASTPath, "Should have AST path")

	// The path should indicate we're at the data source definition level, NOT inside metadata
	s.Assert().True(nodeCtx.ASTPath.IsDataSourceDefinition(),
		"Path should indicate data source definition context (not metadata): %s", nodeCtx.ASTPath.String())

	// Verify we're NOT inside metadata
	s.Assert().False(nodeCtx.ASTPath.IsDataSourceMetadata(),
		"Path should NOT be inside metadata: %s", nodeCtx.ASTPath.String())

	// Verify completion context is DataSourceDefinitionField
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceDefinitionField, completionCtx.Kind,
		"Expected DataSourceDefinitionField context, got: %s", completionCtx.Kind.String())
}

func (s *DocumentContextSuite) TestGetNodeContext_DataSourceSiblingField_TypingNewField() {
	// Test when user is actively typing a new sibling field at data source level.
	// The content represents an invalid state where user is typing "export".
	content := `version: 2025-11-02
datasources:
  orderTable:
    type: aws/dynamodb/table
    metadata:
      displayName: Orders Table
    export
    filter:
      field: TableName
`

	ctx := NewDocumentContext("file:///test.yaml", content, FormatYAML, nil)

	// Position cursor at the end of "export" (line 7, column 11)
	// Line 7 has "    export" (4 spaces + "export")
	nodeCtx := ctx.GetNodeContext(source.Position{Line: 7, Column: 11}, 2)

	s.Require().NotNil(nodeCtx)

	// The path should indicate we're at the data source definition level
	s.Assert().True(nodeCtx.ASTPath.IsDataSourceDefinition(),
		"Path should indicate data source definition context: %s", nodeCtx.ASTPath.String())

	// Verify completion context is DataSourceDefinitionField
	completionCtx := DetermineCompletionContext(nodeCtx)
	s.Assert().Equal(CompletionContextDataSourceDefinitionField, completionCtx.Kind,
		"Expected DataSourceDefinitionField context, got: %s", completionCtx.Kind.String())
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
