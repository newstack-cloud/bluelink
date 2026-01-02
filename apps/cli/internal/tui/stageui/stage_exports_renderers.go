package stageui

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// Ensure StageExportsDetailsRenderer implements splitpane.DetailsRenderer.
var _ splitpane.DetailsRenderer = (*StageExportsDetailsRenderer)(nil)

// StageExportsDetailsRenderer renders export change details for a selected instance.
type StageExportsDetailsRenderer struct{}

// RenderDetails renders the export change details for the selected instance.
func (r *StageExportsDetailsRenderer) RenderDetails(item splitpane.Item, width int, s *styles.Styles) string {
	instanceItem, ok := item.(*StageExportsInstanceItem)
	if !ok || instanceItem == nil {
		return s.Muted.Render("No instance selected")
	}

	sb := strings.Builder{}

	// Header with instance name
	sb.WriteString(s.Header.Render(instanceItem.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Path (for nested children)
	if instanceItem.Path != "" {
		sb.WriteString(s.Muted.Render("Path: "))
		sb.WriteString(instanceItem.Path)
		sb.WriteString("\n\n")
	}

	// Check for export changes
	if instanceItem.Changes == nil || !instanceItem.HasExportChanges() {
		if instanceItem.UnchangedCount > 0 {
			sb.WriteString(s.Muted.Render(fmt.Sprintf("%d unchanged exports", instanceItem.UnchangedCount)))
		} else {
			sb.WriteString(s.Muted.Render("No export changes for this instance"))
		}
		sb.WriteString("\n")
		return sb.String()
	}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	// Render new exports
	if instanceItem.NewCount > 0 {
		sb.WriteString(s.Category.Render("New Exports:"))
		sb.WriteString("\n\n")
		r.renderNewExports(&sb, instanceItem.Changes, width, s, successStyle)
		sb.WriteString("\n")
	}

	// Render modified exports
	if instanceItem.ModifiedCount > 0 {
		sb.WriteString(s.Category.Render("Modified Exports:"))
		sb.WriteString("\n\n")
		r.renderModifiedExports(&sb, instanceItem.Changes, width, s)
		sb.WriteString("\n")
	}

	// Render removed exports
	if instanceItem.RemovedCount > 0 {
		sb.WriteString(s.Category.Render("Removed Exports:"))
		sb.WriteString("\n\n")
		r.renderRemovedExports(&sb, instanceItem.Changes, s)
		sb.WriteString("\n")
	}

	// Render unchanged exports count
	if instanceItem.UnchangedCount > 0 {
		sb.WriteString(s.Muted.Render(fmt.Sprintf("Unchanged: %d exports", instanceItem.UnchangedCount)))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderNewExports renders the new exports section.
// This includes both explicit NewExports and exports in ExportChanges that have nil prevValue
// (which are effectively new exports for new deployments).
func (r *StageExportsDetailsRenderer) renderNewExports(
	sb *strings.Builder,
	bc *changes.BlueprintChanges,
	width int,
	s *styles.Styles,
	successStyle lipgloss.Style,
) {
	// Collect all new exports (explicit NewExports + ExportChanges with nil prevValue)
	allNewExports := make(map[string]provider.FieldChange)

	for name, change := range bc.NewExports {
		allNewExports[name] = change
	}

	// Include exports from ExportChanges that have nil prevValue (new exports for new deployments)
	for name, change := range bc.ExportChanges {
		if change.PrevValue == nil {
			allNewExports[name] = change
		}
	}

	if len(allNewExports) == 0 {
		return
	}

	// Sort export names for consistent ordering
	exportNames := make([]string, 0, len(allNewExports))
	for name := range allNewExports {
		exportNames = append(exportNames, name)
	}
	sort.Strings(exportNames)

	for _, name := range exportNames {
		change := allNewExports[name]
		r.renderExportChange(sb, name, &change, bc.ResolveOnDeploy, true, false, s, successStyle)
	}
}

// renderModifiedExports renders the modified exports section.
// Only exports with a non-nil prevValue are shown as modified.
// Exports with nil prevValue are treated as new exports (for new deployments).
func (r *StageExportsDetailsRenderer) renderModifiedExports(
	sb *strings.Builder,
	bc *changes.BlueprintChanges,
	width int,
	s *styles.Styles,
) {
	if bc.ExportChanges == nil {
		return
	}

	// Collect only exports with non-nil prevValue (true modifications)
	modifiedExports := make(map[string]provider.FieldChange)
	for name, change := range bc.ExportChanges {
		if change.PrevValue != nil {
			modifiedExports[name] = change
		}
	}

	if len(modifiedExports) == 0 {
		return
	}

	// Sort export names for consistent ordering
	exportNames := make([]string, 0, len(modifiedExports))
	for name := range modifiedExports {
		exportNames = append(exportNames, name)
	}
	sort.Strings(exportNames)

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	for _, name := range exportNames {
		change := modifiedExports[name]
		r.renderExportChange(sb, name, &change, bc.ResolveOnDeploy, false, true, s, successStyle)
	}
}

// renderRemovedExports renders the removed exports section.
func (r *StageExportsDetailsRenderer) renderRemovedExports(
	sb *strings.Builder,
	bc *changes.BlueprintChanges,
	s *styles.Styles,
) {
	if len(bc.RemovedExports) == 0 {
		return
	}

	// Sort export names for consistent ordering
	sortedNames := make([]string, len(bc.RemovedExports))
	copy(sortedNames, bc.RemovedExports)
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		line := fmt.Sprintf("  - %s", name)
		sb.WriteString(s.Error.Render(line))
		sb.WriteString("\n")
	}
}

// renderExportChange renders a single export with its metadata and value.
func (r *StageExportsDetailsRenderer) renderExportChange(
	sb *strings.Builder,
	name string,
	change *provider.FieldChange,
	resolveOnDeploy []string,
	isNew bool,
	isModified bool,
	s *styles.Styles,
	successStyle lipgloss.Style,
) {
	// Determine the indicator based on change type
	var indicator string
	var indicatorStyle lipgloss.Style
	if isNew {
		indicator = "+"
		indicatorStyle = successStyle
	} else if isModified {
		indicator = "±"
		indicatorStyle = s.Warning
	}

	// Export name with indicator
	sb.WriteString(indicatorStyle.Render(fmt.Sprintf("  %s %s", indicator, name)))
	sb.WriteString("\n")

	// Field path (source field)
	if change.FieldPath != "" {
		sb.WriteString(s.Muted.Render(fmt.Sprintf("    Field: %s", change.FieldPath)))
		sb.WriteString("\n")
	}

	// Check if value is computed at deploy time
	isComputedAtDeploy := isExportComputedAtDeploy(name, resolveOnDeploy)

	if isModified {
		// For modified exports, show previous and new values
		if change.PrevValue != nil {
			prevValueStr := headless.FormatMappingNode(change.PrevValue)
			sb.WriteString(s.Muted.Render(fmt.Sprintf("    Previous: %s", prevValueStr)))
			sb.WriteString("\n")
		}
		if isComputedAtDeploy {
			sb.WriteString(s.Muted.Render("    New: (known on deploy)"))
			sb.WriteString("\n")
		} else if change.NewValue != nil {
			newValueStr := headless.FormatMappingNode(change.NewValue)
			sb.WriteString(s.Muted.Render(fmt.Sprintf("    New: %s", newValueStr)))
			sb.WriteString("\n")
		}
	} else {
		// For new exports, show just the value
		if isComputedAtDeploy {
			sb.WriteString(s.Muted.Render("    Value: (known on deploy)"))
			sb.WriteString("\n")
		} else if change.NewValue != nil {
			newValueStr := headless.FormatMappingNode(change.NewValue)
			sb.WriteString(s.Muted.Render(fmt.Sprintf("    Value: %s", newValueStr)))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
}

// isExportComputedAtDeploy checks if the export value is computed at deploy time.
func isExportComputedAtDeploy(exportName string, resolveOnDeploy []string) bool {
	path := fmt.Sprintf("exports.%s", exportName)
	return slices.Contains(resolveOnDeploy, path)
}

// StageExportsFooterRenderer renders the footer for the stage exports view.
type StageExportsFooterRenderer struct{}

// Ensure StageExportsFooterRenderer implements splitpane.FooterRenderer.
var _ splitpane.FooterRenderer = (*StageExportsFooterRenderer)(nil)

// RenderFooter renders the exports view footer with navigation hints.
func (r *StageExportsFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	// Navigation help
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" navigate  "))
	sb.WriteString(s.Key.Render("tab"))
	sb.WriteString(s.Muted.Render(" switch pane  "))
	sb.WriteString(s.Key.Render("e"))
	sb.WriteString(s.Muted.Render("/"))
	sb.WriteString(s.Key.Render("esc"))
	sb.WriteString(s.Muted.Render(" close"))
	sb.WriteString("\n")

	return sb.String()
}

// StageExportsHeaderRenderer renders the header for the stage exports view.
type StageExportsHeaderRenderer struct {
	InstanceName string
}

// Ensure StageExportsHeaderRenderer implements splitpane.HeaderRenderer.
var _ splitpane.HeaderRenderer = (*StageExportsHeaderRenderer)(nil)

// RenderHeader renders the exports view header.
func (r *StageExportsHeaderRenderer) RenderHeader(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}

	title := "Export Changes"
	if r.InstanceName != "" {
		title = fmt.Sprintf("Export Changes: %s", r.InstanceName)
	}

	sb.WriteString(s.Header.Render(title))
	sb.WriteString("\n")

	return sb.String()
}
