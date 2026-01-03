package container

import (
	"encoding/json"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ResourceDeployUpdateMessage provides a message containing status updates
// for resources being deployed.
// Deployment messages report on status changes for resources,
// the state of a resource will need to be fetched from the state container
// to get further information about the state of the resource.
type ResourceDeployUpdateMessage struct {
	// InstanceID is the ID of the blueprint instance
	// the message is associated with.
	// As updates are sent for parent and child blueprints,
	// this ID is used to differentiate between them.
	InstanceID string `json:"instanceId"`
	// ResourceID is the globally unique ID of the resource.
	ResourceID string `json:"resourceId"`
	// ResourceName is the logical name of the resource
	// as defined in the source blueprint.
	ResourceName string `json:"resourceName"`
	// Group is the group number the resource belongs to relative to the ordering
	// for components in the current blueprint associated with the instance ID.
	// A group is a collection of items that can be deployed or destroyed at the same time.
	Group int `json:"group"`
	// Status holds the high-level status of the resource.
	Status core.ResourceStatus `json:"status"`
	// PreciseStatus holds the detailed status of the resource.
	PreciseStatus core.PreciseResourceStatus `json:"preciseStatus"`
	// FailureReasons holds a list of reasons why the resource failed to deploy
	// if the status update is for a failure.
	FailureReasons []string `json:"failureReasons,omitempty"`
	// Attempt is the current attempt number for deploying or destroying the resource.
	Attempt int `json:"attempt"`
	// CanRetry indicates if the operation for the resource can be retried
	// after this attempt.
	CanRetry bool `json:"canRetry"`
	// UpdateTimestamp is the unix timestamp in seconds for
	// when the status update occurred.
	UpdateTimestamp int64 `json:"updateTimestamp"`
	// Durations holds duration information for a resource deployment.
	// Duration information is attached on one of the following precise status updates:
	// - PreciseResourceStatusConfigComplete
	// - PreciseResourceStatusCreated
	// - PreciseResourceStatusCreateFailed
	// - PreciseResourceStatusCreateRollbackFailed
	// - PreciseResourceStatusCreateRollbackComplete
	// - PreciseResourceStatusDestroyed
	// - PreciseResourceStatusDestroyFailed
	// - PreciseResourceStatusDestroyRollbackFailed
	// - PreciseResourceStatusDestroyRollbackConfigComplete
	// - PreciseResourceStatusDestroyRollbackComplete
	// - PreciseResourceStatusUpdateConfigComplete
	// - PreciseResourceStatusUpdated
	// - PreciseResourceStatusUpdateFailed
	// - PreciseResourceStatusUpdateRollbackFailed
	// - PreciseResourceStatusUpdateRollbackConfigComplete
	// - PreciseResourceStatusUpdateRollbackComplete
	Durations *state.ResourceCompletionDurations `json:"durations,omitempty"`
}

// ResourceChangesMessage provides a message containing status updates
// for resources being deployed.
// Deployment messages report on status changes for resources,
// the state of a resource will need to be fetched from the state container
// to get further information about the state of the resource.
type LinkDeployUpdateMessage struct {
	// InstanceID is the ID of the blueprint instance
	// the message is associated with.
	// As updates are sent for parent and child blueprints,
	// this ID is used to differentiate between them.
	InstanceID string `json:"instanceId"`
	// LinkID is the globally unique ID of the link.
	LinkID string `json:"linkId"`
	// LinkName is the logic name of the link in the blueprint.
	// This is a combination of the 2 resources that are linked.
	// For example, if a link is between a VPC and a subnet,
	// the link name would be "vpc::subnet".
	LinkName string `json:"linkName"`
	// Status holds the high-level status of the link.
	Status core.LinkStatus `json:"status"`
	// PreciseStatus holds the detailed status of the link.
	PreciseStatus core.PreciseLinkStatus `json:"preciseStatus"`
	// FailureReasons holds a list of reasons why the link failed to deploy
	// if the status update is for a failure.
	FailureReasons []string `json:"failureReasons,omitempty"`
	// Attempt is the current attempt number for applying the changes
	// for the current stage of the link deployment/removal.
	CurrentStageAttempt int `json:"currentStageAttempt"`
	// CanRetryCurrentStage indicates if the operation for the link can be retried
	// after this attempt of the current stage.
	CanRetryCurrentStage bool `json:"canRetryCurrentStage"`
	// UpdateTimestamp is the unix timestamp in seconds for
	// when the status update occurred.
	UpdateTimestamp int64 `json:"updateTimestamp"`
	// Durations holds duration information for a link deployment.
	// Duration information is attached on one of the following precise status updates:
	// - PreciseLinkStatusResourceAUpdated
	// - PreciseLinkStatusResourceAUpdateFailed
	// - PreciseLinkStatusResourceAUpdateRollbackFailed
	// - PreciseLinkStatusResourceAUpdateRollbackComplete
	// - PreciseLinkStatusResourceBUpdated
	// - PreciseLinkStatusResourceBUpdateFailed
	// - PreciseLinkStatusResourceBUpdateRollbackFailed
	// - PreciseLinkStatusResourceBUpdateRollbackComplete
	// - PreciseLinkStatusIntermediaryResourcesUpdated
	// - PreciseLinkStatusIntermediaryResourceUpdateFailed
	// - PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed
	// - PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete
	Durations *state.LinkCompletionDurations `json:"durations,omitempty"`
}

// ChildDeployUpdateMessage provides a message containing status updates
// for child blueprints being deployed.
// Deployment messages report on status changes for child blueprints,
// the state of a child blueprint will need to be fetched from the state container
// to get further information about the state of the child blueprint.
type ChildDeployUpdateMessage struct {
	// ParentInstanceID is the ID of the parent blueprint instance
	// the message is associated with.
	ParentInstanceID string `json:"parentInstanceId"`
	// ChildInstanceID is the ID of the child blueprint instance
	// the message is associated with.
	ChildInstanceID string `json:"childInstanceId"`
	// ChildName is the logical name of the child blueprint
	// as defined in the source blueprint as an include.
	ChildName string `json:"childName"`
	// Group is the group number the child blueprint belongs to relative to the ordering
	// for components in the current blueprint associated with the parent instance ID.
	Group int `json:"group"`
	// Status holds the status of the child blueprint.
	Status core.InstanceStatus `json:"status"`
	// FailureReasons holds a list of reasons why the child blueprint failed to deploy
	// if the status update is for a failure.
	FailureReasons []string `json:"failureReasons,omitempty"`
	// UpdateTimestamp is the unix timestamp in seconds for
	// when the status update occurred.
	UpdateTimestamp int64 `json:"updateTimestamp"`
	// Durations holds duration information for a child blueprint deployment.
	// Duration information is attached on one of the following status updates:
	// - InstanceStatusDeployed
	// - InstanceStatusDeployFailed
	// - InstanceStatusDestroyed
	// - InstanceStatusUpdated
	// - InstanceStatusUpdateFailed
	Durations *state.InstanceCompletionDuration `json:"durations,omitempty"`
}

// DeploymentUpdateMessage provides a message containing a blueprint-wide status update
// for the deployment of a blueprint instance.
type DeploymentUpdateMessage struct {
	// InstanceID is the ID of the blueprint instance
	// the message is associated with.
	InstanceID string `json:"instanceId"`
	// Status holds the status of the instance deployment.
	Status core.InstanceStatus `json:"status"`
	// UpdateTimestamp is the unix timestamp in seconds for
	// when the status update occurred.
	UpdateTimestamp int64 `json:"updateTimestamp"`
}

// DeploymentFinishedMessage provides a message containing the final status
// of the blueprint instance deployment.
type DeploymentFinishedMessage struct {
	// InstanceID is the ID of the blueprint instance
	// the message is associated with.
	InstanceID string `json:"instanceId"`
	// Status holds the status of the instance deployment.
	Status core.InstanceStatus `json:"status"`
	// FailureReasons holds a list of reasons why the instance failed to deploy
	// if the final status is a failure.
	FailureReasons []string `json:"failureReasons,omitempty"`
	// FinishTimestamp is the unix timestamp in seconds for
	// when the deployment finished.
	FinishTimestamp int64 `json:"finishTimestamp"`
	// UpdateTimestamp is the unix timestamp in seconds for
	// when the status update occurred.
	UpdateTimestamp int64 `json:"updateTimestamp"`
	// Durations holds duration information for the blueprint deployment.
	// Duration information is attached on one of the following status updates:
	// - InstanceStatusDeploying (preparation phase duration only)
	// - InstanceStatusDeployed
	// - InstanceStatusDeployFailed
	// - InstanceStatusDeployRollbackFailed
	// - InstanceStatusDeployRollbackComplete
	// - InstanceStatusDestroyed
	// - InstanceStatusDestroyFailed
	// - InstanceStatusDestroyRollbackFailed
	// - InstanceStatusDestroyRollbackComplete
	// - InstanceStatusUpdated
	// - InstanceStatusUpdateFailed
	// - InstanceStatusUpdateRollbackFailed
	// - InstanceStatusUpdateRollbackComplete
	Durations *state.InstanceCompletionDuration `json:"durations,omitempty"`
	// EndOfStream indicates whether this finish event marks the end of the event stream.
	// When false, more events will follow (e.g., auto-rollback events after a failed deployment).
	// When true, no more events will be sent and clients should close their event stream connection.
	EndOfStream bool `json:"endOfStream"`
	// SkippedRollbackItems contains resources and links that were skipped during
	// auto-rollback because they were not in a safe state to rollback.
	// This is only populated for rollback completion events.
	SkippedRollbackItems []SkippedRollbackItem `json:"skippedRollbackItems,omitempty"`
}

// SkippedRollbackItem represents a resource or link that was skipped during
// rollback because it was not in a safe state to roll back.
type SkippedRollbackItem struct {
	// Name is the resource or link name.
	Name string `json:"name"`
	// Type indicates whether this is a "resource" or "link".
	Type string `json:"type"`
	// ChildPath is the path to the child blueprint containing this item,
	// empty string for root-level items.
	ChildPath string `json:"childPath,omitempty"`
	// Status is the current status that prevented rollback.
	Status string `json:"status"`
	// Reason explains why the item was skipped.
	Reason string `json:"reason"`
}

// PreRollbackStateMessage provides a snapshot of instance state before auto-rollback begins.
// This captures the failed deployment state for debugging/auditing before resources are destroyed.
type PreRollbackStateMessage struct {
	// InstanceID is the ID of the blueprint instance.
	InstanceID string `json:"instanceId"`
	// InstanceName is the user-provided name of the blueprint instance.
	InstanceName string `json:"instanceName"`
	// Status is the failed status that triggered rollback (e.g., DeployFailed).
	Status core.InstanceStatus `json:"status"`
	// Resources contains snapshots of all resource states before rollback.
	Resources []ResourceSnapshot `json:"resources"`
	// Links contains snapshots of all link states before rollback.
	Links []LinkSnapshot `json:"links"`
	// Children contains snapshots of all child blueprint states before rollback.
	// Each child snapshot recursively includes its own resources, links, and children.
	Children []ChildSnapshot `json:"children"`
	// FailureReasons contains the reasons for the deployment failure.
	FailureReasons []string `json:"failureReasons"`
	// CapturedAt is the unix timestamp in seconds when the state was captured.
	CapturedAt int64 `json:"capturedAt"`
}

// ResourceSnapshot provides a snapshot of a resource's state before rollback.
type ResourceSnapshot struct {
	// ResourceID is the globally unique ID of the resource.
	ResourceID string `json:"resourceId"`
	// ResourceName is the logical name of the resource in the blueprint.
	ResourceName string `json:"resourceName"`
	// ResourceType is the type of the resource (e.g., "aws/ec2/instance").
	ResourceType string `json:"resourceType"`
	// Status is the high-level status of the resource.
	Status core.ResourceStatus `json:"status"`
	// PreciseStatus is the detailed status of the resource.
	PreciseStatus core.PreciseResourceStatus `json:"preciseStatus"`
	// FailureReasons contains reasons for failure if the resource failed.
	FailureReasons []string `json:"failureReasons,omitempty"`
	// SpecData holds the resolved resource spec including computed fields.
	// This contains the resource outputs/attributes from the provider.
	SpecData *core.MappingNode `json:"specData,omitempty"`
	// ComputedFields lists field paths that are computed at deploy time by the provider.
	ComputedFields []string `json:"computedFields,omitempty"`
}

// LinkSnapshot provides a snapshot of a link's state before rollback.
type LinkSnapshot struct {
	// LinkID is the globally unique ID of the link.
	LinkID string `json:"linkId"`
	// LinkName is the logical name of the link (e.g., "vpc::subnet").
	LinkName string `json:"linkName"`
	// Status is the high-level status of the link.
	Status core.LinkStatus `json:"status"`
	// PreciseStatus is the detailed status of the link.
	PreciseStatus core.PreciseLinkStatus `json:"preciseStatus"`
	// FailureReasons contains reasons for failure if the link failed.
	FailureReasons []string `json:"failureReasons,omitempty"`
}

// ChildSnapshot provides a snapshot of a child blueprint's state before rollback.
// This is recursive - each child includes its own resources, links, and nested children.
type ChildSnapshot struct {
	// ChildInstanceID is the ID of the child blueprint instance.
	ChildInstanceID string `json:"childInstanceId"`
	// ChildName is the logical name of the child blueprint in the parent blueprint.
	ChildName string `json:"childName"`
	// Status is the status of the child blueprint instance.
	Status core.InstanceStatus `json:"status"`
	// Resources contains snapshots of resources in this child blueprint.
	Resources []ResourceSnapshot `json:"resources"`
	// Links contains snapshots of links in this child blueprint.
	Links []LinkSnapshot `json:"links"`
	// Children contains snapshots of nested child blueprints.
	Children []ChildSnapshot `json:"children"`
	// FailureReasons contains reasons for failure if the child blueprint failed.
	FailureReasons []string `json:"failureReasons,omitempty"`
}

// DeployEvent contains an event that is emitted during the deployment process.
// This is used like a sum type to represent the different types of events that can be emitted.
type DeployEvent struct {
	// ResourceUpdateEvent is an event that is emitted when a resource is updated.
	ResourceUpdateEvent *ResourceDeployUpdateMessage
	// LinkUpdateEvent is an event that is emitted when a link is updated.
	LinkUpdateEvent *LinkDeployUpdateMessage
	// ChildUpdateEvent is an event that is emitted when a child blueprint is updated.
	ChildUpdateEvent *ChildDeployUpdateMessage
	// DeploymentUpdateEvent is an event that is emitted when the
	// deployment status of the blueprint instance is updated.
	DeploymentUpdateEvent *DeploymentUpdateMessage
	// FinishEvent is an event that is emitted when the deployment
	// of the blueprint instance has finished.
	FinishEvent *DeploymentFinishedMessage
	// PreRollbackStateEvent is an event emitted before auto-rollback begins,
	// capturing the instance state for debugging/auditing purposes.
	PreRollbackStateEvent *PreRollbackStateMessage
}

type intermediaryDeployEvent struct {
	EventType EventType       `json:"type"`
	Message   json.RawMessage `json:"message"`
}

// EventType is a type that represents the different types of events that can be
// emitted during the deployment process.
type EventType string

const (
	// EventTypeResourceUpdate is an event type that represents a resource update event.
	EventTypeResourceUpdate EventType = "resourceUpdate"
	// EventTypeLinkUpdate is an event type that represents a link update event.
	EventTypeLinkUpdate EventType = "linkUpdate"
	// EventTypeChildUpdate is an event type that represents a child blueprint update event.
	EventTypeChildUpdate EventType = "childUpdate"
	// EventTypeDeploymentUpdate is an event type that represents a
	// blueprint instance deployment update event.
	EventTypeDeploymentUpdate EventType = "deploymentUpdate"
	// EventTypeFinish is an event type that represents a
	// blueprint instance deployment finish event.
	EventTypeFinish EventType = "finish"
	// EventTypePreRollbackState is an event type that represents a
	// pre-rollback state capture event, emitted before auto-rollback begins.
	EventTypePreRollbackState EventType = "preRollbackState"
)

func (e *DeployEvent) MarshalJSON() ([]byte, error) {
	if e.ResourceUpdateEvent != nil {
		return e.marshalEventMessage(
			EventTypeResourceUpdate,
			e.ResourceUpdateEvent,
		)
	}

	if e.LinkUpdateEvent != nil {
		return e.marshalEventMessage(
			EventTypeLinkUpdate,
			e.LinkUpdateEvent,
		)
	}

	if e.ChildUpdateEvent != nil {
		return e.marshalEventMessage(
			EventTypeChildUpdate,
			e.ChildUpdateEvent,
		)
	}

	if e.DeploymentUpdateEvent != nil {
		return e.marshalEventMessage(
			EventTypeDeploymentUpdate,
			e.DeploymentUpdateEvent,
		)
	}

	if e.FinishEvent != nil {
		return e.marshalEventMessage(
			EventTypeFinish,
			e.FinishEvent,
		)
	}

	if e.PreRollbackStateEvent != nil {
		return e.marshalEventMessage(
			EventTypePreRollbackState,
			e.PreRollbackStateEvent,
		)
	}

	return nil, errors.New("no event message set")
}

func (e *DeployEvent) marshalEventMessage(
	eventType EventType,
	eventMessage any,
) ([]byte, error) {
	msgBytes, err := json.Marshal(eventMessage)
	if err != nil {
		return nil, err
	}

	return json.Marshal(intermediaryDeployEvent{
		EventType: eventType,
		Message:   msgBytes,
	})
}

func (e *DeployEvent) UnmarshalJSON(data []byte) error {
	intermediaryEvent := &intermediaryDeployEvent{}
	err := json.Unmarshal(data, intermediaryEvent)
	if err != nil {
		return err
	}

	switch intermediaryEvent.EventType {
	case EventTypeResourceUpdate:
		e.ResourceUpdateEvent = &ResourceDeployUpdateMessage{}
		return json.Unmarshal(intermediaryEvent.Message, e.ResourceUpdateEvent)
	case EventTypeLinkUpdate:
		e.LinkUpdateEvent = &LinkDeployUpdateMessage{}
		return json.Unmarshal(intermediaryEvent.Message, e.LinkUpdateEvent)
	case EventTypeChildUpdate:
		e.ChildUpdateEvent = &ChildDeployUpdateMessage{}
		return json.Unmarshal(intermediaryEvent.Message, e.ChildUpdateEvent)
	case EventTypeDeploymentUpdate:
		e.DeploymentUpdateEvent = &DeploymentUpdateMessage{}
		return json.Unmarshal(intermediaryEvent.Message, e.DeploymentUpdateEvent)
	case EventTypeFinish:
		e.FinishEvent = &DeploymentFinishedMessage{}
		return json.Unmarshal(intermediaryEvent.Message, e.FinishEvent)
	case EventTypePreRollbackState:
		e.PreRollbackStateEvent = &PreRollbackStateMessage{}
		return json.Unmarshal(intermediaryEvent.Message, e.PreRollbackStateEvent)
	}

	return errors.New("no valid event type set")
}
