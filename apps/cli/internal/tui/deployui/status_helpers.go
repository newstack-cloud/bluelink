package deployui

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// Status classification maps and functions for determining element state.
// These are used throughout the deploy UI for exit codes, skipping logic,
// and result collection.

// rollingBackOrFailedStatuses contains instance statuses that indicate
// a rollback is in progress or has completed/failed.
var rollingBackOrFailedStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployRollingBack:       true,
	core.InstanceStatusUpdateRollingBack:       true,
	core.InstanceStatusDestroyRollingBack:      true,
	core.InstanceStatusDeployRollbackFailed:    true,
	core.InstanceStatusUpdateRollbackFailed:    true,
	core.InstanceStatusDestroyRollbackFailed:   true,
	core.InstanceStatusDeployRollbackComplete:  true,
	core.InstanceStatusUpdateRollbackComplete:  true,
	core.InstanceStatusDestroyRollbackComplete: true,
}

// IsRollingBackOrFailedStatus returns true if the instance status indicates
// a rollback is in progress or has completed/failed.
// This is used to mark pending items as skipped in real-time.
func IsRollingBackOrFailedStatus(status core.InstanceStatus) bool {
	return rollingBackOrFailedStatuses[status]
}

// failedStatuses contains instance statuses that should result in a non-zero exit code.
// This includes both failed states and rollback complete states, since a rollback
// indicates the original operation failed (even though the rollback itself succeeded).
// For CI/CD pipelines, a rolled back deployment should not be considered a success.
var failedStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployFailed:  true,
	core.InstanceStatusUpdateFailed:  true,
	core.InstanceStatusDestroyFailed: true,
	// Rollback complete states - the rollback succeeded but the original operation failed
	core.InstanceStatusDeployRollbackComplete:  true,
	core.InstanceStatusUpdateRollbackComplete:  true,
	core.InstanceStatusDestroyRollbackComplete: true,
	// Rollback failed states - both the original operation and the rollback failed
	core.InstanceStatusDeployRollbackFailed:  true,
	core.InstanceStatusUpdateRollbackFailed:  true,
	core.InstanceStatusDestroyRollbackFailed: true,
}

// IsFailedStatus returns true if the instance status should result in a non-zero exit code.
// This includes failed operations and rollback complete states (since a rollback means
// the original operation failed). Used to determine exit code for CI/CD pipelines.
func IsFailedStatus(status core.InstanceStatus) bool {
	return failedStatuses[status]
}

// In-progress status helpers

var inProgressResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusCreating:    true,
	core.ResourceStatusUpdating:    true,
	core.ResourceStatusDestroying:  true,
	core.ResourceStatusRollingBack: true,
}

// IsInProgressResourceStatus returns true if the resource status indicates
// the resource is still being processed (not in a terminal state).
func IsInProgressResourceStatus(status core.ResourceStatus) bool {
	return inProgressResourceStatuses[status]
}

var inProgressInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeploying:           true,
	core.InstanceStatusUpdating:            true,
	core.InstanceStatusDestroying:          true,
	core.InstanceStatusDeployRollingBack:   true,
	core.InstanceStatusUpdateRollingBack:   true,
	core.InstanceStatusDestroyRollingBack:  true,
}

// IsInProgressInstanceStatus returns true if the instance status indicates
// the child blueprint is still being processed (not in a terminal state).
func IsInProgressInstanceStatus(status core.InstanceStatus) bool {
	return inProgressInstanceStatuses[status]
}

var inProgressLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusCreating:           true,
	core.LinkStatusUpdating:           true,
	core.LinkStatusDestroying:         true,
	core.LinkStatusCreateRollingBack:  true,
	core.LinkStatusUpdateRollingBack:  true,
	core.LinkStatusDestroyRollingBack: true,
}

// IsInProgressLinkStatus returns true if the link status indicates
// the link is still being processed (not in a terminal state).
func IsInProgressLinkStatus(status core.LinkStatus) bool {
	return inProgressLinkStatuses[status]
}

// Failed status helpers

var failedResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusCreateFailed:   true,
	core.ResourceStatusUpdateFailed:   true,
	core.ResourceStatusDestroyFailed:  true,
	core.ResourceStatusRollbackFailed: true,
}

// IsFailedResourceStatus returns true if the resource is in a failed state.
func IsFailedResourceStatus(status core.ResourceStatus) bool {
	return failedResourceStatuses[status]
}

var failedInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployFailed:          true,
	core.InstanceStatusUpdateFailed:          true,
	core.InstanceStatusDestroyFailed:         true,
	core.InstanceStatusDeployRollbackFailed:  true,
	core.InstanceStatusUpdateRollbackFailed:  true,
	core.InstanceStatusDestroyRollbackFailed: true,
}

// IsFailedInstanceStatus returns true if the child blueprint is in a failed state.
func IsFailedInstanceStatus(status core.InstanceStatus) bool {
	return failedInstanceStatuses[status]
}

var failedLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusCreateFailed:          true,
	core.LinkStatusUpdateFailed:          true,
	core.LinkStatusDestroyFailed:         true,
	core.LinkStatusCreateRollbackFailed:  true,
	core.LinkStatusUpdateRollbackFailed:  true,
	core.LinkStatusDestroyRollbackFailed: true,
}

// IsFailedLinkStatus returns true if the link is in a failed state.
func IsFailedLinkStatus(status core.LinkStatus) bool {
	return failedLinkStatuses[status]
}

// Interrupted status helpers

var interruptedResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusCreateInterrupted:  true,
	core.ResourceStatusUpdateInterrupted:  true,
	core.ResourceStatusDestroyInterrupted: true,
}

// IsInterruptedResourceStatus returns true if the resource was interrupted.
func IsInterruptedResourceStatus(status core.ResourceStatus) bool {
	return interruptedResourceStatuses[status]
}

var interruptedInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployInterrupted:  true,
	core.InstanceStatusUpdateInterrupted:  true,
	core.InstanceStatusDestroyInterrupted: true,
}

// IsInterruptedInstanceStatus returns true if the child blueprint was interrupted.
func IsInterruptedInstanceStatus(status core.InstanceStatus) bool {
	return interruptedInstanceStatuses[status]
}

var interruptedLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusCreateInterrupted:  true,
	core.LinkStatusUpdateInterrupted:  true,
	core.LinkStatusDestroyInterrupted: true,
}

// IsInterruptedLinkStatus returns true if the link was interrupted.
func IsInterruptedLinkStatus(status core.LinkStatus) bool {
	return interruptedLinkStatuses[status]
}

// Success status helpers

var successResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusCreated:          true,
	core.ResourceStatusUpdated:          true,
	core.ResourceStatusDestroyed:        true,
	core.ResourceStatusRollbackComplete: true,
}

// IsSuccessResourceStatus returns true if the resource completed successfully.
func IsSuccessResourceStatus(status core.ResourceStatus) bool {
	return successResourceStatuses[status]
}

var successInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployed:  true,
	core.InstanceStatusUpdated:   true,
	core.InstanceStatusDestroyed: true,
}

// IsSuccessInstanceStatus returns true if the child blueprint completed successfully.
func IsSuccessInstanceStatus(status core.InstanceStatus) bool {
	return successInstanceStatuses[status]
}

var successLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusCreated:   true,
	core.LinkStatusUpdated:   true,
	core.LinkStatusDestroyed: true,
}

// IsSuccessLinkStatus returns true if the link completed successfully.
func IsSuccessLinkStatus(status core.LinkStatus) bool {
	return successLinkStatuses[status]
}

// Status to action converters

var resourceStatusActions = map[core.ResourceStatus]string{
	core.ResourceStatusCreated:          "created",
	core.ResourceStatusUpdated:          "updated",
	core.ResourceStatusDestroyed:        "destroyed",
	core.ResourceStatusRollbackComplete: "rolled back",
}

// ResourceStatusToAction converts a resource status to a human-readable action string.
func ResourceStatusToAction(status core.ResourceStatus) string {
	if action, ok := resourceStatusActions[status]; ok {
		return action
	}
	return status.String()
}

var instanceStatusActions = map[core.InstanceStatus]string{
	core.InstanceStatusDeployed:  "deployed",
	core.InstanceStatusUpdated:   "updated",
	core.InstanceStatusDestroyed: "destroyed",
}

// InstanceStatusToAction converts an instance status to a human-readable action string.
func InstanceStatusToAction(status core.InstanceStatus) string {
	if action, ok := instanceStatusActions[status]; ok {
		return action
	}
	return status.String()
}

var linkStatusActions = map[core.LinkStatus]string{
	core.LinkStatusCreated:   "created",
	core.LinkStatusUpdated:   "updated",
	core.LinkStatusDestroyed: "destroyed",
}

// LinkStatusToAction converts a link status to a human-readable action string.
func LinkStatusToAction(status core.LinkStatus) string {
	if action, ok := linkStatusActions[status]; ok {
		return action
	}
	return status.String()
}
