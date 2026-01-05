package types

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ChangesetResponse wraps a Changeset with streaming metadata.
type ChangesetResponse struct {
	// LastEventID is the ID of the last event for the channel before this operation started.
	// Clients should pass this to streaming methods to avoid missing events.
	// Omitted from JSON if no events have been generated yet for the channel.
	LastEventID string `json:"lastEventId,omitempty"`
	// Data contains the Changeset.
	Data *manage.Changeset `json:"data"`
}

// BlueprintValidationResponse wraps a BlueprintValidation with streaming metadata.
type BlueprintValidationResponse struct {
	// LastEventID is the ID of the last event for the channel before this operation started.
	// Clients should pass this to streaming methods to avoid missing events.
	// Omitted from JSON if no events have been generated yet for the channel.
	LastEventID string `json:"lastEventId,omitempty"`
	// Data contains the BlueprintValidation.
	Data *manage.BlueprintValidation `json:"data"`
}

// BlueprintInstanceResponse wraps an InstanceState with streaming metadata.
type BlueprintInstanceResponse struct {
	// LastEventID is the ID of the last event for the channel before this operation started.
	// Clients should pass this to streaming methods to avoid missing events.
	// Omitted from JSON if no events have been generated yet for the channel.
	LastEventID string `json:"lastEventId,omitempty"`
	// Data contains the InstanceState.
	Data state.InstanceState `json:"data"`
}
