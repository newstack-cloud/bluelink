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
	// PreciseLinkStatusUpdatingResourceA is used when
	// the configuration for a link is being applied to resource A
	// in the link.
	PreciseLinkStatusUpdatingResourceA
	// PreciseLinkStatusResourceAUpdated is used when
	// the configuration for a link has been applied to resource A
	// in the link.
	PreciseLinkStatusResourceAUpdated
	// PreciseLinkStatusResourceAUpdateFailed is used when
	// the configuration for a link has failed to be applied to resource A
	// in the link.
	PreciseLinkStatusResourceAUpdateFailed
	// PreciseLinkStatusResourceAUpdateRollingBack is used when
	// another change in the same blueprint has failed
	// and the current link for which resource A was successfully
	// updated is being rolled back.
	PreciseLinkStatusResourceAUpdateRollingBack
	// PreciseLinkStatusResourceAUpdateRollbackFailed is used when
	// another change in the same blueprint has failed
	// and the current link for which resource A was successfully
	// updated failed to be rolled back.
	PreciseLinkStatusResourceAUpdateRollbackFailed
	// PreciseLinkStatusResourceAUpdateRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the current link for which resource A was succsefully updated
	// has been rolled back.
	PreciseLinkStatusResourceAUpdateRollbackComplete
	// PreciseLinkStatusUpdatingResourceB is used when
	// the configuration for a link is being applied to resource B
	// in the link.
	PreciseLinkStatusUpdatingResourceB
	// PreciseLinkStatusResourceBUpdated is used when
	// the configuration for a link has been applied to resource B
	// in the link.
	PreciseLinkStatusResourceBUpdated
	// PreciseLinkStatusResourceBUpdateFailed is used when
	// the configuration for a link has failed to be applied to resource B
	// in the link.
	PreciseLinkStatusResourceBUpdateFailed
	// PreciseLinkStatusResourceBUpdateRollingBack is used when
	// another change in the same blueprint has failed
	// and the current link for which resource B was successfully
	// updated is being rolled back.
	PreciseLinkStatusResourceBUpdateRollingBack
	// PreciseLinkStatusResourceBUpdateRollbackFailed is used when
	// another change in the same blueprint has failed
	// and the current link for which resource B was successfully
	// updated failed to be rolled back.
	PreciseLinkStatusResourceBUpdateRollbackFailed
	// PreciseLinkStatusResourceBUpdateRollbackComplete is used when
	// another change in the same blueprint has failed
	// and the current link for which resource B was succsefully updated
	// has been rolled back.
	PreciseLinkStatusResourceBUpdateRollbackComplete
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
	// PreciseLinkStatusResourceAUpdateInterrupted is used when
	// the resource A update for a link was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state is unknown and needs reconciliation.
	PreciseLinkStatusResourceAUpdateInterrupted
	// PreciseLinkStatusResourceBUpdateInterrupted is used when
	// the resource B update for a link was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state is unknown and needs reconciliation.
	PreciseLinkStatusResourceBUpdateInterrupted
	// PreciseLinkStatusIntermediaryResourceUpdateInterrupted is used when
	// the intermediary resources update for a link was interrupted due to deployment
	// being cancelled (e.g., drain timeout after terminal failure).
	// The actual state is unknown and needs reconciliation.
	PreciseLinkStatusIntermediaryResourceUpdateInterrupted
)

var preciseLinkStatusStrings = map[PreciseLinkStatus]string{
	PreciseLinkStatusUnknown:                                    "UNKNOWN",
	PreciseLinkStatusUpdatingResourceA:                          "UPDATING RESOURCE A",
	PreciseLinkStatusResourceAUpdated:                           "RESOURCE A UPDATED",
	PreciseLinkStatusResourceAUpdateFailed:                      "RESOURCE A UPDATE FAILED",
	PreciseLinkStatusResourceAUpdateRollingBack:                 "RESOURCE A UPDATE ROLLING BACK",
	PreciseLinkStatusResourceAUpdateRollbackFailed:              "RESOURCE A UPDATE ROLLBACK FAILED",
	PreciseLinkStatusResourceAUpdateRollbackComplete:            "RESOURCE A UPDATE ROLLBACK COMPLETE",
	PreciseLinkStatusUpdatingResourceB:                          "UPDATING RESOURCE B",
	PreciseLinkStatusResourceBUpdated:                           "RESOURCE B UPDATED",
	PreciseLinkStatusResourceBUpdateFailed:                      "RESOURCE B UPDATE FAILED",
	PreciseLinkStatusResourceBUpdateRollingBack:                 "RESOURCE B UPDATE ROLLING BACK",
	PreciseLinkStatusResourceBUpdateRollbackFailed:              "RESOURCE B UPDATE ROLLBACK FAILED",
	PreciseLinkStatusResourceBUpdateRollbackComplete:            "RESOURCE B UPDATE ROLLBACK COMPLETE",
	PreciseLinkStatusUpdatingIntermediaryResources:              "UPDATING INTERMEDIARY RESOURCES",
	PreciseLinkStatusIntermediaryResourcesUpdated:               "INTERMEDIARY RESOURCES UPDATED",
	PreciseLinkStatusIntermediaryResourceUpdateFailed:           "INTERMEDIARY RESOURCES UPDATE FAILED",
	PreciseLinkStatusIntermediaryResourceUpdateRollingBack:      "INTERMEDIARY RESOURCES UPDATE ROLLING BACK",
	PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed:   "INTERMEDIARY RESOURCES UPDATE ROLLBACK FAILED",
	PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete: "INTERMEDIARY RESOURCES UPDATE ROLLBACK COMPLETE",
	PreciseLinkStatusResourceAUpdateInterrupted:                 "RESOURCE A UPDATE INTERRUPTED",
	PreciseLinkStatusResourceBUpdateInterrupted:                 "RESOURCE B UPDATE INTERRUPTED",
	PreciseLinkStatusIntermediaryResourceUpdateInterrupted:      "INTERMEDIARY RESOURCES UPDATE INTERRUPTED",
}

func (s PreciseLinkStatus) String() string {
	str, ok := preciseLinkStatusStrings[s]
	if !ok {
		return "UNKNOWN"
	}
	return str
}
