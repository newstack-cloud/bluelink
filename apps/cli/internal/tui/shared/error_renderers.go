package shared

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/diagutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// ErrorContext provides context-specific strings for error rendering.
type ErrorContext struct {
	OperationName     string // e.g., "deployment", "change staging"
	FailedHeader      string // e.g., "Failed to start deployment"
	ErrorDuringHeader string // e.g., "Error during deployment"
	IssuesPreamble    string // e.g., "The following issues must be resolved before deployment can proceed:"
}

// DeployErrorContext returns the error context for deployment operations.
func DeployErrorContext() ErrorContext {
	return ErrorContext{
		OperationName:     "deployment",
		FailedHeader:      "Failed to start deployment",
		ErrorDuringHeader: "Error during deployment",
		IssuesPreamble:    "The following issues must be resolved before deployment can proceed:",
	}
}

// StageErrorContext returns the error context for staging operations.
func StageErrorContext() ErrorContext {
	return ErrorContext{
		OperationName:     "change staging",
		FailedHeader:      "Failed to create changeset",
		ErrorDuringHeader: "Error during change staging",
		IssuesPreamble:    "The following issues must be resolved in the blueprint before changes can be staged:",
	}
}

// DestroyErrorContext returns the error context for destroy operations.
func DestroyErrorContext() ErrorContext {
	return ErrorContext{
		OperationName:     "destroy",
		FailedHeader:      "Failed to start destroy",
		ErrorDuringHeader: "Error during destroy",
		IssuesPreamble:    "The following issues must be resolved before destroy can proceed:",
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

// RenderDiagnostic renders a single diagnostic with level styling and suggested actions.
func RenderDiagnostic(diag *core.Diagnostic, s *styles.Styles) string {
	sb := strings.Builder{}

	levelStyle, levelName := getDiagnosticLevelStyle(diag.Level, s)

	sb.WriteString("    ")
	sb.WriteString(levelStyle.Render(levelName))
	if diag.Range != nil && diag.Range.Start.Line > 0 {
		sb.WriteString(s.Muted.Render(fmt.Sprintf(" [line %d, col %d]", diag.Range.Start.Line, diag.Range.Start.Column)))
	}
	sb.WriteString(": ")
	sb.WriteString(diag.Message)
	sb.WriteString("\n")

	if diag.Context != nil && len(diag.Context.SuggestedActions) > 0 {
		sb.WriteString(renderSuggestedActions(diag.Context, s))
	}

	return sb.String()
}

func getDiagnosticLevelStyle(level core.DiagnosticLevel, s *styles.Styles) (lipgloss.Style, string) {
	switch level {
	case core.DiagnosticLevelError:
		return s.Error, "ERROR"
	case core.DiagnosticLevelWarning:
		return s.Warning, "WARNING"
	case core.DiagnosticLevelInfo:
		return s.Info, "INFO"
	default:
		return s.Muted, "unknown"
	}
}

func renderSuggestedActions(ctx *errors.ErrorContext, s *styles.Styles) string {
	if ctx == nil || len(ctx.SuggestedActions) == 0 {
		return ""
	}

	sb := strings.Builder{}
	sb.WriteString(s.Muted.Render("\n      Suggested Actions:\n"))

	for i, action := range ctx.SuggestedActions {
		renderSuggestedAction(&sb, i+1, action, ctx.Metadata, s)
	}

	return sb.String()
}

func renderSuggestedAction(
	sb *strings.Builder,
	index int,
	action errors.SuggestedAction,
	metadata map[string]any,
	s *styles.Styles,
) {
	fmt.Fprintf(sb, "        %d. %s\n", index, s.Info.Render(action.Title))
	if action.Description != "" {
		fmt.Fprintf(sb, "           %s\n", action.Description)
	}

	concrete := diagutils.GetConcreteAction(action, metadata)
	if concrete == nil {
		return
	}

	for _, cmd := range concrete.Commands {
		fmt.Fprintf(sb, "           %s %s\n", s.Muted.Render("Run:"), cmd)
	}
	for _, link := range concrete.Links {
		fmt.Fprintf(sb, "           %s %s\n", s.Muted.Render("See:"), link.URL)
	}
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

// ChangesetTypeMismatchParams holds the parameters for rendering a changeset type mismatch error.
type ChangesetTypeMismatchParams struct {
	// IsDestroyChangeset indicates whether the changeset is a destroy changeset (true)
	// or a deploy changeset (false). This determines the error message direction.
	IsDestroyChangeset bool
	InstanceName       string
	ChangesetID        string
}

// RenderChangesetTypeMismatchError renders an error when attempting to use a changeset
// with the wrong command (e.g., using a destroy changeset with deploy command).
func RenderChangesetTypeMismatchError(params ChangesetTypeMismatchParams, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString("  ")

	var errorMsg, explanation, correctCommand, correctCommandExample, alternativeDesc, alternativeExample string
	if params.IsDestroyChangeset {
		// User tried to deploy with a destroy changeset
		errorMsg = "✗ Cannot deploy using a destroy changeset"
		explanation = "The changeset you specified was created for a destroy operation and cannot\n  be used with the deploy command."
		correctCommand = "1. Use the 'destroy' command to apply this changeset:"
		correctCommandExample = fmt.Sprintf("       bluelink destroy --instance-name %s --change-set-id %s", params.InstanceName, params.ChangesetID)
		alternativeDesc = "2. Create a new changeset for deployment (without --destroy):"
		alternativeExample = fmt.Sprintf("       bluelink stage --instance-name %s", params.InstanceName)
	} else {
		// User tried to destroy with a deploy changeset
		errorMsg = "✗ Cannot destroy using a deploy changeset"
		explanation = "The changeset you specified was created for a deploy operation and cannot\n  be used with the destroy command."
		correctCommand = "1. Use the 'deploy' command to apply this changeset:"
		correctCommandExample = fmt.Sprintf("       bluelink deploy --instance-name %s --change-set-id %s", params.InstanceName, params.ChangesetID)
		alternativeDesc = "2. Create a new changeset for destroy:"
		alternativeExample = fmt.Sprintf("       bluelink stage --instance-name %s --destroy", params.InstanceName)
	}

	sb.WriteString(s.Error.Render(errorMsg))
	sb.WriteString("\n\n")

	sb.WriteString("  ")
	sb.WriteString(s.Muted.Render(explanation))
	sb.WriteString("\n\n")

	sb.WriteString("  ")
	sb.WriteString(s.Muted.Render("To resolve this issue, you can either:"))
	sb.WriteString("\n\n")

	sb.WriteString("    ")
	sb.WriteString(s.Muted.Render(correctCommand))
	sb.WriteString("\n")
	sb.WriteString(correctCommandExample)
	sb.WriteString("\n\n")

	sb.WriteString("    ")
	sb.WriteString(s.Muted.Render(alternativeDesc))
	sb.WriteString("\n")
	sb.WriteString(alternativeExample)
	sb.WriteString("\n")

	sb.WriteString(RenderErrorFooter(s))
	return sb.String()
}
