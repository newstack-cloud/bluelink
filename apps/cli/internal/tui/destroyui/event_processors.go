package destroyui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// Event processing methods for DestroyModel.

func (m *DestroyModel) processEvent(event *types.BlueprintInstanceEvent) {
	printHeadless := m.headlessMode && !m.jsonMode
	shared.DispatchBlueprintEvent(event, shared.BlueprintEventHandlers{
		OnResourceUpdate: func(data *container.ResourceDeployUpdateMessage) {
			m.processResourceUpdate(data)
			if printHeadless {
				m.printHeadlessResourceEvent(data)
			}
		},
		OnChildUpdate: func(data *container.ChildDeployUpdateMessage) {
			m.processChildUpdate(data)
			if printHeadless {
				m.printHeadlessChildEvent(data)
			}
		},
		OnLinkUpdate: func(data *container.LinkDeployUpdateMessage) {
			m.processLinkUpdate(data)
			if printHeadless {
				m.printHeadlessLinkEvent(data)
			}
		},
		OnInstanceUpdate: m.processInstanceUpdate,
	})
}

func (m *DestroyModel) processResourceUpdate(data *container.ResourceDeployUpdateMessage) {
	isRootResource := data.InstanceID == "" || data.InstanceID == m.instanceID
	resourcePath := m.buildItemPath(data.InstanceID, data.ResourceName)

	item := m.lookupOrMigrateResource(resourcePath, data.ResourceName)

	if item == nil {
		item = &ResourceDestroyItem{
			Name:       data.ResourceName,
			ResourceID: data.ResourceID,
			Group:      data.Group,
		}
		// Try to get resource type from pre-destroy instance state
		if m.preDestroyInstanceState != nil {
			if resourceType := m.lookupResourceTypeFromState(data.InstanceID, data.ResourceName); resourceType != "" {
				item.ResourceType = resourceType
			}
		}
		m.resourcesByName[resourcePath] = item
		if isRootResource {
			m.items = append(m.items, DestroyItem{
				Type:     ItemTypeResource,
				Resource: item,
			})
		}
	}

	m.updateResourceItemFromEvent(item, data)
}

func (m *DestroyModel) processChildUpdate(data *container.ChildDeployUpdateMessage) {
	m.trackChildInstanceMapping(data)
	childPath := m.buildInstancePath(data.ParentInstanceID, data.ChildName)

	item := m.lookupOrMigrateChild(childPath, data.ChildName)

	if item == nil {
		item = m.createChildItem(data, childPath)
	}

	m.childNameToInstancePath[data.ChildName] = childPath
	m.updateChildItemFromEvent(item, data)
}

func (m *DestroyModel) processLinkUpdate(data *container.LinkDeployUpdateMessage) {
	isRootLink := data.InstanceID == "" || data.InstanceID == m.instanceID
	linkPath := m.buildItemPath(data.InstanceID, data.LinkName)

	item := m.lookupOrMigrateLink(linkPath, data.LinkName)

	if item == nil {
		item = &LinkDestroyItem{
			LinkID:        data.LinkID,
			LinkName:      data.LinkName,
			ResourceAName: extractResourceAFromLinkName(data.LinkName),
			ResourceBName: extractResourceBFromLinkName(data.LinkName),
		}
		m.linksByName[linkPath] = item
		if isRootLink {
			m.items = append(m.items, DestroyItem{
				Type: ItemTypeLink,
				Link: item,
			})
		}
	}

	m.updateLinkItemFromEvent(item, data)
}

func (m *DestroyModel) processInstanceUpdate(data *container.DeploymentUpdateMessage) {
	m.footerRenderer.CurrentStatus = data.Status

	if IsRollingBackOrFailedStatus(data.Status) && !m.finished {
		m.markPendingItemsAsSkipped()
		m.markInProgressItemsAsInterrupted()
	}
}

// Lookup helpers with migration support

func (m *DestroyModel) lookupOrMigrateResource(path, name string) *ResourceDestroyItem {
	if item, exists := m.resourcesByName[path]; exists {
		return item
	}
	if item, exists := m.resourcesByName[name]; exists {
		delete(m.resourcesByName, name)
		m.resourcesByName[path] = item
		return item
	}
	return nil
}

func (m *DestroyModel) lookupOrMigrateChild(path, name string) *ChildDestroyItem {
	if item, exists := m.childrenByName[path]; exists {
		return item
	}
	if item, exists := m.childrenByName[name]; exists {
		delete(m.childrenByName, name)
		m.childrenByName[path] = item
		return item
	}
	return nil
}

func (m *DestroyModel) lookupOrMigrateLink(path, name string) *LinkDestroyItem {
	if item, exists := m.linksByName[path]; exists {
		return item
	}
	if item, exists := m.linksByName[name]; exists {
		delete(m.linksByName, name)
		m.linksByName[path] = item
		return item
	}
	return nil
}

// Update helpers

func (m *DestroyModel) updateResourceItemFromEvent(item *ResourceDestroyItem, data *container.ResourceDeployUpdateMessage) {
	item.Status = data.Status
	item.PreciseStatus = data.PreciseStatus
	item.FailureReasons = data.FailureReasons
	item.Attempt = data.Attempt
	item.CanRetry = data.CanRetry
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

func (m *DestroyModel) updateChildItemFromEvent(item *ChildDestroyItem, data *container.ChildDeployUpdateMessage) {
	item.Status = data.Status
	item.FailureReasons = data.FailureReasons
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

func (m *DestroyModel) updateLinkItemFromEvent(item *LinkDestroyItem, data *container.LinkDeployUpdateMessage) {
	item.Status = data.Status
	item.PreciseStatus = data.PreciseStatus
	item.FailureReasons = data.FailureReasons
	item.CurrentStageAttempt = data.CurrentStageAttempt
	item.CanRetryCurrentStage = data.CanRetryCurrentStage
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

// Child item creation helper

func (m *DestroyModel) trackChildInstanceMapping(data *container.ChildDeployUpdateMessage) {
	if data.ChildInstanceID != "" && data.ChildName != "" {
		m.instanceIDToChildName[data.ChildInstanceID] = data.ChildName
		m.instanceIDToParentID[data.ChildInstanceID] = data.ParentInstanceID
	}
}

func (m *DestroyModel) createChildItem(data *container.ChildDeployUpdateMessage, childPath string) *ChildDestroyItem {
	isDirectChildOfRoot := data.ParentInstanceID == "" || data.ParentInstanceID == m.instanceID

	childChanges := m.getChildChanges(data.ChildName)

	item := &ChildDestroyItem{
		Name:             data.ChildName,
		ParentInstanceID: data.ParentInstanceID,
		ChildInstanceID:  data.ChildInstanceID,
		Group:            data.Group,
		Changes:          childChanges,
	}
	m.childrenByName[childPath] = item

	if isDirectChildOfRoot {
		m.items = append(m.items, DestroyItem{
			Type:            ItemTypeChild,
			Child:           item,
			Changes:         childChanges,
			childrenByName:  m.childrenByName,
			resourcesByName: m.resourcesByName,
			linksByName:     m.linksByName,
		})
	}

	return item
}

func (m *DestroyModel) getChildChanges(childName string) *changes.BlueprintChanges {
	if m.changesetChanges == nil {
		return nil
	}

	if nc, ok := m.changesetChanges.NewChildren[childName]; ok {
		return &changes.BlueprintChanges{
			NewResources: nc.NewResources,
			NewChildren:  nc.NewChildren,
		}
	}

	if cc, ok := m.changesetChanges.ChildChanges[childName]; ok {
		ccCopy := cc
		return &ccCopy
	}

	return nil
}

// Path building helpers

func (m *DestroyModel) buildInstancePath(parentInstanceID, childName string) string {
	if parentInstanceID == "" || parentInstanceID == m.instanceID {
		return childName
	}

	pathParts := m.buildParentChain(parentInstanceID)
	pathParts = append(pathParts, childName)
	return shared.JoinPath(pathParts)
}

func (m *DestroyModel) buildItemPath(instanceID, itemName string) string {
	if instanceID == "" || instanceID == m.instanceID {
		return itemName
	}

	pathParts := m.buildParentChain(instanceID)
	pathParts = append(pathParts, itemName)
	return shared.JoinPath(pathParts)
}

func (m *DestroyModel) buildParentChain(startInstanceID string) []string {
	var pathParts []string
	currentID := startInstanceID
	for currentID != "" && currentID != m.instanceID {
		if name, ok := m.instanceIDToChildName[currentID]; ok {
			pathParts = append([]string{name}, pathParts...)
			currentID = m.instanceIDToParentID[currentID]
		} else {
			break
		}
	}
	return pathParts
}

// lookupResourceTypeFromState finds the resource type by traversing the pre-destroy instance state.
func (m *DestroyModel) lookupResourceTypeFromState(instanceID, resourceName string) string {
	if m.preDestroyInstanceState == nil {
		return ""
	}

	// Find the correct instance state based on instanceID
	var targetState = m.preDestroyInstanceState
	if instanceID != "" && instanceID != m.instanceID {
		// Need to traverse to find the child instance
		targetState = m.findInstanceStateByID(m.preDestroyInstanceState, instanceID)
		if targetState == nil {
			return ""
		}
	}

	// Look up the resource in the target instance state
	if targetState.ResourceIDs == nil || targetState.Resources == nil {
		return ""
	}

	resourceID, ok := targetState.ResourceIDs[resourceName]
	if !ok {
		return ""
	}

	resourceState, ok := targetState.Resources[resourceID]
	if !ok || resourceState == nil {
		return ""
	}

	return resourceState.Type
}

// findInstanceStateByID recursively searches for an instance state by its ID.
func (m *DestroyModel) findInstanceStateByID(currentState *state.InstanceState, targetID string) *state.InstanceState {
	if currentState == nil {
		return nil
	}

	if currentState.InstanceID == targetID {
		return currentState
	}

	for _, childState := range currentState.ChildBlueprints {
		if found := m.findInstanceStateByID(childState, targetID); found != nil {
			return found
		}
	}

	return nil
}
