package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/stretchr/testify/suite"
)

type NodeContextSuite struct {
	suite.Suite
}

func (s *NodeContextSuite) TestInSubstitution() {
	tests := []struct {
		name       string
		textBefore string
		expected   bool
	}{
		{
			name:       "inside substitution",
			textBefore: "tableName: ${resources.",
			expected:   true,
		},
		{
			name:       "after closed substitution",
			textBefore: "tableName: ${resources.foo} more",
			expected:   false,
		},
		{
			name:       "no substitution",
			textBefore: "tableName: test",
			expected:   false,
		},
		{
			name:       "empty",
			textBefore: "",
			expected:   false,
		},
		{
			name:       "multiple open",
			textBefore: "${a} ${b",
			expected:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := &NodeContext{TextBefore: tt.textBefore}
			s.Assert().Equal(tt.expected, ctx.InSubstitution())
		})
	}
}

func (s *NodeContextSuite) TestGetSubstitutionText() {
	tests := []struct {
		name       string
		textBefore string
		expected   string
	}{
		{
			name:       "basic substitution",
			textBefore: "tableName: ${resources.foo",
			expected:   "resources.foo",
		},
		{
			name:       "not in substitution",
			textBefore: "tableName: test",
			expected:   "",
		},
		{
			name:       "closed substitution",
			textBefore: "tableName: ${resources.foo}",
			expected:   "",
		},
		{
			name:       "nested after closed",
			textBefore: "${a} ${b.c",
			expected:   "b.c",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := &NodeContext{TextBefore: tt.textBefore}
			s.Assert().Equal(tt.expected, ctx.GetSubstitutionText())
		})
	}
}

func (s *NodeContextSuite) TestExtractTextContext() {
	content := `line1
line2 with some text
line3`

	ctx := &NodeContext{}
	ctx.extractTextContext(content, source.Position{Line: 2, Column: 7})

	s.Assert().Equal("line2 with some text", ctx.CurrentLine)
	s.Assert().Equal("line2 ", ctx.TextBefore)
	s.Assert().Equal("with some text", ctx.TextAfter)
	s.Assert().Equal("with", ctx.CurrentWord)
}

func (s *NodeContextSuite) TestExtractTextContext_EdgeCases() {
	tests := []struct {
		name       string
		content    string
		pos        source.Position
		expectLine string
		expectWord string
	}{
		{
			name:       "line out of range",
			content:    "line1\nline2",
			pos:        source.Position{Line: 10, Column: 1},
			expectLine: "",
			expectWord: "",
		},
		{
			name:       "column at end",
			content:    "test",
			pos:        source.Position{Line: 1, Column: 5},
			expectLine: "test",
			expectWord: "test",
		},
		{
			name:       "empty line",
			content:    "\n\n",
			pos:        source.Position{Line: 2, Column: 1},
			expectLine: "",
			expectWord: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := &NodeContext{}
			ctx.extractTextContext(tt.content, tt.pos)
			s.Assert().Equal(tt.expectLine, ctx.CurrentLine)
			s.Assert().Equal(tt.expectWord, ctx.CurrentWord)
		})
	}
}

func (s *NodeContextSuite) TestExtractWordAtPosition() {
	tests := []struct {
		name     string
		line     string
		col      int
		expected string
	}{
		{
			name:     "word in middle",
			line:     "hello world test",
			col:      8,
			expected: "world",
		},
		{
			name:     "word at start",
			line:     "hello world",
			col:      0,
			expected: "hello",
		},
		{
			name:     "word at end",
			line:     "hello world",
			col:      11,
			expected: "world",
		},
		{
			name:     "hyphenated word",
			line:     "my-resource-name",
			col:      5,
			expected: "my-resource-name",
		},
		{
			name:     "dotted path",
			line:     "resources.foo.bar",
			col:      10,
			expected: "resources.foo.bar",
		},
		{
			name:     "empty string",
			line:     "",
			col:      0,
			expected: "",
		},
		{
			name:     "col out of bounds",
			line:     "test",
			col:      10,
			expected: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := extractWordAtPosition(tt.line, tt.col)
			s.Assert().Equal(tt.expected, result)
		})
	}
}

func (s *NodeContextSuite) TestIsWordChar() {
	wordChars := []byte{'a', 'z', 'A', 'Z', '0', '9', '_', '-', '.'}
	for _, c := range wordChars {
		s.Assert().True(isWordChar(c), "expected '%c' to be a word char", c)
	}

	nonWordChars := []byte{' ', '\t', ':', '$', '{', '}', '[', ']', '"', '\''}
	for _, c := range nonWordChars {
		s.Assert().False(isWordChar(c), "expected '%c' not to be a word char", c)
	}
}

func (s *NodeContextSuite) TestIsAtTypeField() {
	tests := []struct {
		name     string
		path     StructuredPath
		expected bool
	}{
		{
			name: "resource type",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myRes"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: true,
		},
		{
			name: "datasource type",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "datasources"},
				{Kind: PathSegmentField, FieldName: "myDS"},
				{Kind: PathSegmentField, FieldName: "type"},
			},
			expected: true,
		},
		{
			name: "not a type field",
			path: StructuredPath{
				{Kind: PathSegmentField, FieldName: "resources"},
				{Kind: PathSegmentField, FieldName: "myRes"},
				{Kind: PathSegmentField, FieldName: "spec"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := &NodeContext{ASTPath: tt.path}
			s.Assert().Equal(tt.expected, ctx.IsAtTypeField())
		})
	}
}

func (s *NodeContextSuite) TestGetResourceName() {
	ctx := &NodeContext{
		ASTPath: StructuredPath{
			{Kind: PathSegmentField, FieldName: "resources"},
			{Kind: PathSegmentField, FieldName: "myTable"},
			{Kind: PathSegmentField, FieldName: "type"},
		},
	}

	name, ok := ctx.GetResourceName()
	s.Assert().True(ok)
	s.Assert().Equal("myTable", name)
}

func (s *NodeContextSuite) TestHasError() {
	ctx := &NodeContext{}
	s.Assert().False(ctx.HasError())

	ctx.UnifiedNode = &UnifiedNode{IsError: false}
	s.Assert().False(ctx.HasError())

	ctx.UnifiedNode = &UnifiedNode{IsError: true}
	s.Assert().True(ctx.HasError())
}

func (s *NodeContextSuite) TestIsEmpty() {
	ctx := &NodeContext{}
	s.Assert().True(ctx.IsEmpty())

	ctx.UnifiedNode = &UnifiedNode{}
	s.Assert().False(ctx.IsEmpty())
}

func (s *NodeContextSuite) TestGetRange() {
	ctx := &NodeContext{}
	s.Assert().Nil(ctx.GetRange())

	ctx.UnifiedNode = &UnifiedNode{
		Range: source.Range{
			Start: &source.Position{Line: 1, Column: 1},
			End:   &source.Position{Line: 1, Column: 10},
		},
	}
	r := ctx.GetRange()
	s.Require().NotNil(r)
	s.Assert().Equal(1, r.Start.Line)
}

func (s *NodeContextSuite) TestGetValue() {
	ctx := &NodeContext{}
	s.Assert().Equal("", ctx.GetValue())

	ctx.UnifiedNode = &UnifiedNode{Value: "test-value"}
	s.Assert().Equal("test-value", ctx.GetValue())
}

func (s *NodeContextSuite) TestGetFieldName() {
	ctx := &NodeContext{}
	s.Assert().Equal("", ctx.GetFieldName())

	ctx.UnifiedNode = &UnifiedNode{FieldName: "myField"}
	s.Assert().Equal("myField", ctx.GetFieldName())
}

func (s *NodeContextSuite) TestGetEnclosingQuoteType() {
	tests := []struct {
		name      string
		ancestors []*UnifiedNode
		expected  QuoteType
	}{
		{
			name:      "no ancestors",
			ancestors: nil,
			expected:  QuoteTypeNone,
		},
		{
			name: "double quote scalar (YAML)",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "double_quote_scalar"},
			},
			expected: QuoteTypeDouble,
		},
		{
			name: "single quote scalar (YAML)",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "single_quote_scalar"},
			},
			expected: QuoteTypeSingle,
		},
		{
			name: "plain scalar (YAML)",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "plain_scalar"},
			},
			expected: QuoteTypeNone,
		},
		{
			name: "block scalar (YAML)",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "block_scalar"},
			},
			expected: QuoteTypeNone,
		},
		{
			name: "string scalar (JSON)",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "object"},
				{TSKind: "string_scalar"},
			},
			expected: QuoteTypeDouble,
		},
		{
			name: "string content (JSON)",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "object"},
				{TSKind: "string_content"},
			},
			expected: QuoteTypeDouble,
		},
		{
			name: "string (JSONC)",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "object"},
				{TSKind: "string"},
			},
			expected: QuoteTypeDouble,
		},
		{
			name: "nested in double quote",
			ancestors: []*UnifiedNode{
				{TSKind: "document"},
				{TSKind: "block_mapping"},
				{TSKind: "double_quote_scalar"},
				{TSKind: "some_inner_node"},
			},
			expected: QuoteTypeDouble,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := &NodeContext{AncestorNodes: tt.ancestors}
			s.Assert().Equal(tt.expected, ctx.GetEnclosingQuoteType())
		})
	}
}

func (s *NodeContextSuite) TestIsAtKeyPosition() {
	tests := []struct {
		name       string
		textBefore string
		expected   bool
	}{
		{
			name:       "empty line",
			textBefore: "",
			expected:   true,
		},
		{
			name:       "whitespace only",
			textBefore: "    ",
			expected:   true,
		},
		{
			name:       "typing a key name",
			textBefore: "  run",
			expected:   true,
		},
		{
			name:       "typing a partial key name",
			textBefore: "  runt",
			expected:   true,
		},
		{
			name:       "after colon with no value - value position",
			textBefore: "fieldName: ",
			expected:   false,
		},
		{
			name:       "after colon with value - value position",
			textBefore: "fieldName: someValue",
			expected:   false,
		},
		{
			name:       "typing in nested value",
			textBefore: "      ORDERS_TABLE: ${resources.ordersTable",
			expected:   false,
		},
		{
			name:       "typing value after quoted key",
			textBefore: `"type": aws/lambda`,
			expected:   false,
		},
		{
			name:       "with leading indent typing key",
			textBefore: "      na",
			expected:   true,
		},
		{
			name:       "colon at end waiting for value",
			textBefore: "name:",
			expected:   false,
		},
		{
			name:       "colon with space waiting for value",
			textBefore: "name: ",
			expected:   false,
		},
		// Multi-line cases
		{
			name:       "new line after mapping - key position",
			textBefore: "spec:\n    ",
			expected:   true,
		},
		{
			name:       "new line with content - typing key",
			textBefore: "spec:\n    run",
			expected:   true,
		},
		{
			name:       "new line after value - key position",
			textBefore: "spec:\n  name: test\n    ",
			expected:   true,
		},
		// JSONC cases
		{
			name:       "after opening brace - key position",
			textBefore: `"spec": { `,
			expected:   true,
		},
		{
			name:       "after comma - key position",
			textBefore: `"name": "test", `,
			expected:   true,
		},
		{
			name:       "after opening brace on new line",
			textBefore: "\"spec\": {\n    ",
			expected:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := &NodeContext{TextBefore: tt.textBefore}
			s.Assert().Equal(tt.expected, ctx.IsAtKeyPosition(), "textBefore: %q", tt.textBefore)
		})
	}
}

func (s *NodeContextSuite) TestIsAtValuePosition() {
	tests := []struct {
		name       string
		textBefore string
		expected   bool
	}{
		{
			name:       "empty line",
			textBefore: "",
			expected:   false,
		},
		{
			name:       "whitespace only",
			textBefore: "    ",
			expected:   false,
		},
		{
			name:       "typing a key name",
			textBefore: "  run",
			expected:   false,
		},
		{
			name:       "after colon with no value",
			textBefore: "fieldName: ",
			expected:   true,
		},
		{
			name:       "after colon typing value",
			textBefore: "fieldName: some",
			expected:   true,
		},
		{
			name:       "after colon with full value",
			textBefore: "fieldName: someValue",
			expected:   true,
		},
		{
			name:       "typing in nested value with substitution",
			textBefore: "      ORDERS_TABLE: ${resources.ordersTable",
			expected:   true,
		},
		{
			name:       "colon at end",
			textBefore: "name:",
			expected:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := &NodeContext{TextBefore: tt.textBefore}
			s.Assert().Equal(tt.expected, ctx.IsAtValuePosition(), "textBefore: %q", tt.textBefore)
		})
	}
}

func (s *NodeContextSuite) TestGetTypedPrefix() {
	tests := []struct {
		name        string
		textBefore  string
		currentWord string
		expected    string
	}{
		{
			name:       "empty line",
			textBefore: "",
			expected:   "",
		},
		{
			name:       "whitespace only",
			textBefore: "    ",
			expected:   "",
		},
		{
			name:       "typing a key name",
			textBefore: "  run",
			expected:   "run",
		},
		{
			name:       "typing longer key name",
			textBefore: "  runtime",
			expected:   "runtime",
		},
		{
			name:       "typing key name with hyphen",
			textBefore: "  my-field",
			expected:   "my-field",
		},
		{
			name:       "after colon typing value",
			textBefore: "fieldName: val",
			expected:   "val",
		},
		{
			name:       "after colon no value yet",
			textBefore: "fieldName: ",
			expected:   "",
		},
		{
			name:        "fallback to current word",
			textBefore:  "some: thing: else",
			currentWord: "else",
			expected:    "else",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := &NodeContext{
				TextBefore:  tt.textBefore,
				CurrentWord: tt.currentWord,
			}
			s.Assert().Equal(tt.expected, ctx.GetTypedPrefix(), "textBefore: %q", tt.textBefore)
		})
	}
}

func TestNodeContextSuite(t *testing.T) {
	suite.Run(t, new(NodeContextSuite))
}
