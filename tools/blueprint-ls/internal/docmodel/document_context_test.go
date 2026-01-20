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
