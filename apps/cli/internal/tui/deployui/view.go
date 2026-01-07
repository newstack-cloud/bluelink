package deployui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
)

// Interactive error rendering methods

func (m DeployModel) renderError(err error) string {
	ctx := shared.DeployErrorContext()

	// Check for validation errors
	if clientErr, isValidation := engineerrors.IsValidationError(err); isValidation {
		return shared.RenderValidationError(clientErr, ctx, m.styles)
	}

	// Check for stream errors
	if streamErr, ok := err.(*engineerrors.StreamError); ok {
		return shared.RenderStreamError(streamErr, ctx, m.styles)
	}

	// Generic error display
	return shared.RenderGenericError(err, "Deployment failed", m.styles)
}

func (m DeployModel) renderDestroyChangesetError() string {
	return shared.RenderChangesetTypeMismatchError(shared.ChangesetTypeMismatchParams{
		IsDestroyChangeset: true,
		InstanceName:       m.instanceName,
		ChangesetID:        m.changesetID,
	}, m.styles)
}

// overviewFooterHeight returns the height of the fixed footer in overview view.
func overviewFooterHeight() int {
	// Footer consists of: separator line + empty line + key hints line + empty line
	return 4
}

// renderOverviewView renders a full-screen deployment summary view.
// This is shown when the user presses 'o' after deployment completes.
// Uses a scrollable viewport for the content with a fixed footer.
func (m DeployModel) renderOverviewView() string {
	sb := strings.Builder{}
	sb.WriteString(m.overviewViewport.View())
	sb.WriteString("\n")
	shared.RenderViewportOverlayFooter(&sb, "o", m.styles)
	return sb.String()
}

// specViewFooterHeight returns the height of the fixed footer in spec view.
func specViewFooterHeight() int {
	// Footer consists of: separator line + empty line + key hints line + empty line
	return 4
}

// renderSpecView renders a full-screen spec view for the currently selected resource.
// This is shown when the user presses 's' while a resource is selected.
func (m DeployModel) renderSpecView() string {
	sb := strings.Builder{}
	sb.WriteString(m.specViewport.View())
	sb.WriteString("\n")
	shared.RenderViewportOverlayFooter(&sb, "s", m.styles)
	return sb.String()
}

// renderSpecContent renders the full spec for a resource (excluding computed fields).
func (m DeployModel) renderSpecContent(resourceState *state.ResourceState, resourceName string) string {
	sb := strings.Builder{}

	// Header
	sb.WriteString("\n")
	sb.WriteString(m.styles.Header.Render("  Resource Spec: " + resourceName))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  " + strings.Repeat("─", 60)))
	sb.WriteString("\n\n")

	if resourceState == nil || resourceState.SpecData == nil {
		sb.WriteString(m.styles.Muted.Render("  No spec data available"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Collect and render non-computed fields with pretty-printed JSON
	fields := outpututil.CollectNonComputedFieldsPretty(resourceState.SpecData, resourceState.ComputedFields)
	if len(fields) == 0 {
		sb.WriteString(m.styles.Muted.Render("  No spec fields available"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Render each field with formatted values
	for _, field := range fields {
		sb.WriteString(m.styles.Category.Render("  " + field.Name + ":"))
		sb.WriteString("\n")

		// Format the value with indentation
		formattedValue := formatSpecValue(field.Value)
		sb.WriteString(formattedValue)
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatSpecValue formats a spec field value with proper indentation.
func formatSpecValue(value string) string {
	// Use headless FormatMappingNode style - just add indentation
	lines := strings.Split(value, "\n")
	sb := strings.Builder{}
	for i, line := range lines {
		sb.WriteString("    " + line)
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// getSelectedResourceState returns the resource state for the currently selected resource.
// It uses the same lookup logic as the renderer to find state from multiple sources.
func (m DeployModel) getSelectedResourceState() (*state.ResourceState, string) {
	selectedItem := m.splitPane.SelectedItem()
	if selectedItem == nil {
		return nil, ""
	}

	deployItem, ok := selectedItem.(*DeployItem)
	if !ok || deployItem.Type != ItemTypeResource || deployItem.Resource == nil {
		return nil, ""
	}

	resourceName := deployItem.Resource.Name
	path := deployItem.Path

	// Try post-deploy state first (uses path to traverse child blueprints)
	if m.postDeployInstanceState != nil {
		if resourceState := findResourceStateByPath(m.postDeployInstanceState, path, resourceName); resourceState != nil {
			return resourceState, resourceName
		}
	}

	// Try pre-deploy state for items with no changes
	if m.preDeployInstanceState != nil {
		if resourceState := findResourceStateByPath(m.preDeployInstanceState, path, resourceName); resourceState != nil {
			return resourceState, resourceName
		}
	}

	// Try the resource state field directly (populated when building items)
	if deployItem.Resource.ResourceState != nil {
		return deployItem.Resource.ResourceState, resourceName
	}

	// Fall back to changeset state
	if deployItem.Resource.Changes != nil &&
		deployItem.Resource.Changes.AppliedResourceInfo.CurrentResourceState != nil {
		return deployItem.Resource.Changes.AppliedResourceInfo.CurrentResourceState, resourceName
	}

	return nil, resourceName
}

// renderOverviewContent renders the scrollable content for the deployment overview viewport.
// This includes the header, instance info, successful operations, failures, and interruptions.
func (m DeployModel) renderOverviewContent() string {
	sb := strings.Builder{}
	contentWidth := m.width - 4 // Leave margin for padding

	// Header
	sb.WriteString("\n")
	sb.WriteString(m.styles.Header.Render("  Deployment Summary"))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  " + strings.Repeat("─", 60)))
	sb.WriteString("\n\n")

	// Instance info
	sb.WriteString(m.styles.Muted.Render("  Instance ID: "))
	sb.WriteString(m.styles.Selected.Render(m.instanceID))
	sb.WriteString("\n")
	if m.instanceName != "" {
		sb.WriteString(m.styles.Muted.Render("  Instance Name: "))
		sb.WriteString(m.styles.Selected.Render(m.instanceName))
		sb.WriteString("\n")
	}
	sb.WriteString(m.styles.Muted.Render("  Status: "))
	sb.WriteString(m.renderFinalStatusBadge())
	sb.WriteString("\n")

	// Durations (from post-deploy instance state)
	m.renderOverviewDurations(&sb)

	sb.WriteString("\n")

	// Render successful operations first
	m.renderSuccessfulElements(&sb)

	// Render structured element failures with root cause details
	m.renderElementFailuresWithWrapping(&sb, contentWidth)

	// Render interrupted elements in a separate section
	m.renderInterruptedElementsWithPath(&sb)

	// Render skipped rollback items if any
	m.renderSkippedRollbackItems(&sb, contentWidth)

	return sb.String()
}

// renderOverviewDurations renders the deployment duration information.
func (m DeployModel) renderOverviewDurations(sb *strings.Builder) {
	if m.postDeployInstanceState == nil {
		return
	}

	durations := m.postDeployInstanceState.Durations
	if durations == nil {
		return
	}

	hasDurations := false

	if durations.PrepareDuration != nil && *durations.PrepareDuration > 0 {
		if !hasDurations {
			sb.WriteString("\n")
			hasDurations = true
		}
		sb.WriteString(m.styles.Muted.Render("  Prepare Duration: "))
		sb.WriteString(outpututil.FormatDuration(*durations.PrepareDuration))
		sb.WriteString("\n")
	}

	if durations.TotalDuration != nil && *durations.TotalDuration > 0 {
		if !hasDurations {
			sb.WriteString("\n")
		}
		sb.WriteString(m.styles.Muted.Render("  Total Duration: "))
		sb.WriteString(outpututil.FormatDuration(*durations.TotalDuration))
		sb.WriteString("\n")
	}
}

// renderFinalStatusBadge returns a styled badge for the final deployment status.
func (m DeployModel) renderFinalStatusBadge() string {
	switch m.finalStatus {
	case core.InstanceStatusDeployed, core.InstanceStatusUpdated, core.InstanceStatusDestroyed:
		successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())
		return successStyle.Render(m.finalStatus.String())
	case core.InstanceStatusDeployFailed, core.InstanceStatusUpdateFailed, core.InstanceStatusDestroyFailed:
		return m.styles.Error.Render(m.finalStatus.String())
	case core.InstanceStatusDeployRollbackComplete, core.InstanceStatusUpdateRollbackComplete, core.InstanceStatusDestroyRollbackComplete:
		return m.styles.Warning.Render(m.finalStatus.String())
	default:
		return m.styles.Muted.Render(m.finalStatus.String())
	}
}

// renderSuccessfulElements renders the successful operations section.
func (m DeployModel) renderSuccessfulElements(sb *strings.Builder) {
	if len(m.successfulElements) == 0 {
		return
	}

	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())
	elementLabel := sdkstrings.Pluralize(len(m.successfulElements), "Operation", "Operations")
	sb.WriteString(successStyle.Render(fmt.Sprintf("  %d Successful %s:", len(m.successfulElements), elementLabel)))
	sb.WriteString("\n\n")

	for _, elem := range m.successfulElements {
		sb.WriteString(successStyle.Render("  ✓ "))
		sb.WriteString(m.styles.Selected.Render(elem.ElementPath))
		if elem.Action != "" {
			sb.WriteString(m.styles.Muted.Render(" (" + elem.Action + ")"))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

// renderElementFailuresWithWrapping renders element failures with text wrapping
// and full element paths for the scrollable error details view.
func (m DeployModel) renderElementFailuresWithWrapping(sb *strings.Builder, contentWidth int) {
	shared.RenderElementFailures(sb, m.elementFailures, contentWidth, false, m.styles)
}

// renderInterruptedElementsWithPath renders interrupted elements with full paths.
func (m DeployModel) renderInterruptedElementsWithPath(sb *strings.Builder) {
	if len(m.interruptedElements) == 0 {
		return
	}

	elementLabel := sdkstrings.Pluralize(len(m.interruptedElements), "Element", "Elements")
	sb.WriteString(m.styles.Warning.Render(fmt.Sprintf("  %d %s Interrupted:", len(m.interruptedElements), elementLabel)))
	sb.WriteString("\n\n")

	for _, elem := range m.interruptedElements {
		sb.WriteString(m.styles.Warning.Render("  ⏹ "))
		sb.WriteString(m.styles.Selected.Render(elem.ElementPath))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("    These elements were interrupted and their state is unknown."))
	sb.WriteString("\n")
}

// renderSkippedRollbackItems renders items that were skipped during rollback.
func (m DeployModel) renderSkippedRollbackItems(sb *strings.Builder, contentWidth int) {
	if len(m.skippedRollbackItems) == 0 {
		return
	}

	itemLabel := sdkstrings.Pluralize(len(m.skippedRollbackItems), "Item", "Items")
	sb.WriteString(m.styles.Warning.Render(fmt.Sprintf("  %d %s Skipped During Rollback:", len(m.skippedRollbackItems), itemLabel)))
	sb.WriteString("\n\n")

	reasonWidth := contentWidth - 8

	for _, item := range m.skippedRollbackItems {
		itemPath := item.Name
		if item.ChildPath != "" {
			itemPath = item.ChildPath + "." + item.Name
		}
		sb.WriteString(m.styles.Warning.Render("  ⚠ "))
		sb.WriteString(m.styles.Selected.Render(itemPath))
		sb.WriteString(m.styles.Muted.Render(fmt.Sprintf(" (%s)", item.Type)))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Muted.Render(fmt.Sprintf("      Status: %s", item.Status)))
		sb.WriteString("\n")

		wrappedLines := outpututil.WrapTextLines(item.Reason, reasonWidth)
		for i, line := range wrappedLines {
			if i == 0 {
				sb.WriteString(m.styles.Muted.Render("      Reason: "))
			} else {
				sb.WriteString("              ")
			}
			sb.WriteString(m.styles.Muted.Render(line))
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("    These items were not in a safe state to rollback and were excluded."))
	sb.WriteString("\n")
}

// preRollbackStateFooterHeight returns the height of the fixed footer in pre-rollback state view.
func preRollbackStateFooterHeight() int {
	return 4
}

// renderPreRollbackStateView renders a full-screen pre-rollback state view.
func (m DeployModel) renderPreRollbackStateView() string {
	sb := strings.Builder{}
	sb.WriteString(m.preRollbackStateViewport.View())
	sb.WriteString("\n")
	shared.RenderViewportOverlayFooter(&sb, "r", m.styles)
	return sb.String()
}

// renderPreRollbackStateContent renders the scrollable content for the pre-rollback state viewport.
func (m DeployModel) renderPreRollbackStateContent() string {
	if m.preRollbackState == nil {
		return m.styles.Muted.Render("  No pre-rollback state available")
	}

	sb := strings.Builder{}
	data := m.preRollbackState

	m.renderPreRollbackHeader(&sb, data)
	m.renderPreRollbackFailures(&sb, data)
	m.renderPreRollbackSections(&sb, data)

	return sb.String()
}

func (m DeployModel) renderPreRollbackHeader(sb *strings.Builder, data *container.PreRollbackStateMessage) {
	sb.WriteString("\n")
	sb.WriteString(m.styles.Header.Render("  Pre-Rollback State"))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  " + strings.Repeat("─", 60)))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Instance ID: "))
	sb.WriteString(m.styles.Selected.Render(data.InstanceID))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  Instance Name: "))
	sb.WriteString(m.styles.Selected.Render(data.InstanceName))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  Status: "))
	sb.WriteString(m.styles.Error.Render(data.Status.String()))
	sb.WriteString("\n\n")
}

func (m DeployModel) renderPreRollbackFailures(sb *strings.Builder, data *container.PreRollbackStateMessage) {
	if len(data.FailureReasons) == 0 {
		return
	}

	sb.WriteString(m.styles.Error.Render("  Failure Reasons:"))
	sb.WriteString("\n")

	wrapWidth := m.preRollbackStateViewport.Width - 8
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	for _, reason := range data.FailureReasons {
		wrappedLines := outpututil.WrapTextLines(reason, wrapWidth)
		for i, line := range wrappedLines {
			if i == 0 {
				sb.WriteString(m.styles.Error.Render("    • " + line))
			} else {
				sb.WriteString(m.styles.Error.Render("      " + line))
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")
}

func (m DeployModel) renderPreRollbackSections(sb *strings.Builder, data *container.PreRollbackStateMessage) {
	if len(data.Resources) > 0 {
		sb.WriteString(m.styles.Category.Render(fmt.Sprintf("  Resources (%d):", len(data.Resources))))
		sb.WriteString("\n\n")
		for _, r := range data.Resources {
			m.renderResourceSnapshot(sb, &r, "    ")
		}
	}

	if len(data.Links) > 0 {
		sb.WriteString(m.styles.Category.Render(fmt.Sprintf("  Links (%d):", len(data.Links))))
		sb.WriteString("\n\n")
		for _, l := range data.Links {
			m.renderLinkSnapshot(sb, &l, "    ")
		}
	}

	if len(data.Children) > 0 {
		sb.WriteString(m.styles.Category.Render(fmt.Sprintf("  Children (%d):", len(data.Children))))
		sb.WriteString("\n\n")
		for _, c := range data.Children {
			m.renderChildSnapshot(sb, &c, "    ")
		}
	}
}

// renderResourceSnapshot renders a resource snapshot in the pre-rollback state view.
func (m DeployModel) renderResourceSnapshot(sb *strings.Builder, r *container.ResourceSnapshot, indent string) {
	statusStyle := m.getResourceStatusStyle(r.Status)

	sb.WriteString(indent)
	sb.WriteString(m.styles.Selected.Render(r.ResourceName))
	sb.WriteString("\n")

	sb.WriteString(indent)
	sb.WriteString(m.styles.Muted.Render("  Type: "))
	sb.WriteString(r.ResourceType)
	sb.WriteString("\n")

	sb.WriteString(indent)
	sb.WriteString(m.styles.Muted.Render("  Status: "))
	sb.WriteString(statusStyle.Render(r.Status.String()))
	sb.WriteString("\n")

	if len(r.FailureReasons) > 0 {
		wrapWidth := max(m.preRollbackStateViewport.Width-len(indent)-6, 20)
		for _, reason := range r.FailureReasons {
			wrappedLines := outpututil.WrapTextLines(reason, wrapWidth)
			for i, line := range wrappedLines {
				sb.WriteString(indent)
				if i == 0 {
					sb.WriteString(m.styles.Error.Render("  • " + line))
				} else {
					sb.WriteString(m.styles.Error.Render("    " + line))
				}
				sb.WriteString("\n")
			}
		}
	}

	// Render outputs section if spec data is available
	if r.SpecData != nil {
		fields := outpututil.CollectOutputFields(r.SpecData, r.ComputedFields)
		if len(fields) > 0 {
			sb.WriteString(indent)
			sb.WriteString(m.styles.Category.Render("  Outputs:"))
			sb.WriteString("\n")
			for _, field := range fields {
				sb.WriteString(indent)
				sb.WriteString(m.styles.Muted.Render(fmt.Sprintf("    %s: %s", field.Name, field.Value)))
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString("\n")
}

// renderLinkSnapshot renders a link snapshot in the pre-rollback state view.
func (m DeployModel) renderLinkSnapshot(sb *strings.Builder, l *container.LinkSnapshot, indent string) {
	statusStyle := m.getLinkStatusStyle(l.Status)

	sb.WriteString(indent)
	sb.WriteString(m.styles.Selected.Render(l.LinkName))
	sb.WriteString("\n")

	sb.WriteString(indent)
	sb.WriteString(m.styles.Muted.Render("  Status: "))
	sb.WriteString(statusStyle.Render(l.Status.String()))
	sb.WriteString("\n")

	if len(l.FailureReasons) > 0 {
		wrapWidth := m.preRollbackStateViewport.Width - len(indent) - 6
		if wrapWidth < 20 {
			wrapWidth = 20
		}
		for _, reason := range l.FailureReasons {
			wrappedLines := outpututil.WrapTextLines(reason, wrapWidth)
			for i, line := range wrappedLines {
				sb.WriteString(indent)
				if i == 0 {
					sb.WriteString(m.styles.Error.Render("  • " + line))
				} else {
					sb.WriteString(m.styles.Error.Render("    " + line))
				}
				sb.WriteString("\n")
			}
		}
	}
	sb.WriteString("\n")
}

// renderChildSnapshot renders a child blueprint snapshot in the pre-rollback state view.
func (m DeployModel) renderChildSnapshot(sb *strings.Builder, c *container.ChildSnapshot, indent string) {
	statusStyle := m.getInstanceStatusStyle(c.Status)

	sb.WriteString(indent)
	sb.WriteString(m.styles.Selected.Render(c.ChildName))
	sb.WriteString("\n")

	sb.WriteString(indent)
	sb.WriteString(m.styles.Muted.Render("  Status: "))
	sb.WriteString(statusStyle.Render(c.Status.String()))
	sb.WriteString("\n")

	if len(c.FailureReasons) > 0 {
		for _, reason := range c.FailureReasons {
			sb.WriteString(indent)
			sb.WriteString(m.styles.Error.Render("  • " + reason))
			sb.WriteString("\n")
		}
	}

	// Nested resources
	if len(c.Resources) > 0 {
		sb.WriteString(indent)
		sb.WriteString(m.styles.Muted.Render(fmt.Sprintf("  Resources (%d):", len(c.Resources))))
		sb.WriteString("\n")
		for _, r := range c.Resources {
			m.renderResourceSnapshot(sb, &r, indent+"    ")
		}
	}

	// Nested links
	if len(c.Links) > 0 {
		sb.WriteString(indent)
		sb.WriteString(m.styles.Muted.Render(fmt.Sprintf("  Links (%d):", len(c.Links))))
		sb.WriteString("\n")
		for _, l := range c.Links {
			m.renderLinkSnapshot(sb, &l, indent+"    ")
		}
	}

	// Nested children (recursive)
	if len(c.Children) > 0 {
		sb.WriteString(indent)
		sb.WriteString(m.styles.Muted.Render(fmt.Sprintf("  Children (%d):", len(c.Children))))
		sb.WriteString("\n")
		for _, nested := range c.Children {
			m.renderChildSnapshot(sb, &nested, indent+"    ")
		}
	}

	sb.WriteString("\n")
}

// getResourceStatusStyle returns the appropriate style for a resource status.
func (m DeployModel) getResourceStatusStyle(status core.ResourceStatus) lipgloss.Style {
	switch {
	case IsFailedResourceStatus(status):
		return m.styles.Error
	case IsInterruptedResourceStatus(status):
		return m.styles.Warning
	case IsSuccessResourceStatus(status):
		return lipgloss.NewStyle().Foreground(m.styles.Palette.Success())
	default:
		return m.styles.Muted
	}
}

// getLinkStatusStyle returns the appropriate style for a link status.
func (m DeployModel) getLinkStatusStyle(status core.LinkStatus) lipgloss.Style {
	switch {
	case IsFailedLinkStatus(status):
		return m.styles.Error
	case IsInterruptedLinkStatus(status):
		return m.styles.Warning
	case IsSuccessLinkStatus(status):
		return lipgloss.NewStyle().Foreground(m.styles.Palette.Success())
	default:
		return m.styles.Muted
	}
}

// getInstanceStatusStyle returns the appropriate style for an instance status.
func (m DeployModel) getInstanceStatusStyle(status core.InstanceStatus) lipgloss.Style {
	switch {
	case IsFailedInstanceStatus(status):
		return m.styles.Error
	case IsInterruptedInstanceStatus(status):
		return m.styles.Warning
	case IsSuccessInstanceStatus(status):
		return lipgloss.NewStyle().Foreground(m.styles.Palette.Success())
	default:
		return m.styles.Muted
	}
}
