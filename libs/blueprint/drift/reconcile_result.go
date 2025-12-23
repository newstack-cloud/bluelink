package drift

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ReconcileResult represents the result of checking an interrupted resource
// with full state details to allow the user to review before applying reconciliation.
type ReconcileResult struct {
	// ResourceID is the unique identifier for the resource.
	ResourceID string
	// ResourceName is the logical name of the resource.
	ResourceName string
	// ResourceType is the type of the resource (e.g., "aws/s3Bucket").
	ResourceType string
	// OldStatus is the status the resource had before reconciliation
	// (typically an interrupted status).
	OldStatus core.PreciseResourceStatus
	// NewStatus is the status determined from fetching external state.
	// This will be the actual status of the resource (e.g., created, create_failed).
	NewStatus core.PreciseResourceStatus
	// ExternalState contains the actual cloud state if the resource exists.
	// This will be nil if the resource doesn't exist in the cloud.
	ExternalState *core.MappingNode
	// PersistedState contains what we had in our state before interruption.
	PersistedState *core.MappingNode
	// StateChanges shows the diff between persisted and external state.
	// Generated using the same change detection as drift checking.
	StateChanges *provider.Changes
}

// HasStateChanges returns true if the reconcile result has detected
// changes between the persisted and external state.
func (r *ReconcileResult) HasStateChanges() bool {
	if r.StateChanges == nil {
		return false
	}
	return len(r.StateChanges.ModifiedFields) > 0 ||
		len(r.StateChanges.NewFields) > 0 ||
		len(r.StateChanges.RemovedFields) > 0
}

// ResourceExists returns true if the resource was found in the external state.
func (r *ReconcileResult) ResourceExists() bool {
	return r.ExternalState != nil
}
