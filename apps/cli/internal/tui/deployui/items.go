package deployui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// Ensure DeployItem implements splitpane.Item.
var _ splitpane.Item = (*DeployItem)(nil)

// GetID returns a unique identifier for the item.
func (i *DeployItem) GetID() string {
	switch i.Type {
	case ItemTypeResource:
		if i.Resource != nil {
			return i.Resource.Name
		}
	case ItemTypeChild:
		if i.Child != nil {
			return i.Child.Name
		}
	case ItemTypeLink:
		if i.Link != nil {
			return i.Link.LinkName
		}
	}
	return ""
}

// GetName returns the display name for the item.
func (i *DeployItem) GetName() string {
	return i.GetID()
}

// GetIcon returns a status icon for the item.
func (i *DeployItem) GetIcon(selected bool) string {
	return i.getIconChar()
}

func (i *DeployItem) getIconChar() string {
	switch i.Type {
	case ItemTypeResource:
		if i.Resource != nil {
			if i.Resource.Skipped {
				return shared.IconSkipped
			}
			if i.Resource.Action == ActionNoChange {
				return shared.IconNoChange
			}
			return shared.ResourceStatusIcon(i.Resource.Status)
		}
	case ItemTypeChild:
		if i.Child != nil {
			if i.Child.Skipped {
				return shared.IconSkipped
			}
			if i.Child.Action == ActionNoChange {
				return shared.IconNoChange
			}
			return shared.InstanceStatusIcon(i.Child.Status)
		}
	case ItemTypeLink:
		if i.Link != nil {
			if i.Link.Skipped {
				return shared.IconSkipped
			}
			if i.Link.Action == ActionNoChange {
				return shared.IconNoChange
			}
			return shared.LinkStatusIcon(i.Link.Status)
		}
	}
	return shared.IconPending
}


// GetIconStyled returns a styled icon for the item.
func (i *DeployItem) GetIconStyled(s *styles.Styles, styled bool) string {
	icon := i.getIconChar()
	if !styled {
		return icon
	}

	switch i.Type {
	case ItemTypeResource:
		if i.Resource != nil {
			if i.Resource.Skipped {
				return s.Warning.Render(icon)
			}
			if i.Resource.Action == ActionNoChange {
				return s.Muted.Render(icon)
			}
			return shared.StyleResourceIcon(icon, i.Resource.Status, s)
		}
	case ItemTypeChild:
		if i.Child != nil {
			if i.Child.Skipped {
				return s.Warning.Render(icon)
			}
			if i.Child.Action == ActionNoChange {
				return s.Muted.Render(icon)
			}
			return shared.StyleInstanceIcon(icon, i.Child.Status, s)
		}
	case ItemTypeLink:
		if i.Link != nil {
			if i.Link.Skipped {
				return s.Warning.Render(icon)
			}
			if i.Link.Action == ActionNoChange {
				return s.Muted.Render(icon)
			}
			return shared.StyleLinkIcon(icon, i.Link.Status, s)
		}
	}
	return icon
}

// GetAction returns the action badge text.
func (i *DeployItem) GetAction() string {
	switch i.Type {
	case ItemTypeResource:
		if i.Resource != nil {
			return string(i.Resource.Action)
		}
	case ItemTypeChild:
		if i.Child != nil {
			return string(i.Child.Action)
		}
	case ItemTypeLink:
		if i.Link != nil {
			return string(i.Link.Action)
		}
	}
	return ""
}

// GetDepth returns the nesting depth for indentation.
func (i *DeployItem) GetDepth() int {
	return i.Depth
}

// GetParentID returns the parent item ID.
func (i *DeployItem) GetParentID() string {
	return i.ParentChild
}

// GetItemType returns the type for section grouping.
func (i *DeployItem) GetItemType() string {
	return string(i.Type)
}

// IsExpandable returns true if the item can be expanded in-place.
// A child can be expanded if it has either Changes (from changeset) or InstanceState (for unchanged items).
func (i *DeployItem) IsExpandable() bool {
	return i.Type == ItemTypeChild && (i.Changes != nil || i.InstanceState != nil)
}

// CanDrillDown returns true if the item can be drilled into.
// A child can be drilled into if it has either Changes (from changeset) or InstanceState (for unchanged items).
func (i *DeployItem) CanDrillDown() bool {
	return i.Type == ItemTypeChild && (i.Changes != nil || i.InstanceState != nil)
}

// GetChildren returns child items when expanded.
// Uses the Changes data from the changeset to build the hierarchy,
// and also includes unchanged items from InstanceState.
func (i *DeployItem) GetChildren() []splitpane.Item {
	if i.Type != ItemTypeChild {
		return nil
	}

	// Need either Changes or InstanceState to build children
	if i.Changes == nil && i.InstanceState == nil {
		return nil
	}

	// Check if the parent child is skipped - all children inherit this status
	parentSkipped := i.Child != nil && i.Child.Skipped

	// Track which items have been added from changes
	addedResources := make(map[string]bool)
	addedChildren := make(map[string]bool)
	addedLinks := make(map[string]bool)

	var items []splitpane.Item

	// First add items from changes (if any)
	if i.Changes != nil {
		items = i.appendChildResourceItems(items, parentSkipped, addedResources)
		items = i.appendNestedChildItems(items, parentSkipped, addedChildren)
		items = i.appendChildLinkItems(items, parentSkipped, addedLinks)
	}

	// Then add unchanged items from InstanceState
	items = i.appendUnchangedItemsFromInstanceState(items, parentSkipped, addedResources, addedChildren, addedLinks)

	// Finally, add items discovered from shared maps (for streaming scenarios where items
	// exist in the lookup maps but aren't yet in Changes or InstanceState)
	items = i.appendItemsFromSharedMaps(items, parentSkipped, addedResources, addedChildren, addedLinks)

	return items
}

// appendChildResourceItems adds resource items from this child's changes.
// It uses the shared lookup maps to ensure status updates are reflected in the UI.
// It also tracks which resources have been added in the addedResources map.
func (i *DeployItem) appendChildResourceItems(items []splitpane.Item, parentSkipped bool, addedResources map[string]bool) []splitpane.Item {
	childChanges := i.Changes

	// New resources
	for name := range childChanges.NewResources {
		resourceItem, resourcePath := i.getOrCreateResourceItem(name, ActionCreate, parentSkipped)
		items = append(items, &DeployItem{
			Type:            ItemTypeResource,
			Resource:        resourceItem,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            resourcePath,
			InstanceState:   i.InstanceState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedResources[name] = true
	}

	// Changed resources
	for name, rc := range childChanges.ResourceChanges {
		rcCopy := rc
		// Determine the action based on whether there are actual field changes
		// If only link changes, treat as no-change for the resource itself
		action := ActionUpdate
		if rc.MustRecreate {
			action = ActionRecreate
		} else if !provider.ChangesHasFieldChanges(&rcCopy) {
			// Resource has only link changes, no field changes
			action = ActionNoChange
		}
		resourceItem, resourcePath := i.getOrCreateResourceItemWithChanges(name, action, parentSkipped, &rcCopy)
		items = append(items, &DeployItem{
			Type:            ItemTypeResource,
			Resource:        resourceItem,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            resourcePath,
			InstanceState:   i.InstanceState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedResources[name] = true
	}

	// Removed resources
	for _, name := range childChanges.RemovedResources {
		resourceItem, resourcePath := i.getOrCreateResourceItem(name, ActionDelete, parentSkipped)
		items = append(items, &DeployItem{
			Type:            ItemTypeResource,
			Resource:        resourceItem,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            resourcePath,
			InstanceState:   i.InstanceState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedResources[name] = true
	}

	return items
}

// getOrCreateResourceItem looks up a resource item from the shared map, or creates one if it doesn't exist.
// It uses path-based keys to uniquely identify resources across different child blueprints.
func (i *DeployItem) getOrCreateResourceItem(name string, action ActionType, skipped bool) (*ResourceDeployItem, string) {
	// Build path-based key: parentPath/resourceName
	resourcePath := i.BuildChildPath(name)

	if i.resourcesByName != nil {
		// First try path-based lookup
		if existing, ok := i.resourcesByName[resourcePath]; ok {
			existing.Skipped = skipped
			return existing, resourcePath
		}
		// Fall back to simple name lookup for backwards compatibility
		if existing, ok := i.resourcesByName[name]; ok {
			existing.Skipped = skipped
			return existing, resourcePath
		}
	}
	// Create a new item if not found
	newItem := &ResourceDeployItem{
		Name:    name,
		Action:  action,
		Skipped: skipped,
	}
	// Store in the shared map using path-based key
	if i.resourcesByName != nil {
		i.resourcesByName[resourcePath] = newItem
	}
	return newItem, resourcePath
}

// getOrCreateResourceItemWithChanges looks up or creates a resource item and populates it
// with data from the provider.Changes. This is used for resources in ResourceChanges
// to ensure we have access to ResourceID, ResourceType, and CurrentResourceState.
func (i *DeployItem) getOrCreateResourceItemWithChanges(name string, action ActionType, skipped bool, changes *provider.Changes) (*ResourceDeployItem, string) {
	// Build path-based key: parentPath/resourceName
	resourcePath := i.BuildChildPath(name)

	if i.resourcesByName != nil {
		// First try path-based lookup
		if existing, ok := i.resourcesByName[resourcePath]; ok {
			existing.Skipped = skipped
			// Update with changes data if not already set
			if existing.Changes == nil && changes != nil {
				existing.Changes = changes
				i.populateResourceItemFromChanges(existing, changes)
			}
			return existing, resourcePath
		}
		// Fall back to simple name lookup for backwards compatibility
		if existing, ok := i.resourcesByName[name]; ok {
			existing.Skipped = skipped
			// Update with changes data if not already set
			if existing.Changes == nil && changes != nil {
				existing.Changes = changes
				i.populateResourceItemFromChanges(existing, changes)
			}
			return existing, resourcePath
		}
	}

	// Create a new item if not found
	newItem := &ResourceDeployItem{
		Name:    name,
		Action:  action,
		Skipped: skipped,
		Changes: changes,
	}
	// Populate from changes data
	if changes != nil {
		i.populateResourceItemFromChanges(newItem, changes)
	}
	// Store in the shared map using path-based key
	if i.resourcesByName != nil {
		i.resourcesByName[resourcePath] = newItem
	}
	return newItem, resourcePath
}

// populateResourceItemFromChanges fills in resource item fields from provider.Changes data.
func (i *DeployItem) populateResourceItemFromChanges(item *ResourceDeployItem, changes *provider.Changes) {
	if changes == nil {
		return
	}
	info := &changes.AppliedResourceInfo
	if item.ResourceID == "" && info.ResourceID != "" {
		item.ResourceID = info.ResourceID
	}
	if item.ResourceState == nil && info.CurrentResourceState != nil {
		item.ResourceState = info.CurrentResourceState
		// Also extract resource type from current state if available
		if item.ResourceType == "" && info.CurrentResourceState.Type != "" {
			item.ResourceType = info.CurrentResourceState.Type
		}
	}
}

// BuildChildPath builds a path for a child element based on this item's path.
func (i *DeployItem) BuildChildPath(childName string) string {
	if i.Path == "" {
		// If parent has no path, use the parent's ID (child name) as the base
		if i.Child != nil {
			return i.Child.Name + "/" + childName
		}
		return childName
	}
	return i.Path + "/" + childName
}

// appendNestedChildItems adds nested child blueprint items from this child's changes.
// It uses the shared lookup maps to ensure status updates are reflected in the UI.
// It also tracks which children have been added in the addedChildren map.
func (i *DeployItem) appendNestedChildItems(items []splitpane.Item, parentSkipped bool, addedChildren map[string]bool) []splitpane.Item {
	childChanges := i.Changes

	// New children - convert NewBlueprintDefinition to BlueprintChanges
	for name, nc := range childChanges.NewChildren {
		nestedChanges := &changes.BlueprintChanges{
			NewResources: nc.NewResources,
			NewChildren:  nc.NewChildren,
		}
		// Look up or create the shared ChildDeployItem
		childItem, childPath := i.getOrCreateChildItem(name, ActionCreate, nestedChanges, parentSkipped)
		items = append(items, &DeployItem{
			Type:            ItemTypeChild,
			Child:           childItem,
			Changes:         nestedChanges,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            childPath,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedChildren[name] = true
	}

	// Changed children - get nested instance state if available
	for name, cc := range childChanges.ChildChanges {
		ccCopy := cc
		// Look up or create the shared ChildDeployItem
		childItem, childPath := i.getOrCreateChildItem(name, ActionUpdate, &ccCopy, parentSkipped)

		// Get nested instance state if available
		var nestedInstanceState *state.InstanceState
		if i.InstanceState != nil && i.InstanceState.ChildBlueprints != nil {
			nestedInstanceState = i.InstanceState.ChildBlueprints[name]
		}

		items = append(items, &DeployItem{
			Type:            ItemTypeChild,
			Child:           childItem,
			Changes:         &ccCopy,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            childPath,
			InstanceState:   nestedInstanceState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedChildren[name] = true
	}

	// Removed children
	for _, name := range childChanges.RemovedChildren {
		// Look up or create the shared ChildDeployItem
		childItem, childPath := i.getOrCreateChildItem(name, ActionDelete, nil, parentSkipped)
		items = append(items, &DeployItem{
			Type:            ItemTypeChild,
			Child:           childItem,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            childPath,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedChildren[name] = true
	}

	return items
}

// getOrCreateChildItem looks up a child item from the shared map, or creates one if it doesn't exist.
// It uses path-based keys to uniquely identify children across different parent blueprints.
func (i *DeployItem) getOrCreateChildItem(name string, action ActionType, nestedChanges *changes.BlueprintChanges, skipped bool) (*ChildDeployItem, string) {
	// Build path-based key: parentPath/childName
	childPath := i.BuildChildPath(name)

	if i.childrenByName != nil {
		// First try path-based lookup
		if existing, ok := i.childrenByName[childPath]; ok {
			existing.Skipped = skipped
			return existing, childPath
		}
		// Fall back to simple name lookup for backwards compatibility
		if existing, ok := i.childrenByName[name]; ok {
			existing.Skipped = skipped
			return existing, childPath
		}
	}
	// Create a new item if not found
	newItem := &ChildDeployItem{
		Name:    name,
		Action:  action,
		Changes: nestedChanges,
		Skipped: skipped,
	}
	// Store in the shared map using path-based key
	if i.childrenByName != nil {
		i.childrenByName[childPath] = newItem
	}
	return newItem, childPath
}

// appendChildLinkItems adds link items from this child's changes.
// Links are found within resource changes as NewOutboundLinks, OutboundLinkChanges, and RemovedOutboundLinks.
// It uses the shared lookup maps to ensure status updates are reflected in the UI.
// It also tracks which links have been added in the addedLinks map.
func (i *DeployItem) appendChildLinkItems(items []splitpane.Item, parentSkipped bool, addedLinks map[string]bool) []splitpane.Item {
	childChanges := i.Changes

	// Extract links from new resources
	for resourceAName, resourceChanges := range childChanges.NewResources {
		for resourceBName := range resourceChanges.NewOutboundLinks {
			linkName := resourceAName + "::" + resourceBName
			linkItem, linkPath := i.getOrCreateLinkItem(linkName, resourceAName, resourceBName, ActionCreate, parentSkipped)
			items = append(items, &DeployItem{
				Type:            ItemTypeLink,
				Link:            linkItem,
				ParentChild:     i.GetID(),
				Depth:           i.Depth + 1,
				Path:            linkPath,
				InstanceState:   i.InstanceState,
				childrenByName:  i.childrenByName,
				resourcesByName: i.resourcesByName,
				linksByName:     i.linksByName,
			})
			addedLinks[linkName] = true
		}
	}

	// Extract links from changed resources
	for resourceAName, resourceChanges := range childChanges.ResourceChanges {
		// New outbound links from changed resources
		for resourceBName := range resourceChanges.NewOutboundLinks {
			linkName := resourceAName + "::" + resourceBName
			linkItem, linkPath := i.getOrCreateLinkItem(linkName, resourceAName, resourceBName, ActionCreate, parentSkipped)
			items = append(items, &DeployItem{
				Type:            ItemTypeLink,
				Link:            linkItem,
				ParentChild:     i.GetID(),
				Depth:           i.Depth + 1,
				Path:            linkPath,
				InstanceState:   i.InstanceState,
				childrenByName:  i.childrenByName,
				resourcesByName: i.resourcesByName,
				linksByName:     i.linksByName,
			})
			addedLinks[linkName] = true
		}

		// Changed outbound links
		for resourceBName := range resourceChanges.OutboundLinkChanges {
			linkName := resourceAName + "::" + resourceBName
			linkItem, linkPath := i.getOrCreateLinkItem(linkName, resourceAName, resourceBName, ActionUpdate, parentSkipped)
			items = append(items, &DeployItem{
				Type:            ItemTypeLink,
				Link:            linkItem,
				ParentChild:     i.GetID(),
				Depth:           i.Depth + 1,
				Path:            linkPath,
				InstanceState:   i.InstanceState,
				childrenByName:  i.childrenByName,
				resourcesByName: i.resourcesByName,
				linksByName:     i.linksByName,
			})
			addedLinks[linkName] = true
		}

		// Removed outbound links
		for _, linkName := range resourceChanges.RemovedOutboundLinks {
			linkItem, linkPath := i.getOrCreateLinkItem(
				linkName,
				ExtractResourceAFromLinkName(linkName),
				ExtractResourceBFromLinkName(linkName),
				ActionDelete,
				parentSkipped,
			)
			items = append(items, &DeployItem{
				Type:            ItemTypeLink,
				Link:            linkItem,
				ParentChild:     i.GetID(),
				Depth:           i.Depth + 1,
				Path:            linkPath,
				InstanceState:   i.InstanceState,
				childrenByName:  i.childrenByName,
				resourcesByName: i.resourcesByName,
				linksByName:     i.linksByName,
			})
			addedLinks[linkName] = true
		}
	}

	// Also check top-level RemovedLinks
	for _, linkName := range childChanges.RemovedLinks {
		linkItem, linkPath := i.getOrCreateLinkItem(
			linkName,
			ExtractResourceAFromLinkName(linkName),
			ExtractResourceBFromLinkName(linkName),
			ActionDelete,
			parentSkipped,
		)
		items = append(items, &DeployItem{
			Type:            ItemTypeLink,
			Link:            linkItem,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            linkPath,
			InstanceState:   i.InstanceState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedLinks[linkName] = true
	}

	return items
}

// appendUnchangedItemsFromInstanceState adds items from InstanceState that have no changes.
// This ensures all resources and children are visible in the navigation even if they have no changes.
func (i *DeployItem) appendUnchangedItemsFromInstanceState(
	items []splitpane.Item,
	parentSkipped bool,
	addedResources map[string]bool,
	addedChildren map[string]bool,
	addedLinks map[string]bool,
) []splitpane.Item {
	if i.InstanceState == nil {
		return items
	}

	// Determine the default action for new items based on the parent's action.
	// If the parent is in inspect mode (ActionInspect), children should also use ActionInspect.
	defaultAction := i.GetDefaultChildAction()

	// Add resources from instance state that have no changes
	for _, resourceState := range i.InstanceState.Resources {
		if addedResources[resourceState.Name] {
			continue
		}
		resourceItem, resourcePath := i.getOrCreateResourceItemFromState(resourceState, defaultAction, parentSkipped)
		items = append(items, &DeployItem{
			Type:            ItemTypeResource,
			Resource:        resourceItem,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            resourcePath,
			InstanceState:   i.InstanceState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedResources[resourceState.Name] = true
	}

	// Add child blueprints from instance state that have no changes
	for name, childState := range i.InstanceState.ChildBlueprints {
		if addedChildren[name] {
			continue
		}
		childItem, childPath := i.getOrCreateChildItemFromState(name, defaultAction, parentSkipped)
		items = append(items, &DeployItem{
			Type:            ItemTypeChild,
			Child:           childItem,
			Changes:         childItem.Changes,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            childPath,
			InstanceState:   childState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedChildren[name] = true
	}

	// Add links from instance state that have no changes
	for linkName, linkState := range i.InstanceState.Links {
		if addedLinks[linkName] {
			continue
		}
		linkItem, linkPath := i.getOrCreateLinkItemFromState(linkName, linkState, defaultAction, parentSkipped)
		items = append(items, &DeployItem{
			Type:            ItemTypeLink,
			Link:            linkItem,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            linkPath,
			InstanceState:   i.InstanceState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedLinks[linkName] = true
	}

	return items
}

// GetDefaultChildAction returns the action to use for child items created from state.
// If the parent has ActionInspect, children inherit that; otherwise ActionNoChange.
func (i *DeployItem) GetDefaultChildAction() ActionType {
	if i.Child != nil && i.Child.Action == ActionInspect {
		return ActionInspect
	}
	return ActionNoChange
}

// appendItemsFromSharedMaps discovers items from the shared lookup maps that haven't been
// added from Changes or InstanceState. This handles the streaming scenario where items
// are added to the maps via events but the parent's Changes/InstanceState don't yet reflect them.
func (i *DeployItem) appendItemsFromSharedMaps(
	items []splitpane.Item,
	parentSkipped bool,
	addedResources map[string]bool,
	addedChildren map[string]bool,
	addedLinks map[string]bool,
) []splitpane.Item {
	if i.Child == nil {
		return items
	}

	pathPrefix := i.BuildChildPath("")
	defaultAction := i.GetDefaultChildAction()

	items = i.appendResourcesFromSharedMaps(items, pathPrefix, parentSkipped, addedResources, defaultAction)
	items = i.appendChildrenFromSharedMaps(items, pathPrefix, parentSkipped, addedChildren, defaultAction)
	items = i.appendLinksFromSharedMaps(items, pathPrefix, parentSkipped, addedLinks, defaultAction)

	return items
}

func (i *DeployItem) appendResourcesFromSharedMaps(
	items []splitpane.Item,
	pathPrefix string,
	parentSkipped bool,
	addedResources map[string]bool,
	defaultAction ActionType,
) []splitpane.Item {
	if i.resourcesByName == nil {
		return items
	}

	for path, resourceItem := range i.resourcesByName {
		if !i.IsDirectChild(path, pathPrefix) {
			continue
		}
		if addedResources[resourceItem.Name] {
			continue
		}
		resourceItem.Skipped = parentSkipped
		items = append(items, &DeployItem{
			Type:            ItemTypeResource,
			Resource:        resourceItem,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            path,
			InstanceState:   i.InstanceState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedResources[resourceItem.Name] = true
	}
	return items
}

func (i *DeployItem) appendChildrenFromSharedMaps(
	items []splitpane.Item,
	pathPrefix string,
	parentSkipped bool,
	addedChildren map[string]bool,
	defaultAction ActionType,
) []splitpane.Item {
	if i.childrenByName == nil {
		return items
	}

	for path, childItem := range i.childrenByName {
		if !i.IsDirectChild(path, pathPrefix) {
			continue
		}
		if addedChildren[childItem.Name] {
			continue
		}
		childItem.Skipped = parentSkipped
		items = append(items, &DeployItem{
			Type:            ItemTypeChild,
			Child:           childItem,
			Changes:         childItem.Changes,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            path,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedChildren[childItem.Name] = true
	}
	return items
}

func (i *DeployItem) appendLinksFromSharedMaps(
	items []splitpane.Item,
	pathPrefix string,
	parentSkipped bool,
	addedLinks map[string]bool,
	defaultAction ActionType,
) []splitpane.Item {
	if i.linksByName == nil {
		return items
	}

	for path, linkItem := range i.linksByName {
		if !i.IsDirectChild(path, pathPrefix) {
			continue
		}
		if addedLinks[linkItem.LinkName] {
			continue
		}
		linkItem.Skipped = parentSkipped
		items = append(items, &DeployItem{
			Type:            ItemTypeLink,
			Link:            linkItem,
			ParentChild:     i.GetID(),
			Depth:           i.Depth + 1,
			Path:            path,
			InstanceState:   i.InstanceState,
			childrenByName:  i.childrenByName,
			resourcesByName: i.resourcesByName,
			linksByName:     i.linksByName,
		})
		addedLinks[linkItem.LinkName] = true
	}
	return items
}

// IsDirectChild checks if the given path represents a direct child of the pathPrefix.
// A direct child has exactly one path component after the prefix.
// Example: pathPrefix="parent/", path="parent/child" -> true
// Example: pathPrefix="parent/", path="parent/child/grandchild" -> false
func (i *DeployItem) IsDirectChild(path, pathPrefix string) bool {
	if len(path) <= len(pathPrefix) {
		return false
	}
	if path[:len(pathPrefix)] != pathPrefix {
		return false
	}
	// Check there's no additional slash in the remainder
	remainder := path[len(pathPrefix):]
	for _, c := range remainder {
		if c == '/' {
			return false
		}
	}
	return true
}

// getOrCreateLinkItem looks up a link item from the shared map, or creates one if it doesn't exist.
// It uses path-based keys to uniquely identify links across different child blueprints.
func (i *DeployItem) getOrCreateLinkItem(linkName, resourceAName, resourceBName string, action ActionType, skipped bool) (*LinkDeployItem, string) {
	// Build path-based key: parentPath/linkName
	linkPath := i.BuildChildPath(linkName)

	if i.linksByName != nil {
		// First try path-based lookup
		if existing, ok := i.linksByName[linkPath]; ok {
			existing.Skipped = skipped
			return existing, linkPath
		}
		// Fall back to simple name lookup for backwards compatibility
		if existing, ok := i.linksByName[linkName]; ok {
			existing.Skipped = skipped
			return existing, linkPath
		}
	}
	// Create a new item if not found
	newItem := &LinkDeployItem{
		LinkName:      linkName,
		ResourceAName: resourceAName,
		ResourceBName: resourceBName,
		Action:        action,
		Skipped:       skipped,
	}
	// Store in the shared map using path-based key
	if i.linksByName != nil {
		i.linksByName[linkPath] = newItem
	}
	return newItem, linkPath
}

// getOrCreateResourceItemFromState looks up a resource item from the shared map, or creates one
// using instance state data. It handles migration from simple name keys to path-based keys
// and hydrates the item with state data.
func (i *DeployItem) getOrCreateResourceItemFromState(resourceState *state.ResourceState, action ActionType, skipped bool) (*ResourceDeployItem, string) {
	resourcePath := i.BuildChildPath(resourceState.Name)

	if i.resourcesByName != nil {
		// First try path-based lookup
		if existing, ok := i.resourcesByName[resourcePath]; ok {
			existing.Skipped = skipped
			i.hydrateResourceItemFromState(existing, resourceState)
			return existing, resourcePath
		}
		// Fall back to simple name lookup and migrate to path-based key
		if existing, ok := i.resourcesByName[resourceState.Name]; ok {
			existing.Skipped = skipped
			delete(i.resourcesByName, resourceState.Name)
			i.resourcesByName[resourcePath] = existing
			i.hydrateResourceItemFromState(existing, resourceState)
			return existing, resourcePath
		}
	}

	// Create new item from state
	newItem := &ResourceDeployItem{
		Name:          resourceState.Name,
		ResourceID:    resourceState.ResourceID,
		ResourceType:  resourceState.Type,
		Action:        action,
		Status:        resourceState.Status,
		ResourceState: resourceState,
		Skipped:       skipped,
	}
	if i.resourcesByName != nil {
		i.resourcesByName[resourcePath] = newItem
	}
	return newItem, resourcePath
}

// hydrateResourceItemFromState fills in resource item fields from state if not already set.
func (i *DeployItem) hydrateResourceItemFromState(item *ResourceDeployItem, resourceState *state.ResourceState) {
	if item.ResourceState == nil {
		item.ResourceState = resourceState
	}
	if item.ResourceID == "" {
		item.ResourceID = resourceState.ResourceID
	}
	if item.ResourceType == "" {
		item.ResourceType = resourceState.Type
	}
}

// getOrCreateChildItemFromState looks up a child item from the shared map, or creates one
// using instance state data. It handles migration from simple name keys to path-based keys.
func (i *DeployItem) getOrCreateChildItemFromState(name string, action ActionType, skipped bool) (*ChildDeployItem, string) {
	childPath := i.BuildChildPath(name)

	if i.childrenByName != nil {
		// First try path-based lookup
		if existing, ok := i.childrenByName[childPath]; ok {
			existing.Skipped = skipped
			if existing.Changes == nil {
				existing.Changes = &changes.BlueprintChanges{}
			}
			return existing, childPath
		}
		// Fall back to simple name lookup and migrate to path-based key
		if existing, ok := i.childrenByName[name]; ok {
			existing.Skipped = skipped
			if existing.Changes == nil {
				existing.Changes = &changes.BlueprintChanges{}
			}
			delete(i.childrenByName, name)
			i.childrenByName[childPath] = existing
			return existing, childPath
		}
	}

	// Create new item
	newItem := &ChildDeployItem{
		Name:    name,
		Action:  action,
		Skipped: skipped,
		Changes: &changes.BlueprintChanges{},
	}
	if i.childrenByName != nil {
		i.childrenByName[childPath] = newItem
	}
	return newItem, childPath
}

// getOrCreateLinkItemFromState looks up a link item from the shared map, or creates one
// using instance state data. It handles migration from simple name keys to path-based keys.
func (i *DeployItem) getOrCreateLinkItemFromState(linkName string, linkState *state.LinkState, action ActionType, skipped bool) (*LinkDeployItem, string) {
	linkPath := i.BuildChildPath(linkName)

	if i.linksByName != nil {
		// First try path-based lookup
		if existing, ok := i.linksByName[linkPath]; ok {
			existing.Skipped = skipped
			if existing.LinkID == "" {
				existing.LinkID = linkState.LinkID
			}
			return existing, linkPath
		}
		// Fall back to simple name lookup and migrate to path-based key
		if existing, ok := i.linksByName[linkName]; ok {
			existing.Skipped = skipped
			if existing.LinkID == "" {
				existing.LinkID = linkState.LinkID
			}
			delete(i.linksByName, linkName)
			i.linksByName[linkPath] = existing
			return existing, linkPath
		}
	}

	// Create new item from state
	newItem := &LinkDeployItem{
		LinkID:        linkState.LinkID,
		LinkName:      linkName,
		ResourceAName: ExtractResourceAFromLinkName(linkName),
		ResourceBName: ExtractResourceBFromLinkName(linkName),
		Action:        action,
		Status:        linkState.Status,
		Skipped:       skipped,
	}
	if i.linksByName != nil {
		i.linksByName[linkPath] = newItem
	}
	return newItem, linkPath
}

// ToSplitPaneItems converts a slice of DeployItems to splitpane.Items.
func ToSplitPaneItems(items []DeployItem) []splitpane.Item {
	result := make([]splitpane.Item, len(items))
	for idx := range items {
		result[idx] = &items[idx]
	}
	return result
}

// MakeChildDeployItem creates a DeployItem for a child blueprint with lookup maps.
// This allows child blueprints to properly navigate and display their nested items.
func MakeChildDeployItem(
	child *ChildDeployItem,
	childChanges *changes.BlueprintChanges,
	instanceState *state.InstanceState,
	childrenByName map[string]*ChildDeployItem,
	resourcesByName map[string]*ResourceDeployItem,
	linksByName map[string]*LinkDeployItem,
) DeployItem {
	return DeployItem{
		Type:            ItemTypeChild,
		Child:           child,
		Changes:         childChanges,
		InstanceState:   instanceState,
		childrenByName:  childrenByName,
		resourcesByName: resourcesByName,
		linksByName:     linksByName,
	}
}
