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
	Rollback bool `json:"rollback"`
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
	Rollback bool `json:"rollback"`
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
