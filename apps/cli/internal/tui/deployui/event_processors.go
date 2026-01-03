package deployui

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
)

// Event processing methods for DeployModel.
// These methods handle incoming deployment events and update the model state.

func (m *DeployModel) processResourceUpdate(data *container.ResourceDeployUpdateMessage) {
	isRootResource := data.InstanceID == "" || data.InstanceID == m.instanceID
	resourcePath := m.buildResourcePath(data.InstanceID, data.ResourceName)

	item := m.lookupOrMigrateResource(resourcePath, data.ResourceName)

	if item == nil {
		item = &ResourceDeployItem{
			Name:       data.ResourceName,
			ResourceID: data.ResourceID,
			Group:      data.Group,
		}
		m.resourcesByName[resourcePath] = item
		if isRootResource {
			m.items = append(m.items, DeployItem{
				Type:     ItemTypeResource,
				Resource: item,
			})
		}
	}

	m.updateResourceItemFromEvent(item, data)
}

func (m *DeployModel) processChildUpdate(data *container.ChildDeployUpdateMessage) {
	m.trackChildInstanceMapping(data)
	childPath := m.buildInstancePath(data.ParentInstanceID, data.ChildName)

	item := m.lookupOrMigrateChild(childPath, data.ChildName)

	if item == nil {
		item = m.createChildItem(data, childPath)
	}

	m.childNameToInstancePath[data.ChildName] = childPath
	m.updateChildItemFromEvent(item, data)
}

func (m *DeployModel) processLinkUpdate(data *container.LinkDeployUpdateMessage) {
	isRootLink := data.InstanceID == "" || data.InstanceID == m.instanceID
	linkPath := m.buildResourcePath(data.InstanceID, data.LinkName)

	item := m.lookupOrMigrateLink(linkPath, data.LinkName)

	if item == nil {
		item = &LinkDeployItem{
			LinkID:        data.LinkID,
			LinkName:      data.LinkName,
			ResourceAName: extractResourceAFromLinkName(data.LinkName),
			ResourceBName: extractResourceBFromLinkName(data.LinkName),
		}
		m.linksByName[linkPath] = item
		if isRootLink {
			m.items = append(m.items, DeployItem{
				Type: ItemTypeLink,
				Link: item,
			})
		}
	}

	m.updateLinkItemFromEvent(item, data)
}

func (m *DeployModel) processInstanceUpdate(data *container.DeploymentUpdateMessage) {
	m.footerRenderer.CurrentStatus = data.Status

	if IsRollingBackOrFailedStatus(data.Status) && !m.finished {
		m.markPendingItemsAsSkipped()
		m.markInProgressItemsAsInterrupted()
	}
}

func (m *DeployModel) processPreRollbackState(data *container.PreRollbackStateMessage) {
	m.preRollbackState = data
	m.footerRenderer.HasPreRollbackState = true
}

// Lookup helpers with migration support

func (m *DeployModel) lookupOrMigrateResource(path, name string) *ResourceDeployItem {
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

func (m *DeployModel) lookupOrMigrateChild(path, name string) *ChildDeployItem {
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

func (m *DeployModel) lookupOrMigrateLink(path, name string) *LinkDeployItem {
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

func (m *DeployModel) updateResourceItemFromEvent(item *ResourceDeployItem, data *container.ResourceDeployUpdateMessage) {
	status, preciseStatus := data.Status, data.PreciseStatus
	if IsInterruptedResourceStatus(data.Status) {
		status, preciseStatus = determineResourceInterruptedStatusFromAction(item.Action, data.Status)
	}
	item.Status = status
	item.PreciseStatus = preciseStatus
	item.FailureReasons = data.FailureReasons
	item.Attempt = data.Attempt
	item.CanRetry = data.CanRetry
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

func (m *DeployModel) updateChildItemFromEvent(item *ChildDeployItem, data *container.ChildDeployUpdateMessage) {
	status := data.Status
	if IsInterruptedInstanceStatus(data.Status) {
		status = determineChildInterruptedStatusFromAction(item.Action, data.Status)
	}
	item.Status = status
	item.FailureReasons = data.FailureReasons
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

func (m *DeployModel) updateLinkItemFromEvent(item *LinkDeployItem, data *container.LinkDeployUpdateMessage) {
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

func (m *DeployModel) trackChildInstanceMapping(data *container.ChildDeployUpdateMessage) {
	if data.ChildInstanceID != "" && data.ChildName != "" {
		m.instanceIDToChildName[data.ChildInstanceID] = data.ChildName
		m.instanceIDToParentID[data.ChildInstanceID] = data.ParentInstanceID
	}
}

func (m *DeployModel) createChildItem(data *container.ChildDeployUpdateMessage, childPath string) *ChildDeployItem {
	isDirectChildOfRoot := data.ParentInstanceID == "" || data.ParentInstanceID == m.instanceID

	childChanges := m.getChildChanges(data.ChildName)

	item := &ChildDeployItem{
		Name:             data.ChildName,
		ParentInstanceID: data.ParentInstanceID,
		ChildInstanceID:  data.ChildInstanceID,
		Group:            data.Group,
		Changes:          childChanges,
	}
	m.childrenByName[childPath] = item

	if isDirectChildOfRoot {
		m.items = append(m.items, DeployItem{
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

func (m *DeployModel) getChildChanges(childName string) *changes.BlueprintChanges {
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

// buildInstancePath builds a path from instance ID to the child name.
// For root instance resources, returns just the name.
// For nested children, returns a path like "parentChild/childName".
func (m *DeployModel) buildInstancePath(parentInstanceID, childName string) string {
	if parentInstanceID == "" || parentInstanceID == m.instanceID {
		return childName
	}

	pathParts := m.buildParentChain(parentInstanceID)
	pathParts = append(pathParts, childName)
	return joinPath(pathParts)
}

// buildResourcePath builds a path for a resource based on its instance ID.
// For root instance resources, returns just the resource name.
// For nested resources, returns a path like "parentChild/childName/resourceName".
func (m *DeployModel) buildResourcePath(instanceID, resourceName string) string {
	if instanceID == "" || instanceID == m.instanceID {
		return resourceName
	}

	pathParts := m.buildParentChain(instanceID)
	pathParts = append(pathParts, resourceName)
	return joinPath(pathParts)
}

func (m *DeployModel) buildParentChain(startInstanceID string) []string {
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

func joinPath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "/" + parts[i]
	}
	return result
}
