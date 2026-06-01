package docmodel

import (
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
)

type TokensToUnifiedSuite struct {
	suite.Suite
}

func childByFieldName(node *UnifiedNode, fieldName string) *UnifiedNode {
	if node == nil {
		return nil
	}
	for _, child := range node.Children {
		if child.FieldName == fieldName {
			return child
		}
	}
	return nil
}

// Walks the canonical field-name hierarchy (e.g. resources,
// saveOrderFunction, spec, functionName).
func nodeAtPath(root *UnifiedNode, fields ...string) *UnifiedNode {
	current := root
	for _, field := range fields {
		current = childByFieldName(current, field)
		if current == nil {
			return nil
		}
	}

	return current
}

func (s *TokensToUnifiedSuite) Test_resource_canonical_shape() {
	content := `version "2025-11-02"

resource saveOrderFunction: aws/lambda/function {
    spec {
        functionName = "saveOrderFunction"
        timeout = 120
    }
}
`
	root, err := ParseBlueprintLangToUnified(content)
	s.Require().NoError(err)
	s.Require().NotNil(root)
	s.Assert().Equal(NodeKindMapping, root.Kind)

	version := childByFieldName(root, "version")
	s.Require().NotNil(version)
	s.Assert().Equal(NodeKindScalar, version.Kind)
	s.Assert().Equal("2025-11-02", version.Value)

	typeNode := nodeAtPath(root, "resources", "saveOrderFunction", "type")
	s.Require().NotNil(typeNode, "expected /resources/saveOrderFunction/type")
	s.Assert().Equal("aws/lambda/function", typeNode.Value)
	s.Assert().Equal("/resources/saveOrderFunction/type", typeNode.Path())

	functionName := nodeAtPath(root, "resources", "saveOrderFunction", "spec", "functionName")
	s.Require().NotNil(functionName, "expected /resources/saveOrderFunction/spec/functionName")
	s.Assert().Equal(NodeKindScalar, functionName.Kind)
	s.Assert().Equal("saveOrderFunction", functionName.Value)
	s.Assert().Equal("/resources/saveOrderFunction/spec/functionName", functionName.Path())

	timeout := nodeAtPath(root, "resources", "saveOrderFunction", "spec", "timeout")
	s.Require().NotNil(timeout)
	s.Assert().Equal("!!int", timeout.Tag)
}

func (s *TokensToUnifiedSuite) Test_variable_value_and_object_shapes() {
	content := `version "2025-11-02"

variable region: string {
    default = "us-east-1"
    allowedValues = ["us-east-1", "us-west-2"]
}

value featureFlags: object {
    value = {
        enabled = true,
        retryCount = 3
    }
}
`
	root, err := ParseBlueprintLangToUnified(content)
	s.Require().NoError(err)

	varType := nodeAtPath(root, "variables", "region", "type")
	s.Require().NotNil(varType)
	s.Assert().Equal("string", varType.Value)

	allowed := nodeAtPath(root, "variables", "region", "allowedValues")
	s.Require().NotNil(allowed)
	s.Assert().Equal(NodeKindSequence, allowed.Kind)
	s.Require().Len(allowed.Children, 2)
	s.Assert().Equal(0, allowed.Children[0].Index)
	s.Assert().Equal(1, allowed.Children[1].Index)

	valueObj := nodeAtPath(root, "values", "featureFlags", "value")
	s.Require().NotNil(valueObj)
	s.Assert().Equal(NodeKindMapping, valueObj.Kind)
	enabled := childByFieldName(valueObj, "enabled")
	s.Require().NotNil(enabled)
	s.Assert().Equal("!!bool", enabled.Tag)
}

func (s *TokensToUnifiedSuite) Test_datasource_filter_and_exports() {
	content := `version "2025-11-02"

data network: aws/vpc {
    filter "state" == "available"

    export vpcId: string
    export region as vpcRegion: string
}
`
	root, err := ParseBlueprintLangToUnified(content)
	s.Require().NoError(err)

	dsType := nodeAtPath(root, "datasources", "network", "type")
	s.Require().NotNil(dsType)
	s.Assert().Equal("aws/vpc", dsType.Value)

	vpcId := nodeAtPath(root, "datasources", "network", "exports", "vpcId", "type")
	s.Require().NotNil(vpcId, "expected /datasources/network/exports/vpcId/type")
	s.Assert().Equal("string", vpcId.Value)

	aliasFor := nodeAtPath(root, "datasources", "network", "exports", "vpcRegion", "aliasFor")
	s.Require().NotNil(aliasFor, "aliased export should record aliasFor")
	s.Assert().Equal("region", aliasFor.Value)

	filters := nodeAtPath(root, "datasources", "network", "filters")
	s.Require().NotNil(filters)
	s.Assert().Equal(NodeKindSequence, filters.Kind)
	s.Require().Len(filters.Children, 1)
	s.Assert().Equal("available", childByFieldName(filters.Children[0], "search").Value)
}

func (s *TokensToUnifiedSuite) Test_select_by_label_becomes_link_selector() {
	content := `version "2025-11-02"

resource fn: aws/lambda/function {
    select by label {
        service = "ordersApi"
        exclude = [testFn]
    }
    spec {
        functionName = "fn"
    }
}
`
	root, err := ParseBlueprintLangToUnified(content)
	s.Require().NoError(err)

	byLabel := nodeAtPath(root, "resources", "fn", "linkSelector", "byLabel", "service")
	s.Require().NotNil(byLabel, "expected linkSelector/byLabel/service")
	s.Assert().Equal("ordersApi", byLabel.Value)

	exclude := nodeAtPath(root, "resources", "fn", "linkSelector", "exclude")
	s.Require().NotNil(exclude, "expected linkSelector/exclude")
	s.Assert().Equal(NodeKindSequence, exclude.Kind)
}

func collectSymbolNames(symbols []lsp.DocumentSymbol, names map[string]bool) {
	for _, sym := range symbols {
		names[sym.Name] = true
		collectSymbolNames(sym.Children, names)
	}
}

func (s *TokensToUnifiedSuite) Test_produces_document_symbols() {
	content := `version "2025-11-02"

resource saveOrderFunction: aws/lambda/function {
    spec {
        functionName = "saveOrderFunction"
    }
}
`
	root, err := ParseBlueprintLangToUnified(content)
	s.Require().NoError(err)

	symbols := BuildDocumentSymbols(root, strings.Count(content, "\n")+1)
	s.Require().NotEmpty(symbols)

	names := map[string]bool{}
	collectSymbolNames(symbols, names)
	s.Assert().True(names["version"], "expected a version symbol")
	s.Assert().True(names["resources"], "expected a resources symbol")
	s.Assert().True(names["saveOrderFunction"], "expected the resource name as a nested symbol")
}

func (s *TokensToUnifiedSuite) Test_detects_duplicate_keys_in_spec() {
	content := `version "2025-11-02"

resource fn: aws/lambda/function {
    spec {
        functionName = "a"
        functionName = "b"
    }
}
`
	root, err := ParseBlueprintLangToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(root)
	s.Require().NotNil(result)
	foundDup := false
	for _, dupErr := range result.Errors {
		if dupErr.Key == "functionName" {
			foundDup = true
		}
	}
	s.Assert().True(foundDup, "expected duplicate functionName to be detected")
}

func (s *TokensToUnifiedSuite) Test_cursor_context_resolves_resource_type() {
	content := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n" +
		"    spec {\n" +
		"        functionName = \"x\"\n" +
		"    }\n" +
		"}\n"

	docCtx := NewDocumentContext("file:///test.bp", content, FormatBlueprintLang, nil)

	// Cursor inside the element type "aws/lambda/function" on line 2.
	cursorCtx := docCtx.GetCursorContext(source.Position{Line: 2, Column: 16}, 0)
	s.Require().NotNil(cursorCtx)
	s.Assert().Equal("/resources/fn/type", cursorCtx.StructuralPath.String())

	completion := DetermineCompletionContext(cursorCtx)
	s.Assert().Equal(CompletionContextResourceType, completion.Kind)
}

func (s *TokensToUnifiedSuite) Test_cursor_context_resolves_spec_field() {
	content := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n" +
		"    spec {\n" +
		"        functionName = \"x\"\n" +
		"    }\n" +
		"}\n"

	docCtx := NewDocumentContext("file:///test.bp", content, FormatBlueprintLang, nil)

	// Cursor in the "functionName" value (line 4, inside the "x" string) resolves
	// to the full spec-field path at a value position.
	cursorCtx := docCtx.GetCursorContext(source.Position{Line: 4, Column: 25}, 0)
	s.Require().NotNil(cursorCtx)
	s.Assert().Equal(
		"/resources/fn/spec/functionName",
		cursorCtx.StructuralPath.String(),
	)
}

func (s *TokensToUnifiedSuite) Test_blueprint_top_level_is_key_position() {
	// An empty document: the cursor at the root is a (declaration) key position.
	docCtx := NewDocumentContext("file:///test.bp", "", FormatBlueprintLang, nil)
	cursorCtx := docCtx.GetCursorContext(source.Position{Line: 1, Column: 1}, 0)
	s.Require().NotNil(cursorCtx)
	s.Assert().True(cursorCtx.IsAtKeyPosition(), "top level should be a key position")
	s.Assert().Equal(SyntacticStyleBlueprint, cursorCtx.Syntactic.Style)
	s.Assert().True(cursorCtx.StructuralPath.IsEmpty())
}

func (s *TokensToUnifiedSuite) Test_blueprint_empty_line_in_spec_is_key_position() {
	content := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n" +
		"    spec {\n" +
		"        \n" +
		"    }\n" +
		"}\n"
	docCtx := NewDocumentContext("file:///test.bp", content, FormatBlueprintLang, nil)

	// Blank line 4 inside the spec block.
	cursorCtx := docCtx.GetCursorContext(source.Position{Line: 4, Column: 9}, 0)
	s.Require().NotNil(cursorCtx)
	s.Assert().Equal("/resources/fn/spec", cursorCtx.StructuralPath.String())
	s.Assert().True(cursorCtx.IsAtKeyPosition(), "empty line in spec is a key position")
}

func (s *TokensToUnifiedSuite) Test_blueprint_partial_key_reparents_to_container() {
	content := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n" +
		"    desc\n" +
		"}\n"
	docCtx := NewDocumentContext("file:///test.bp", content, FormatBlueprintLang, nil)

	// Cursor at the end of the partial key "desc" on line 3 (col 9).
	cursorCtx := docCtx.GetCursorContext(source.Position{Line: 3, Column: 9}, 0)
	s.Require().NotNil(cursorCtx)
	s.Assert().True(cursorCtx.IsAtKeyPosition())
	s.Assert().Equal("/resources/fn", cursorCtx.StructuralPath.String(),
		"a partial field key should resolve against its enclosing block")
}

func (s *TokensToUnifiedSuite) Test_blueprint_value_position_after_equals() {
	content := "version \"2025-11-02\"\n" +
		"variable region: string {\n" +
		"    default = \n" +
		"}\n"
	docCtx := NewDocumentContext("file:///test.bp", content, FormatBlueprintLang, nil)

	// Cursor just after "default = " on line 3.
	cursorCtx := docCtx.GetCursorContext(source.Position{Line: 3, Column: 15}, 0)
	s.Require().NotNil(cursorCtx)
	s.Assert().True(cursorCtx.IsAtValuePosition(), "after '=' is a value position")
}

func (s *TokensToUnifiedSuite) Test_blueprint_bare_dotted_annotation_key_is_single_segment() {
	content := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n" +
		"    metadata {\n" +
		"        annotations = {\n" +
		"            aws.lambda.dynamodb.accessType = \"read\"\n" +
		"        }\n" +
		"    }\n" +
		"}\n"
	root, err := ParseBlueprintLangToUnified(content)
	s.Require().NoError(err)

	// The dotted key must be a single path segment, not split at the first dot.
	annotation := nodeAtPath(
		root, "resources", "fn", "metadata", "annotations", "aws.lambda.dynamodb.accessType",
	)
	s.Require().NotNil(annotation,
		"expected the bare dotted key as a single /metadata/annotations segment")
	s.Assert().Equal("read", annotation.Value)
	s.Assert().Equal(
		"/resources/fn/metadata/annotations/aws.lambda.dynamodb.accessType",
		annotation.Path(),
	)
}

func (s *TokensToUnifiedSuite) Test_blueprint_unclosed_object_covers_trailing_blank_line() {
	// In-progress edit: the annotations object is never closed.
	content := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n" +
		"    metadata {\n" +
		"        annotations = {\n" +
		"            "
	docCtx := NewDocumentContext("file:///test.bp", content, FormatBlueprintLang, nil)

	// Cursor on the trailing blank line (line 5) must resolve inside annotations.
	cursorCtx := docCtx.GetCursorContext(source.Position{Line: 5, Column: 13}, 0)
	s.Require().NotNil(cursorCtx)
	s.Assert().Equal(
		"/resources/fn/metadata/annotations",
		cursorCtx.StructuralPath.String(),
	)
}

func TestTokensToUnifiedSuite(t *testing.T) {
	suite.Run(t, new(TokensToUnifiedSuite))
}
