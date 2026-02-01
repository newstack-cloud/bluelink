package diagnostichelpers

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// BlueprintToLSP deals with transforming blueprint diagnostics to LSP diagnostics.
// When showAnyTypeWarnings is false, warning diagnostics tagged with
// ErrorReasonCodeAnyTypeWarning are filtered out.
func BlueprintToLSP(
	bpDiagnostics []*core.Diagnostic,
	showAnyTypeWarnings bool,
) []lsp.Diagnostic {
	lspDiagnostics := []lsp.Diagnostic{}
	source := "blueprint-validator"

	for _, bpDiagnostic := range bpDiagnostics {
		if !showAnyTypeWarnings && isAnyTypeWarning(bpDiagnostic) {
			continue
		}

		severity := lsp.DiagnosticSeverityInformation
		switch bpDiagnostic.Level {
		case core.DiagnosticLevelWarning:
			severity = lsp.DiagnosticSeverityWarning
		case core.DiagnosticLevelError:
			severity = lsp.DiagnosticSeverityError
		}

		// Build enhanced message with context if available
		message := bpDiagnostic.Message
		if bpDiagnostic.Context != nil {
			message = formatDiagnosticWithContext(bpDiagnostic.Message, bpDiagnostic.Context)
		}

		diag := lsp.Diagnostic{
			Severity: &severity,
			Message:  message,
			Source:   &source,
			Range: lspDiagnosticRangeFromBlueprintDiagnosticRange(
				bpDiagnostic.Range,
			),
		}

		// Add Code if ReasonCode is available in context
		if bpDiagnostic.Context != nil && bpDiagnostic.Context.ReasonCode != "" {
			code := string(bpDiagnostic.Context.ReasonCode)
			diag.Code = &lsp.IntOrString{StrVal: &code}
		}

		lspDiagnostics = append(lspDiagnostics, diag)
	}

	return lspDiagnostics
}

// isAnyTypeWarning checks if a diagnostic is a warning about a substitution
// resolving to the "any" type. Only warning-level diagnostics are matched.
func isAnyTypeWarning(diag *core.Diagnostic) bool {
	return diag.Level == core.DiagnosticLevelWarning &&
		diag.Context != nil &&
		diag.Context.ReasonCode == errors.ErrorReasonCodeAnyTypeWarning
}

// formatDiagnosticWithContext formats a diagnostic message with its ErrorContext,
// including suggested actions in a plain text format suitable for editor diagnostics.
func formatDiagnosticWithContext(message string, ctx *errors.ErrorContext) string {
	if ctx == nil || len(ctx.SuggestedActions) == 0 {
		return message
	}

	sb := strings.Builder{}
	sb.WriteString(message)
	sb.WriteString("\n\nSuggested Actions:\n")

	for i, action := range ctx.SuggestedActions {
		sb.WriteString(fmt.Sprintf("  %d. %s", i+1, action.Title))
		if action.Description != "" {
			sb.WriteString(fmt.Sprintf(": %s", action.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
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
