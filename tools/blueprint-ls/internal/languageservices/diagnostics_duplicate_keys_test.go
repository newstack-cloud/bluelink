package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
)

type DuplicateKeyDiagnosticsSuite struct {
	suite.Suite
}

func (s *DuplicateKeyDiagnosticsSuite) TestDuplicateKeysToDiagnostics_Nil() {
	result := DuplicateKeysToDiagnostics(nil)
	s.Assert().Nil(result)
}

func (s *DuplicateKeyDiagnosticsSuite) TestDuplicateKeysToDiagnostics_Empty() {
	result := DuplicateKeysToDiagnostics(&docmodel.DuplicateKeyResult{})
	s.Assert().Nil(result)
}

func (s *DuplicateKeyDiagnosticsSuite) TestDuplicateKeysToDiagnostics_SingleDuplicate() {
	result := &docmodel.DuplicateKeyResult{
		Errors: []*docmodel.DuplicateKeyError{
			{
				Key:     "myKey",
				IsFirst: true,
				Range: source.Range{
					Start: &source.Position{Line: 5, Column: 3},
					End:   &source.Position{Line: 5, Column: 8},
				},
			},
			{
				Key:     "myKey",
				IsFirst: false,
				Range: source.Range{
					Start: &source.Position{Line: 10, Column: 3},
					End:   &source.Position{Line: 10, Column: 8},
				},
			},
		},
	}

	diagnostics := DuplicateKeysToDiagnostics(result)
	s.Require().Len(diagnostics, 2)

	// First occurrence message
	s.Assert().Contains(diagnostics[0].Message, "first occurrence")
	s.Assert().Contains(diagnostics[0].Message, "myKey")
	s.Assert().Equal(lsp.UInteger(4), diagnostics[0].Range.Start.Line) // 0-based

	// Second occurrence message
	s.Assert().Contains(diagnostics[1].Message, "Duplicate key 'myKey'")
	s.Assert().NotContains(diagnostics[1].Message, "first occurrence")
	s.Assert().Equal(lsp.UInteger(9), diagnostics[1].Range.Start.Line)
}

func (s *DuplicateKeyDiagnosticsSuite) TestDuplicateKeysToDiagnostics_Severity() {
	result := &docmodel.DuplicateKeyResult{
		Errors: []*docmodel.DuplicateKeyError{
			{
				Key:     "key",
				IsFirst: true,
				Range: source.Range{
					Start: &source.Position{Line: 1, Column: 1},
					End:   &source.Position{Line: 1, Column: 5},
				},
			},
		},
	}

	diagnostics := DuplicateKeysToDiagnostics(result)
	s.Require().Len(diagnostics, 1)
	s.Assert().Equal(lsp.DiagnosticSeverityError, *diagnostics[0].Severity)
}

func (s *DuplicateKeyDiagnosticsSuite) TestDuplicateKeysToDiagnostics_HasCodeAndSource() {
	result := &docmodel.DuplicateKeyResult{
		Errors: []*docmodel.DuplicateKeyError{
			{
				Key:     "key",
				IsFirst: true,
				Range: source.Range{
					Start: &source.Position{Line: 1, Column: 1},
					End:   &source.Position{Line: 1, Column: 5},
				},
			},
		},
	}

	diagnostics := DuplicateKeysToDiagnostics(result)
	s.Require().Len(diagnostics, 1)
	s.Assert().NotNil(diagnostics[0].Code)
	s.Assert().Equal(DiagnosticCodeDuplicateKey, *diagnostics[0].Code.StrVal)
	s.Assert().NotNil(diagnostics[0].Source)
	s.Assert().Equal(DiagnosticSourceSyntax, *diagnostics[0].Source)
}

func (s *DuplicateKeyDiagnosticsSuite) TestDuplicateKeysToDiagnostics_UsesKeyRange() {
	keyRange := &source.Range{
		Start: &source.Position{Line: 5, Column: 3},
		End:   &source.Position{Line: 5, Column: 8},
	}
	result := &docmodel.DuplicateKeyResult{
		Errors: []*docmodel.DuplicateKeyError{
			{
				Key:     "key",
				IsFirst: true,
				Range: source.Range{
					Start: &source.Position{Line: 5, Column: 1},
					End:   &source.Position{Line: 6, Column: 10},
				},
				KeyRange: keyRange,
			},
		},
	}

	diagnostics := DuplicateKeysToDiagnostics(result)
	s.Require().Len(diagnostics, 1)
	// Should use KeyRange (column 3) not Range (column 1)
	s.Assert().Equal(lsp.UInteger(2), diagnostics[0].Range.Start.Character)
}

func (s *DuplicateKeyDiagnosticsSuite) TestSourceRangeToLSP_NilRange() {
	result := sourceRangeToLSP(nil)
	s.Assert().Equal(lsp.UInteger(0), result.Start.Line)
	s.Assert().Equal(lsp.UInteger(0), result.Start.Character)
	s.Assert().Equal(lsp.UInteger(1), result.End.Line)
}

func (s *DuplicateKeyDiagnosticsSuite) TestSourceRangeToLSP_NilStart() {
	result := sourceRangeToLSP(&source.Range{})
	s.Assert().Equal(lsp.UInteger(0), result.Start.Line)
	s.Assert().Equal(lsp.UInteger(0), result.Start.Character)
}

func (s *DuplicateKeyDiagnosticsSuite) TestSourceRangeToLSP_NoEnd() {
	result := sourceRangeToLSP(&source.Range{
		Start: &source.Position{Line: 5, Column: 3},
	})
	s.Assert().Equal(lsp.UInteger(4), result.Start.Line)
	s.Assert().Equal(lsp.UInteger(2), result.Start.Character)
	s.Assert().Equal(lsp.UInteger(5), result.End.Line) // start.Line + 1
	s.Assert().Equal(lsp.UInteger(0), result.End.Character)
}

func TestDuplicateKeyDiagnosticsSuite(t *testing.T) {
	suite.Run(t, new(DuplicateKeyDiagnosticsSuite))
}
