package languageservices

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

const (
	DiagnosticCodeDuplicateKey = "duplicate-key"
	DiagnosticSourceSyntax     = "blueprint-syntax"
)

// DuplicateKeysToDiagnostics converts duplicate key detection results to LSP diagnostics.
func DuplicateKeysToDiagnostics(result *docmodel.DuplicateKeyResult) []lsp.Diagnostic {
	if result == nil || len(result.Errors) == 0 {
		return nil
	}

	diagnostics := make([]lsp.Diagnostic, 0, len(result.Errors))
	severity := lsp.DiagnosticSeverityError
	diagSource := DiagnosticSourceSyntax
	code := DiagnosticCodeDuplicateKey

	for _, err := range result.Errors {
		var message string
		if err.IsFirst {
			message = fmt.Sprintf("Duplicate key '%s' (first occurrence)", err.Key)
		} else {
			message = fmt.Sprintf("Duplicate key '%s'", err.Key)
		}

		diagRange := duplicateKeyRangeToLSP(err)

		diagnostics = append(diagnostics, lsp.Diagnostic{
			Range:    diagRange,
			Severity: &severity,
			Code:     &lsp.IntOrString{StrVal: &code},
			Source:   &diagSource,
			Message:  message,
		})
	}

	return diagnostics
}

func duplicateKeyRangeToLSP(err *docmodel.DuplicateKeyError) lsp.Range {
	// Prefer KeyRange for precise highlighting of just the key text
	if err.KeyRange != nil && err.KeyRange.Start != nil {
		return sourceRangeToLSP(err.KeyRange)
	}
	return sourceRangeToLSP(&err.Range)
}

func sourceRangeToLSP(r *source.Range) lsp.Range {
	if r == nil || r.Start == nil {
		return lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 1, Character: 0},
		}
	}

	start := lsp.Position{
		Line:      lsp.UInteger(r.Start.Line - 1),
		Character: lsp.UInteger(r.Start.Column - 1),
	}

	var end lsp.Position
	if r.End != nil {
		end = lsp.Position{
			Line:      lsp.UInteger(r.End.Line - 1),
			Character: lsp.UInteger(r.End.Column - 1),
		}
	} else {
		end = lsp.Position{
			Line:      start.Line + 1,
			Character: 0,
		}
	}

	return lsp.Range{Start: start, End: end}
}
