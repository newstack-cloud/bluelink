package languageservices

import (
	"testing"

	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/assert"
)

func severityPtr(s lsp.DiagnosticSeverity) *lsp.DiagnosticSeverity {
	return &s
}

func TestDeduplicateDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		input    []lsp.Diagnostic
		expected int
	}{
		{
			name:     "empty list",
			input:    []lsp.Diagnostic{},
			expected: 0,
		},
		{
			name: "single diagnostic",
			input: []lsp.Diagnostic{
				{
					Range: lsp.Range{
						Start: lsp.Position{Line: 1, Character: 0},
						End:   lsp.Position{Line: 1, Character: 10},
					},
					Severity: severityPtr(lsp.DiagnosticSeverityError),
					Message:  "error message",
				},
			},
			expected: 1,
		},
		{
			name: "two different diagnostics",
			input: []lsp.Diagnostic{
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
			},
			expected: 2,
		},
		{
			name: "two identical diagnostics",
			input: []lsp.Diagnostic{
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
			},
			expected: 1,
		},
		{
			name: "same message different range",
			input: []lsp.Diagnostic{
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
			},
			expected: 2,
		},
		{
			name: "same range different severity",
			input: []lsp.Diagnostic{
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
			},
			expected: 2,
		},
		{
			name: "multiple duplicates",
			input: []lsp.Diagnostic{
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
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateDiagnostics(tt.input)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestDiagnosticKey(t *testing.T) {
	diag := lsp.Diagnostic{
		Range: lsp.Range{
			Start: lsp.Position{Line: 5, Character: 10},
			End:   lsp.Position{Line: 5, Character: 20},
		},
		Severity: severityPtr(lsp.DiagnosticSeverityError),
		Message:  "test error",
	}

	key := diagnosticKey(diag)
	assert.Equal(t, "5:10-5:20|1|test error", key)
}
