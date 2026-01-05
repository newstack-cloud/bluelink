package destroyui

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// Status classification maps and functions for determining element state.

// rollingBackOrFailedStatuses contains instance statuses that indicate
// a rollback is in progress or has completed/failed.
var rollingBackOrFailedStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDestroyRollingBack:      true,
	core.InstanceStatusDestroyRollbackFailed:   true,
	core.InstanceStatusDestroyRollbackComplete: true,
}

// IsRollingBackOrFailedStatus returns true if the instance status indicates
// a rollback is in progress or has completed/failed.
func IsRollingBackOrFailedStatus(status core.InstanceStatus) bool {
	return rollingBackOrFailedStatuses[status]
}

// IsRollingBackStatus returns true if the instance is actively rolling back.
func IsRollingBackStatus(status core.InstanceStatus) bool {
	return status == core.InstanceStatusDestroyRollingBack
}

// failedStatuses contains instance statuses that should result in a non-zero exit code.
var failedStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDestroyFailed:           true,
	core.InstanceStatusDestroyRollbackComplete: true,
	core.InstanceStatusDestroyRollbackFailed:   true,
}

// IsFailedStatus returns true if the instance status should result in a non-zero exit code.
func IsFailedStatus(status core.InstanceStatus) bool {
	return failedStatuses[status]
}

// In-progress status helpers

var inProgressResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusDestroying:  true,
	core.ResourceStatusRollingBack: true,
}

// IsInProgressResourceStatus returns true if the resource status indicates
// the resource is still being processed (not in a terminal state).
func IsInProgressResourceStatus(status core.ResourceStatus) bool {
	return inProgressResourceStatuses[status]
}

var inProgressInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDestroying:         true,
	core.InstanceStatusDestroyRollingBack: true,
}

// IsInProgressInstanceStatus returns true if the instance status indicates
// the child blueprint is still being processed (not in a terminal state).
func IsInProgressInstanceStatus(status core.InstanceStatus) bool {
	return inProgressInstanceStatuses[status]
}

var inProgressLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusDestroying:         true,
	core.LinkStatusDestroyRollingBack: true,
}

// IsInProgressLinkStatus returns true if the link status indicates
// the link is still being processed (not in a terminal state).
func IsInProgressLinkStatus(status core.LinkStatus) bool {
	return inProgressLinkStatuses[status]
}

// Failed status helpers

var failedResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusDestroyFailed:  true,
	core.ResourceStatusRollbackFailed: true,
}

// IsFailedResourceStatus returns true if the resource is in a failed state.
func IsFailedResourceStatus(status core.ResourceStatus) bool {
	return failedResourceStatuses[status]
}

var failedInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDestroyFailed:         true,
	core.InstanceStatusDestroyRollbackFailed: true,
}

// IsFailedInstanceStatus returns true if the child blueprint is in a failed state.
func IsFailedInstanceStatus(status core.InstanceStatus) bool {
	return failedInstanceStatuses[status]
}

var failedLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusDestroyFailed:         true,
	core.LinkStatusDestroyRollbackFailed: true,
}

// IsFailedLinkStatus returns true if the link is in a failed state.
func IsFailedLinkStatus(status core.LinkStatus) bool {
	return failedLinkStatuses[status]
}

// Interrupted status helpers

var interruptedResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusDestroyInterrupted: true,
}

// IsInterruptedResourceStatus returns true if the resource was interrupted.
func IsInterruptedResourceStatus(status core.ResourceStatus) bool {
	return interruptedResourceStatuses[status]
}

var interruptedInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDestroyInterrupted: true,
}

// IsInterruptedInstanceStatus returns true if the child blueprint was interrupted.
func IsInterruptedInstanceStatus(status core.InstanceStatus) bool {
	return interruptedInstanceStatuses[status]
}

var interruptedLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusDestroyInterrupted: true,
}

// IsInterruptedLinkStatus returns true if the link was interrupted.
func IsInterruptedLinkStatus(status core.LinkStatus) bool {
	return interruptedLinkStatuses[status]
}

// Success status helpers

var successResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusDestroyed:        true,
	core.ResourceStatusRollbackComplete: true,
}

// IsSuccessResourceStatus returns true if the resource completed successfully.
func IsSuccessResourceStatus(status core.ResourceStatus) bool {
	return successResourceStatuses[status]
}

var successInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDestroyed: true,
}

// IsSuccessInstanceStatus returns true if the child blueprint completed successfully.
func IsSuccessInstanceStatus(status core.InstanceStatus) bool {
	return successInstanceStatuses[status]
}

var successLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusDestroyed: true,
}

// IsSuccessLinkStatus returns true if the link completed successfully.
func IsSuccessLinkStatus(status core.LinkStatus) bool {
	return successLinkStatuses[status]
}

// Status to action converters

var resourceStatusActions = map[core.ResourceStatus]string{
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
	core.LinkStatusDestroyed: "destroyed",
}

// LinkStatusToAction converts a link status to a human-readable action string.
func LinkStatusToAction(status core.LinkStatus) string {
	if action, ok := linkStatusActions[status]; ok {
		return action
	}
	return status.String()
}
