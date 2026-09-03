package core

// LinkStatus is used to represent the current state of a link
// in a blueprint instance.
type LinkStatus int

const (
	// LinkStatusUnknown is used when we can't
	// determine an accurate status for a link.
	LinkStatusUnknown LinkStatus = iota
	// LinkStatusCreating is used when
	// an initial link deployment is currently in progress.
	LinkStatusCreating
	// LinkStatusCreated is used when
	// a link has been deployed
	// successfully.
	LinkStatusCreated
	// LinkStatusCreateFailed is used when
	// the first creation of a link failed.
	LinkStatusCreateFailed
	// LinkStatusCreateRollingBack is used when
	// another change in the same blueprint has failed
	// and the creation of the current link is being rolled back.
	LinkStatusCreateRollingBack
	// LinkStatusCreateRollbackFailed is used when
	// another element change in the same blueprint has failed
	// and the creation of the current link could not be rolled back.
	LinkStatusCreateRollbackFailed
	// LinkStatusCreateRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the creation of the current link has been rolled back.
	LinkStatusCreateRollbackComplete
	// LinkStatusDestroying is used when
	// a link is in the process of being destroyed.
	LinkStatusDestroying
	// LinkStatusDestroyed is used when
	// a link has been destroyed.
	LinkStatusDestroyed
	// LinkStatusDestroyFailed is used when
	// the destruction of a link fails.
	LinkStatusDestroyFailed
	// LinkStatusDestroyRollingBack is used when
	// another change in the same blueprint has failed
	// and the removal of the current link is being rolled back.
	LinkStatusDestroyRollingBack
	// LinkStatusDestroyRollbackFailed is used when
	// another resource change in the same blueprint has failed
	// and the removal of the current link could not be rolled back.
	LinkStatusDestroyRollbackFailed
	// LinkStatusDestroyRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the removal of the current link has been rolled back.
	LinkStatusDestroyRollbackComplete
	// LinkStatusUpdating is used when
	// a link is being updated.
	LinkStatusUpdating
	// LinkStatusUpdated is used when a link
	// has been sucessfully updated.
	LinkStatusUpdated
	// LinkStatusUpdateFailed is used when a link
	// has failed to update.
	LinkStatusUpdateFailed
	// LinkStatusUpdateRollingBack is used when
	// another change in the same blueprint has failed
	// and the latest changes made to
	// the current link are being rolled back.
	LinkStatusUpdateRollingBack
	// LinkStatusUpdateRollbackFailed is used when
	// another resource change in the same blueprint has failed
	// and the latest changes made to
	// the current link could not be rolled back.
	LinkStatusUpdateRollbackFailed
	// LinkStatusUpdateRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the latest changes made to
	// the current link have been rolled back.
	LinkStatusUpdateRollbackComplete
	// LinkStatusCreateInterrupted is used when
	// a link creation was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state of the link is unknown and needs reconciliation.
	LinkStatusCreateInterrupted
	// LinkStatusUpdateInterrupted is used when
	// a link update was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state of the link is unknown and needs reconciliation.
	LinkStatusUpdateInterrupted
	// LinkStatusDestroyInterrupted is used when
	// a link destruction was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state of the link is unknown and needs reconciliation.
	LinkStatusDestroyInterrupted
)

var linkStatusStrings = map[LinkStatus]string{
	LinkStatusUnknown:                 "UNKNOWN",
	LinkStatusCreating:                "CREATING",
	LinkStatusCreated:                 "CREATED",
	LinkStatusCreateFailed:            "CREATE FAILED",
	LinkStatusCreateRollingBack:       "CREATE ROLLING BACK",
	LinkStatusCreateRollbackFailed:    "CREATE ROLLBACK FAILED",
	LinkStatusCreateRollbackComplete:  "CREATE ROLLBACK COMPLETE",
	LinkStatusDestroying:              "DESTROYING",
	LinkStatusDestroyed:               "DESTROYED",
	LinkStatusDestroyFailed:           "DESTROY FAILED",
	LinkStatusDestroyRollingBack:      "DESTROY ROLLING BACK",
	LinkStatusDestroyRollbackFailed:   "DESTROY ROLLBACK FAILED",
	LinkStatusDestroyRollbackComplete: "DESTROY ROLLBACK COMPLETE",
	LinkStatusUpdating:                "UPDATING",
	LinkStatusUpdated:                 "UPDATED",
	LinkStatusUpdateFailed:            "UPDATE FAILED",
	LinkStatusUpdateRollingBack:       "UPDATE ROLLING BACK",
	LinkStatusUpdateRollbackFailed:    "UPDATE ROLLBACK FAILED",
	LinkStatusUpdateRollbackComplete:  "UPDATE ROLLBACK COMPLETE",
	LinkStatusCreateInterrupted:       "CREATE INTERRUPTED",
	LinkStatusUpdateInterrupted:       "UPDATE INTERRUPTED",
	LinkStatusDestroyInterrupted:      "DESTROY INTERRUPTED",
}

func (s LinkStatus) String() string {
	str, ok := linkStatusStrings[s]
	if !ok {
		return "UNKNOWN"
	}
	return str
}

// PreciseLinkStatus is used to represent a more precise
// current state of a link in a blueprint instance.
type PreciseLinkStatus int

const (
	// PreciseLinkStatusUnknown is used when we can't
	// determine an accurate status for a link.
	PreciseLinkStatusUnknown PreciseLinkStatus = iota
	// PreciseLinkStatusUpdatingLinkedResources is used when
	// the configuration for a link is being applied to the blueprint-declared
	// resources the link relates.
	PreciseLinkStatusUpdatingLinkedResources
	// PreciseLinkStatusLinkedResourcesUpdated is used when
	// the configuration for a link has been applied to the blueprint-declared
	// resources the link relates.
	PreciseLinkStatusLinkedResourcesUpdated
	// PreciseLinkStatusLinkedResourcesUpdateFailed is used when
	// the configuration for a link has failed to be applied to the
	// blueprint-declared resources the link relates.
	PreciseLinkStatusLinkedResourcesUpdateFailed
	// PreciseLinkStatusLinkedResourcesUpdateRollingBack is used when
	// another change in the same blueprint has failed
	// and the current link for which the resources it relates were
	// successfully updated is being rolled back.
	PreciseLinkStatusLinkedResourcesUpdateRollingBack
	// PreciseLinkStatusLinkedResourcesUpdateRollbackFailed is used when
	// another change in the same blueprint has failed
	// and the current link for which the resources it relates were
	// successfully updated failed to be rolled back.
	PreciseLinkStatusLinkedResourcesUpdateRollbackFailed
	// PreciseLinkStatusLinkedResourcesUpdateRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the current link for which the resources it relates were
	// successfully updated has been rolled back.
	PreciseLinkStatusLinkedResourcesUpdateRollbackComplete
	// PreciseLinkStatusUpdatingIntermediaryResources is used when
	// intermediary resources are being created, updated or destroyed.
	// This status is a high level indication of progress,
	// the status of each intermediary resource should be checked
	// to determine the exact state of each intermediary resource
	// in the link.
	PreciseLinkStatusUpdatingIntermediaryResources
	// PreciseLinkStatusIntermediaryResourcesUpdated is used when
	// all intermediary resources have been successfully updated,
	// created or destroyed.
	PreciseLinkStatusIntermediaryResourcesUpdated
	// PreciseLinkStatusIntermediaryResourceUpdateFailed is used when
	// an intermediary resource has failed to be updated, created or destroyed.
	PreciseLinkStatusIntermediaryResourceUpdateFailed
	// PreciseLinkStatusIntermediaryResourceUpdateRollingBack is used when
	// another change in the same blueprint has failed
	// and the current link for which intermediary resources were successfully
	// updated is being rolled back.
	PreciseLinkStatusIntermediaryResourceUpdateRollingBack
	// PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed is used when
	// another change in the same blueprint has failed
	// and the current link for which intermediary resources were successfully
	// updated failed to be rolled back.
	PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed
	// PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the current link for which intermediary resources were succsefully updated
	// has been rolled back.
	PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete
	// PreciseLinkStatusLinkedResourcesUpdateInterrupted is used when
	// the update of the resources a link relates was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state is unknown and needs reconciliation.
	PreciseLinkStatusLinkedResourcesUpdateInterrupted
	// PreciseLinkStatusIntermediaryResourceUpdateInterrupted is used when
	// the intermediary resources update for a link was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state is unknown and needs reconciliation.
	PreciseLinkStatusIntermediaryResourceUpdateInterrupted
)

var preciseLinkStatusStrings = map[PreciseLinkStatus]string{
	PreciseLinkStatusUnknown:                                    "UNKNOWN",
	PreciseLinkStatusUpdatingLinkedResources:                    "UPDATING LINKED RESOURCES",
	PreciseLinkStatusLinkedResourcesUpdated:                     "LINKED RESOURCES UPDATED",
	PreciseLinkStatusLinkedResourcesUpdateFailed:                "LINKED RESOURCES UPDATE FAILED",
	PreciseLinkStatusLinkedResourcesUpdateRollingBack:           "LINKED RESOURCES UPDATE ROLLING BACK",
	PreciseLinkStatusLinkedResourcesUpdateRollbackFailed:        "LINKED RESOURCES UPDATE ROLLBACK FAILED",
	PreciseLinkStatusLinkedResourcesUpdateRollbackComplete:      "LINKED RESOURCES UPDATE ROLLBACK COMPLETE",
	PreciseLinkStatusUpdatingIntermediaryResources:              "UPDATING INTERMEDIARY RESOURCES",
	PreciseLinkStatusIntermediaryResourcesUpdated:               "INTERMEDIARY RESOURCES UPDATED",
	PreciseLinkStatusIntermediaryResourceUpdateFailed:           "INTERMEDIARY RESOURCES UPDATE FAILED",
	PreciseLinkStatusIntermediaryResourceUpdateRollingBack:      "INTERMEDIARY RESOURCES UPDATE ROLLING BACK",
	PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed:   "INTERMEDIARY RESOURCES UPDATE ROLLBACK FAILED",
	PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete: "INTERMEDIARY RESOURCES UPDATE ROLLBACK COMPLETE",
	PreciseLinkStatusLinkedResourcesUpdateInterrupted:           "LINKED RESOURCES UPDATE INTERRUPTED",
	PreciseLinkStatusIntermediaryResourceUpdateInterrupted:      "INTERMEDIARY RESOURCES UPDATE INTERRUPTED",
}

func (s PreciseLinkStatus) String() string {
	str, ok := preciseLinkStatusStrings[s]
	if !ok {
		return "UNKNOWN"
	}
	return str
}

// LinkStatusIsUnsuccessfulCreate returns true if the link status indicates
// that a link creation was attempted but never completed successfully.
// This is used during change staging to determine if a link should be treated
// as "new" (requiring creation) rather than existing (requiring update).
func LinkStatusIsUnsuccessfulCreate(status LinkStatus) bool {
	return status == LinkStatusCreateFailed ||
		status == LinkStatusCreateInterrupted ||
		status == LinkStatusCreating
}

// LinkStatusIsSafeToRollback returns true if the link is in a state where
// rolling back changes is safe and expected to succeed.
// Safe states are those where the link operation completed successfully:
// - Created: link was successfully created (can be destroyed in rollback)
// - Updated: link was successfully updated (can be reverted in rollback)
// - Destroyed: link was successfully destroyed (can be recreated in rollback)
func LinkStatusIsSafeToRollback(status LinkStatus) bool {
	return status == LinkStatusCreated ||
		status == LinkStatusUpdated ||
		status == LinkStatusDestroyed
}
