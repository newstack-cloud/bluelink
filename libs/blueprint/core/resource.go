package core

// ResourceStatus is used to represent the current state of a resource
// in a blueprint instance.
type ResourceStatus int

const (
	// ResourceStatusUnknown is used when we can't
	// determine an accurate status for a resource.
	ResourceStatusUnknown ResourceStatus = iota
	// ResourceStatusCreating is used when
	// an initial resource deployment is currently in progress.
	ResourceStatusCreating
	// ResourceStatusCreated is used when
	// a resource has been deployed
	// successfully.
	ResourceStatusCreated
	// ResourceStatusCreateFailed is used when
	// the first creation of a resource failed.
	ResourceStatusCreateFailed
	// ResourceStatusDestroying is used when
	// a resource is in the process of being destroyed.
	ResourceStatusDestroying
	// ResourceStatusDestroyed is used when
	// a resource has been destroyed.
	ResourceStatusDestroyed
	// ResourceStatusDestroyFailed is used when
	// the destruction of a resource fails.
	ResourceStatusDestroyFailed
	// ResourceStatusUpdating is used when
	// a resource is being updated.
	ResourceStatusUpdating
	// ResourceStatusUpdated is used when a resource
	// has been successfully updated.
	ResourceStatusUpdated
	// ResourceStatusUpdateFailed is used when a resource
	// has failed to update.
	ResourceStatusUpdateFailed
	// ResourceStatusRollingBack is used when
	// another change in the same blueprint has failed
	// and the latest change involving the current resource
	// is being rolled back.
	ResourceStatusRollingBack
	// ResourceStatusRollbackFailed is used when
	// another change in the same blueprint has failed
	// and the latest change involving the current resource
	// could not be rolled back.
	ResourceStatusRollbackFailed
	// ResourceStatusRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the latest change involving the current resource
	// has been rolled back.
	ResourceStatusRollbackComplete
	// ResourceStatusCreateInterrupted is used when
	// a resource creation was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state of the resource is unknown and needs reconciliation.
	ResourceStatusCreateInterrupted
	// ResourceStatusUpdateInterrupted is used when
	// a resource update was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state of the resource is unknown and needs reconciliation.
	ResourceStatusUpdateInterrupted
	// ResourceStatusDestroyInterrupted is used when
	// a resource destruction was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state of the resource is unknown and needs reconciliation.
	ResourceStatusDestroyInterrupted
	// ResourceStatusRetained is used when a resource has been removed
	// from the blueprint's managed state without destroying the underlying
	// infrastructure in the provider, as a result of the resource having
	// a removal policy of "retain".
	ResourceStatusRetained
)

var resourceStatusStrings = map[ResourceStatus]string{
	ResourceStatusUnknown:            "UNKNOWN",
	ResourceStatusCreating:           "CREATING",
	ResourceStatusCreated:            "CREATED",
	ResourceStatusCreateFailed:       "CREATE FAILED",
	ResourceStatusDestroying:         "DESTROYING",
	ResourceStatusDestroyed:          "DESTROYED",
	ResourceStatusDestroyFailed:      "DESTROY FAILED",
	ResourceStatusUpdating:           "UPDATING",
	ResourceStatusUpdated:            "UPDATED",
	ResourceStatusUpdateFailed:       "UPDATE FAILED",
	ResourceStatusRollingBack:        "ROLLING BACK",
	ResourceStatusRollbackFailed:     "ROLLBACK FAILED",
	ResourceStatusRollbackComplete:   "ROLLBACK COMPLETE",
	ResourceStatusCreateInterrupted:  "CREATE INTERRUPTED",
	ResourceStatusUpdateInterrupted:  "UPDATE INTERRUPTED",
	ResourceStatusDestroyInterrupted: "DESTROY INTERRUPTED",
	ResourceStatusRetained:           "RETAINED",
}

func (s ResourceStatus) String() string {
	str, ok := resourceStatusStrings[s]
	if !ok {
		return "UNKNOWN"
	}
	return str
}

// PreciseResourceStatus is used to represent a more precise
// current state of a resource in a blueprint instance.
// This is used to allow the container "engine" to be more efficient
// in deploying a blueprint, by avoiding blocking on resource finalisation
// that isn't always needed to be able to successfully deploy the resources
// that are dependent on the resource in question.
//
// Most of these statuses are both reported on the deployment event stream and recorded as
// the resource's status in blueprint instance state. The link contribution statuses are
// reported on the event stream only, and are never persisted in instance state, since the
// update they describe is not a deployment of the resource that the blueprint asked for.
// IsLinkContributionStatus reports which is which.
type PreciseResourceStatus int

const (
	// PreciseResourceStatusUnknown is used when we can't
	// determine an accurate status for a resource.
	PreciseResourceStatusUnknown PreciseResourceStatus = iota
	// PreciseResourceStatusCreating is used when
	// an initial resource deployment is currently in progress.
	PreciseResourceStatusCreating
	// PreciseResourceStatusConfigComplete is used when
	// a resource has been configured successfully.
	// What this means is that the resource has been created
	// but is not yet in a stable state.
	// For example, an application in a container orchestration service
	// has been created but is not yet up and running.
	PreciseResourceStatusConfigComplete
	// ResourceStatusCreated is used when
	// a resource has been deployed
	// successfully.
	// This is used when a resource is in a stable state.
	PreciseResourceStatusCreated
	// ResourceStatusCreateFailed is used when
	// the first creation of a resource failed.
	PreciseResourceStatusCreateFailed
	// PreciseResourceStatusCreateRollingBack is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// created is being rolled back.
	PreciseResourceStatusCreateRollingBack
	// PreciseResourceStatusCreateRollbackFailed is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// created could not be rolled back.
	PreciseResourceStatusCreateRollbackFailed
	// PreciseResourceStatusCreateRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// created has been rolled back.
	PreciseResourceStatusCreateRollbackComplete
	// ResourceStatusDestroying is used when
	// a resource is in the process of being destroyed.
	PreciseResourceStatusDestroying
	// ResourceStatusDestroyed is used when
	// a resource has been destroyed.
	PreciseResourceStatusDestroyed
	// ResourceStatusDestroyFailed is used when
	// the destruction of a resource fails.
	PreciseResourceStatusDestroyFailed
	// PreciseResourceStatusDestroyRollingBack is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// removed is being rolled back.
	// This involves recreating the resource from the previous state.
	PreciseResourceStatusDestroyRollingBack
	// PreciseResourceStatusDestroyRollbackFailed is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// removed could not be rolled back (recreated).
	PreciseResourceStatusDestroyRollbackFailed
	// PreciseResourceStatusDestroyRollbackConfigComplete is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// removed has been rolled back (recreated)
	// but is not yet in a stable state.
	PreciseResourceStatusDestroyRollbackConfigComplete
	// PreciseResourceStatusDestroyRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// removed has been rolled back (recreated).
	PreciseResourceStatusDestroyRollbackComplete
	// ResourceStatusUpdating is used when
	// a resource is being updated.
	PreciseResourceStatusUpdating
	// PreciseResourceStatusUpdateConfigComplete is used when
	// a resource being updated has been configured successfully.
	// What this means is that the resource has been updated
	// but is not yet in a stable state.
	// For example, an application in a container orchestration service
	// has been updated but the new version is not yet up and running.
	PreciseResourceStatusUpdateConfigComplete
	// ResourceStatusUpdated is used when a resource
	// has been sucessfully updated.
	PreciseResourceStatusUpdated
	// ResourceStatusUpdateFailed is used when a resource
	// has failed to update.
	PreciseResourceStatusUpdateFailed
	// PreciseResourceStatusUpdateRollingBack is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// updated is being rolled back to the previous state.
	PreciseResourceStatusUpdateRollingBack
	// PreciseResourceStatusUpdateRollbackFailed is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// updated could not be rolled back.
	PreciseResourceStatusUpdateRollbackFailed
	// PreciseResourceStatusUpdateRollbackConfigComplete is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// updated has been rolled back to the previous state
	// but is not yet in a stable state.
	PreciseResourceStatusUpdateRollbackConfigComplete
	// PreciseResourceStatusUpdateRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the current resource that was successfully
	// updated has been rolled back to the previous state.
	PreciseResourceStatusUpdateRollbackComplete
	// PreciseResourceStatusCreateInterrupted is used when
	// a resource creation was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state of the resource is unknown and needs reconciliation.
	PreciseResourceStatusCreateInterrupted
	// PreciseResourceStatusUpdateInterrupted is used when
	// a resource update was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state of the resource is unknown and needs reconciliation.
	PreciseResourceStatusUpdateInterrupted
	// PreciseResourceStatusDestroyInterrupted is used when
	// a resource destruction was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state of the resource is unknown and needs reconciliation.
	PreciseResourceStatusDestroyInterrupted
	// PreciseResourceStatusRetained is used when a resource has been removed
	// from the blueprint's managed state without destroying the underlying
	// infrastructure in the provider, as a result of the resource having
	// a removal policy of "retain".
	PreciseResourceStatusRetained
	// PreciseResourceStatusUpdatingLinkContributions is used when a resource is being
	// updated to carry what the links that contribute to it need its spec to say, rather
	// than because anything the blueprint declares about it changed.
	//
	// A resource can reach this without appearing in the change set at all: an execution
	// role several links write to is ordinarily untouched by a deployment that changes
	// one of those links.
	//
	// This and the two statuses that follow it are reported on the deployment event stream
	// and never recorded as the resource's status in instance state. A resource's recorded
	// status is that of its own deployment, and an update carrying link contributions is
	// not one, it is a write the framework makes on behalf of the links that need it,
	// which leaves the resource exactly as deployed prior to the link contributions.
	PreciseResourceStatusUpdatingLinkContributions
	// PreciseResourceStatusLinkContributionsUpdated is used when a resource has been
	// updated with what the links contributing to it need its spec to say.
	PreciseResourceStatusLinkContributionsUpdated
	// PreciseResourceStatusLinkContributionsUpdateFailed is used when a resource could
	// not be updated with what the links contributing to it need its spec to say.
	PreciseResourceStatusLinkContributionsUpdateFailed
)

// IsLinkContributionStatus reports whether the status describes an update carrying what the
// links contributing to a resource need its spec to say.
//
// These statuses are reported on the deployment event stream and are never recorded as a
// resource's status in blueprint instance state, so a status enum documenting what instance
// state can hold excludes them, and one documenting the event stream does not.
func (s PreciseResourceStatus) IsLinkContributionStatus() bool {
	return s == PreciseResourceStatusUpdatingLinkContributions ||
		s == PreciseResourceStatusLinkContributionsUpdated ||
		s == PreciseResourceStatusLinkContributionsUpdateFailed
}

var preciseResourceStatusStrings = map[PreciseResourceStatus]string{
	PreciseResourceStatusUnknown:                       "UNKNOWN",
	PreciseResourceStatusUpdatingLinkContributions:     "UPDATING LINK CONTRIBUTIONS",
	PreciseResourceStatusLinkContributionsUpdated:      "LINK CONTRIBUTIONS UPDATED",
	PreciseResourceStatusLinkContributionsUpdateFailed: "LINK CONTRIBUTIONS UPDATE FAILED",
	PreciseResourceStatusCreating:                      "CREATING",
	PreciseResourceStatusConfigComplete:                "CONFIG COMPLETE",
	PreciseResourceStatusCreated:                       "CREATED",
	PreciseResourceStatusCreateFailed:                  "CREATE FAILED",
	PreciseResourceStatusCreateRollingBack:             "CREATE ROLLING BACK",
	PreciseResourceStatusCreateRollbackFailed:          "CREATE ROLLBACK FAILED",
	PreciseResourceStatusCreateRollbackComplete:        "CREATE ROLLBACK COMPLETE",
	PreciseResourceStatusDestroying:                    "DESTROYING",
	PreciseResourceStatusDestroyed:                     "DESTROYED",
	PreciseResourceStatusDestroyFailed:                 "DESTROY FAILED",
	PreciseResourceStatusDestroyRollingBack:            "DESTROY ROLLING BACK",
	PreciseResourceStatusDestroyRollbackFailed:         "DESTROY ROLLBACK FAILED",
	PreciseResourceStatusDestroyRollbackConfigComplete: "DESTROY ROLLBACK CONFIG COMPLETE",
	PreciseResourceStatusDestroyRollbackComplete:       "DESTROY ROLLBACK COMPLETE",
	PreciseResourceStatusUpdating:                      "UPDATING",
	PreciseResourceStatusUpdateConfigComplete:          "UPDATE CONFIG COMPLETE",
	PreciseResourceStatusUpdated:                       "UPDATED",
	PreciseResourceStatusUpdateFailed:                  "UPDATE FAILED",
	PreciseResourceStatusUpdateRollingBack:             "UPDATE ROLLING BACK",
	PreciseResourceStatusUpdateRollbackFailed:          "UPDATE ROLLBACK FAILED",
	PreciseResourceStatusUpdateRollbackConfigComplete:  "UPDATE ROLLBACK CONFIG COMPLETE",
	PreciseResourceStatusUpdateRollbackComplete:        "UPDATE ROLLBACK COMPLETE",
	PreciseResourceStatusCreateInterrupted:             "CREATE INTERRUPTED",
	PreciseResourceStatusUpdateInterrupted:             "UPDATE INTERRUPTED",
	PreciseResourceStatusDestroyInterrupted:            "DESTROY INTERRUPTED",
	PreciseResourceStatusRetained:                      "RETAINED",
}

func (s PreciseResourceStatus) String() string {
	str, ok := preciseResourceStatusStrings[s]
	if !ok {
		return "UNKNOWN"
	}
	return str
}

// ResourceStatusIsUnsuccessfulCreate returns true if the resource status indicates
// that a resource creation was attempted but never completed successfully.
// This is used during change staging to determine if a resource should be treated
// as "new" (requiring creation) rather than existing (requiring update).
func ResourceStatusIsUnsuccessfulCreate(status ResourceStatus) bool {
	return status == ResourceStatusCreateFailed ||
		status == ResourceStatusCreateInterrupted ||
		status == ResourceStatusCreating
}

// ResourceStatusIsSafeToRollback returns true if the resource is in a state where
// rolling back changes is safe and expected to succeed.
// Safe states are those where the resource operation completed successfully:
// - Created: resource was successfully created (can be destroyed in rollback)
// - Updated: resource was successfully updated (can be reverted in rollback)
// - Destroyed: resource was successfully destroyed (can be recreated in rollback)
func ResourceStatusIsSafeToRollback(status ResourceStatus) bool {
	return status == ResourceStatusCreated ||
		status == ResourceStatusUpdated ||
		status == ResourceStatusDestroyed
}

// PreciseResourceStatusIsSafeToRollback returns true if the resource is in a precise
// state where rolling back changes is safe and expected to succeed.
// This includes both fully complete and config-complete states, as config-complete
// indicates the resource was successfully configured even if not yet stable.
func PreciseResourceStatusIsSafeToRollback(preciseStatus PreciseResourceStatus) bool {
	return preciseStatus == PreciseResourceStatusCreated ||
		preciseStatus == PreciseResourceStatusConfigComplete ||
		preciseStatus == PreciseResourceStatusUpdated ||
		preciseStatus == PreciseResourceStatusUpdateConfigComplete ||
		preciseStatus == PreciseResourceStatusDestroyed
}
