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
	Error                 error
}

type DeployEventWrapper struct {
	DeployEvent *container.DeployEvent
	DeployError error
}
