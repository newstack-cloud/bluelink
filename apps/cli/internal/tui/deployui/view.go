package deployui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// Headless rendering methods

func (m *DeployModel) printHeadlessHeader() {
	w := m.printer.Writer()
	w.Println("Starting deployment...")
	w.Printf("Instance ID: %s\n", m.instanceID)
	if m.instanceName != "" {
		w.Printf("Instance Name: %s\n", m.instanceName)
	}
	w.Printf("Changeset: %s\n", m.changesetID)
	w.DoubleSeparator(72)
	w.PrintlnEmpty()
}

func (m *DeployModel) printHeadlessResourceEvent(data *container.ResourceDeployUpdateMessage) {
	statusIcon := resourceStatusHeadlessIcon(data.Status)
	statusText := resourceStatusHeadlessText(data.Status)
	m.printer.ProgressItem(statusIcon, "resource", data.ResourceName, statusText, "")
}

func resourceStatusHeadlessIcon(status core.ResourceStatus) string {
	switch status {
	case core.ResourceStatusCreating, core.ResourceStatusUpdating, core.ResourceStatusDestroying:
		return "..."
	case core.ResourceStatusCreated, core.ResourceStatusUpdated, core.ResourceStatusDestroyed:
		return "OK"
	case core.ResourceStatusCreateFailed, core.ResourceStatusUpdateFailed, core.ResourceStatusDestroyFailed:
		return "ERR"
	case core.ResourceStatusRollingBack:
		return "<-"
	case core.ResourceStatusRollbackFailed:
		return "!!"
	case core.ResourceStatusRollbackComplete:
		return "<>"
	default:
		return "  "
	}
}

func resourceStatusHeadlessText(status core.ResourceStatus) string {
	switch status {
	case core.ResourceStatusCreating:
		return "creating"
	case core.ResourceStatusCreated:
		return "created"
	case core.ResourceStatusCreateFailed:
		return "create failed"
	case core.ResourceStatusUpdating:
		return "updating"
	case core.ResourceStatusUpdated:
		return "updated"
	case core.ResourceStatusUpdateFailed:
		return "update failed"
	case core.ResourceStatusDestroying:
		return "destroying"
	case core.ResourceStatusDestroyed:
		return "destroyed"
	case core.ResourceStatusDestroyFailed:
		return "destroy failed"
	case core.ResourceStatusRollingBack:
		return "rolling back"
	case core.ResourceStatusRollbackFailed:
		return "rollback failed"
	case core.ResourceStatusRollbackComplete:
		return "rolled back"
	default:
		return "pending"
	}
}

func (m *DeployModel) printHeadlessChildEvent(data *container.ChildDeployUpdateMessage) {
	statusIcon := instanceStatusHeadlessIcon(data.Status)
	statusText := instanceStatusHeadlessText(data.Status)
	m.printer.ProgressItem(statusIcon, "child", data.ChildName, statusText, "")
}

func instanceStatusHeadlessIcon(status core.InstanceStatus) string {
	switch status {
	case core.InstanceStatusPreparing:
		return "  "
	case core.InstanceStatusDeploying, core.InstanceStatusUpdating, core.InstanceStatusDestroying:
		return "..."
	case core.InstanceStatusDeployed, core.InstanceStatusUpdated, core.InstanceStatusDestroyed:
		return "OK"
	case core.InstanceStatusDeployFailed, core.InstanceStatusUpdateFailed, core.InstanceStatusDestroyFailed:
		return "ERR"
	case core.InstanceStatusDeployRollingBack, core.InstanceStatusUpdateRollingBack, core.InstanceStatusDestroyRollingBack:
		return "<-"
	case core.InstanceStatusDeployRollbackFailed, core.InstanceStatusUpdateRollbackFailed, core.InstanceStatusDestroyRollbackFailed:
		return "!!"
	case core.InstanceStatusDeployRollbackComplete, core.InstanceStatusUpdateRollbackComplete, core.InstanceStatusDestroyRollbackComplete:
		return "<>"
	default:
		return "  "
	}
}

func instanceStatusHeadlessText(status core.InstanceStatus) string {
	switch status {
	case core.InstanceStatusPreparing:
		return "preparing"
	case core.InstanceStatusDeploying:
		return "deploying"
	case core.InstanceStatusDeployed:
		return "deployed"
	case core.InstanceStatusDeployFailed:
		return "deploy failed"
	case core.InstanceStatusUpdating:
		return "updating"
	case core.InstanceStatusUpdated:
		return "updated"
	case core.InstanceStatusUpdateFailed:
		return "update failed"
	case core.InstanceStatusDestroying:
		return "destroying"
	case core.InstanceStatusDestroyed:
		return "destroyed"
	case core.InstanceStatusDestroyFailed:
		return "destroy failed"
	case core.InstanceStatusDeployRollingBack:
		return "rolling back deploy"
	case core.InstanceStatusDeployRollbackFailed:
		return "deploy rollback failed"
	case core.InstanceStatusDeployRollbackComplete:
		return "deploy rolled back"
	case core.InstanceStatusUpdateRollingBack:
		return "rolling back update"
	case core.InstanceStatusUpdateRollbackFailed:
		return "update rollback failed"
	case core.InstanceStatusUpdateRollbackComplete:
		return "update rolled back"
	case core.InstanceStatusDestroyRollingBack:
		return "rolling back destroy"
	case core.InstanceStatusDestroyRollbackFailed:
		return "destroy rollback failed"
	case core.InstanceStatusDestroyRollbackComplete:
		return "destroy rolled back"
	case core.InstanceStatusNotDeployed:
		return "not deployed"
	default:
		return "unknown"
	}
}

func (m *DeployModel) printHeadlessLinkEvent(data *container.LinkDeployUpdateMessage) {
	statusIcon := linkStatusHeadlessIcon(data.Status)
	statusText := linkStatusHeadlessText(data.Status)
	m.printer.ProgressItem(statusIcon, "link", data.LinkName, statusText, "")
}

func linkStatusHeadlessIcon(status core.LinkStatus) string {
	switch status {
	case core.LinkStatusCreating, core.LinkStatusUpdating, core.LinkStatusDestroying:
		return "..."
	case core.LinkStatusCreated, core.LinkStatusUpdated, core.LinkStatusDestroyed:
		return "OK"
	case core.LinkStatusCreateFailed, core.LinkStatusUpdateFailed, core.LinkStatusDestroyFailed:
		return "ERR"
	case core.LinkStatusCreateRollingBack, core.LinkStatusUpdateRollingBack, core.LinkStatusDestroyRollingBack:
		return "<-"
	case core.LinkStatusCreateRollbackFailed, core.LinkStatusUpdateRollbackFailed, core.LinkStatusDestroyRollbackFailed:
		return "!!"
	case core.LinkStatusCreateRollbackComplete, core.LinkStatusUpdateRollbackComplete, core.LinkStatusDestroyRollbackComplete:
		return "<>"
	default:
		return "  "
	}
}

func linkStatusHeadlessText(status core.LinkStatus) string {
	switch status {
	case core.LinkStatusCreating:
		return "creating"
	case core.LinkStatusCreated:
		return "created"
	case core.LinkStatusCreateFailed:
		return "create failed"
	case core.LinkStatusUpdating:
		return "updating"
	case core.LinkStatusUpdated:
		return "updated"
	case core.LinkStatusUpdateFailed:
		return "update failed"
	case core.LinkStatusDestroying:
		return "destroying"
	case core.LinkStatusDestroyed:
		return "destroyed"
	case core.LinkStatusDestroyFailed:
		return "destroy failed"
	case core.LinkStatusCreateRollingBack:
		return "rolling back create"
	case core.LinkStatusCreateRollbackFailed:
		return "create rollback failed"
	case core.LinkStatusCreateRollbackComplete:
		return "create rolled back"
	case core.LinkStatusUpdateRollingBack:
		return "rolling back update"
	case core.LinkStatusUpdateRollbackFailed:
		return "update rollback failed"
	case core.LinkStatusUpdateRollbackComplete:
		return "update rolled back"
	case core.LinkStatusDestroyRollingBack:
		return "rolling back destroy"
	case core.LinkStatusDestroyRollbackFailed:
		return "destroy rollback failed"
	case core.LinkStatusDestroyRollbackComplete:
		return "destroy rolled back"
	default:
		return "pending"
	}
}

func (m *DeployModel) printHeadlessSummary() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.DoubleSeparator(72)

	statusText := instanceStatusHeadlessText(m.finalStatus)
	w.Printf("Deployment %s\n", statusText)
	w.DoubleSeparator(72)
	w.PrintlnEmpty()

	// Count completed items
	resourceCount := len(m.resourcesByName)
	childCount := len(m.childrenByName)
	linkCount := len(m.linksByName)

	w.Printf("Complete: %d %s, %d %s, %d %s\n",
		resourceCount, sdkstrings.Pluralize(resourceCount, "resource", "resources"),
		childCount, sdkstrings.Pluralize(childCount, "child", "children"),
		linkCount, sdkstrings.Pluralize(linkCount, "link", "links"))
	w.PrintlnEmpty()

	w.Printf("Instance ID: %s\n", m.instanceID)
	if m.instanceName != "" {
		w.Printf("Instance Name: %s\n", m.instanceName)
	}
	w.PrintlnEmpty()
}

func (m *DeployModel) printHeadlessError(err error) {
	w := m.printer.Writer()
	w.PrintlnEmpty()

	// Check for validation errors
	if clientErr, isValidation := engineerrors.IsValidationError(err); isValidation {
		m.printHeadlessValidationError(clientErr)
		return
	}

	// Check for stream errors
	if streamErr, ok := err.(*engineerrors.StreamError); ok {
		m.printHeadlessStreamError(streamErr)
		return
	}

	// Generic error
	w.Println("ERR Deployment failed")
	w.PrintlnEmpty()
	w.Printf("  Error: %s\n", err.Error())
}

func (m *DeployModel) printHeadlessValidationError(clientErr *engineerrors.ClientError) {
	w := m.printer.Writer()
	w.Println("ERR Failed to start deployment")
	w.PrintlnEmpty()
	w.Println("The following issues must be resolved before deployment can proceed:")
	w.PrintlnEmpty()

	if len(clientErr.ValidationErrors) > 0 {
		w.Println("Validation Errors:")
		w.SingleSeparator(72)
		for _, valErr := range clientErr.ValidationErrors {
			location := valErr.Location
			if location == "" {
				location = "unknown"
			}
			w.Printf("  - %s: %s\n", location, valErr.Message)
		}
		w.PrintlnEmpty()
	}

	if len(clientErr.ValidationDiagnostics) > 0 {
		w.Println("Blueprint Diagnostics:")
		w.SingleSeparator(72)
		for _, diag := range clientErr.ValidationDiagnostics {
			m.printHeadlessDiagnostic(diag)
		}
		w.PrintlnEmpty()
	}

	if len(clientErr.ValidationErrors) == 0 && len(clientErr.ValidationDiagnostics) == 0 {
		w.Printf("  %s\n", clientErr.Message)
	}
}

func (m *DeployModel) printHeadlessStreamError(streamErr *engineerrors.StreamError) {
	w := m.printer.Writer()
	w.Println("ERR Error during deployment")
	w.PrintlnEmpty()
	w.Println("The following issues occurred during deployment:")
	w.PrintlnEmpty()
	w.Printf("  %s\n", streamErr.Event.Message)
	w.PrintlnEmpty()

	if len(streamErr.Event.Diagnostics) > 0 {
		w.Println("Diagnostics:")
		w.SingleSeparator(72)
		for _, diag := range streamErr.Event.Diagnostics {
			m.printHeadlessDiagnostic(diag)
		}
		w.PrintlnEmpty()
	}
}

func (m *DeployModel) printHeadlessDiagnostic(diag *core.Diagnostic) {
	w := m.printer.Writer()

	levelName := "INFO"
	switch diag.Level {
	case core.DiagnosticLevelError:
		levelName = "ERROR"
	case core.DiagnosticLevelWarning:
		levelName = "WARNING"
	}

	line, col := 0, 0
	if diag.Range != nil {
		line = diag.Range.Start.Line
		col = diag.Range.Start.Column
	}

	if line > 0 {
		w.Printf("  [%s] line %d, col %d: %s\n", levelName, line, col, diag.Message)
	} else {
		w.Printf("  [%s] %s\n", levelName, diag.Message)
	}
}

// Interactive error rendering methods

func (m DeployModel) renderError(err error) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	// Check for validation errors
	if clientErr, isValidation := engineerrors.IsValidationError(err); isValidation {
		return m.renderValidationError(clientErr)
	}

	// Check for stream errors
	if streamErr, ok := err.(*engineerrors.StreamError); ok {
		return m.renderStreamError(streamErr)
	}

	// Generic error display
	sb.WriteString(m.styles.Error.Render("  ✗ Deployment failed\n\n"))
	sb.WriteString(m.styles.Error.Render(fmt.Sprintf("    %s\n", err.Error())))
	sb.WriteString(m.renderErrorFooter())
	return sb.String()
}

func (m DeployModel) renderValidationError(clientErr *engineerrors.ClientError) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Error.Render("  ✗ Failed to start deployment\n\n"))

	sb.WriteString(m.styles.Muted.Render("  The following issues must be resolved before deployment can proceed:\n\n"))

	if len(clientErr.ValidationErrors) > 0 {
		sb.WriteString(m.styles.Category.Render("  Validation Errors:\n"))
		for _, valErr := range clientErr.ValidationErrors {
			location := valErr.Location
			if location == "" {
				location = "unknown"
			}
			sb.WriteString(m.styles.Error.Render(fmt.Sprintf("    • %s: ", location)))
			sb.WriteString(fmt.Sprintf("%s\n", valErr.Message))
		}
		sb.WriteString("\n")
	}

	if len(clientErr.ValidationDiagnostics) > 0 {
		sb.WriteString(m.styles.Category.Render("  Blueprint Diagnostics:\n"))
		for _, diag := range clientErr.ValidationDiagnostics {
			sb.WriteString(m.renderDiagnostic(diag))
		}
		sb.WriteString("\n")
	}

	if len(clientErr.ValidationErrors) == 0 && len(clientErr.ValidationDiagnostics) == 0 {
		sb.WriteString(m.styles.Error.Render(fmt.Sprintf("    %s\n", clientErr.Message)))
	}

	sb.WriteString(m.renderErrorFooter())
	return sb.String()
}

func (m DeployModel) renderStreamError(streamErr *engineerrors.StreamError) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Error.Render("  ✗ Error during deployment\n\n"))

	sb.WriteString(m.styles.Muted.Render("  The following issues occurred during deployment:\n\n"))
	sb.WriteString(fmt.Sprintf("    %s\n\n", streamErr.Event.Message))

	if len(streamErr.Event.Diagnostics) > 0 {
		sb.WriteString(m.styles.Category.Render("  Diagnostics:\n"))
		for _, diag := range streamErr.Event.Diagnostics {
			sb.WriteString(m.renderDiagnostic(diag))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(m.renderErrorFooter())
	return sb.String()
}

func (m DeployModel) renderDiagnostic(diag *core.Diagnostic) string {
	sb := strings.Builder{}

	var levelStyle lipgloss.Style
	levelName := "unknown"
	switch diag.Level {
	case core.DiagnosticLevelError:
		levelStyle = m.styles.Error
		levelName = "ERROR"
	case core.DiagnosticLevelWarning:
		levelStyle = m.styles.Warning
		levelName = "WARNING"
	case core.DiagnosticLevelInfo:
		levelStyle = m.styles.Info
		levelName = "INFO"
	default:
		levelStyle = m.styles.Muted
	}

	sb.WriteString("    ")
	sb.WriteString(levelStyle.Render(levelName))
	if diag.Range != nil && diag.Range.Start.Line > 0 {
		sb.WriteString(m.styles.Muted.Render(fmt.Sprintf(" [line %d, col %d]", diag.Range.Start.Line, diag.Range.Start.Column)))
	}
	sb.WriteString(": ")
	sb.WriteString(diag.Message)
	sb.WriteString("\n")

	return sb.String()
}

func (m DeployModel) renderErrorFooter() string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	keyStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Primary()).Bold(true)
	sb.WriteString(m.styles.Muted.Render("  Press "))
	sb.WriteString(keyStyle.Render("q"))
	sb.WriteString(m.styles.Muted.Render(" to quit"))
	sb.WriteString("\n")
	return sb.String()
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

	// Scrollable viewport content
	sb.WriteString(m.overviewViewport.View())
	sb.WriteString("\n")

	// Fixed footer with navigation help
	sb.WriteString(m.styles.Muted.Render("  " + strings.Repeat("─", 60)))
	sb.WriteString("\n")
	keyStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Primary()).Bold(true)
	sb.WriteString(m.styles.Muted.Render("  Press "))
	sb.WriteString(keyStyle.Render("↑/↓"))
	sb.WriteString(m.styles.Muted.Render(" to scroll  "))
	sb.WriteString(keyStyle.Render("esc"))
	sb.WriteString(m.styles.Muted.Render("/"))
	sb.WriteString(keyStyle.Render("o"))
	sb.WriteString(m.styles.Muted.Render(" to return  "))
	sb.WriteString(keyStyle.Render("q"))
	sb.WriteString(m.styles.Muted.Render(" to quit"))
	sb.WriteString("\n")

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

	// Scrollable viewport content
	sb.WriteString(m.specViewport.View())
	sb.WriteString("\n")

	// Fixed footer with navigation help
	sb.WriteString(m.styles.Muted.Render("  " + strings.Repeat("─", 60)))
	sb.WriteString("\n")
	keyStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Primary()).Bold(true)
	sb.WriteString(m.styles.Muted.Render("  Press "))
	sb.WriteString(keyStyle.Render("↑/↓"))
	sb.WriteString(m.styles.Muted.Render(" to scroll  "))
	sb.WriteString(keyStyle.Render("esc"))
	sb.WriteString(m.styles.Muted.Render("/"))
	sb.WriteString(keyStyle.Render("s"))
	sb.WriteString(m.styles.Muted.Render(" to return  "))
	sb.WriteString(keyStyle.Render("q"))
	sb.WriteString(m.styles.Muted.Render(" to quit"))
	sb.WriteString("\n")

	return sb.String()
}

// renderSpecContent renders the full spec for a resource (excluding computed fields).
func (m DeployModel) renderSpecContent(resourceState *state.ResourceState, resourceName string) string {
	sb := strings.Builder{}
	contentWidth := m.width - 4 // Leave margin for padding

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
		formattedValue := formatSpecValue(field.Value, contentWidth-4)
		sb.WriteString(formattedValue)
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatSpecValue formats a spec field value with proper indentation.
func formatSpecValue(value string, width int) string {
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
	if len(m.elementFailures) == 0 {
		return
	}

	failureLabel := sdkstrings.Pluralize(len(m.elementFailures), "Failure", "Failures")
	sb.WriteString(m.styles.Error.Render(fmt.Sprintf("  %d %s:", len(m.elementFailures), failureLabel)))
	sb.WriteString("\n\n")

	// Calculate available width for failure reasons (after indent)
	reasonWidth := contentWidth - 8 // 6 chars indent + 2 chars bullet

	for _, failure := range m.elementFailures {
		sb.WriteString(m.styles.Error.Render("  ✗ "))
		sb.WriteString(m.styles.Selected.Render(failure.ElementPath))
		sb.WriteString("\n")
		renderFailureReasons(sb, failure.FailureReasons, reasonWidth, m.styles)
		sb.WriteString("\n")
	}
}

func renderFailureReasons(sb *strings.Builder, reasons []string, width int, styles *stylespkg.Styles) {
	for _, reason := range reasons {
		wrappedLines := wrapText(reason, width)
		for i, line := range wrappedLines {
			sb.WriteString("      ")
			if i == 0 {
				sb.WriteString(styles.Error.Render("• "))
			} else {
				sb.WriteString("  ")
			}
			sb.WriteString(styles.Error.Render(line))
			sb.WriteString("\n")
		}
	}
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

// wrapText wraps a string to fit within the specified width.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)

	return lines
}

// printHeadlessDriftDetected prints drift information in headless mode.
func (m *DeployModel) printHeadlessDriftDetected() {
	printer := driftui.NewHeadlessDriftPrinter(m.printer, m.driftContext)
	printer.PrintDriftDetected(m.driftResult)
}
