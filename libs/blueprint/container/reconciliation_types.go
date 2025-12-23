package container

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ReconciliationType indicates why reconciliation is needed for an element.
type ReconciliationType string

const (
	// ReconciliationTypeInterrupted indicates the element was left in an interrupted
	// state after a deployment was cancelled (e.g., drain timeout after terminal failure).
	ReconciliationTypeInterrupted ReconciliationType = "interrupted"
	// ReconciliationTypeDrift indicates the element has drifted from the expected state
	// (external cloud state differs from persisted state).
	ReconciliationTypeDrift ReconciliationType = "drift"
	// ReconciliationTypeStateRefresh indicates a manual state refresh was requested
	// to sync persisted state with external cloud state.
	ReconciliationTypeStateRefresh ReconciliationType = "state_refresh"
)

// ReconciliationScope controls which elements to check for reconciliation.
type ReconciliationScope string

const (
	// ReconciliationScopeAll checks all resources and links in the instance.
	ReconciliationScopeAll ReconciliationScope = "all"
	// ReconciliationScopeInterrupted only checks elements currently in an interrupted state.
	ReconciliationScopeInterrupted ReconciliationScope = "interrupted"
	// ReconciliationScopeSpecific checks only the named resources and/or links.
	ReconciliationScopeSpecific ReconciliationScope = "specific"
)

// ReconciliationAction indicates what action to take for an element during reconciliation.
type ReconciliationAction string

const (
	// ReconciliationActionAcceptExternal accepts the external cloud state as truth
	// and updates persisted state to match.
	ReconciliationActionAcceptExternal ReconciliationAction = "accept_external"
	// ReconciliationActionUpdateStatus only updates the element's status without
	// modifying the persisted state data.
	ReconciliationActionUpdateStatus ReconciliationAction = "update_status"
	// ReconciliationActionMarkFailed marks the element as failed when the external
	// state indicates the operation did not complete successfully.
	ReconciliationActionMarkFailed ReconciliationAction = "mark_failed"
)

// CheckReconciliationInput specifies what to check for reconciliation.
type CheckReconciliationInput struct {
	// InstanceID is the ID of the blueprint instance to check.
	InstanceID string
	// Scope controls which elements to check.
	Scope ReconciliationScope
	// ResourceNames specifies which resources to check when Scope is ReconciliationScopeSpecific.
	// Ignored for other scopes.
	ResourceNames []string
	// LinkNames specifies which links to check when Scope is ReconciliationScopeSpecific.
	// Ignored for other scopes.
	LinkNames []string
}

// ReconciliationCheckResult contains all elements needing reconciliation.
type ReconciliationCheckResult struct {
	// InstanceID is the ID of the blueprint instance that was checked.
	InstanceID string
	// Resources contains reconciliation details for each resource that needs attention.
	Resources []ResourceReconcileResult
	// Links contains reconciliation details for each link that needs attention.
	Links []LinkReconcileResult
	// HasInterrupted is true if any elements are in an interrupted state.
	HasInterrupted bool
	// HasDrift is true if any elements have drifted from expected state.
	HasDrift bool
}

// ResourceReconcileResult contains reconciliation details for a single resource.
type ResourceReconcileResult struct {
	// ResourceID is the unique identifier for the resource.
	ResourceID string
	// ResourceName is the logical name of the resource in the blueprint.
	ResourceName string
	// ResourceType is the provider resource type (e.g., "aws/s3Bucket").
	ResourceType string
	// Type indicates why this resource needs reconciliation.
	Type ReconciliationType
	// OldStatus is the status the resource had before reconciliation check.
	OldStatus core.PreciseResourceStatus
	// NewStatus is the status determined from fetching external state.
	NewStatus core.PreciseResourceStatus
	// ExternalState contains the current cloud state if the resource exists.
	// This will be nil if the resource doesn't exist in the cloud.
	ExternalState *core.MappingNode
	// PersistedState contains what we have in our state store.
	PersistedState *core.MappingNode
	// Changes shows the detailed diff between persisted and external state.
	// Generated using the same change detection as drift checking.
	Changes *provider.Changes
	// ResourceExists indicates whether the resource was found in the cloud.
	ResourceExists bool
	// RecommendedAction is the suggested action based on the reconciliation analysis.
	RecommendedAction ReconciliationAction
}

// HasStateChanges returns true if the reconcile result has detected
// changes between the persisted and external state.
func (r *ResourceReconcileResult) HasStateChanges() bool {
	if r.Changes == nil {
		return false
	}
	return len(r.Changes.ModifiedFields) > 0 ||
		len(r.Changes.NewFields) > 0 ||
		len(r.Changes.RemovedFields) > 0
}

// LinkReconcileResult contains reconciliation details for a single link.
// Link reconciliation is derived from connected resource states and
// ResourceDataMappings rather than direct external state fetching.
type LinkReconcileResult struct {
	// LinkID is the unique identifier for the link.
	LinkID string
	// LinkName is the logical name of the link (format: "{resourceA}::{resourceB}").
	LinkName string
	// Type indicates why this link needs reconciliation.
	Type ReconciliationType
	// OldStatus is the status the link had before reconciliation check.
	OldStatus core.PreciseLinkStatus
	// NewStatus is the status determined from analyzing connected resources.
	NewStatus core.PreciseLinkStatus
	// ResourceAChanges contains changes attributed to this link on ResourceA.
	// Populated from ResourceDataMappings when resource drift is detected.
	ResourceAChanges *provider.Changes
	// ResourceBChanges contains changes attributed to this link on ResourceB.
	// Populated from ResourceDataMappings when resource drift is detected.
	ResourceBChanges *provider.Changes
	// IntermediaryChanges contains reconciliation details for intermediary resources
	// owned by this link. Key is the intermediary resource name.
	// Note: Intermediary resource reconciliation requires future provider interface changes.
	IntermediaryChanges map[string]*IntermediaryReconcileResult
	// RecommendedAction is the suggested action based on the reconciliation analysis.
	RecommendedAction ReconciliationAction
	// LinkDataUpdates contains the pre-computed updates to apply to link.Data when
	// accepting external state. This is derived from ResourceDataMappings during the
	// check phase, making it easy for callers to construct LinkReconcileAction.
	// Key is the linkDataPath (e.g., "resourceA.handler"), value is the external value.
	LinkDataUpdates map[string]*core.MappingNode
}

// IntermediaryReconcileResult contains reconciliation details for an intermediary resource
// owned by a link.
type IntermediaryReconcileResult struct {
	// Name is the name of the intermediary resource.
	Name string
	// Type is the type of the intermediary resource.
	Type string
	// ExternalState contains the current cloud state if the resource exists.
	ExternalState *core.MappingNode
	// PersistedState contains what we have in our state store.
	PersistedState *core.MappingNode
	// Changes shows the detailed diff between persisted and external state.
	Changes *provider.Changes
	// Exists indicates whether the intermediary resource was found in the cloud.
	Exists bool
}

// ApplyReconciliationInput specifies what reconciliation actions to apply.
type ApplyReconciliationInput struct {
	// InstanceID is the ID of the blueprint instance to reconcile.
	InstanceID string
	// ResourceActions specifies the actions to take for each resource.
	ResourceActions []ResourceReconcileAction
	// LinkActions specifies the actions to take for each link.
	LinkActions []LinkReconcileAction
}

// ResourceReconcileAction specifies the action to take for a resource.
type ResourceReconcileAction struct {
	// ResourceID is the unique identifier for the resource.
	ResourceID string
	// Action is the reconciliation action to apply.
	Action ReconciliationAction
	// ExternalState is required when Action is ReconciliationActionAcceptExternal.
	// This is the state that will be persisted.
	ExternalState *core.MappingNode
	// NewStatus is the status to set for the resource.
	NewStatus core.PreciseResourceStatus
}

// LinkReconcileAction specifies the action to take for a link.
type LinkReconcileAction struct {
	// LinkID is the unique identifier for the link.
	LinkID string
	// Action is the reconciliation action to apply.
	Action ReconciliationAction
	// NewStatus is the status to set for the link.
	NewStatus core.PreciseLinkStatus
	// LinkDataUpdates contains updates to apply to link.Data when Action is
	// ReconciliationActionAcceptExternal. This is used to sync link.Data with
	// external resource state when drift is detected via ResourceDataMappings.
	// Key is the linkDataPath (e.g., "resourceA.handler"), value is the new external value.
	LinkDataUpdates map[string]*core.MappingNode
	// IntermediaryActions specifies actions for each intermediary resource.
	// Key is the intermediary resource ID.
	IntermediaryActions map[string]*IntermediaryReconcileAction
}

// IntermediaryReconcileAction specifies the action to take for an intermediary resource.
type IntermediaryReconcileAction struct {
	// IntermediaryID is the unique identifier for the intermediary resource.
	IntermediaryID string
	// Action is the reconciliation action to apply.
	Action ReconciliationAction
	// ExternalState is required when Action is ReconciliationActionAcceptExternal.
	// This is the state that will be persisted.
	ExternalState *core.MappingNode
	// NewStatus is the status to set for the intermediary resource.
	NewStatus core.PreciseResourceStatus
}

// ApplyReconciliationResult contains the outcome of applying reconciliation.
type ApplyReconciliationResult struct {
	// InstanceID is the ID of the blueprint instance that was reconciled.
	InstanceID string
	// ResourcesUpdated is the number of resources that were successfully updated.
	ResourcesUpdated int
	// LinksUpdated is the number of links that were successfully updated.
	LinksUpdated int
	// Errors contains any errors that occurred during reconciliation.
	Errors []ReconciliationError
}

// ReconciliationError captures an error for a specific element during reconciliation.
type ReconciliationError struct {
	// ElementID is the unique identifier for the element that failed.
	ElementID string
	// ElementName is the logical name of the element.
	ElementName string
	// ElementType is "resource" or "link".
	ElementType string
	// Error is the error message.
	Error string
}
