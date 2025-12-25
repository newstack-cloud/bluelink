package testutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
)

type ChangeStagingEvent struct {
	ResourceChangesEvent  *container.ResourceChangesMessage
	ChildChangesEvent     *container.ChildChangesMessage
	LinkChangesEvent      *container.LinkChangesMessage
	FinalBlueprintChanges *changes.BlueprintChanges
	DriftDetectedEvent    *DriftDetectedEventData
	Error                 error
}

// DriftDetectedEventData represents the data from a drift detected event.
type DriftDetectedEventData struct {
	Message              string                                `json:"message"`
	ReconciliationResult *container.ReconciliationCheckResult `json:"reconciliationResult"`
	Timestamp            int64                                 `json:"timestamp"`
}

type DeployEventWrapper struct {
	DeployEvent *container.DeployEvent
	DeployError error
}
