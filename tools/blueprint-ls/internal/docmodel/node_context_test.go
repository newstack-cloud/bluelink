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

func TestNodeContextSuite(t *testing.T) {
	suite.Run(t, new(NodeContextSuite))
}
