package languageservices

import (
	"testing"

	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
)

type DiagnosticsSuite struct {
	suite.Suite
}

func severityPtr(s lsp.DiagnosticSeverity) *lsp.DiagnosticSeverity {
	return &s
}

func (s *DiagnosticsSuite) TestDeduplicateDiagnostics_EmptyList() {
	result := deduplicateDiagnostics([]lsp.Diagnostic{})
	s.Len(result, 0)
}

func (s *DiagnosticsSuite) TestDeduplicateDiagnostics_SingleDiagnostic() {
	input := []lsp.Diagnostic{
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error message",
		},
	}
	result := deduplicateDiagnostics(input)
	s.Len(result, 1)
}

func (s *DiagnosticsSuite) TestDeduplicateDiagnostics_TwoDifferentDiagnostics() {
	input := []lsp.Diagnostic{
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error message 1",
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 2, Character: 0},
				End:   lsp.Position{Line: 2, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error message 2",
		},
	}
	result := deduplicateDiagnostics(input)
	s.Len(result, 2)
}

func (s *DiagnosticsSuite) TestDeduplicateDiagnostics_TwoIdenticalDiagnostics() {
	input := []lsp.Diagnostic{
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error message",
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error message",
		},
	}
	result := deduplicateDiagnostics(input)
	s.Len(result, 1)
}

func (s *DiagnosticsSuite) TestDeduplicateDiagnostics_SameMessageDifferentRange() {
	input := []lsp.Diagnostic{
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error message",
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 2, Character: 0},
				End:   lsp.Position{Line: 2, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error message",
		},
	}
	result := deduplicateDiagnostics(input)
	s.Len(result, 2)
}

func (s *DiagnosticsSuite) TestDeduplicateDiagnostics_SameRangeDifferentSeverity() {
	input := []lsp.Diagnostic{
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error message",
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityWarning),
			Message:  "error message",
		},
	}
	result := deduplicateDiagnostics(input)
	s.Len(result, 2)
}

func (s *DiagnosticsSuite) TestDeduplicateDiagnostics_MultipleDuplicates() {
	input := []lsp.Diagnostic{
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error 1",
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error 1",
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 2, Character: 0},
				End:   lsp.Position{Line: 2, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error 2",
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 2, Character: 0},
				End:   lsp.Position{Line: 2, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error 2",
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 2, Character: 0},
				End:   lsp.Position{Line: 2, Character: 10},
			},
			Severity: severityPtr(lsp.DiagnosticSeverityError),
			Message:  "error 2",
		},
	}
	result := deduplicateDiagnostics(input)
	s.Len(result, 2)
}

func (s *DiagnosticsSuite) TestDiagnosticKey() {
	diag := lsp.Diagnostic{
		Range: lsp.Range{
			Start: lsp.Position{Line: 5, Character: 10},
			End:   lsp.Position{Line: 5, Character: 20},
		},
		Severity: severityPtr(lsp.DiagnosticSeverityError),
		Message:  "test error",
	}

	key := diagnosticKey(diag)
	s.Equal("5:10-5:20|1|test error", key)
}

func TestDiagnosticsSuite(t *testing.T) {
	suite.Run(t, new(DiagnosticsSuite))
}
