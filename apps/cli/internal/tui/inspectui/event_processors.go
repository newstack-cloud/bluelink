package inspectui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/deployui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// processResourceUpdate processes a resource update event from the stream.
// When a resource already has ResourceState from the initial fetch, we only update
// status if the event represents a progression (not a regression from replayed events).
func (m *InspectModel) processResourceUpdate(data *container.ResourceDeployUpdateMessage) {
	isRootResource := data.InstanceID == "" || data.InstanceID == m.instanceID
	resourcePath := m.buildResourcePath(data.InstanceID, data.ResourceName)

	item := m.lookupOrMigrateResource(resourcePath, data.ResourceName)

	if item != nil {
		m.updateResourceItemFromEvent(item, data)
	} else {
		// Get instance state for this resource's parent instance
		instanceState := m.getInstanceStateForResource(data.InstanceID)

		// Try to find ResourceState from instance state for new items
		var resourceState *state.ResourceState
		if instanceState != nil {
			resourceState = findResourceStateByName(instanceState, data.ResourceName)
		}

		// Determine the correct status - prefer state store if available
		status := data.Status
		preciseStatus := data.PreciseStatus
		if resourceState != nil {
			status = resourceState.Status
		}

		item = &deployui.ResourceDeployItem{
			Name:           data.ResourceName,
			ResourceID:     data.ResourceID,
			ResourceType:   getResourceTypeFromState(resourceState),
			Action:         shared.ActionInspect,
			Status:         status,
			PreciseStatus:  preciseStatus,
			FailureReasons: data.FailureReasons,
			Attempt:        data.Attempt,
			CanRetry:       data.CanRetry,
			Durations:      data.Durations,
			ResourceState:  resourceState,
		}
		m.resourcesByName[resourcePath] = item

		// Only add root resources to the top-level items list
		if isRootResource {
			m.items = append(m.items, deployui.DeployItem{
				Type:          deployui.ItemTypeResource,
				Resource:      item,
				InstanceState: instanceState,
			})
		}
	}

	// Update footer status from instance state if available
	if m.instanceState != nil {
		m.footerRenderer.CurrentStatus = m.instanceState.Status
	}
}

// updateResourceItemFromEvent updates an existing resource item from event data.
func (m *InspectModel) updateResourceItemFromEvent(item *deployui.ResourceDeployItem, data *container.ResourceDeployUpdateMessage) {
	// If we have authoritative ResourceState from the state store, use its status
	// to prevent replayed events from regressing the displayed status.
	if item.ResourceState != nil {
		if shouldUpdateResourceStatus(item.ResourceState.Status, data.Status) {
			item.Status = data.Status
			item.PreciseStatus = data.PreciseStatus
		}
	} else {
		item.Status = data.Status
		item.PreciseStatus = data.PreciseStatus
	}

	if data.ResourceID != "" {
		item.ResourceID = data.ResourceID
	}
	item.FailureReasons = data.FailureReasons
	item.Attempt = data.Attempt
	item.CanRetry = data.CanRetry
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

// getResourceTypeFromState extracts the resource type from ResourceState if available.
func getResourceTypeFromState(resourceState *state.ResourceState) string {
	if resourceState != nil {
		return resourceState.Type
	}
	return ""
}

// processChildUpdate processes a child blueprint update event from the stream.
func (m *InspectModel) processChildUpdate(data *container.ChildDeployUpdateMessage) {
	m.trackChildInstanceMapping(data)
	childPath := m.buildInstancePath(data.ParentInstanceID, data.ChildName)

	item := m.lookupOrMigrateChild(childPath, data.ChildName)

	if item != nil {
		m.updateChildItemFromEvent(item, data)
	} else {
		item = m.createChildItem(data, childPath)
	}

	m.childNameToInstancePath[data.ChildName] = childPath
}

// findChildState finds the child instance state from the instance state.
func (m *InspectModel) findChildState(childName string) *state.InstanceState {
	if m.instanceState == nil || m.instanceState.ChildBlueprints == nil {
		return nil
	}
	return m.instanceState.ChildBlueprints[childName]
}

// processLinkUpdate processes a link update event from the stream.
func (m *InspectModel) processLinkUpdate(data *container.LinkDeployUpdateMessage) {
	isRootLink := data.InstanceID == "" || data.InstanceID == m.instanceID
	linkPath := m.buildResourcePath(data.InstanceID, data.LinkName)

	item := m.lookupOrMigrateLink(linkPath, data.LinkName)

	if item != nil {
		m.updateLinkItemFromEvent(item, data)
	} else {
		// Get instance state for this link's parent instance
		instanceState := m.getInstanceStateForResource(data.InstanceID)

		// Try to find link state from instance state for new items
		var linkState *state.LinkState
		if instanceState != nil {
			linkState = m.findLinkStateInInstance(instanceState, data.LinkName)
		}

		// Determine the correct status - prefer state store if available
		status := data.Status
		preciseStatus := data.PreciseStatus
		linkID := data.LinkID
		if linkState != nil {
			status = linkState.Status
			if linkState.LinkID != "" {
				linkID = linkState.LinkID
			}
		}

		item = &deployui.LinkDeployItem{
			LinkID:               linkID,
			LinkName:             data.LinkName,
			ResourceAName:        extractResourceAFromLinkName(data.LinkName),
			ResourceBName:        extractResourceBFromLinkName(data.LinkName),
			Action:               shared.ActionInspect,
			Status:               status,
			PreciseStatus:        preciseStatus,
			FailureReasons:       data.FailureReasons,
			CurrentStageAttempt:  data.CurrentStageAttempt,
			CanRetryCurrentStage: data.CanRetryCurrentStage,
		}
		m.linksByName[linkPath] = item

		// Only add root links to the top-level items list
		if isRootLink {
			m.items = append(m.items, deployui.DeployItem{
				Type:          deployui.ItemTypeLink,
				Link:          item,
				InstanceState: instanceState,
			})
		}
	}
}

// updateLinkItemFromEvent updates an existing link item from event data.
func (m *InspectModel) updateLinkItemFromEvent(item *deployui.LinkDeployItem, data *container.LinkDeployUpdateMessage) {
	// Look up current link state to prevent status regression
	linkState := m.findLinkState(data.LinkName)
	if linkState == nil || shouldUpdateLinkStatus(linkState.Status, data.Status) {
		item.Status = data.Status
		item.PreciseStatus = data.PreciseStatus
	}
	if data.LinkID != "" {
		item.LinkID = data.LinkID
	}
	item.FailureReasons = data.FailureReasons
	item.CurrentStageAttempt = data.CurrentStageAttempt
	item.CanRetryCurrentStage = data.CanRetryCurrentStage
}

// findLinkState finds the link state from the instance state.
func (m *InspectModel) findLinkState(linkName string) *state.LinkState {
	if m.instanceState == nil || m.instanceState.Links == nil {
		return nil
	}
	return m.instanceState.Links[linkName]
}

// processInstanceUpdate processes an instance update event from the stream.
func (m *InspectModel) processInstanceUpdate(data *container.DeploymentUpdateMessage) {
	if m.instanceState != nil {
		m.instanceState.Status = data.Status
	}
	m.footerRenderer.CurrentStatus = data.Status
}

// shouldUpdateResourceStatus returns true if the event status should replace the current status.
// This prevents replayed historical events from regressing the displayed status.
// For example, if a resource shows "Created" from the state store, we shouldn't
// overwrite it with "Creating" from a replayed event.
func shouldUpdateResourceStatus(currentStatus, eventStatus core.ResourceStatus) bool {
	// If the event status is a terminal status (completed or failed), always update
	if isTerminalResourceStatus(eventStatus) {
		return true
	}

	// If the current status is already terminal, don't regress to in-progress
	if isTerminalResourceStatus(currentStatus) {
		return false
	}

	// Both are non-terminal, allow the update
	return true
}

// isTerminalResourceStatus returns true if the status represents a completed state.
func isTerminalResourceStatus(status core.ResourceStatus) bool {
	switch status {
	case core.ResourceStatusCreated,
		core.ResourceStatusUpdated,
		core.ResourceStatusDestroyed,
		core.ResourceStatusCreateFailed,
		core.ResourceStatusUpdateFailed,
		core.ResourceStatusDestroyFailed,
		core.ResourceStatusRollbackComplete,
		core.ResourceStatusRollbackFailed,
		core.ResourceStatusCreateInterrupted,
		core.ResourceStatusUpdateInterrupted,
		core.ResourceStatusDestroyInterrupted:
		return true
	default:
		return false
	}
}

// shouldUpdateChildStatus returns true if the event status should replace the current child status.
func shouldUpdateChildStatus(currentStatus, eventStatus core.InstanceStatus) bool {
	if isTerminalInstanceStatus(eventStatus) {
		return true
	}
	if isTerminalInstanceStatus(currentStatus) {
		return false
	}
	return true
}

// isTerminalInstanceStatus returns true if the instance status represents a completed state.
func isTerminalInstanceStatus(status core.InstanceStatus) bool {
	switch status {
	case core.InstanceStatusDeployed,
		core.InstanceStatusUpdated,
		core.InstanceStatusDestroyed,
		core.InstanceStatusDeployFailed,
		core.InstanceStatusUpdateFailed,
		core.InstanceStatusDestroyFailed,
		core.InstanceStatusDeployRollbackComplete,
		core.InstanceStatusUpdateRollbackComplete,
		core.InstanceStatusDestroyRollbackComplete,
		core.InstanceStatusDeployRollbackFailed,
		core.InstanceStatusUpdateRollbackFailed,
		core.InstanceStatusDestroyRollbackFailed,
		core.InstanceStatusDeployInterrupted,
		core.InstanceStatusUpdateInterrupted,
		core.InstanceStatusDestroyInterrupted:
		return true
	default:
		return false
	}
}

// shouldUpdateLinkStatus returns true if the event status should replace the current link status.
func shouldUpdateLinkStatus(currentStatus, eventStatus core.LinkStatus) bool {
	if isTerminalLinkStatus(eventStatus) {
		return true
	}
	if isTerminalLinkStatus(currentStatus) {
		return false
	}
	return true
}

// isTerminalLinkStatus returns true if the link status represents a completed state.
func isTerminalLinkStatus(status core.LinkStatus) bool {
	switch status {
	case core.LinkStatusCreated,
		core.LinkStatusUpdated,
		core.LinkStatusDestroyed,
		core.LinkStatusCreateFailed,
		core.LinkStatusUpdateFailed,
		core.LinkStatusDestroyFailed,
		core.LinkStatusCreateRollbackComplete,
		core.LinkStatusUpdateRollbackComplete,
		core.LinkStatusDestroyRollbackComplete,
		core.LinkStatusCreateRollbackFailed,
		core.LinkStatusUpdateRollbackFailed,
		core.LinkStatusDestroyRollbackFailed,
		core.LinkStatusCreateInterrupted,
		core.LinkStatusUpdateInterrupted,
		core.LinkStatusDestroyInterrupted:
		return true
	default:
		return false
	}
}

// Path building helpers - delegated to shared.PathBuilder

func (m *InspectModel) buildInstancePath(parentInstanceID, childName string) string {
	return m.pathBuilder.BuildInstancePath(parentInstanceID, childName)
}

func (m *InspectModel) buildResourcePath(instanceID, resourceName string) string {
	return m.pathBuilder.BuildItemPath(instanceID, resourceName)
}

// Lookup helpers with migration support

func (m *InspectModel) lookupOrMigrateResource(path, name string) *deployui.ResourceDeployItem {
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

func (m *InspectModel) lookupOrMigrateChild(path, name string) *deployui.ChildDeployItem {
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

func (m *InspectModel) lookupOrMigrateLink(path, name string) *deployui.LinkDeployItem {
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

// Child instance tracking - delegated to shared.PathBuilder

func (m *InspectModel) trackChildInstanceMapping(data *container.ChildDeployUpdateMessage) {
	m.pathBuilder.TrackChildInstance(data.ChildInstanceID, data.ChildName, data.ParentInstanceID)
}

func (m *InspectModel) createChildItem(data *container.ChildDeployUpdateMessage, childPath string) *deployui.ChildDeployItem {
	isDirectChildOfRoot := data.ParentInstanceID == "" || data.ParentInstanceID == m.instanceID

	// Try to find child state from instance state
	childState := m.findChildState(data.ChildName)

	// Determine the correct status - prefer state store if available
	status := data.Status
	if childState != nil {
		status = childState.Status
	}

	item := &deployui.ChildDeployItem{
		Name:             data.ChildName,
		ParentInstanceID: data.ParentInstanceID,
		ChildInstanceID:  data.ChildInstanceID,
		Action:           shared.ActionInspect,
		Status:           status,
		FailureReasons:   data.FailureReasons,
		Changes:          &changes.BlueprintChanges{},
	}
	m.childrenByName[childPath] = item

	if isDirectChildOfRoot {
		m.items = append(m.items, deployui.MakeChildDeployItem(
			item,
			&changes.BlueprintChanges{},
			childState,
			m.childrenByName,
			m.resourcesByName,
			m.linksByName,
		))
	}

	return item
}

func (m *InspectModel) updateChildItemFromEvent(item *deployui.ChildDeployItem, data *container.ChildDeployUpdateMessage) {
	// Look up current child state to prevent status regression
	childState := m.findChildState(data.ChildName)
	if childState == nil || shouldUpdateChildStatus(childState.Status, data.Status) {
		item.Status = data.Status
	}
	if data.ChildInstanceID != "" {
		item.ChildInstanceID = data.ChildInstanceID
	}
	if data.ParentInstanceID != "" {
		item.ParentInstanceID = data.ParentInstanceID
	}
	item.FailureReasons = data.FailureReasons
}

// Instance state lookup helpers

// getInstanceStateForResource returns the instance state that contains the given resource.
func (m *InspectModel) getInstanceStateForResource(instanceID string) *state.InstanceState {
	if instanceID == "" || instanceID == m.instanceID {
		return m.instanceState
	}

	// Look up the child name from the instance ID using the path builder
	childName, ok := m.pathBuilder.InstanceIDToChildName[instanceID]
	if !ok {
		return m.instanceState
	}

	// Find the child instance state
	return m.findChildState(childName)
}

// findLinkStateInInstance finds a link state within a specific instance state.
func (m *InspectModel) findLinkStateInInstance(instanceState *state.InstanceState, linkName string) *state.LinkState {
	if instanceState == nil || instanceState.Links == nil {
		return nil
	}
	return instanceState.Links[linkName]
}
