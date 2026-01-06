package shared

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// BlueprintEventHandlers holds callback functions for handling blueprint instance events.
type BlueprintEventHandlers struct {
	OnResourceUpdate func(data *container.ResourceDeployUpdateMessage)
	OnChildUpdate    func(data *container.ChildDeployUpdateMessage)
	OnLinkUpdate     func(data *container.LinkDeployUpdateMessage)
	OnInstanceUpdate func(data *container.DeploymentUpdateMessage)
	// OnPreRollbackState is optional - only used by deploy model
	OnPreRollbackState func(data *container.PreRollbackStateMessage)
}

// DispatchBlueprintEvent dispatches a blueprint instance event to the appropriate handler.
// Returns true if the event was handled, false otherwise.
func DispatchBlueprintEvent(event *types.BlueprintInstanceEvent, handlers BlueprintEventHandlers) bool {
	if resourceData, ok := event.AsResourceUpdate(); ok {
		if handlers.OnResourceUpdate != nil {
			handlers.OnResourceUpdate(resourceData)
		}
		return true
	}

	if childData, ok := event.AsChildUpdate(); ok {
		if handlers.OnChildUpdate != nil {
			handlers.OnChildUpdate(childData)
		}
		return true
	}

	if linkData, ok := event.AsLinkUpdate(); ok {
		if handlers.OnLinkUpdate != nil {
			handlers.OnLinkUpdate(linkData)
		}
		return true
	}

	if instanceData, ok := event.AsInstanceUpdate(); ok {
		if handlers.OnInstanceUpdate != nil {
			handlers.OnInstanceUpdate(instanceData)
		}
		return true
	}

	if preRollbackData, ok := event.AsPreRollbackState(); ok {
		if handlers.OnPreRollbackState != nil {
			handlers.OnPreRollbackState(preRollbackData)
		}
		return true
	}

	return false
}
