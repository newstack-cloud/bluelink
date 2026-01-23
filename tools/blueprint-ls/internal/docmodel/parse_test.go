package docmodel

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type ParseSuite struct {
	suite.Suite
}

func (s *ParseSuite) TestParseYAMLToUnified() {
	yamlContent := `
version: 2021-12-18
resources:
  myTable:
    type: aws/dynamodb/table
    spec:
      tableName: test-table
`

	node, err := ParseYAMLToUnified(yamlContent)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	s.Assert().Equal(NodeKindDocument, node.Kind)
	s.Assert().False(node.IsError)

	err = testhelpers.Snapshot(toSnapshotAST(node))
	s.Require().NoError(err)
}

func (s *ParseSuite) TestParseYAMLToUnified_WithErrors() {
	yamlContent := `
resources:
  myTable:
    type: aws/dynamodb/table
    spec:
      tableName: ${
`

	node, err := ParseYAMLToUnified(yamlContent)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	err = testhelpers.Snapshot(toSnapshotAST(node))
	s.Require().NoError(err)
}

func (s *ParseSuite) TestParseJSONCToUnified() {
	jsonContent := `{
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

	node, err := ParseJSONCToUnified(jsonContent)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	s.Assert().Equal(NodeKindDocument, node.Kind)
	s.Assert().False(node.IsError)
	s.Require().NotEmpty(node.Children)
	s.Assert().Equal(NodeKindMapping, node.Children[0].Kind)

	err = testhelpers.Snapshot(toSnapshotAST(node))
	s.Require().NoError(err)
}

func (s *ParseSuite) TestParseJSONCToUnified_WithComments() {
	jsonContent := `{
  // This is a comment
  "version": "2021-12-18",
  /* Multi-line
     comment */
  "resources": {}
}`

	node, err := ParseJSONCToUnified(jsonContent)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	s.Assert().Equal(NodeKindDocument, node.Kind)

	err = testhelpers.Snapshot(toSnapshotAST(node))
	s.Require().NoError(err)
}

func (s *ParseSuite) TestParseJSONCToUnified_WithErrors() {
	jsonContent := `{
  "version": "2021-12-18",
  "resources": {
`

	node, err := ParseJSONCToUnified(jsonContent)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	s.Assert().Equal(NodeKindDocument, node.Kind)
	s.Require().NotEmpty(node.Children)

	err = testhelpers.Snapshot(toSnapshotAST(node))
	s.Require().NoError(err)
}

func (s *ParseSuite) TestParseYAML_PositionAccuracy() {
	yamlContent := `version: 2021-12-18
resources:
  myTable:
    type: aws/dynamodb/table`

	node, err := ParseYAMLToUnified(yamlContent)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	s.Assert().Equal(1, node.Range.Start.Line)

	err = testhelpers.Snapshot(toSnapshotAST(node))
	s.Require().NoError(err)
}

func (s *ParseSuite) TestParseJSON_PositionAccuracy() {
	jsonContent := `{
  "version": "2021-12-18",
  "resources": {
    "myTable": {
      "type": "aws/dynamodb/table"
    }
  }
}`

	node, err := ParseJSONCToUnified(jsonContent)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	s.Assert().Equal(1, node.Range.Start.Line)
	s.Assert().Equal(1, node.Range.Start.Column)

	err = testhelpers.Snapshot(toSnapshotAST(node))
	s.Require().NoError(err)
}

func (s *ParseSuite) TestParseYAML_EmptyMappingValue() {
	// Test that spec: with no value creates an empty mapping node
	// This is important for indentation-based lookups to find spec as a parent
	yamlContent := `version: 2025-11-02
resources:
  saveOrder:
    type: aws/lambda/function
    spec:

`

	node, err := ParseYAMLToUnified(yamlContent)
	s.Require().NoError(err)
	s.Require().NotNil(node)

	// The first child should be the document mapping
	s.Require().NotEmpty(node.Children, "Document should have children")
	docMapping := node.Children[0]

	// Navigate to resources -> saveOrder -> spec
	var resourcesNode *UnifiedNode
	for _, child := range docMapping.Children {
		if child.FieldName == "resources" {
			resourcesNode = child
			break
		}
	}
	s.Require().NotNil(resourcesNode, "Should have resources node")

	var saveOrderNode *UnifiedNode
	for _, child := range resourcesNode.Children {
		if child.FieldName == "saveOrder" {
			saveOrderNode = child
			break
		}
	}
	s.Require().NotNil(saveOrderNode, "Should have saveOrder node")

	// Find spec node
	var specNode *UnifiedNode
	for _, child := range saveOrderNode.Children {
		if child.FieldName == "spec" {
			specNode = child
			break
		}
	}

	// spec: should exist as an empty mapping node
	s.Require().NotNil(specNode, "Should have spec node even when empty")
	s.Assert().Equal(NodeKindMapping, specNode.Kind, "spec should be a mapping node")
	s.Assert().Empty(specNode.Children, "spec should have no children")

	// Verify position information
	s.Assert().NotNil(specNode.Range.Start, "spec should have start position")
	s.Assert().Equal(5, specNode.Range.Start.Line, "spec should be on line 5")
}

func TestParseSuite(t *testing.T) {
	suite.Run(t, new(ParseSuite))
}
