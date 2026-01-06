package shared

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// Icon characters used across deploy and destroy UIs.
const (
	IconPending     = "○"
	IconInProgress  = "◐"
	IconSuccess     = "✓"
	IconFailed      = "✗"
	IconRollingBack = "↺"
	IconRollbackFailed   = "⚠"
	IconRollbackComplete = "⟲"
	IconInterrupted      = "⏹"
	IconSkipped          = "⊘"
	IconNoChange         = "─"
)

// ResourceStatusIcon returns an icon character for the given resource status.
// This handles all resource statuses (create, update, destroy).
func ResourceStatusIcon(status core.ResourceStatus) string {
	switch status {
	case core.ResourceStatusCreating, core.ResourceStatusUpdating, core.ResourceStatusDestroying:
		return IconInProgress
	case core.ResourceStatusCreated, core.ResourceStatusUpdated, core.ResourceStatusDestroyed:
		return IconSuccess
	case core.ResourceStatusCreateFailed, core.ResourceStatusUpdateFailed, core.ResourceStatusDestroyFailed:
		return IconFailed
	case core.ResourceStatusRollingBack:
		return IconRollingBack
	case core.ResourceStatusRollbackFailed:
		return IconRollbackFailed
	case core.ResourceStatusRollbackComplete:
		return IconRollbackComplete
	case core.ResourceStatusCreateInterrupted, core.ResourceStatusUpdateInterrupted, core.ResourceStatusDestroyInterrupted:
		return IconInterrupted
	default:
		return IconPending
	}
}

// InstanceStatusIcon returns an icon character for the given instance status.
// This handles all instance statuses (deploy, update, destroy).
func InstanceStatusIcon(status core.InstanceStatus) string {
	switch status {
	case core.InstanceStatusPreparing:
		return IconPending
	case core.InstanceStatusDeploying, core.InstanceStatusUpdating, core.InstanceStatusDestroying:
		return IconInProgress
	case core.InstanceStatusDeployed, core.InstanceStatusUpdated, core.InstanceStatusDestroyed:
		return IconSuccess
	case core.InstanceStatusDeployFailed, core.InstanceStatusUpdateFailed, core.InstanceStatusDestroyFailed:
		return IconFailed
	case core.InstanceStatusDeployRollingBack, core.InstanceStatusUpdateRollingBack, core.InstanceStatusDestroyRollingBack:
		return IconRollingBack
	case core.InstanceStatusDeployRollbackFailed, core.InstanceStatusUpdateRollbackFailed, core.InstanceStatusDestroyRollbackFailed:
		return IconRollbackFailed
	case core.InstanceStatusDeployRollbackComplete, core.InstanceStatusUpdateRollbackComplete, core.InstanceStatusDestroyRollbackComplete:
		return IconRollbackComplete
	case core.InstanceStatusDeployInterrupted, core.InstanceStatusUpdateInterrupted, core.InstanceStatusDestroyInterrupted:
		return IconInterrupted
	default:
		return IconPending
	}
}

// LinkStatusIcon returns an icon character for the given link status.
// This handles all link statuses (create, update, destroy).
func LinkStatusIcon(status core.LinkStatus) string {
	switch status {
	case core.LinkStatusCreating, core.LinkStatusUpdating, core.LinkStatusDestroying:
		return IconInProgress
	case core.LinkStatusCreated, core.LinkStatusUpdated, core.LinkStatusDestroyed:
		return IconSuccess
	case core.LinkStatusCreateFailed, core.LinkStatusUpdateFailed, core.LinkStatusDestroyFailed:
		return IconFailed
	case core.LinkStatusCreateRollingBack, core.LinkStatusUpdateRollingBack, core.LinkStatusDestroyRollingBack:
		return IconRollingBack
	case core.LinkStatusCreateRollbackFailed, core.LinkStatusUpdateRollbackFailed, core.LinkStatusDestroyRollbackFailed:
		return IconRollbackFailed
	case core.LinkStatusCreateRollbackComplete, core.LinkStatusUpdateRollbackComplete, core.LinkStatusDestroyRollbackComplete:
		return IconRollbackComplete
	case core.LinkStatusCreateInterrupted, core.LinkStatusUpdateInterrupted, core.LinkStatusDestroyInterrupted:
		return IconInterrupted
	default:
		return IconPending
	}
}

// StyleResourceIcon returns a styled icon string for the given resource status.
func StyleResourceIcon(icon string, status core.ResourceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.ResourceStatusCreating, core.ResourceStatusUpdating, core.ResourceStatusDestroying:
		return s.Info.Render(icon)
	case core.ResourceStatusCreated, core.ResourceStatusUpdated, core.ResourceStatusDestroyed:
		return successStyle.Render(icon)
	case core.ResourceStatusCreateFailed, core.ResourceStatusUpdateFailed, core.ResourceStatusDestroyFailed,
		core.ResourceStatusRollbackFailed:
		return s.Error.Render(icon)
	case core.ResourceStatusRollingBack:
		return s.Warning.Render(icon)
	case core.ResourceStatusRollbackComplete:
		return s.Muted.Render(icon)
	default:
		return s.Muted.Render(icon)
	}
}

// StyleInstanceIcon returns a styled icon string for the given instance status.
func StyleInstanceIcon(icon string, status core.InstanceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.InstanceStatusDeploying, core.InstanceStatusUpdating, core.InstanceStatusDestroying:
		return s.Info.Render(icon)
	case core.InstanceStatusDeployed, core.InstanceStatusUpdated, core.InstanceStatusDestroyed:
		return successStyle.Render(icon)
	case core.InstanceStatusDeployFailed, core.InstanceStatusUpdateFailed, core.InstanceStatusDestroyFailed,
		core.InstanceStatusDeployRollbackFailed, core.InstanceStatusUpdateRollbackFailed, core.InstanceStatusDestroyRollbackFailed:
		return s.Error.Render(icon)
	case core.InstanceStatusDeployRollingBack, core.InstanceStatusUpdateRollingBack, core.InstanceStatusDestroyRollingBack:
		return s.Warning.Render(icon)
	case core.InstanceStatusDeployRollbackComplete, core.InstanceStatusUpdateRollbackComplete, core.InstanceStatusDestroyRollbackComplete:
		return s.Muted.Render(icon)
	default:
		return s.Muted.Render(icon)
	}
}

// StyleLinkIcon returns a styled icon string for the given link status.
func StyleLinkIcon(icon string, status core.LinkStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.LinkStatusCreating, core.LinkStatusUpdating, core.LinkStatusDestroying:
		return s.Info.Render(icon)
	case core.LinkStatusCreated, core.LinkStatusUpdated, core.LinkStatusDestroyed:
		return successStyle.Render(icon)
	case core.LinkStatusCreateFailed, core.LinkStatusUpdateFailed, core.LinkStatusDestroyFailed,
		core.LinkStatusCreateRollbackFailed, core.LinkStatusUpdateRollbackFailed, core.LinkStatusDestroyRollbackFailed:
		return s.Error.Render(icon)
	case core.LinkStatusCreateRollingBack, core.LinkStatusUpdateRollingBack, core.LinkStatusDestroyRollingBack:
		return s.Warning.Render(icon)
	case core.LinkStatusCreateRollbackComplete, core.LinkStatusUpdateRollbackComplete, core.LinkStatusDestroyRollbackComplete:
		return s.Muted.Render(icon)
	default:
		return s.Muted.Render(icon)
	}
}
