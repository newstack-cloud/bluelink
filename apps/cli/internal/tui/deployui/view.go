package deployui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
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
