package shared

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// RenderResourceStatus renders a styled resource status string.
func RenderResourceStatus(status core.ResourceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.ResourceStatusCreating:
		return s.Info.Render("Creating")
	case core.ResourceStatusCreated:
		return successStyle.Render("Created")
	case core.ResourceStatusCreateFailed:
		return s.Error.Render("Create Failed")
	case core.ResourceStatusUpdating:
		return s.Info.Render("Updating")
	case core.ResourceStatusUpdated:
		return successStyle.Render("Updated")
	case core.ResourceStatusUpdateFailed:
		return s.Error.Render("Update Failed")
	case core.ResourceStatusDestroying:
		return s.Info.Render("Destroying")
	case core.ResourceStatusDestroyed:
		return successStyle.Render("Destroyed")
	case core.ResourceStatusDestroyFailed:
		return s.Error.Render("Destroy Failed")
	case core.ResourceStatusRollingBack:
		return s.Warning.Render("Rolling Back")
	case core.ResourceStatusRollbackFailed:
		return s.Error.Render("Rollback Failed")
	case core.ResourceStatusRollbackComplete:
		return s.Muted.Render("Rolled Back")
	case core.ResourceStatusCreateInterrupted,
		core.ResourceStatusUpdateInterrupted,
		core.ResourceStatusDestroyInterrupted:
		return s.Warning.Render("Interrupted")
	default:
		return s.Muted.Render("Unknown")
	}
}

// RenderInstanceStatus renders a styled instance status string.
func RenderInstanceStatus(status core.InstanceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.InstanceStatusPreparing:
		return s.Muted.Render("Preparing")
	case core.InstanceStatusDeploying:
		return s.Info.Render("Deploying")
	case core.InstanceStatusDeployed:
		return successStyle.Render("Deployed")
	case core.InstanceStatusDeployFailed:
		return s.Error.Render("Deploy Failed")
	case core.InstanceStatusUpdating:
		return s.Info.Render("Updating")
	case core.InstanceStatusUpdated:
		return successStyle.Render("Updated")
	case core.InstanceStatusUpdateFailed:
		return s.Error.Render("Update Failed")
	case core.InstanceStatusDestroying:
		return s.Info.Render("Destroying")
	case core.InstanceStatusDestroyed:
		return successStyle.Render("Destroyed")
	case core.InstanceStatusDestroyFailed:
		return s.Error.Render("Destroy Failed")
	case core.InstanceStatusDeployRollingBack:
		return s.Warning.Render("Rolling Back Deploy")
	case core.InstanceStatusDeployRollbackFailed:
		return s.Error.Render("Deploy Rollback Failed")
	case core.InstanceStatusDeployRollbackComplete:
		return s.Muted.Render("Deploy Rolled Back")
	case core.InstanceStatusUpdateRollingBack:
		return s.Warning.Render("Rolling Back Update")
	case core.InstanceStatusUpdateRollbackFailed:
		return s.Error.Render("Update Rollback Failed")
	case core.InstanceStatusUpdateRollbackComplete:
		return s.Muted.Render("Update Rolled Back")
	case core.InstanceStatusDestroyRollingBack:
		return s.Warning.Render("Rolling Back Destroy")
	case core.InstanceStatusDestroyRollbackFailed:
		return s.Error.Render("Destroy Rollback Failed")
	case core.InstanceStatusDestroyRollbackComplete:
		return s.Muted.Render("Destroy Rolled Back")
	case core.InstanceStatusNotDeployed:
		return s.Muted.Render("Not Deployed")
	case core.InstanceStatusDeployInterrupted,
		core.InstanceStatusUpdateInterrupted,
		core.InstanceStatusDestroyInterrupted:
		return s.Warning.Render("Interrupted")
	default:
		return s.Muted.Render("Unknown")
	}
}

// RenderLinkStatus renders a styled link status string.
func RenderLinkStatus(status core.LinkStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.LinkStatusCreating:
		return s.Info.Render("Creating")
	case core.LinkStatusCreated:
		return successStyle.Render("Created")
	case core.LinkStatusCreateFailed:
		return s.Error.Render("Create Failed")
	case core.LinkStatusUpdating:
		return s.Info.Render("Updating")
	case core.LinkStatusUpdated:
		return successStyle.Render("Updated")
	case core.LinkStatusUpdateFailed:
		return s.Error.Render("Update Failed")
	case core.LinkStatusDestroying:
		return s.Info.Render("Destroying")
	case core.LinkStatusDestroyed:
		return successStyle.Render("Destroyed")
	case core.LinkStatusDestroyFailed:
		return s.Error.Render("Destroy Failed")
	case core.LinkStatusCreateRollingBack:
		return s.Warning.Render("Rolling Back Create")
	case core.LinkStatusCreateRollbackFailed:
		return s.Error.Render("Create Rollback Failed")
	case core.LinkStatusCreateRollbackComplete:
		return s.Muted.Render("Create Rolled Back")
	case core.LinkStatusUpdateRollingBack:
		return s.Warning.Render("Rolling Back Update")
	case core.LinkStatusUpdateRollbackFailed:
		return s.Error.Render("Update Rollback Failed")
	case core.LinkStatusUpdateRollbackComplete:
		return s.Muted.Render("Update Rolled Back")
	case core.LinkStatusDestroyRollingBack:
		return s.Warning.Render("Rolling Back Destroy")
	case core.LinkStatusDestroyRollbackFailed:
		return s.Error.Render("Destroy Rollback Failed")
	case core.LinkStatusDestroyRollbackComplete:
		return s.Muted.Render("Destroy Rolled Back")
	case core.LinkStatusCreateInterrupted,
		core.LinkStatusUpdateInterrupted,
		core.LinkStatusDestroyInterrupted:
		return s.Warning.Render("Interrupted")
	default:
		return s.Muted.Render("Unknown")
	}
}

// FormatLinkStatus returns a lowercase human-readable status string for a link.
func FormatLinkStatus(status core.LinkStatus) string {
	switch status {
	case core.LinkStatusCreated:
		return "created"
	case core.LinkStatusUpdated:
		return "updated"
	case core.LinkStatusDestroyed:
		return "destroyed"
	case core.LinkStatusCreating:
		return "creating"
	case core.LinkStatusUpdating:
		return "updating"
	case core.LinkStatusDestroying:
		return "destroying"
	case core.LinkStatusCreateFailed, core.LinkStatusUpdateFailed, core.LinkStatusDestroyFailed:
		return "failed"
	default:
		return status.String()
	}
}

// FormatPreciseResourceStatus returns a human-readable string for a precise resource status.
func FormatPreciseResourceStatus(status core.PreciseResourceStatus) string {
	switch status {
	case core.PreciseResourceStatusCreating:
		return "Creating resource..."
	case core.PreciseResourceStatusConfigComplete:
		return "Configuration applied, waiting for stability"
	case core.PreciseResourceStatusCreated:
		return "Resource created and stable"
	case core.PreciseResourceStatusCreateFailed:
		return "Failed to create resource"
	case core.PreciseResourceStatusUpdating:
		return "Updating resource..."
	case core.PreciseResourceStatusUpdateConfigComplete:
		return "Update applied, waiting for stability"
	case core.PreciseResourceStatusUpdated:
		return "Resource updated and stable"
	case core.PreciseResourceStatusUpdateFailed:
		return "Failed to update resource"
	case core.PreciseResourceStatusDestroying:
		return "Destroying resource..."
	case core.PreciseResourceStatusDestroyed:
		return "Resource destroyed"
	case core.PreciseResourceStatusDestroyFailed:
		return "Failed to destroy resource"
	case core.PreciseResourceStatusCreateRollingBack:
		return "Rolling back resource creation..."
	case core.PreciseResourceStatusCreateRollbackFailed:
		return "Failed to roll back resource creation"
	case core.PreciseResourceStatusCreateRollbackComplete:
		return "Resource creation rolled back"
	case core.PreciseResourceStatusUpdateRollingBack:
		return "Rolling back resource update..."
	case core.PreciseResourceStatusUpdateRollbackFailed:
		return "Failed to roll back resource update"
	case core.PreciseResourceStatusUpdateRollbackConfigComplete:
		return "Update rollback applied, waiting for stability"
	case core.PreciseResourceStatusUpdateRollbackComplete:
		return "Resource update rolled back"
	case core.PreciseResourceStatusDestroyRollingBack:
		return "Rolling back resource destruction..."
	case core.PreciseResourceStatusDestroyRollbackFailed:
		return "Failed to roll back resource destruction"
	case core.PreciseResourceStatusDestroyRollbackConfigComplete:
		return "Destruction rollback applied, waiting for stability"
	case core.PreciseResourceStatusDestroyRollbackComplete:
		return "Resource destruction rolled back"
	case core.PreciseResourceStatusCreateInterrupted:
		return "Resource creation was interrupted (actual state unknown)"
	case core.PreciseResourceStatusUpdateInterrupted:
		return "Resource update was interrupted (actual state unknown)"
	case core.PreciseResourceStatusDestroyInterrupted:
		return "Resource destruction was interrupted (actual state unknown)"
	default:
		return "Pending"
	}
}

// FormatPreciseLinkStatus returns a human-readable string for a precise link status.
func FormatPreciseLinkStatus(status core.PreciseLinkStatus) string {
	switch status {
	case core.PreciseLinkStatusUpdatingResourceA:
		return "Updating resource A..."
	case core.PreciseLinkStatusResourceAUpdated:
		return "Resource A updated"
	case core.PreciseLinkStatusResourceAUpdateFailed:
		return "Failed to update resource A"
	case core.PreciseLinkStatusResourceAUpdateRollingBack:
		return "Rolling back resource A update..."
	case core.PreciseLinkStatusResourceAUpdateRollbackFailed:
		return "Failed to roll back resource A update"
	case core.PreciseLinkStatusResourceAUpdateRollbackComplete:
		return "Resource A update rolled back"
	case core.PreciseLinkStatusUpdatingResourceB:
		return "Updating resource B..."
	case core.PreciseLinkStatusResourceBUpdated:
		return "Resource B updated"
	case core.PreciseLinkStatusResourceBUpdateFailed:
		return "Failed to update resource B"
	case core.PreciseLinkStatusResourceBUpdateRollingBack:
		return "Rolling back resource B update..."
	case core.PreciseLinkStatusResourceBUpdateRollbackFailed:
		return "Failed to roll back resource B update"
	case core.PreciseLinkStatusResourceBUpdateRollbackComplete:
		return "Resource B update rolled back"
	case core.PreciseLinkStatusUpdatingIntermediaryResources:
		return "Updating intermediary resources..."
	case core.PreciseLinkStatusIntermediaryResourcesUpdated:
		return "Intermediary resources updated"
	case core.PreciseLinkStatusIntermediaryResourceUpdateFailed:
		return "Failed to update intermediary resources"
	case core.PreciseLinkStatusIntermediaryResourceUpdateRollingBack:
		return "Rolling back intermediary resources..."
	case core.PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed:
		return "Failed to roll back intermediary resources"
	case core.PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete:
		return "Intermediary resources rolled back"
	case core.PreciseLinkStatusResourceAUpdateInterrupted:
		return "Resource A update was interrupted (actual state unknown)"
	case core.PreciseLinkStatusResourceBUpdateInterrupted:
		return "Resource B update was interrupted (actual state unknown)"
	case core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted:
		return "Intermediary resource update was interrupted (actual state unknown)"
	default:
		return "Pending"
	}
}
