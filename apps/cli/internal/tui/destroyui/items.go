package destroyui

import (
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// Ensure DestroyItem implements splitpane.Item.
var _ splitpane.Item = (*DestroyItem)(nil)

// GetID returns a unique identifier for the item.
func (d *DestroyItem) GetID() string {
	switch d.Type {
	case ItemTypeResource:
		if d.Resource != nil {
			return d.Resource.Name
		}
	case ItemTypeChild:
		if d.Child != nil {
			return d.Child.Name
		}
	case ItemTypeLink:
		if d.Link != nil {
			return d.Link.LinkName
		}
	}
	return ""
}

// GetName returns the display name for the item.
func (d *DestroyItem) GetName() string {
	return d.GetID()
}

// GetIcon returns a status icon for the item.
func (d *DestroyItem) GetIcon(selected bool) string {
	return d.getIconChar()
}

func (d *DestroyItem) getIconChar() string {
	switch d.Type {
	case ItemTypeResource:
		if d.Resource != nil {
			if d.Resource.Skipped {
				return shared.IconSkipped
			}
			if d.Resource.Action == ActionNoChange {
				return shared.IconNoChange
			}
			return shared.ResourceStatusIcon(d.Resource.Status)
		}
	case ItemTypeChild:
		if d.Child != nil {
			if d.Child.Skipped {
				return shared.IconSkipped
			}
			if d.Child.Action == ActionNoChange {
				return shared.IconNoChange
			}
			return shared.InstanceStatusIcon(d.Child.Status)
		}
	case ItemTypeLink:
		if d.Link != nil {
			if d.Link.Skipped {
				return shared.IconSkipped
			}
			if d.Link.Action == ActionNoChange {
				return shared.IconNoChange
			}
			return shared.LinkStatusIcon(d.Link.Status)
		}
	}
	return shared.IconPending
}

// GetIconStyled returns a styled icon for the item.
func (d *DestroyItem) GetIconStyled(s *styles.Styles, styled bool) string {
	icon := d.getIconChar()
	if !styled {
		return icon
	}

	switch d.Type {
	case ItemTypeResource:
		if d.Resource != nil {
			if d.Resource.Skipped {
				return s.Warning.Render(icon)
			}
			if d.Resource.Action == ActionNoChange {
				return s.Muted.Render(icon)
			}
			return shared.StyleResourceIcon(icon, d.Resource.Status, s)
		}
	case ItemTypeChild:
		if d.Child != nil {
			if d.Child.Skipped {
				return s.Warning.Render(icon)
			}
			if d.Child.Action == ActionNoChange {
				return s.Muted.Render(icon)
			}
			return shared.StyleInstanceIcon(icon, d.Child.Status, s)
		}
	case ItemTypeLink:
		if d.Link != nil {
			if d.Link.Skipped {
				return s.Warning.Render(icon)
			}
			if d.Link.Action == ActionNoChange {
				return s.Muted.Render(icon)
			}
			return shared.StyleLinkIcon(icon, d.Link.Status, s)
		}
	}
	return icon
}

// GetAction returns the action badge text.
func (d *DestroyItem) GetAction() string {
	switch d.Type {
	case ItemTypeResource:
		if d.Resource != nil {
			return string(d.Resource.Action)
		}
	case ItemTypeChild:
		if d.Child != nil {
			return string(d.Child.Action)
		}
	case ItemTypeLink:
		if d.Link != nil {
			return string(d.Link.Action)
		}
	}
	return ""
}

// GetDepth returns the nesting depth for indentation.
func (d *DestroyItem) GetDepth() int {
	return d.Depth
}

// GetParentID returns the parent item ID.
func (d *DestroyItem) GetParentID() string {
	return d.ParentChild
}

// GetItemType returns the type for section grouping.
func (d *DestroyItem) GetItemType() string {
	return string(d.Type)
}

// IsExpandable returns true if the item can be expanded in-place.
func (d *DestroyItem) IsExpandable() bool {
	return d.Type == ItemTypeChild && (d.Changes != nil || d.InstanceState != nil)
}

// CanDrillDown returns true if the item can be drilled into.
func (d *DestroyItem) CanDrillDown() bool {
	return d.Type == ItemTypeChild && (d.Changes != nil || d.InstanceState != nil)
}

// GetChildren returns child items when expanded.
func (d *DestroyItem) GetChildren() []splitpane.Item {
	if d.Type != ItemTypeChild {
		return nil
	}

	if d.Changes == nil && d.InstanceState == nil {
		return nil
	}

	parentSkipped := d.Child != nil && d.Child.Skipped
	var items []splitpane.Item

	// Add items from changes if available
	if d.Changes != nil {
		items = d.appendResourceItems(items, parentSkipped)
		items = d.appendChildItems(items, parentSkipped)
	}

	return items
}

func (d *DestroyItem) appendResourceItems(items []splitpane.Item, parentSkipped bool) []splitpane.Item {
	// Handle removed resources (the primary case for destroy)
	for _, name := range d.Changes.RemovedResources {
		resourceItem := d.getOrCreateResourceItem(name, ActionDelete, parentSkipped)
		resourcePath := d.buildChildPath(name)
		items = append(items, &DestroyItem{
			Type:            ItemTypeResource,
			Resource:        resourceItem,
			ParentChild:     d.GetID(),
			Depth:           d.Depth + 1,
			Path:            resourcePath,
			InstanceState:   d.InstanceState,
			childrenByName:  d.childrenByName,
			resourcesByName: d.resourcesByName,
			linksByName:     d.linksByName,
		})
	}

	// Handle resources with changes (for partial destroys or updates)
	for name := range d.Changes.ResourceChanges {
		resourceItem := d.getOrCreateResourceItem(name, ActionUpdate, parentSkipped)
		resourcePath := d.buildChildPath(name)
		items = append(items, &DestroyItem{
			Type:            ItemTypeResource,
			Resource:        resourceItem,
			ParentChild:     d.GetID(),
			Depth:           d.Depth + 1,
			Path:            resourcePath,
			InstanceState:   d.InstanceState,
			childrenByName:  d.childrenByName,
			resourcesByName: d.resourcesByName,
			linksByName:     d.linksByName,
		})
	}

	return items
}

func (d *DestroyItem) appendChildItems(items []splitpane.Item, parentSkipped bool) []splitpane.Item {
	// Handle removed children
	for _, name := range d.Changes.RemovedChildren {
		childItem := d.getOrCreateChildItem(name, ActionDelete, nil, parentSkipped)
		childPath := d.buildChildPath(name)
		items = append(items, &DestroyItem{
			Type:            ItemTypeChild,
			Child:           childItem,
			ParentChild:     d.GetID(),
			Depth:           d.Depth + 1,
			Path:            childPath,
			childrenByName:  d.childrenByName,
			resourcesByName: d.resourcesByName,
			linksByName:     d.linksByName,
		})
	}

	// Handle child changes
	for name, cc := range d.Changes.ChildChanges {
		ccCopy := cc
		childItem := d.getOrCreateChildItem(name, ActionUpdate, &ccCopy, parentSkipped)
		childPath := d.buildChildPath(name)
		items = append(items, &DestroyItem{
			Type:            ItemTypeChild,
			Child:           childItem,
			Changes:         &ccCopy,
			ParentChild:     d.GetID(),
			Depth:           d.Depth + 1,
			Path:            childPath,
			childrenByName:  d.childrenByName,
			resourcesByName: d.resourcesByName,
			linksByName:     d.linksByName,
		})
	}

	return items
}

func (d *DestroyItem) getOrCreateResourceItem(name string, action ActionType, skipped bool) *ResourceDestroyItem {
	resourcePath := d.buildChildPath(name)

	if d.resourcesByName != nil {
		if existing, ok := d.resourcesByName[resourcePath]; ok {
			existing.Skipped = skipped
			return existing
		}
		if existing, ok := d.resourcesByName[name]; ok {
			existing.Skipped = skipped
			return existing
		}
	}

	newItem := &ResourceDestroyItem{
		Name:    name,
		Action:  action,
		Skipped: skipped,
	}
	if d.resourcesByName != nil {
		d.resourcesByName[resourcePath] = newItem
	}
	return newItem
}

func (d *DestroyItem) getOrCreateChildItem(name string, action ActionType, changes *changes.BlueprintChanges, skipped bool) *ChildDestroyItem {
	childPath := d.buildChildPath(name)

	if d.childrenByName != nil {
		if existing, ok := d.childrenByName[childPath]; ok {
			existing.Skipped = skipped
			return existing
		}
		if existing, ok := d.childrenByName[name]; ok {
			existing.Skipped = skipped
			return existing
		}
	}

	newItem := &ChildDestroyItem{
		Name:    name,
		Action:  action,
		Changes: changes,
		Skipped: skipped,
	}
	if d.childrenByName != nil {
		d.childrenByName[childPath] = newItem
	}
	return newItem
}

func (d *DestroyItem) buildChildPath(childName string) string {
	if d.Path == "" {
		if d.Child != nil {
			return d.Child.Name + "/" + childName
		}
		return childName
	}
	return d.Path + "/" + childName
}
