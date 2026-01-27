package blueprint

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParseSuite struct {
	suite.Suite
}

func (s *ParseSuite) TestParseYAMLNode_ValidYAML() {
	content := `
version: 2024-01-01
resources:
  myResource:
    type: aws/lambda/function
`
	node, err := ParseYAMLNode(content)
	s.NoError(err)
	s.NotNil(node)
	s.Equal("", node.Value) // Root document node has empty value
}

func (s *ParseSuite) TestParseYAMLNode_SimpleKeyValue() {
	content := "key: value"
	node, err := ParseYAMLNode(content)
	s.NoError(err)
	s.NotNil(node)
	// The root node is a document node containing a mapping
	s.NotEmpty(node.Content)
}

func (s *ParseSuite) TestParseYAMLNode_InvalidYAML() {
	content := `
key: value
  badindent: invalid
`
	node, err := ParseYAMLNode(content)
	s.Error(err)
	s.Nil(node)
}

func (s *ParseSuite) TestParseYAMLNode_EmptyString() {
	content := ""
	node, err := ParseYAMLNode(content)
	s.NoError(err)
	s.NotNil(node)
}

func (s *ParseSuite) TestParseYAMLNode_ComplexNestedStructure() {
	content := `
root:
  level1:
    level2:
      - item1
      - item2
    another: value
`
	node, err := ParseYAMLNode(content)
	s.NoError(err)
	s.NotNil(node)
	s.NotEmpty(node.Content)
}

func (s *ParseSuite) TestParseYAMLNode_WithComments() {
	content := `
# This is a comment
key: value # inline comment
`
	node, err := ParseYAMLNode(content)
	s.NoError(err)
	s.NotNil(node)
}

func (s *ParseSuite) TestParseJWCCNode_ValidJSON() {
	content := `{"key": "value", "number": 42}`
	node, err := ParseJWCCNode(content)
	s.NoError(err)
	s.NotNil(node)
}

func (s *ParseSuite) TestParseJWCCNode_ValidJSONCWithComments() {
	content := `{
		// This is a comment
		"key": "value",
		/* multi-line
		   comment */
		"number": 42
	}`
	node, err := ParseJWCCNode(content)
	s.NoError(err)
	s.NotNil(node)
}

func (s *ParseSuite) TestParseJWCCNode_ValidJSONCWithTrailingCommas() {
	content := `{
		"key": "value",
		"array": [1, 2, 3,],
	}`
	node, err := ParseJWCCNode(content)
	s.NoError(err)
	s.NotNil(node)
}

func (s *ParseSuite) TestParseJWCCNode_InvalidJSON() {
	content := `{key: value}` // Missing quotes around key
	node, err := ParseJWCCNode(content)
	s.Error(err)
	s.Nil(node)
}

func (s *ParseSuite) TestParseJWCCNode_EmptyObject() {
	content := `{}`
	node, err := ParseJWCCNode(content)
	s.NoError(err)
	s.NotNil(node)
}

func (s *ParseSuite) TestParseJWCCNode_ComplexNestedStructure() {
	content := `{
		"root": {
			"level1": {
				"level2": ["item1", "item2"],
				"another": "value"
			}
		}
	}`
	node, err := ParseJWCCNode(content)
	s.NoError(err)
	s.NotNil(node)
}

func TestParseSuite(t *testing.T) {
	suite.Run(t, new(ParseSuite))
}
