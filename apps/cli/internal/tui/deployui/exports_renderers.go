package deployui

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// Ensure ExportsDetailsRenderer implements splitpane.DetailsRenderer.
var _ splitpane.DetailsRenderer = (*ExportsDetailsRenderer)(nil)

// ExportsDetailsRenderer renders export details for a selected instance.
type ExportsDetailsRenderer struct{}

// RenderDetails renders the export details for the selected instance.
func (r *ExportsDetailsRenderer) RenderDetails(item splitpane.Item, width int, s *styles.Styles) string {
	instanceItem, ok := item.(*ExportsInstanceItem)
	if !ok || instanceItem == nil {
		return s.Muted.Render("No instance selected")
	}

	sb := strings.Builder{}

	// Header with instance name
	sb.WriteString(s.Header.Render(instanceItem.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", width-4)))
	sb.WriteString("\n\n")

	// Instance ID
	if instanceItem.InstanceID != "" {
		sb.WriteString(s.Muted.Render("Instance ID: "))
		sb.WriteString(instanceItem.InstanceID)
		sb.WriteString("\n")
	}

	// Path (for nested children)
	if instanceItem.Path != "" {
		sb.WriteString(s.Muted.Render("Path: "))
		sb.WriteString(instanceItem.Path)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Check for exports
	if instanceItem.InstanceState == nil || len(instanceItem.InstanceState.Exports) == 0 {
		sb.WriteString(s.Muted.Render("No exports defined for this instance"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Render exports
	sb.WriteString(s.Category.Render("Exports:"))
	sb.WriteString("\n\n")

	fields := outpututil.CollectExportFieldsPretty(instanceItem.InstanceState.Exports)
	for _, field := range fields {
		r.renderExportField(&sb, field, width, s)
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderExportField renders a single export with all its metadata.
func (r *ExportsDetailsRenderer) renderExportField(sb *strings.Builder, field outpututil.ExportField, width int, s *styles.Styles) {
	// Export name as sub-header
	sb.WriteString(s.Selected.Render("  " + field.Name))
	sb.WriteString("\n")

	// Type
	sb.WriteString(s.Muted.Render(fmt.Sprintf("    Type: %s", field.Type)))
	sb.WriteString("\n")

	// Source field
	if field.Field != "" {
		sb.WriteString(s.Muted.Render(fmt.Sprintf("    Field: %s", field.Field)))
		sb.WriteString("\n")
	}

	// Description (if present)
	if field.Description != "" {
		sb.WriteString(s.Muted.Render(fmt.Sprintf("    Description: %s", field.Description)))
		sb.WriteString("\n")
	}

	// Value with proper formatting
	sb.WriteString(s.Muted.Render("    Value:"))
	sb.WriteString("\n")

	// Format the value with indentation
	formattedValue := formatExportValue(field.Value, width-8)
	sb.WriteString(formattedValue)
}

// formatExportValue formats an export value with proper indentation and text wrapping.
func formatExportValue(value string, maxWidth int) string {
	if value == "" || value == "null" {
		return "      null\n"
	}

	// Account for the 6-space indentation prefix
	wrapWidth := maxWidth - 6
	if wrapWidth < 20 {
		wrapWidth = 20 // Minimum wrap width
	}

	lines := strings.Split(value, "\n")
	sb := strings.Builder{}
	for _, line := range lines {
		// Wrap long lines
		if len(line) > wrapWidth {
			wrappedLine := outpututil.WrapText(line, wrapWidth)
			wrappedLines := strings.Split(wrappedLine, "\n")
			for _, wl := range wrappedLines {
				sb.WriteString("      ")
				sb.WriteString(wl)
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString("      ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// ExportsFooterRenderer renders the footer for the exports view.
type ExportsFooterRenderer struct{}

// Ensure ExportsFooterRenderer implements splitpane.FooterRenderer.
var _ splitpane.FooterRenderer = (*ExportsFooterRenderer)(nil)

// RenderFooter renders the exports view footer with navigation hints.
func (r *ExportsFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	shared.RenderExportsFooterNavigation(&sb, s)
	return sb.String()
}

// ExportsHeaderRenderer renders the header for the exports view.
type ExportsHeaderRenderer struct {
	InstanceName string
}

// Ensure ExportsHeaderRenderer implements splitpane.HeaderRenderer.
var _ splitpane.HeaderRenderer = (*ExportsHeaderRenderer)(nil)

// RenderHeader renders the exports view header.
func (r *ExportsHeaderRenderer) RenderHeader(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}

	title := "Instance Exports"
	if r.InstanceName != "" {
		title = fmt.Sprintf("Exports: %s", r.InstanceName)
	}

	sb.WriteString(s.Header.Render(title))
	sb.WriteString("\n")

	return sb.String()
}
