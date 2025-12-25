package deploymentsv1

import (
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// CreateChangesetRequestPayload represents the payload
// for creating a new change set and start a new change staging process.
type CreateChangesetRequestPayload struct {
	resolve.BlueprintDocumentInfo
	// The ID of an existing blueprint instance to stage changes for.
	// If this is not provided and an instance name is not provided,
	// a change set for a new blueprint instance deployment will be created.
	// This should be left empty if the `instanceName` field is provided.
	InstanceID string `json:"instanceId"`
	// The user-defined name of an existing blueprint instance to stage changes for.
	// If this is not provided an an instance ID is not provided, a change set for a new
	// blueprint instance deployment will be created.
	// This should be left empty if the `instanceId` field is provided.
	InstanceName string `json:"instanceName"`
	// If true, the change set will be created for a destroy operation.
	// This will only be used if the `instanceId` or `instanceName` fields are provided.
	// If this is not provided, the default value will be false.
	Destroy bool `json:"destroy"`
	// SkipDriftCheck, when true, skips drift detection during change staging.
	SkipDriftCheck bool `json:"skipDriftCheck"`
	// Config values for the change staging process
	// that will be used in plugins and passed into the blueprint.
	Config *types.BlueprintOperationConfig `json:"config"`
}

// BlueprintInstanceRequestPayload represents the payload
// for creating and updating blueprint instances which in turn starts
// the deployment process for new or existing blueprint instances.
type BlueprintInstanceRequestPayload struct {
	resolve.BlueprintDocumentInfo
	// The user-defined name for the blueprint instance.
	// This is required when creating a new blueprint instance and must be unique.
	// This should be left empty when updating an existing instance.
	InstanceName string `json:"instanceName"`
	// The ID of the change set to use to deploy the blueprint instance.
	// When deploying blueprint instances,
	// a change set is used instead of the deployment process re-computing the changes
	// that need to be applied.
	// The source blueprint document is still required in addition to a change set to finish
	// resolving substitutions that can only be resolved at deploy time and for deployment
	// orchestration.
	// The source blueprint document is not used to compute changes at the deployment stage.
	ChangeSetID string `json:"changeSetId" validate:"required"`
	// If true, and a new blueprint instance is being created,
	// the creation of the blueprint instance will be treated as a rollback operation
	// for a previously destroyed blueprint instance.
	// If true, and an existing blueprint instance is being updated,
	// the update will be treated as a rollback operation for the previous state.
	AsRollback bool `json:"asRollback"`
	// If true, the deployment will automatically rollback on failure.
	// Auto-rollback is supported for:
	//
	// - New deployments (DeployFailed): Destroys partially created resources,
	//   ensuring a clean state where users can fix issues and retry.
	//
	// - Updates (UpdateFailed): Reverts to the previous instance state by
	//   generating and deploying a reverse changeset that undoes the failed changes.
	//
	// - Destroys (DestroyFailed): Recreates destroyed resources from the previous
	//   instance state using a reverse changeset.
	//
	// Rollback operations have auto-rollback disabled to prevent infinite loops.
	AutoRollback bool `json:"autoRollback"`
	// Force bypasses state validation checks that prevent deployment when the instance
	// is already in an active state (e.g., Deploying, Updating).
	// This is an escape hatch for recovering from stuck states where the instance
	// is in an inconsistent state due to a crash or unexpected termination.
	Force bool `json:"force"`
	// Config values for the deployment process
	// that will be used in plugins and passed into the blueprint.
	Config *types.BlueprintOperationConfig `json:"config"`
}

// BlueprintInstanceDestroyRequestPayload represents the payload
// for destroying a blueprint instance.
type BlueprintInstanceDestroyRequestPayload struct {
	// The ID of the change set to use to destroy the blueprint instance.
	// When destroying a blueprint instance,
	// a change set is used instead of the destroy process re-computing the changes
	// that need to be applied.
	ChangeSetID string `json:"changeSetId" validate:"required"`
	// If true, destroying the blueprint instance will be treated as a rollback
	// for the initial deployment of the blueprint instance.
	// This will usually be set to true when rolling back a recent first time
	// deployment that needs to be rolled back due to failure in a parent
	// blueprint instance.
	AsRollback bool `json:"asRollback"`
	// Force continues the destroy operation even if individual resource/link/child
	// destruction fails, and removes the blueprint instance record from state
	// regardless of whether all resources were successfully destroyed.
	// This is useful for removing instances where underlying resources were manually
	// deleted or when a provider is unavailable.
	Force bool `json:"force"`
	// Config values for the destroy process
	// that will be used in plugins.
	Config *types.BlueprintOperationConfig `json:"config"`
}

type errorMessageEvent struct {
	Message     string             `json:"message"`
	Diagnostics []*core.Diagnostic `json:"diagnostics"`
	Timestamp   int64              `json:"timestamp"`
}

type resourceChangesEventWithTimestamp struct {
	container.ResourceChangesMessage
	Timestamp int64 `json:"timestamp"`
}

type childChangesEventWithTimestamp struct {
	container.ChildChangesMessage
	Timestamp int64 `json:"timestamp"`
}
type linkChangesEventWithTimestamp struct {
	container.LinkChangesMessage
	Timestamp int64 `json:"timestamp"`
}

type changeStagingCompleteEvent struct {
	Changes   *changes.BlueprintChanges `json:"changes"`
	Timestamp int64                     `json:"timestamp"`
}

// CheckReconciliationRequestPayload represents the payload for checking
// reconciliation status of a blueprint instance.
type CheckReconciliationRequestPayload struct {
	resolve.BlueprintDocumentInfo
	// Scope controls which elements to check.
	// Valid values: "all" (default), "interrupted", "specific"
	Scope string `json:"scope"`
	// ResourceNames specifies which resources to check when Scope is "specific".
	// Ignored for other scopes.
	ResourceNames []string `json:"resourceNames,omitempty"`
	// LinkNames specifies which links to check when Scope is "specific".
	// Ignored for other scopes.
	LinkNames []string `json:"linkNames,omitempty"`
	// IncludeChildren controls whether to recursively check child blueprints.
	// If nil or not provided, defaults to true.
	IncludeChildren *bool `json:"includeChildren,omitempty"`
	// ChildPath limits the scope to resources/links within a specific child blueprint path.
	// Used when Scope is "specific".
	// Format: "childA" for first level, "childA.childB" for nested.
	ChildPath string `json:"childPath,omitempty"`
	// Config values for the reconciliation check
	// that will be used in plugins.
	Config *types.BlueprintOperationConfig `json:"config" validate:"required"`
}

// ApplyReconciliationRequestPayload represents the payload for applying
// reconciliation actions to a blueprint instance.
type ApplyReconciliationRequestPayload struct {
	resolve.BlueprintDocumentInfo
	// ResourceActions specifies the actions to take for each resource.
	ResourceActions []ResourceReconcileActionPayload `json:"resourceActions,omitempty"`
	// LinkActions specifies the actions to take for each link.
	LinkActions []LinkReconcileActionPayload `json:"linkActions,omitempty"`
	// Config values for the reconciliation apply
	// that will be used in plugins.
	Config *types.BlueprintOperationConfig `json:"config" validate:"required"`
}

// ResourceReconcileActionPayload specifies the action to take for a resource.
type ResourceReconcileActionPayload struct {
	// ResourceID is the unique identifier for the resource.
	ResourceID string `json:"resourceId" validate:"required"`
	// ChildPath is the path to the child blueprint containing this resource.
	// Empty for resources in the parent blueprint.
	// Format: "childA" for first level, "childA.childB" for nested.
	ChildPath string `json:"childPath,omitempty"`
	// Action is the reconciliation action to apply.
	// Valid values: "accept_external", "update_status", "mark_failed"
	Action string `json:"action" validate:"required"`
	// ExternalState is required when Action is "accept_external".
	// This is the state that will be persisted.
	ExternalState *core.MappingNode `json:"externalState,omitempty"`
	// NewStatus is the status to set for the resource.
	NewStatus string `json:"newStatus" validate:"required"`
}

// LinkReconcileActionPayload specifies the action to take for a link.
type LinkReconcileActionPayload struct {
	// LinkID is the unique identifier for the link.
	LinkID string `json:"linkId" validate:"required"`
	// ChildPath is the path to the child blueprint containing this link.
	// Empty for links in the parent blueprint.
	// Format: "childA" for first level, "childA.childB" for nested.
	ChildPath string `json:"childPath,omitempty"`
	// Action is the reconciliation action to apply.
	// Valid values: "accept_external", "update_status", "mark_failed"
	Action string `json:"action" validate:"required"`
	// NewStatus is the status to set for the link.
	NewStatus string `json:"newStatus" validate:"required"`
	// LinkDataUpdates contains updates to apply to link.Data when Action is
	// "accept_external". This is used to sync link.Data with
	// external resource state when drift is detected via ResourceDataMappings.
	// Key is the linkDataPath (e.g., "resourceA.handler"), value is the new external value.
	LinkDataUpdates map[string]*core.MappingNode `json:"linkDataUpdates,omitempty"`
	// IntermediaryActions specifies actions for each intermediary resource.
	// Key is the intermediary resource ID.
	IntermediaryActions map[string]*IntermediaryReconcileActionPayload `json:"intermediaryActions,omitempty"`
}

// IntermediaryReconcileActionPayload specifies the action to take for an intermediary resource.
type IntermediaryReconcileActionPayload struct {
	// Action is the reconciliation action to apply.
	// Valid values: "accept_external", "update_status", "mark_failed"
	Action string `json:"action" validate:"required"`
	// ExternalState is required when Action is "accept_external".
	// This is the state that will be persisted.
	ExternalState *core.MappingNode `json:"externalState,omitempty"`
	// NewStatus is the status to set for the intermediary resource.
	NewStatus string `json:"newStatus" validate:"required"`
}

// DriftBlockedResponse is returned when an operation is blocked due to drift detection.
type DriftBlockedResponse struct {
	// Message explains why the operation was blocked.
	Message string `json:"message"`
	// InstanceID is the ID of the blueprint instance.
	InstanceID string `json:"instanceId"`
	// ChangesetID is the ID of the changeset that detected drift (if applicable).
	ChangesetID string `json:"changesetId,omitempty"`
	// ReconciliationResult contains the full drift/interrupted state detection result.
	// This allows clients to see exactly what drifted without making a separate API call.
	ReconciliationResult *container.ReconciliationCheckResult `json:"reconciliationResult,omitempty"`
	// Hint provides guidance on how to proceed.
	Hint string `json:"hint"`
}

// driftDetectedEvent is the event payload sent when drift or interrupted state is detected
// during change staging.
type driftDetectedEvent struct {
	// Message explains what was detected.
	Message string `json:"message"`
	// ReconciliationResult contains the full reconciliation check result.
	ReconciliationResult *container.ReconciliationCheckResult `json:"reconciliationResult"`
	// Timestamp is the unix timestamp when drift was detected.
	Timestamp int64 `json:"timestamp"`
}

const (
	// eventTypeDriftDetected is the event type for drift/interrupted detection during change staging.
	eventTypeDriftDetected = "driftDetected"
)
