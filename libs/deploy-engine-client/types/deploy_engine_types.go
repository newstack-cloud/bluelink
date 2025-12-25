package types

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// BlueprintValidationEvent holds the data for a blueprint validation
// event that is sent to a validation stream.
type BlueprintValidationEvent struct {
	core.Diagnostic
	// ID of the blueprint validation event,
	// this is useful for clients that want to persist events
	// to a database or other storage.
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	// End indicates whether or not the event is the last event
	// in the stream.
	End bool `json:"end"`
}

// CreateBlueprintValidationPayload represents the payload
// for creating a new blueprint validation.
type CreateBlueprintValidationPayload struct {
	BlueprintDocumentInfo
	// Config values for the validation process
	// that will be used in plugins and passed into the blueprint.
	Config *BlueprintOperationConfig `json:"config"`
}

// CreateBlueprintValidationQuery represents options
// for creating a new blueprint validation.
// This holds optional query fields that map to query string parameters
// that can be used to control the behaviour of the validation process.
type CreateBlueprintValidationQuery struct {
	// CheckBlueprintVars indicates whether or not to check
	// the blueprint variables provided in the request payload
	// as part of the validation process.
	CheckBlueprintVars bool
	// CheckPluginConfig indicates whether or not to check
	// the plugin configuration provided in the request payload
	// as part of the validation process.
	// If set to true, the plugin configuration will be validated
	// against the plugin schemas for each provider and transformer
	// for which configuration is provided in the request.
	CheckPluginConfig bool
}

// CreateChangesetPayload represents the payload
// for creating a new change set and starting the change staging process.
type CreateChangesetPayload struct {
	BlueprintDocumentInfo
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
	// Drift detection checks for external changes to resources that were made
	// outside of the deploy engine.
	SkipDriftCheck bool `json:"skipDriftCheck"`
	// Config values for the change staging process
	// that will be used in plugins and passed into the blueprint.
	Config *BlueprintOperationConfig `json:"config"`
}

// ChangeStagingEventType is the type of change staging event
// that is sent to a change staging stream for a change set.
type ChangeStagingEventType string

const (
	// ChangeStagingEventTypeResourceChanges is the type of change staging event
	// that is sent to a change staging stream for a change set
	// when the changes to a resource have been computed.
	ChangeStagingEventTypeResourceChanges ChangeStagingEventType = "resourceChanges"
	// ChangeStagingEventTypeChildChanges is the type of change staging event
	// that is sent to a change staging stream for a change set
	// when the changes to a child blueprint have been computed.
	ChangeStagingEventTypeChildChanges ChangeStagingEventType = "childChanges"
	// ChangeStagingEventTypeLinkChanges is the type of change staging event
	// that is sent to a change staging stream for a change set
	// when the changes to a link between two resources have been computed.
	ChangeStagingEventTypeLinkChanges ChangeStagingEventType = "linkChanges"
	// ChangeStagingEventTypeCompleteChanges is the type of change staging event
	// that is sent to a change staging stream for a change set
	// when the change staging process has been completed.
	ChangeStagingEventTypeCompleteChanges ChangeStagingEventType = "changeStagingComplete"
)

// ChangeStagingEvent holds the data for change staging event
// that is sent to a change staging stream for a change set.
type ChangeStagingEvent struct {
	// ID of the change staging event,
	// this is useful for clients that want to persist events
	// to a database or other storage.
	ID string `json:"id"`
	// ResourceChanges is populated when the change staging event
	// is for when the changes to a resource have been computed.
	ResourceChanges *ResourceChangesEventData `json:"resourceChanges"`
	// ChildChanges is populated when the change staging event
	// is for when the changes to a child blueprint have been computed.
	ChildChanges *ChildChangesEventData `json:"childChanges"`
	// LinkChanges is populated when the change staging event
	// is for when the changes to a link between two resources have been computed.
	LinkChanges *LinkChangesEventData `json:"linkChanges"`
	// CompleteChanges is populated when change staging has been completed,
	// this contains the full set of changes.
	CompleteChanges *CompleteChangesEventData `json:"completeChanges"`
	// DriftDetected is populated when drift or interrupted state is detected
	// during change staging. When this is populated, the change staging process
	// is blocked and the changeset will have DRIFT_DETECTED status.
	DriftDetected *DriftDetectedEventData `json:"driftDetected"`
}

// GetType is a helper method that derives the type of a change staging event
// that matches the `event` field in the raw event stream.
func (c *ChangeStagingEvent) GetType() ChangeStagingEventType {
	switch {
	case c.ResourceChanges != nil:
		return ChangeStagingEventTypeResourceChanges
	case c.ChildChanges != nil:
		return ChangeStagingEventTypeChildChanges
	case c.LinkChanges != nil:
		return ChangeStagingEventTypeLinkChanges
	case c.CompleteChanges != nil:
		return ChangeStagingEventTypeCompleteChanges
	case c.DriftDetected != nil:
		return ChangeStagingEventTypeDriftDetected
	default:
		return ""
	}
}

// AsResourceChanges is a helper method that returns the resource changes
// event data if the change staging event is of type resource changes.
func (c *ChangeStagingEvent) AsResourceChanges() (*ResourceChangesEventData, bool) {
	if c.ResourceChanges != nil {
		return c.ResourceChanges, true
	}

	return nil, false
}

// AsChildChanges is a helper method that returns the child changes
// event data if the change staging event is of type child changes.
func (c *ChangeStagingEvent) AsChildChanges() (*ChildChangesEventData, bool) {
	if c.ChildChanges != nil {
		return c.ChildChanges, true
	}

	return nil, false
}

// AsLinkChanges is a helper method that returns the link changes
// event data if the change staging event is of type link changes.
func (c *ChangeStagingEvent) AsLinkChanges() (*LinkChangesEventData, bool) {
	if c.LinkChanges != nil {
		return c.LinkChanges, true
	}

	return nil, false
}

// AsCompleteChanges is a helper method that returns the complete changes
// event data if the change staging event is of type complete changes.
func (c *ChangeStagingEvent) AsCompleteChanges() (*CompleteChangesEventData, bool) {
	if c.CompleteChanges != nil {
		return c.CompleteChanges, true
	}

	return nil, false
}

// AsDriftDetected is a helper method that returns the drift detected
// event data if the change staging event is of type drift detected.
func (c *ChangeStagingEvent) AsDriftDetected() (*DriftDetectedEventData, bool) {
	if c.DriftDetected != nil {
		return c.DriftDetected, true
	}

	return nil, false
}

// ResourceChangesEventData holds the data for a resource changes event
// that is sent to a change staging stream for a change set.
type ResourceChangesEventData struct {
	container.ResourceChangesMessage
	Timestamp int64 `json:"timestamp"`
}

// ChildChangesEventData holds the data for a child changes event
// that is sent to a change staging stream for a change set.
type ChildChangesEventData struct {
	container.ChildChangesMessage
	Timestamp int64 `json:"timestamp"`
}

// LinkChangesEventData holds the data for a link changes event
// that is sent to a change staging stream for a change set.
type LinkChangesEventData struct {
	container.LinkChangesMessage
	Timestamp int64 `json:"timestamp"`
}

// CompleteChangesEventData holds the data for a complete changes event
// that is sent to a change staging stream for a change set.
type CompleteChangesEventData struct {
	Changes   *changes.BlueprintChanges `json:"changes"`
	Timestamp int64                     `json:"timestamp"`
}

// BlueprintInstancePayload represents the payload
// for creating or updating a blueprint instance
// and starting the deployment process.
type BlueprintInstancePayload struct {
	BlueprintDocumentInfo
	// A unique, user-defined name for the blueprint instance.
	// This name must be unique across all blueprint instances in the system
	// and can be used as an alternative to the instance ID when referencing
	// the blueprint instance in other API calls.
	// This is required when creating a new blueprint instance.
	InstanceName string `json:"instanceName,omitempty"`
	// The ID of the change set to use to deploy the blueprint instance.
	// When deploying blueprint instances,
	// a change set is used instead of the deployment process re-computing the changes
	// that need to be applied.
	// The source blueprint document is still required in addition to a change set to finish
	// resolving substitutions that can only be resolved at deploy time and for deployment
	// orchestration.
	// The source blueprint document is not used to compute changes at the deployment stage.
	ChangeSetID string `json:"changeSetId"`
	// If true, and a new blueprint instance is being created,
	// the creation of the blueprint instance will be treated as a rollback operation
	// for a previously destroyed blueprint instance.
	// If true, and an existing blueprint instance is being updated,
	// the update will be treated as a rollback operation for the previous state.
	AsRollback bool `json:"asRollback"`
	// If true, the deployment will automatically rollback on failure.
	// For new deployments, this destroys partially created resources.
	// For updates, this reverts to the previous instance state.
	// For destroys, this recreates destroyed resources from the previous state.
	AutoRollback bool `json:"autoRollback"`
	// Force bypasses state validation checks that prevent deployment when the instance
	// is already in an active state (e.g., Deploying, Updating).
	// This is an escape hatch for recovering from stuck states.
	Force bool `json:"force"`
	// Config values for the deployment process
	// that will be used in plugins and passed into the blueprint.
	Config *BlueprintOperationConfig `json:"config"`
}

// DestroyBlueprintInstancePayload represents the payload
// for starting the destroy process for a blueprint instance.
type DestroyBlueprintInstancePayload struct {
	// The ID of the change set to use to destroy the blueprint instance.
	// When destroying a blueprint instance,
	// a change set is used instead of the destroy process re-computing the changes
	// that need to be applied.
	ChangeSetID string `json:"changeSetId"`
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
	Config *BlueprintOperationConfig `json:"config"`
}

// BlueprintInstanceEvent holds the data for a deployment event
// that is sent to a blueprint instance stream.
// This event type is used for both deploying and destroying
// blueprint instances.
type BlueprintInstanceEvent struct {
	// ID of the blueprint instance event,
	// this is useful for clients that want to persist events
	// to a database or other storage.
	ID string `json:"id"`
	container.DeployEvent
}

func (e *BlueprintInstanceEvent) String() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("event[%s]: id: %s", e.GetType(), e.ID))

	if e.ChildUpdateEvent != nil {
		childUpdateEvent := e.ChildUpdateEvent
		sb.WriteString(
			fmt.Sprintf(
				"child: %s status: %s",
				childUpdateEvent.ChildName,
				childUpdateEvent.Status.String(),
			),
		)
	}

	if e.ResourceUpdateEvent != nil {
		resourceUpdateEvent := e.ResourceUpdateEvent
		sb.WriteString(
			fmt.Sprintf(
				" resource: %s status: %s",
				resourceUpdateEvent.ResourceName,
				resourceUpdateEvent.Status.String(),
			),
		)
	}

	if e.LinkUpdateEvent != nil {
		linkUpdateEvent := e.LinkUpdateEvent
		sb.WriteString(
			fmt.Sprintf(
				" link: %s status: %s",
				linkUpdateEvent.LinkName,
				linkUpdateEvent.Status.String(),
			),
		)
	}

	if e.DeploymentUpdateEvent != nil {
		deploymentUpdateEvent := e.DeploymentUpdateEvent
		sb.WriteString(
			fmt.Sprintf(
				" deployment status: %s",
				deploymentUpdateEvent.Status.String(),
			),
		)
	}

	if e.FinishEvent != nil {
		finishEvent := e.FinishEvent
		sb.WriteString(
			fmt.Sprintf(
				" finish status: %s",
				finishEvent.Status.String(),
			),
		)
	}

	return sb.String()
}

// BlueprintInstanceEventType is the type of deployment event
// that is sent to a blueprint instance stream.
type BlueprintInstanceEventType string

const (
	// BlueprintInstanceEventTypeResourceUpdate is the type of deployment event
	// that is sent to a blueprint instance stream
	// when there is a change in status of a resource deployment or removal.
	BlueprintInstanceEventTypeResourceUpdate BlueprintInstanceEventType = "resource"
	// BlueprintInstanceEventTypeChildUpdate is the type of deployment event
	// that is sent to a blueprint instance stream
	// when there is a change in status of a child blueprint deployment or removal.
	BlueprintInstanceEventTypeChildUpdate BlueprintInstanceEventType = "child"
	// BlueprintInstanceEventTypeLinkUpdate is the type of deployment event
	// that is sent to a blueprint instance stream
	// when there is a change in status of a link deployment or removal.
	BlueprintInstanceEventTypeLinkUpdate BlueprintInstanceEventType = "link"
	// BlueprintInstanceEventTypeInstanceUpdate is the type of deployment event
	// that is sent to a blueprint instance stream
	// when there is a change in status of the overall
	// blueprint instance deployment or removal.
	BlueprintInstanceEventTypeInstanceUpdate BlueprintInstanceEventType = "instanceUpdate"
	// BlueprintInstanceEventTypeDeployFinished is the type of deployment event
	// that is sent to a blueprint instance stream
	// when the deployment process has been completed either successfully or with errors.
	BlueprintInstanceEventTypeDeployFinished BlueprintInstanceEventType = "finish"
)

// GetType is a helper method that derives the type of a blueprint instance event
// that matches the `event` field in the raw event stream.
func (c *BlueprintInstanceEvent) GetType() BlueprintInstanceEventType {
	switch {
	case c.ResourceUpdateEvent != nil:
		return BlueprintInstanceEventTypeResourceUpdate
	case c.ChildUpdateEvent != nil:
		return BlueprintInstanceEventTypeChildUpdate
	case c.LinkUpdateEvent != nil:
		return BlueprintInstanceEventTypeLinkUpdate
	case c.DeploymentUpdateEvent != nil:
		return BlueprintInstanceEventTypeInstanceUpdate
	case c.FinishEvent != nil:
		return BlueprintInstanceEventTypeDeployFinished
	default:
		return ""
	}
}

// AsResourceUpdate is a helper method that returns the resource update
// event data if th blueprint instance event is a resource update.
func (c *BlueprintInstanceEvent) AsResourceUpdate() (*container.ResourceDeployUpdateMessage, bool) {
	if c.ResourceUpdateEvent != nil {
		return c.ResourceUpdateEvent, true
	}

	return nil, false
}

// AsChildUpdate is a helper method that returns the child update
// event data if the blueprint instance event is a child blueprint update.
func (c *BlueprintInstanceEvent) AsChildUpdate() (*container.ChildDeployUpdateMessage, bool) {
	if c.ChildUpdateEvent != nil {
		return c.ChildUpdateEvent, true
	}

	return nil, false
}

// AsLinkUpdate is a helper method that returns the link update
// event data if the blueprint instance event is an update to a link
// between two resources.
func (c *BlueprintInstanceEvent) AsLinkUpdate() (*container.LinkDeployUpdateMessage, bool) {
	if c.LinkUpdateEvent != nil {
		return c.LinkUpdateEvent, true
	}

	return nil, false
}

// AsInstanceUpdate is a helper method that returns the instance update
// event data if the blueprint instance event is an update to the
// overall blueprint instance deployment or removal.
func (c *BlueprintInstanceEvent) AsInstanceUpdate() (*container.DeploymentUpdateMessage, bool) {
	if c.DeploymentUpdateEvent != nil {
		return c.DeploymentUpdateEvent, true
	}

	return nil, false
}

// AsFinish is a helper method that returns the finish
// event data if the blueprint instance event is to mark the deployment
// or removal process as finished.
func (c *BlueprintInstanceEvent) AsFinish() (*container.DeploymentFinishedMessage, bool) {
	if c.FinishEvent != nil {
		return c.FinishEvent, true
	}

	return nil, false
}

// BlueprintDocumentInfo is a type that provides
// information about the location of a source blueprint document.
type BlueprintDocumentInfo struct {
	// FileSourceScheme is the file source scheme
	// to determine where the blueprint document is located.
	// This can one of the following:
	//
	// `file`: The blueprint document is located on the local file system of the Deploy Engine server.
	// `s3`: The blueprint document is located in an S3 bucket.
	// `gcs`: The blueprint document is located in a Google Cloud Storage bucket.
	// `azureblob`: The blueprint document is located in an Azure Blob Storage container.
	// `https`: The blueprint document is located via a public HTTPS URL.
	//
	// For remote source authentication, the Deploy Engine server will need to be configured
	// with the appropriate credentials to access the remote source.
	// Authentication is not supported `https` sources.
	//
	// If not provided, the default value of `file` will be used.
	FileSourceScheme string `json:"fileSourceScheme"`
	// Directory where the blueprint document is located.
	// For `file` sources, this must be an absolute path to the directory
	// on the local file system of the Deploy Engine server.
	// An example for a `file` source would be `/path/to/blueprint-directory`.
	// For `s3`, `gcs` and `azureblob` sources, this must be the path to the
	// virtual directory where the first path segment is the bucket/container name
	// and the rest of the path is the path to the virtual directory.
	//
	// An example for a remote object storage source would be
	/// `bucket-name/path/to/blueprint-directory`.
	// For `https` sources, this must be the base URL to the blueprint document
	// excluding the scheme.
	// An example for a `https` source would be `example.com/path/to/blueprint-directory`.
	Directory string `json:"directory"`
	// BlueprintFile is the name of the blueprint file to validate.
	//
	// If not provided, the default value of `project.blueprint.yml` will be used.
	BlueprintFile string `json:"blueprintFile"`
	// BlueprintLocationMetadata is a mapping of string keys to
	// scalar values that hold additional information about the location
	// of the blueprint document.
	// For example, this can be used to specify the region of the bucket/container
	// where the blueprint document is located in a cloud storage service.
	BlueprintLocationMetadata map[string]any `json:"blueprintLocationMetadata"`
}

// BlueprintOperationConfig is the data type for configuration that can be provided
// in HTTP requests for actions that are carried out for blueprints.
// These values will be merged with the default values either defined in
// plugins or in the blueprint itself.
type BlueprintOperationConfig struct {
	Providers          map[string]map[string]*core.ScalarValue `json:"providers"`
	Transformers       map[string]map[string]*core.ScalarValue `json:"transformers"`
	ContextVariables   map[string]*core.ScalarValue            `json:"contextVariables"`
	BlueprintVariables map[string]*core.ScalarValue            `json:"blueprintVariables"`
	Dependencies       map[string]string                       `json:"dependencies,omitempty"`
}

// StreamErrorMessageEvent holds the data for an error event
// that is sent to a stream when an unexpected error occurs
// in the change staging or deployment processes.
type StreamErrorMessageEvent struct {
	// ID of the error event,
	// this is useful for clients that want to persist events
	// to a database or other storage.
	ID          string             `json:"id"`
	Message     string             `json:"message"`
	Diagnostics []*core.Diagnostic `json:"diagnostics"`
	Timestamp   int64              `json:"timestamp"`
}

// CheckReconciliationPayload represents the payload for checking
// reconciliation status of a blueprint instance.
type CheckReconciliationPayload struct {
	BlueprintDocumentInfo
	// Scope controls which elements to check.
	// Valid values: "all" (default), "interrupted", "specific"
	Scope string `json:"scope"`
	// ResourceNames specifies which resources to check when Scope is "specific".
	// Ignored for other scopes.
	ResourceNames []string `json:"resourceNames,omitempty"`
	// LinkNames specifies which links to check when Scope is "specific".
	// Ignored for other scopes.
	LinkNames []string `json:"linkNames,omitempty"`
	// Config values for the reconciliation check
	// that will be used in plugins.
	Config *BlueprintOperationConfig `json:"config"`
}

// ApplyReconciliationPayload represents the payload for applying
// reconciliation actions to a blueprint instance.
type ApplyReconciliationPayload struct {
	BlueprintDocumentInfo
	// ResourceActions specifies the actions to take for each resource.
	ResourceActions []ResourceReconcileActionPayload `json:"resourceActions,omitempty"`
	// LinkActions specifies the actions to take for each link.
	LinkActions []LinkReconcileActionPayload `json:"linkActions,omitempty"`
	// Config values for the reconciliation apply
	// that will be used in plugins.
	Config *BlueprintOperationConfig `json:"config"`
}

// ResourceReconcileActionPayload specifies the action to take for a resource.
type ResourceReconcileActionPayload struct {
	// ResourceID is the unique identifier for the resource.
	ResourceID string `json:"resourceId"`
	// Action is the reconciliation action to apply.
	// Valid values: "accept_external", "update_status", "mark_failed"
	Action string `json:"action"`
	// ExternalState is required when Action is "accept_external".
	// This is the state that will be persisted.
	ExternalState *core.MappingNode `json:"externalState,omitempty"`
	// NewStatus is the status to set for the resource.
	NewStatus string `json:"newStatus"`
}

// LinkReconcileActionPayload specifies the action to take for a link.
type LinkReconcileActionPayload struct {
	// LinkID is the unique identifier for the link.
	LinkID string `json:"linkId"`
	// Action is the reconciliation action to apply.
	// Valid values: "accept_external", "update_status", "mark_failed"
	Action string `json:"action"`
	// NewStatus is the status to set for the link.
	NewStatus string `json:"newStatus"`
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
	Action string `json:"action"`
	// ExternalState is required when Action is "accept_external".
	// This is the state that will be persisted.
	ExternalState *core.MappingNode `json:"externalState,omitempty"`
	// NewStatus is the status to set for the intermediary resource.
	NewStatus string `json:"newStatus"`
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

// DriftDetectedEventData holds the data for a drift detected event
// that is sent to a change staging stream when drift or interrupted state is detected.
type DriftDetectedEventData struct {
	// Message explains what was detected.
	Message string `json:"message"`
	// ReconciliationResult contains the full reconciliation check result.
	ReconciliationResult *container.ReconciliationCheckResult `json:"reconciliationResult"`
	// Timestamp is the unix timestamp when drift was detected.
	Timestamp int64 `json:"timestamp"`
}

const (
	// ChangeStagingEventTypeDriftDetected is the type of change staging event
	// that is sent to a change staging stream when drift or interrupted state
	// is detected during change staging.
	ChangeStagingEventTypeDriftDetected ChangeStagingEventType = "driftDetected"
)
