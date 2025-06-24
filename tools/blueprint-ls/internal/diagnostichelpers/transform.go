package diagnostichelpers

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// BlueprintToLSP deals with transforming blueprint diagnostics to LSP diagnostics.
func BlueprintToLSP(bpDiagnostics []*core.Diagnostic) []lsp.Diagnostic {
	lspDiagnostics := []lsp.Diagnostic{}

	for _, bpDiagnostic := range bpDiagnostics {
		severity := lsp.DiagnosticSeverityInformation
		if bpDiagnostic.Level == core.DiagnosticLevelWarning {
			severity = lsp.DiagnosticSeverityWarning
		} else if bpDiagnostic.Level == core.DiagnosticLevelError {
			severity = lsp.DiagnosticSeverityError
		}

		lspDiagnostics = append(lspDiagnostics, lsp.Diagnostic{
			Severity: &severity,
			Message:  bpDiagnostic.Message,
			Range: lspDiagnosticRangeFromBlueprintDiagnosticRange(
				bpDiagnostic.Range,
			),
		})
	}

	return lspDiagnostics
}

func lspDiagnosticRangeFromBlueprintDiagnosticRange(bpRange *core.DiagnosticRange) lsp.Range {
	if bpRange == nil {
		return lsp.Range{
			Start: lsp.Position{
				Line:      0,
				Character: 0,
			},
			End: lsp.Position{
				Line:      1,
				Character: 0,
			},
		}
	}

	start := lspPositionFromSourceMeta(bpRange.Start, nil, bpRange.ColumnAccuracy)
	end := lspPositionFromSourceMeta(bpRange.End, &start, bpRange.ColumnAccuracy)

	return lsp.Range{
		Start: start,
		End:   end,
	}
}

func lspPositionFromSourceMeta(
	sourceMeta *source.Meta,
	startPos *lsp.Position,
	columnAccuracy *substitutions.ColumnAccuracy,
) lsp.Position {
	if sourceMeta == nil && startPos == nil {
		return lsp.Position{
			Line:      0,
			Character: 0,
		}
	}

	if sourceMeta == nil && startPos != nil {
		return lsp.Position{
			Line:      startPos.Line + 1,
			Character: 0,
		}
	}

	// When columnAccuracy is nil, it is assumed this diagnostic is not in a substitution
	// context.
	if columnAccuracy != nil && *columnAccuracy == substitutions.ColumnAccuracyApproximate {
		return lsp.Position{
			// LSP offsets are 0-based, the blueprint package uses 1-based offsets.
			Line:      lsp.UInteger(sourceMeta.Line - 1),
			Character: lsp.UInteger(0),
		}
	}

	return lsp.Position{
		// LSP offsets are 0-based, the blueprint package uses 1-based offsets.
		Line:      lsp.UInteger(sourceMeta.Line - 1),
		Character: lsp.UInteger(sourceMeta.Column - 1),
	}
}
