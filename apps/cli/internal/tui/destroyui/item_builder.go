package destroyui

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// itemBuilder encapsulates the state needed for building destroy items.
type itemBuilder struct {
	resourcesByName map[string]*ResourceDestroyItem
	childrenByName  map[string]*ChildDestroyItem
	linksByName     map[string]*LinkDestroyItem
	instanceState   *state.InstanceState
	addedResources  map[string]bool
	addedChildren   map[string]bool
	addedLinks      map[string]bool
}

// buildItemsFromChangeset creates the initial item list from changeset data.
func buildItemsFromChangeset(
	changesetChanges *changes.BlueprintChanges,
	resourcesByName map[string]*ResourceDestroyItem,
	childrenByName map[string]*ChildDestroyItem,
	linksByName map[string]*LinkDestroyItem,
	instanceState *state.InstanceState,
) []DestroyItem {
	b := &itemBuilder{
		resourcesByName: resourcesByName,
		childrenByName:  childrenByName,
		linksByName:     linksByName,
		instanceState:   instanceState,
		addedResources:  make(map[string]bool),
		addedChildren:   make(map[string]bool),
		addedLinks:      make(map[string]bool),
	}

	var items []DestroyItem

	if changesetChanges != nil {
		items = b.appendResourceItems(items, changesetChanges)
		items = b.appendChildItems(items, changesetChanges)
		items = b.appendLinkItems(items, changesetChanges)
	}

	return items
}

// appendResourceItems adds resource items from changeset.
func (b *itemBuilder) appendResourceItems(items []DestroyItem, bpChanges *changes.BlueprintChanges) []DestroyItem {
	// For destroy, RemovedResources is the primary source
	for _, name := range bpChanges.RemovedResources {
		var resourceState *state.ResourceState
		if b.instanceState != nil {
			resourceState = findResourceState(b.instanceState, name)
		}
		item := &ResourceDestroyItem{
			Name:          name,
			Action:        ActionDelete,
			ResourceState: resourceState,
		}
		if resourceState != nil {
			item.ResourceType = resourceState.Type
			item.ResourceID = resourceState.ResourceID
		}
		b.resourcesByName[name] = item
		items = append(items, DestroyItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: b.instanceState,
		})
		b.addedResources[name] = true
	}

	// Also handle resources with changes (for partial operations)
	for name, rc := range bpChanges.ResourceChanges {
		if b.addedResources[name] {
			continue
		}
		var resourceState *state.ResourceState
		if b.instanceState != nil {
			resourceState = findResourceState(b.instanceState, name)
		}
		item := &ResourceDestroyItem{
			Name:          name,
			Action:        ActionUpdate,
			Changes:       &rc,
			ResourceState: resourceState,
		}
		if resourceState != nil {
			item.ResourceType = resourceState.Type
			item.ResourceID = resourceState.ResourceID
		}
		b.resourcesByName[name] = item
		items = append(items, DestroyItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: b.instanceState,
		})
		b.addedResources[name] = true
	}

	return items
}

// appendChildItems adds child blueprint items from changeset.
func (b *itemBuilder) appendChildItems(items []DestroyItem, bpChanges *changes.BlueprintChanges) []DestroyItem {
	// For destroy, RemovedChildren is the primary source
	for _, name := range bpChanges.RemovedChildren {
		// Get nested instance state for navigation
		var nestedInstanceState *state.InstanceState
		if b.instanceState != nil && b.instanceState.ChildBlueprints != nil {
			nestedInstanceState = b.instanceState.ChildBlueprints[name]
		}

		// Build changes from instance state for navigation into removed children
		var childChanges *changes.BlueprintChanges
		if nestedInstanceState != nil {
			childChanges = buildChangesFromInstanceState(nestedInstanceState)
		}

		item := &ChildDestroyItem{
			Name:    name,
			Action:  ActionDelete,
			Changes: childChanges,
		}
		b.childrenByName[name] = item
		items = append(items, DestroyItem{
			Type:            ItemTypeChild,
			Child:           item,
			Changes:         childChanges,
			InstanceState:   nestedInstanceState,
			childrenByName:  b.childrenByName,
			resourcesByName: b.resourcesByName,
			linksByName:     b.linksByName,
		})
		b.addedChildren[name] = true
	}

	// Also handle children with changes
	for name, cc := range bpChanges.ChildChanges {
		if b.addedChildren[name] {
			continue
		}
		ccCopy := cc
		var nestedInstanceState *state.InstanceState
		if b.instanceState != nil && b.instanceState.ChildBlueprints != nil {
			nestedInstanceState = b.instanceState.ChildBlueprints[name]
		}
		item := &ChildDestroyItem{
			Name:    name,
			Action:  ActionUpdate,
			Changes: &ccCopy,
		}
		b.childrenByName[name] = item
		items = append(items, DestroyItem{
			Type:            ItemTypeChild,
			Child:           item,
			Changes:         &ccCopy,
			InstanceState:   nestedInstanceState,
			childrenByName:  b.childrenByName,
			resourcesByName: b.resourcesByName,
			linksByName:     b.linksByName,
		})
		b.addedChildren[name] = true
	}

	return items
}

// appendLinkItems adds link items from changeset.
func (b *itemBuilder) appendLinkItems(items []DestroyItem, bpChanges *changes.BlueprintChanges) []DestroyItem {
	// For destroy, RemovedLinks is the primary source
	for _, linkName := range bpChanges.RemovedLinks {
		item := &LinkDestroyItem{
			LinkName:      linkName,
			ResourceAName: extractResourceAFromLinkName(linkName),
			ResourceBName: extractResourceBFromLinkName(linkName),
			Action:        ActionDelete,
		}
		b.linksByName[linkName] = item
		items = append(items, DestroyItem{
			Type: ItemTypeLink,
			Link: item,
		})
		b.addedLinks[linkName] = true
	}

	return items
}

// findResourceState finds a resource state by name using the instance state's
// ResourceIDs map to look up the resource ID, then retrieves the state from Resources.
func findResourceState(instanceState *state.InstanceState, name string) *state.ResourceState {
	if instanceState == nil || instanceState.ResourceIDs == nil || instanceState.Resources == nil {
		return nil
	}
	resourceID, ok := instanceState.ResourceIDs[name]
	if !ok {
		return nil
	}
	return instanceState.Resources[resourceID]
}

// buildChangesFromInstanceState creates a BlueprintChanges structure from instance state.
// This is used to enable navigation into removed child blueprints by treating all
// elements in the instance state as items to be removed during destroy.
func buildChangesFromInstanceState(instanceState *state.InstanceState) *changes.BlueprintChanges {
	if instanceState == nil {
		return nil
	}

	bpChanges := &changes.BlueprintChanges{
		RemovedResources: make([]string, 0),
		RemovedChildren:  make([]string, 0),
		RemovedLinks:     make([]string, 0),
		ChildChanges:     make(map[string]changes.BlueprintChanges),
	}

	// Add all resources as removed
	for name := range instanceState.ResourceIDs {
		bpChanges.RemovedResources = append(bpChanges.RemovedResources, name)
	}

	// Add all links as removed
	for linkName := range instanceState.Links {
		bpChanges.RemovedLinks = append(bpChanges.RemovedLinks, linkName)
	}

	// Add all children as removed, recursively building their changes
	for childName, childState := range instanceState.ChildBlueprints {
		bpChanges.RemovedChildren = append(bpChanges.RemovedChildren, childName)
		if childState != nil {
			childChanges := buildChangesFromInstanceState(childState)
			if childChanges != nil {
				bpChanges.ChildChanges[childName] = *childChanges
			}
		}
	}

	return bpChanges
}
