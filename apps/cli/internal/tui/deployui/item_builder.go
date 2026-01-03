package deployui

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// itemBuilder encapsulates the state needed for building deploy items.
// This pattern reduces parameter counts by grouping related data together.
type itemBuilder struct {
	resourcesByName map[string]*ResourceDeployItem
	childrenByName  map[string]*ChildDeployItem
	linksByName     map[string]*LinkDeployItem
	instanceState   *state.InstanceState
	addedResources  map[string]bool
	addedChildren   map[string]bool
	addedLinks      map[string]bool
}

// buildItemsFromChangeset creates the initial item list from changeset data.
// This provides the proper hierarchy (resources, children, links) from the start.
// It also includes items with no changes from the instance state.
func buildItemsFromChangeset(
	changesetChanges *changes.BlueprintChanges,
	resourcesByName map[string]*ResourceDeployItem,
	childrenByName map[string]*ChildDeployItem,
	linksByName map[string]*LinkDeployItem,
	instanceState *state.InstanceState,
) []DeployItem {
	b := &itemBuilder{
		resourcesByName: resourcesByName,
		childrenByName:  childrenByName,
		linksByName:     linksByName,
		instanceState:   instanceState,
		addedResources:  make(map[string]bool),
		addedChildren:   make(map[string]bool),
		addedLinks:      make(map[string]bool),
	}

	var items []DeployItem

	if changesetChanges != nil {
		items = b.appendResourceItems(items, changesetChanges)
		items = b.appendChildItems(items, changesetChanges)
		items = b.appendLinkItems(items, changesetChanges)

		b.populateNestedItems(changesetChanges)
	}

	items = b.appendNoChangeItems(items)

	return items
}

// appendResourceItems adds resource items from changeset.
func (b *itemBuilder) appendResourceItems(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	items = b.appendNewResources(items, changes)
	items = b.appendChangedResources(items, changes)
	items = b.appendRemovedResources(items, changes)
	return items
}

func (b *itemBuilder) appendNewResources(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	for name, rc := range changes.NewResources {
		rcCopy := rc
		item := &ResourceDeployItem{
			Name:    name,
			Action:  ActionCreate,
			Changes: &rcCopy,
		}
		b.resourcesByName[name] = item
		items = append(items, DeployItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: b.instanceState,
		})
		b.addedResources[name] = true
	}
	return items
}

func (b *itemBuilder) appendChangedResources(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	for name, rc := range changes.ResourceChanges {
		rcCopy := rc
		item := buildChangedResourceItem(name, &rcCopy, b.instanceState)
		b.resourcesByName[name] = item
		items = append(items, DeployItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: b.instanceState,
		})
		b.addedResources[name] = true
	}
	return items
}

func (b *itemBuilder) appendRemovedResources(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	for _, name := range changes.RemovedResources {
		var resourceState *state.ResourceState
		if b.instanceState != nil {
			resourceState = findResourceState(b.instanceState, name)
		}
		item := &ResourceDeployItem{
			Name:          name,
			Action:        ActionDelete,
			ResourceState: resourceState,
		}
		b.resourcesByName[name] = item
		items = append(items, DeployItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: b.instanceState,
		})
		b.addedResources[name] = true
	}
	return items
}

// appendChildItems adds child blueprint items from changeset.
func (b *itemBuilder) appendChildItems(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	items = b.appendNewChildren(items, changes)
	items = b.appendChangedChildren(items, changes)
	items = b.appendRecreateChildren(items, changes)
	items = b.appendRemovedChildren(items, changes)
	return items
}

func (b *itemBuilder) appendNewChildren(items []DeployItem, bpChanges *changes.BlueprintChanges) []DeployItem {
	for name, nc := range bpChanges.NewChildren {
		childChanges := &changes.BlueprintChanges{
			NewResources: nc.NewResources,
			NewChildren:  nc.NewChildren,
		}
		item := &ChildDeployItem{
			Name:    name,
			Action:  ActionCreate,
			Changes: childChanges,
		}
		b.childrenByName[name] = item
		items = append(items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			Changes:         childChanges,
			childrenByName:  b.childrenByName,
			resourcesByName: b.resourcesByName,
			linksByName:     b.linksByName,
		})
		b.addedChildren[name] = true
	}
	return items
}

func (b *itemBuilder) appendChangedChildren(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	for name, cc := range changes.ChildChanges {
		ccCopy := cc
		var nestedInstanceState *state.InstanceState
		if b.instanceState != nil && b.instanceState.ChildBlueprints != nil {
			nestedInstanceState = b.instanceState.ChildBlueprints[name]
		}
		item := &ChildDeployItem{
			Name:    name,
			Action:  ActionUpdate,
			Changes: &ccCopy,
		}
		b.childrenByName[name] = item
		items = append(items, DeployItem{
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

func (b *itemBuilder) appendRecreateChildren(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	for _, name := range changes.RecreateChildren {
		item := &ChildDeployItem{
			Name:   name,
			Action: ActionRecreate,
		}
		b.childrenByName[name] = item
		items = append(items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			childrenByName:  b.childrenByName,
			resourcesByName: b.resourcesByName,
			linksByName:     b.linksByName,
		})
		b.addedChildren[name] = true
	}
	return items
}

func (b *itemBuilder) appendRemovedChildren(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	for _, name := range changes.RemovedChildren {
		item := &ChildDeployItem{
			Name:   name,
			Action: ActionDelete,
		}
		b.childrenByName[name] = item
		items = append(items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			childrenByName:  b.childrenByName,
			resourcesByName: b.resourcesByName,
			linksByName:     b.linksByName,
		})
		b.addedChildren[name] = true
	}
	return items
}

// appendLinkItems adds link items from changeset.
func (b *itemBuilder) appendLinkItems(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	items = b.appendLinksFromNewResources(items, changes)
	items = b.appendLinksFromChangedResources(items, changes)
	items = b.appendRemovedLinks(items, changes)
	return items
}

func (b *itemBuilder) appendLinksFromNewResources(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	for resourceAName, resourceChanges := range changes.NewResources {
		for resourceBName := range resourceChanges.NewOutboundLinks {
			linkName := resourceAName + "::" + resourceBName
			item := &LinkDeployItem{
				LinkName:      linkName,
				ResourceAName: resourceAName,
				ResourceBName: resourceBName,
				Action:        ActionCreate,
			}
			b.linksByName[linkName] = item
			items = append(items, DeployItem{
				Type: ItemTypeLink,
				Link: item,
			})
			b.addedLinks[linkName] = true
		}
	}
	return items
}

func (b *itemBuilder) appendLinksFromChangedResources(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	for resourceAName, resourceChanges := range changes.ResourceChanges {
		items = b.appendLinksFromChangedResource(items, resourceAName, resourceChanges)
	}
	return items
}

func (b *itemBuilder) appendLinksFromChangedResource(items []DeployItem, resourceAName string, rc provider.Changes) []DeployItem {
	// New outbound links
	for resourceBName := range rc.NewOutboundLinks {
		linkName := resourceAName + "::" + resourceBName
		item := &LinkDeployItem{
			LinkName:      linkName,
			ResourceAName: resourceAName,
			ResourceBName: resourceBName,
			Action:        ActionCreate,
		}
		b.linksByName[linkName] = item
		items = append(items, DeployItem{Type: ItemTypeLink, Link: item})
		b.addedLinks[linkName] = true
	}

	// Changed outbound links
	for resourceBName := range rc.OutboundLinkChanges {
		linkName := resourceAName + "::" + resourceBName
		item := &LinkDeployItem{
			LinkName:      linkName,
			ResourceAName: resourceAName,
			ResourceBName: resourceBName,
			Action:        ActionUpdate,
		}
		b.linksByName[linkName] = item
		items = append(items, DeployItem{Type: ItemTypeLink, Link: item})
		b.addedLinks[linkName] = true
	}

	// Removed outbound links
	for _, linkName := range rc.RemovedOutboundLinks {
		item := &LinkDeployItem{
			LinkName:      linkName,
			ResourceAName: extractResourceAFromLinkName(linkName),
			ResourceBName: extractResourceBFromLinkName(linkName),
			Action:        ActionDelete,
		}
		b.linksByName[linkName] = item
		items = append(items, DeployItem{Type: ItemTypeLink, Link: item})
		b.addedLinks[linkName] = true
	}

	return items
}

func (b *itemBuilder) appendRemovedLinks(items []DeployItem, changes *changes.BlueprintChanges) []DeployItem {
	for _, linkName := range changes.RemovedLinks {
		if _, exists := b.linksByName[linkName]; !exists {
			item := &LinkDeployItem{
				LinkName:      linkName,
				ResourceAName: extractResourceAFromLinkName(linkName),
				ResourceBName: extractResourceBFromLinkName(linkName),
				Action:        ActionDelete,
			}
			b.linksByName[linkName] = item
			items = append(items, DeployItem{Type: ItemTypeLink, Link: item})
			b.addedLinks[linkName] = true
		}
	}
	return items
}

// appendNoChangeItems adds items from instance state that have no changes.
func (b *itemBuilder) appendNoChangeItems(items []DeployItem) []DeployItem {
	if b.instanceState == nil {
		return items
	}

	items = b.appendNoChangeResources(items)
	items = b.appendNoChangeChildren(items)
	items = b.appendNoChangeLinks(items)

	return items
}

func (b *itemBuilder) appendNoChangeResources(items []DeployItem) []DeployItem {
	for _, resourceState := range b.instanceState.Resources {
		if b.addedResources[resourceState.Name] {
			continue
		}
		item := &ResourceDeployItem{
			Name:          resourceState.Name,
			ResourceID:    resourceState.ResourceID,
			ResourceType:  resourceState.Type,
			Action:        ActionNoChange,
			ResourceState: resourceState,
		}
		b.resourcesByName[resourceState.Name] = item
		items = append(items, DeployItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: b.instanceState,
		})
	}
	return items
}

func (b *itemBuilder) appendNoChangeChildren(items []DeployItem) []DeployItem {
	for name, childState := range b.instanceState.ChildBlueprints {
		if b.addedChildren[name] {
			continue
		}
		item := &ChildDeployItem{
			Name:    name,
			Action:  ActionNoChange,
			Changes: &changes.BlueprintChanges{},
		}
		b.childrenByName[name] = item
		items = append(items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			Changes:         &changes.BlueprintChanges{},
			InstanceState:   childState,
			childrenByName:  b.childrenByName,
			resourcesByName: b.resourcesByName,
			linksByName:     b.linksByName,
		})
	}
	return items
}

func (b *itemBuilder) appendNoChangeLinks(items []DeployItem) []DeployItem {
	for linkName, linkState := range b.instanceState.Links {
		if b.addedLinks[linkName] {
			continue
		}
		item := &LinkDeployItem{
			LinkID:        linkState.LinkID,
			LinkName:      linkName,
			ResourceAName: extractResourceAFromLinkName(linkName),
			ResourceBName: extractResourceBFromLinkName(linkName),
			Action:        ActionNoChange,
			Status:        linkState.Status,
		}
		b.linksByName[linkName] = item
		items = append(items, DeployItem{
			Type:          ItemTypeLink,
			Link:          item,
			InstanceState: b.instanceState,
		})
	}
	return items
}

// populateNestedItems pre-populates nested items into the maps.
func (b *itemBuilder) populateNestedItems(bpChanges *changes.BlueprintChanges) {
	for _, nc := range bpChanges.NewChildren {
		childChanges := &changes.BlueprintChanges{
			NewResources: nc.NewResources,
			NewChildren:  nc.NewChildren,
		}
		b.populateNestedFromChangeset(childChanges)
	}
	for _, cc := range bpChanges.ChildChanges {
		ccCopy := cc
		b.populateNestedFromChangeset(&ccCopy)
	}
}

// populateNestedFromChangeset recursively walks the changeset hierarchy and adds all
// nested children and resources to the shared lookup maps.
func (b *itemBuilder) populateNestedFromChangeset(blueprintChanges *changes.BlueprintChanges) {
	if blueprintChanges == nil {
		return
	}

	b.populateNestedNewChildren(blueprintChanges)
	b.populateNestedChangedChildren(blueprintChanges)
}

func (b *itemBuilder) populateNestedNewChildren(blueprintChanges *changes.BlueprintChanges) {
	for name, nc := range blueprintChanges.NewChildren {
		childChanges := &changes.BlueprintChanges{
			NewResources: nc.NewResources,
			NewChildren:  nc.NewChildren,
		}

		if _, exists := b.childrenByName[name]; !exists {
			b.childrenByName[name] = &ChildDeployItem{
				Name:    name,
				Action:  ActionCreate,
				Changes: childChanges,
			}
		}

		for resourceName, rc := range nc.NewResources {
			if _, exists := b.resourcesByName[resourceName]; !exists {
				rcCopy := rc
				b.resourcesByName[resourceName] = &ResourceDeployItem{
					Name:    resourceName,
					Action:  ActionCreate,
					Changes: &rcCopy,
				}
			}
		}

		b.populateNestedFromChangeset(childChanges)
	}
}

func (b *itemBuilder) populateNestedChangedChildren(blueprintChanges *changes.BlueprintChanges) {
	for name, cc := range blueprintChanges.ChildChanges {
		ccCopy := cc

		if _, exists := b.childrenByName[name]; !exists {
			b.childrenByName[name] = &ChildDeployItem{
				Name:    name,
				Action:  ActionUpdate,
				Changes: &ccCopy,
			}
		}

		b.addNestedResourcesFromChildChanges(&ccCopy)
		b.populateNestedFromChangeset(&ccCopy)
	}
}

func (b *itemBuilder) addNestedResourcesFromChildChanges(cc *changes.BlueprintChanges) {
	for resourceName, rc := range cc.NewResources {
		if _, exists := b.resourcesByName[resourceName]; !exists {
			rcCopy := rc
			b.resourcesByName[resourceName] = &ResourceDeployItem{
				Name:    resourceName,
				Action:  ActionCreate,
				Changes: &rcCopy,
			}
		}
	}
	for resourceName, rc := range cc.ResourceChanges {
		if _, exists := b.resourcesByName[resourceName]; !exists {
			rcCopy := rc
			item := buildNestedChangedResourceItem(resourceName, &rcCopy)
			b.resourcesByName[resourceName] = item
		}
	}
	for _, resourceName := range cc.RemovedResources {
		if _, exists := b.resourcesByName[resourceName]; !exists {
			b.resourcesByName[resourceName] = &ResourceDeployItem{
				Name:   resourceName,
				Action: ActionDelete,
			}
		}
	}
}

// Helper functions

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

// buildChangedResourceItem creates a ResourceDeployItem for a changed resource.
func buildChangedResourceItem(name string, rc *provider.Changes, instanceState *state.InstanceState) *ResourceDeployItem {
	action := determineResourceAction(rc)

	var resourceState *state.ResourceState
	if instanceState != nil {
		resourceState = findResourceState(instanceState, name)
	}

	resourceID, resourceType := extractResourceInfo(rc)

	return &ResourceDeployItem{
		Name:          name,
		Action:        action,
		Changes:       rc,
		ResourceID:    resourceID,
		ResourceType:  resourceType,
		ResourceState: resourceState,
	}
}

func determineResourceAction(rc *provider.Changes) ActionType {
	if rc.MustRecreate {
		return ActionRecreate
	}
	if !provider.ChangesHasFieldChanges(rc) {
		return ActionNoChange
	}
	return ActionUpdate
}

func extractResourceInfo(rc *provider.Changes) (string, string) {
	var resourceID, resourceType string

	if rc.AppliedResourceInfo.ResourceID != "" {
		resourceID = rc.AppliedResourceInfo.ResourceID
	}
	if rc.AppliedResourceInfo.CurrentResourceState != nil {
		if resourceType == "" && rc.AppliedResourceInfo.CurrentResourceState.Type != "" {
			resourceType = rc.AppliedResourceInfo.CurrentResourceState.Type
		}
	}

	return resourceID, resourceType
}

// buildNestedChangedResourceItem creates a ResourceDeployItem for a nested changed resource.
func buildNestedChangedResourceItem(resourceName string, rc *provider.Changes) *ResourceDeployItem {
	action := determineResourceAction(rc)
	resourceID, resourceType := extractResourceInfo(rc)

	var resourceState *state.ResourceState
	if rc.AppliedResourceInfo.CurrentResourceState != nil {
		resourceState = rc.AppliedResourceInfo.CurrentResourceState
	}

	return &ResourceDeployItem{
		Name:          resourceName,
		Action:        action,
		Changes:       rc,
		ResourceID:    resourceID,
		ResourceType:  resourceType,
		ResourceState: resourceState,
	}
}
