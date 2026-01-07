package deployui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
)

// EventProcessorState contains the state needed for event processing operations.
// This allows the event processing helper functions to be tested independently.
type EventProcessorState struct {
	InstanceID              string
	ResourcesByName         map[string]*ResourceDeployItem
	ChildrenByName          map[string]*ChildDeployItem
	LinksByName             map[string]*LinkDeployItem
	InstanceIDToChildName   map[string]string
	InstanceIDToParentID    map[string]string
	ChildNameToInstancePath map[string]string
	ChangesetChanges        *changes.BlueprintChanges
	FooterRenderer          *DeployFooterRenderer
	Items                   []DeployItem
}

// NewEventProcessorState creates a new EventProcessorState with initialized maps.
func NewEventProcessorState(instanceID string) *EventProcessorState {
	return &EventProcessorState{
		InstanceID:              instanceID,
		ResourcesByName:         make(map[string]*ResourceDeployItem),
		ChildrenByName:          make(map[string]*ChildDeployItem),
		LinksByName:             make(map[string]*LinkDeployItem),
		InstanceIDToChildName:   make(map[string]string),
		InstanceIDToParentID:    make(map[string]string),
		ChildNameToInstancePath: make(map[string]string),
		FooterRenderer:          &DeployFooterRenderer{},
	}
}

// BuildResourcePath builds a path for a resource based on its instance ID.
// For root instance resources, returns just the resource name.
// For nested resources, returns a path like "parentChild/childName/resourceName".
func BuildResourcePath(state *EventProcessorState, instanceID, resourceName string) string {
	if instanceID == "" || instanceID == state.InstanceID {
		return resourceName
	}

	pathParts := BuildParentChain(state, instanceID)
	pathParts = append(pathParts, resourceName)
	return shared.JoinPath(pathParts)
}

// BuildInstancePath builds a path from instance ID to the child name.
// For root instance resources, returns just the name.
// For nested children, returns a path like "parentChild/childName".
func BuildInstancePath(state *EventProcessorState, parentInstanceID, childName string) string {
	if parentInstanceID == "" || parentInstanceID == state.InstanceID {
		return childName
	}

	pathParts := BuildParentChain(state, parentInstanceID)
	pathParts = append(pathParts, childName)
	return shared.JoinPath(pathParts)
}

// BuildParentChain constructs the path parts from the root to the given instance ID.
func BuildParentChain(state *EventProcessorState, startInstanceID string) []string {
	var pathParts []string
	currentID := startInstanceID
	for currentID != "" && currentID != state.InstanceID {
		if name, ok := state.InstanceIDToChildName[currentID]; ok {
			pathParts = append([]string{name}, pathParts...)
			currentID = state.InstanceIDToParentID[currentID]
		} else {
			break
		}
	}
	return pathParts
}

// LookupOrMigrateResource looks up a resource by path, migrating from name-based key if needed.
func LookupOrMigrateResource(state *EventProcessorState, path, name string) *ResourceDeployItem {
	if item, exists := state.ResourcesByName[path]; exists {
		return item
	}
	if item, exists := state.ResourcesByName[name]; exists {
		delete(state.ResourcesByName, name)
		state.ResourcesByName[path] = item
		return item
	}
	return nil
}

// LookupOrMigrateChild looks up a child by path, migrating from name-based key if needed.
func LookupOrMigrateChild(state *EventProcessorState, path, name string) *ChildDeployItem {
	if item, exists := state.ChildrenByName[path]; exists {
		return item
	}
	if item, exists := state.ChildrenByName[name]; exists {
		delete(state.ChildrenByName, name)
		state.ChildrenByName[path] = item
		return item
	}
	return nil
}

// LookupOrMigrateLink looks up a link by path, migrating from name-based key if needed.
func LookupOrMigrateLink(state *EventProcessorState, path, name string) *LinkDeployItem {
	if item, exists := state.LinksByName[path]; exists {
		return item
	}
	if item, exists := state.LinksByName[name]; exists {
		delete(state.LinksByName, name)
		state.LinksByName[path] = item
		return item
	}
	return nil
}

// TrackChildInstanceMapping records the mapping from a child instance ID to its name and parent.
func TrackChildInstanceMapping(state *EventProcessorState, data *container.ChildDeployUpdateMessage) {
	if data.ChildInstanceID != "" && data.ChildName != "" {
		state.InstanceIDToChildName[data.ChildInstanceID] = data.ChildName
		state.InstanceIDToParentID[data.ChildInstanceID] = data.ParentInstanceID
	}
}

// GetChildChanges returns the blueprint changes for a child, if available.
func GetChildChanges(state *EventProcessorState, childName string) *changes.BlueprintChanges {
	if state.ChangesetChanges == nil {
		return nil
	}

	if nc, ok := state.ChangesetChanges.NewChildren[childName]; ok {
		return &changes.BlueprintChanges{
			NewResources: nc.NewResources,
			NewChildren:  nc.NewChildren,
		}
	}

	if cc, ok := state.ChangesetChanges.ChildChanges[childName]; ok {
		ccCopy := cc
		return &ccCopy
	}

	return nil
}

// ProcessResourceUpdate handles a resource update event.
func ProcessResourceUpdate(state *EventProcessorState, data *container.ResourceDeployUpdateMessage) {
	isRootResource := data.InstanceID == "" || data.InstanceID == state.InstanceID
	resourcePath := BuildResourcePath(state, data.InstanceID, data.ResourceName)

	item := LookupOrMigrateResource(state, resourcePath, data.ResourceName)

	if item == nil {
		item = &ResourceDeployItem{
			Name:       data.ResourceName,
			ResourceID: data.ResourceID,
			Group:      data.Group,
		}
		state.ResourcesByName[resourcePath] = item
		if isRootResource {
			state.Items = append(state.Items, DeployItem{
				Type:     ItemTypeResource,
				Resource: item,
			})
		}
	}

	UpdateResourceItemFromEvent(item, data)
}

// ProcessChildUpdate handles a child blueprint update event.
func ProcessChildUpdate(state *EventProcessorState, data *container.ChildDeployUpdateMessage) {
	TrackChildInstanceMapping(state, data)
	childPath := BuildInstancePath(state, data.ParentInstanceID, data.ChildName)

	item := LookupOrMigrateChild(state, childPath, data.ChildName)

	if item == nil {
		item = CreateChildItem(state, data, childPath)
	}

	state.ChildNameToInstancePath[data.ChildName] = childPath
	UpdateChildItemFromEvent(item, data)
}

// ProcessLinkUpdate handles a link update event.
func ProcessLinkUpdate(state *EventProcessorState, data *container.LinkDeployUpdateMessage) {
	isRootLink := data.InstanceID == "" || data.InstanceID == state.InstanceID
	linkPath := BuildResourcePath(state, data.InstanceID, data.LinkName)

	item := LookupOrMigrateLink(state, linkPath, data.LinkName)

	if item == nil {
		item = &LinkDeployItem{
			LinkID:        data.LinkID,
			LinkName:      data.LinkName,
			ResourceAName: ExtractResourceAFromLinkName(data.LinkName),
			ResourceBName: ExtractResourceBFromLinkName(data.LinkName),
		}
		state.LinksByName[linkPath] = item
		if isRootLink {
			state.Items = append(state.Items, DeployItem{
				Type: ItemTypeLink,
				Link: item,
			})
		}
	}

	UpdateLinkItemFromEvent(item, data)
}

// ProcessInstanceUpdate handles an instance status update event.
func ProcessInstanceUpdate(state *EventProcessorState, data *container.DeploymentUpdateMessage) {
	state.FooterRenderer.CurrentStatus = data.Status
}

// CreateChildItem creates a new child deploy item.
func CreateChildItem(state *EventProcessorState, data *container.ChildDeployUpdateMessage, childPath string) *ChildDeployItem {
	isDirectChildOfRoot := data.ParentInstanceID == "" || data.ParentInstanceID == state.InstanceID

	childChanges := GetChildChanges(state, data.ChildName)

	item := &ChildDeployItem{
		Name:             data.ChildName,
		ParentInstanceID: data.ParentInstanceID,
		ChildInstanceID:  data.ChildInstanceID,
		Group:            data.Group,
		Changes:          childChanges,
	}
	state.ChildrenByName[childPath] = item

	if isDirectChildOfRoot {
		state.Items = append(state.Items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			Changes:         childChanges,
			childrenByName:  state.ChildrenByName,
			resourcesByName: state.ResourcesByName,
			linksByName:     state.LinksByName,
		})
	}

	return item
}

// UpdateResourceItemFromEvent updates a resource item with data from an event.
func UpdateResourceItemFromEvent(item *ResourceDeployItem, data *container.ResourceDeployUpdateMessage) {
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

// UpdateChildItemFromEvent updates a child item with data from an event.
func UpdateChildItemFromEvent(item *ChildDeployItem, data *container.ChildDeployUpdateMessage) {
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

// UpdateLinkItemFromEvent updates a link item with data from an event.
func UpdateLinkItemFromEvent(item *LinkDeployItem, data *container.LinkDeployUpdateMessage) {
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
