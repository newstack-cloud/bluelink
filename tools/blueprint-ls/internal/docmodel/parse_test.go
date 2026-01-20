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

func TestParseSuite(t *testing.T) {
	suite.Run(t, new(ParseSuite))
}
