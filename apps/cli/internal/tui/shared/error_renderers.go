package shared

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// ErrorContext provides context-specific strings for error rendering.
type ErrorContext struct {
	OperationName      string // e.g., "deployment", "change staging"
	FailedHeader       string // e.g., "Failed to start deployment"
	ErrorDuringHeader  string // e.g., "Error during deployment"
	IssuesPreamble     string // e.g., "The following issues must be resolved before deployment can proceed:"
}

// DeployErrorContext returns the error context for deployment operations.
func DeployErrorContext() ErrorContext {
	return ErrorContext{
		OperationName:      "deployment",
		FailedHeader:       "Failed to start deployment",
		ErrorDuringHeader:  "Error during deployment",
		IssuesPreamble:     "The following issues must be resolved before deployment can proceed:",
	}
}

// StageErrorContext returns the error context for staging operations.
func StageErrorContext() ErrorContext {
	return ErrorContext{
		OperationName:      "change staging",
		FailedHeader:       "Failed to create changeset",
		ErrorDuringHeader:  "Error during change staging",
		IssuesPreamble:     "The following issues must be resolved in the blueprint before changes can be staged:",
	}
}

// DestroyErrorContext returns the error context for destroy operations.
func DestroyErrorContext() ErrorContext {
	return ErrorContext{
		OperationName:      "destroy",
		FailedHeader:       "Failed to start destroy",
		ErrorDuringHeader:  "Error during destroy",
		IssuesPreamble:     "The following issues must be resolved before destroy can proceed:",
	}
}

// RenderErrorFooter renders a standard "Press q to quit" footer.
func RenderErrorFooter(s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	keyStyle := lipgloss.NewStyle().Foreground(s.Palette.Primary()).Bold(true)
	sb.WriteString(s.Muted.Render("  Press "))
	sb.WriteString(keyStyle.Render("q"))
	sb.WriteString(s.Muted.Render(" to quit"))
	sb.WriteString("\n")
	return sb.String()
}

// RenderDiagnostic renders a single diagnostic with level styling.
func RenderDiagnostic(diag *core.Diagnostic, s *styles.Styles) string {
	sb := strings.Builder{}

	var levelStyle lipgloss.Style
	levelName := "unknown"
	switch diag.Level {
	case core.DiagnosticLevelError:
		levelStyle = s.Error
		levelName = "ERROR"
	case core.DiagnosticLevelWarning:
		levelStyle = s.Warning
		levelName = "WARNING"
	case core.DiagnosticLevelInfo:
		levelStyle = s.Info
		levelName = "INFO"
	default:
		levelStyle = s.Muted
	}

	sb.WriteString("    ")
	sb.WriteString(levelStyle.Render(levelName))
	if diag.Range != nil && diag.Range.Start.Line > 0 {
		sb.WriteString(s.Muted.Render(fmt.Sprintf(" [line %d, col %d]", diag.Range.Start.Line, diag.Range.Start.Column)))
	}
	sb.WriteString(": ")
	sb.WriteString(diag.Message)
	sb.WriteString("\n")

	return sb.String()
}

// RenderValidationError renders a validation error with diagnostics.
func RenderValidationError(clientErr *engineerrors.ClientError, ctx ErrorContext, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(s.Error.Render("  ✗ " + ctx.FailedHeader + "\n\n"))

	sb.WriteString(s.Muted.Render("  " + ctx.IssuesPreamble + "\n\n"))

	if len(clientErr.ValidationErrors) > 0 {
		sb.WriteString(s.Category.Render("  Validation Errors:"))
		sb.WriteString("\n")
		for _, valErr := range clientErr.ValidationErrors {
			location := valErr.Location
			if location == "" {
				location = "unknown"
			}
			sb.WriteString(s.Error.Render(fmt.Sprintf("    • %s: ", location)))
			sb.WriteString(fmt.Sprintf("%s\n", valErr.Message))
		}
		sb.WriteString("\n")
	}

	if len(clientErr.ValidationDiagnostics) > 0 {
		sb.WriteString(s.Category.Render("  Blueprint Diagnostics:"))
		sb.WriteString("\n")
		for _, diag := range clientErr.ValidationDiagnostics {
			sb.WriteString(RenderDiagnostic(diag, s))
		}
		sb.WriteString("\n")
	}

	if len(clientErr.ValidationErrors) == 0 && len(clientErr.ValidationDiagnostics) == 0 {
		sb.WriteString(s.Error.Render(fmt.Sprintf("    %s\n", clientErr.Message)))
	}

	sb.WriteString(RenderErrorFooter(s))
	return sb.String()
}

// RenderStreamError renders a stream error with diagnostics.
func RenderStreamError(streamErr *engineerrors.StreamError, ctx ErrorContext, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(s.Error.Render("  ✗ " + ctx.ErrorDuringHeader + "\n\n"))

	sb.WriteString(s.Muted.Render("  The following issues occurred during " + ctx.OperationName + ":\n\n"))
	sb.WriteString(fmt.Sprintf("    %s\n\n", streamErr.Event.Message))

	if len(streamErr.Event.Diagnostics) > 0 {
		sb.WriteString(s.Category.Render("  Diagnostics:"))
		sb.WriteString("\n")
		for _, diag := range streamErr.Event.Diagnostics {
			sb.WriteString(RenderDiagnostic(diag, s))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(RenderErrorFooter(s))
	return sb.String()
}

// RenderGenericError renders a generic error with the operation context.
func RenderGenericError(err error, operationFailedHeader string, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(s.Error.Render("  ✗ " + operationFailedHeader + "\n\n"))
	sb.WriteString(s.Error.Render(fmt.Sprintf("    %s\n", err.Error())))
	sb.WriteString(RenderErrorFooter(s))
	return sb.String()
}
