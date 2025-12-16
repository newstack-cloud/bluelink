package deployui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
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
				return "⊘" // Skipped indicator
			}
			return resourceStatusIcon(i.Resource.Status)
		}
	case ItemTypeChild:
		if i.Child != nil {
			if i.Child.Skipped {
				return "⊘" // Skipped indicator
			}
			return instanceStatusIcon(i.Child.Status)
		}
	case ItemTypeLink:
		if i.Link != nil {
			if i.Link.Skipped {
				return "⊘" // Skipped indicator
			}
			return linkStatusIcon(i.Link.Status)
		}
	}
	return "○"
}

func resourceStatusIcon(status core.ResourceStatus) string {
	switch status {
	case core.ResourceStatusCreating, core.ResourceStatusUpdating, core.ResourceStatusDestroying:
		return "◐"
	case core.ResourceStatusCreated, core.ResourceStatusUpdated, core.ResourceStatusDestroyed:
		return "✓"
	case core.ResourceStatusCreateFailed, core.ResourceStatusUpdateFailed, core.ResourceStatusDestroyFailed:
		return "✗"
	case core.ResourceStatusRollingBack:
		return "↺"
	case core.ResourceStatusRollbackFailed:
		return "⚠"
	case core.ResourceStatusRollbackComplete:
		return "⟲"
	default:
		return "○" // Pending/Unknown
	}
}

func instanceStatusIcon(status core.InstanceStatus) string {
	switch status {
	case core.InstanceStatusPreparing:
		return "○"
	case core.InstanceStatusDeploying, core.InstanceStatusUpdating, core.InstanceStatusDestroying:
		return "◐"
	case core.InstanceStatusDeployed, core.InstanceStatusUpdated, core.InstanceStatusDestroyed:
		return "✓"
	case core.InstanceStatusDeployFailed, core.InstanceStatusUpdateFailed, core.InstanceStatusDestroyFailed:
		return "✗"
	case core.InstanceStatusDeployRollingBack, core.InstanceStatusUpdateRollingBack, core.InstanceStatusDestroyRollingBack:
		return "↺"
	case core.InstanceStatusDeployRollbackFailed, core.InstanceStatusUpdateRollbackFailed, core.InstanceStatusDestroyRollbackFailed:
		return "⚠"
	case core.InstanceStatusDeployRollbackComplete, core.InstanceStatusUpdateRollbackComplete, core.InstanceStatusDestroyRollbackComplete:
		return "⟲"
	default:
		return "○"
	}
}

func linkStatusIcon(status core.LinkStatus) string {
	switch status {
	case core.LinkStatusCreating, core.LinkStatusUpdating, core.LinkStatusDestroying:
		return "◐"
	case core.LinkStatusCreated, core.LinkStatusUpdated, core.LinkStatusDestroyed:
		return "✓"
	case core.LinkStatusCreateFailed, core.LinkStatusUpdateFailed, core.LinkStatusDestroyFailed:
		return "✗"
	case core.LinkStatusCreateRollingBack, core.LinkStatusUpdateRollingBack, core.LinkStatusDestroyRollingBack:
		return "↺"
	case core.LinkStatusCreateRollbackFailed, core.LinkStatusUpdateRollbackFailed, core.LinkStatusDestroyRollbackFailed:
		return "⚠"
	case core.LinkStatusCreateRollbackComplete, core.LinkStatusUpdateRollbackComplete, core.LinkStatusDestroyRollbackComplete:
		return "⟲"
	default:
		return "○"
	}
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
			return styleResourceIcon(icon, i.Resource.Status, s)
		}
	case ItemTypeChild:
		if i.Child != nil {
			if i.Child.Skipped {
				return s.Warning.Render(icon)
			}
			return styleInstanceIcon(icon, i.Child.Status, s)
		}
	case ItemTypeLink:
		if i.Link != nil {
			if i.Link.Skipped {
				return s.Warning.Render(icon)
			}
			return styleLinkIcon(icon, i.Link.Status, s)
		}
	}
	return icon
}

func styleResourceIcon(icon string, status core.ResourceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.ResourceStatusCreating, core.ResourceStatusUpdating, core.ResourceStatusDestroying:
		return s.Info.Render(icon)
	case core.ResourceStatusCreated, core.ResourceStatusUpdated, core.ResourceStatusDestroyed:
		return successStyle.Render(icon)
	case core.ResourceStatusCreateFailed, core.ResourceStatusUpdateFailed, core.ResourceStatusDestroyFailed,
		core.ResourceStatusRollbackFailed:
		return s.Error.Render(icon)
	case core.ResourceStatusRollingBack:
		return s.Warning.Render(icon)
	case core.ResourceStatusRollbackComplete:
		return s.Muted.Render(icon)
	default:
		return s.Muted.Render(icon)
	}
}

func styleInstanceIcon(icon string, status core.InstanceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.InstanceStatusDeploying, core.InstanceStatusUpdating, core.InstanceStatusDestroying:
		return s.Info.Render(icon)
	case core.InstanceStatusDeployed, core.InstanceStatusUpdated, core.InstanceStatusDestroyed:
		return successStyle.Render(icon)
	case core.InstanceStatusDeployFailed, core.InstanceStatusUpdateFailed, core.InstanceStatusDestroyFailed,
		core.InstanceStatusDeployRollbackFailed, core.InstanceStatusUpdateRollbackFailed, core.InstanceStatusDestroyRollbackFailed:
		return s.Error.Render(icon)
	case core.InstanceStatusDeployRollingBack, core.InstanceStatusUpdateRollingBack, core.InstanceStatusDestroyRollingBack:
		return s.Warning.Render(icon)
	case core.InstanceStatusDeployRollbackComplete, core.InstanceStatusUpdateRollbackComplete, core.InstanceStatusDestroyRollbackComplete:
		return s.Muted.Render(icon)
	default:
		return s.Muted.Render(icon)
	}
}

func styleLinkIcon(icon string, status core.LinkStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.LinkStatusCreating, core.LinkStatusUpdating, core.LinkStatusDestroying:
		return s.Info.Render(icon)
	case core.LinkStatusCreated, core.LinkStatusUpdated, core.LinkStatusDestroyed:
		return successStyle.Render(icon)
	case core.LinkStatusCreateFailed, core.LinkStatusUpdateFailed, core.LinkStatusDestroyFailed,
		core.LinkStatusCreateRollbackFailed, core.LinkStatusUpdateRollbackFailed, core.LinkStatusDestroyRollbackFailed:
		return s.Error.Render(icon)
	case core.LinkStatusCreateRollingBack, core.LinkStatusUpdateRollingBack, core.LinkStatusDestroyRollingBack:
		return s.Warning.Render(icon)
	case core.LinkStatusCreateRollbackComplete, core.LinkStatusUpdateRollbackComplete, core.LinkStatusDestroyRollbackComplete:
		return s.Muted.Render(icon)
	default:
		return s.Muted.Render(icon)
	}
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
func (i *DeployItem) IsExpandable() bool {
	return i.Type == ItemTypeChild && i.Changes != nil
}

// CanDrillDown returns true if the item can be drilled into.
func (i *DeployItem) CanDrillDown() bool {
	return i.Type == ItemTypeChild && i.Changes != nil
}

// GetChildren returns child items when expanded.
// Uses the Changes data from the changeset to build the hierarchy.
func (i *DeployItem) GetChildren() []splitpane.Item {
	if i.Type != ItemTypeChild || i.Changes == nil {
		return nil
	}

	var items []splitpane.Item
	items = i.appendChildResourceItems(items)
	items = i.appendNestedChildItems(items)
	items = i.appendChildLinkItems(items)
	return items
}

// appendChildResourceItems adds resource items from this child's changes.
func (i *DeployItem) appendChildResourceItems(items []splitpane.Item) []splitpane.Item {
	childChanges := i.Changes

	// New resources
	for name := range childChanges.NewResources {
		items = append(items, &DeployItem{
			Type: ItemTypeResource,
			Resource: &ResourceDeployItem{
				Name:   name,
				Action: ActionCreate,
			},
			ParentChild: i.GetID(),
			Depth:       i.Depth + 1,
		})
	}

	// Changed resources
	for name, rc := range childChanges.ResourceChanges {
		action := ActionUpdate
		if rc.MustRecreate {
			action = ActionRecreate
		}
		items = append(items, &DeployItem{
			Type: ItemTypeResource,
			Resource: &ResourceDeployItem{
				Name:   name,
				Action: action,
			},
			ParentChild: i.GetID(),
			Depth:       i.Depth + 1,
		})
	}

	// Removed resources
	for _, name := range childChanges.RemovedResources {
		items = append(items, &DeployItem{
			Type: ItemTypeResource,
			Resource: &ResourceDeployItem{
				Name:   name,
				Action: ActionDelete,
			},
			ParentChild: i.GetID(),
			Depth:       i.Depth + 1,
		})
	}

	return items
}

// appendNestedChildItems adds nested child blueprint items from this child's changes.
func (i *DeployItem) appendNestedChildItems(items []splitpane.Item) []splitpane.Item {
	childChanges := i.Changes

	// New children - convert NewBlueprintDefinition to BlueprintChanges
	for name, nc := range childChanges.NewChildren {
		nestedChanges := &changes.BlueprintChanges{
			NewResources: nc.NewResources,
			NewChildren:  nc.NewChildren,
		}
		items = append(items, &DeployItem{
			Type: ItemTypeChild,
			Child: &ChildDeployItem{
				Name:    name,
				Action:  ActionCreate,
				Changes: nestedChanges,
			},
			Changes:     nestedChanges,
			ParentChild: i.GetID(),
			Depth:       i.Depth + 1,
		})
	}

	// Changed children
	for name, cc := range childChanges.ChildChanges {
		ccCopy := cc
		items = append(items, &DeployItem{
			Type: ItemTypeChild,
			Child: &ChildDeployItem{
				Name:    name,
				Action:  ActionUpdate,
				Changes: &ccCopy,
			},
			Changes:     &ccCopy,
			ParentChild: i.GetID(),
			Depth:       i.Depth + 1,
		})
	}

	// Removed children
	for _, name := range childChanges.RemovedChildren {
		items = append(items, &DeployItem{
			Type: ItemTypeChild,
			Child: &ChildDeployItem{
				Name:   name,
				Action: ActionDelete,
			},
			ParentChild: i.GetID(),
			Depth:       i.Depth + 1,
		})
	}

	return items
}

// appendChildLinkItems adds link items from this child's changes.
// Links are found within resource changes as NewOutboundLinks, OutboundLinkChanges, and RemovedOutboundLinks.
func (i *DeployItem) appendChildLinkItems(items []splitpane.Item) []splitpane.Item {
	childChanges := i.Changes

	// Extract links from new resources
	for resourceAName, resourceChanges := range childChanges.NewResources {
		for resourceBName := range resourceChanges.NewOutboundLinks {
			linkName := resourceAName + "::" + resourceBName
			items = append(items, &DeployItem{
				Type: ItemTypeLink,
				Link: &LinkDeployItem{
					LinkName:      linkName,
					ResourceAName: resourceAName,
					ResourceBName: resourceBName,
					Action:        ActionCreate,
				},
				ParentChild: i.GetID(),
				Depth:       i.Depth + 1,
			})
		}
	}

	// Extract links from changed resources
	for resourceAName, resourceChanges := range childChanges.ResourceChanges {
		// New outbound links from changed resources
		for resourceBName := range resourceChanges.NewOutboundLinks {
			linkName := resourceAName + "::" + resourceBName
			items = append(items, &DeployItem{
				Type: ItemTypeLink,
				Link: &LinkDeployItem{
					LinkName:      linkName,
					ResourceAName: resourceAName,
					ResourceBName: resourceBName,
					Action:        ActionCreate,
				},
				ParentChild: i.GetID(),
				Depth:       i.Depth + 1,
			})
		}

		// Changed outbound links
		for resourceBName := range resourceChanges.OutboundLinkChanges {
			linkName := resourceAName + "::" + resourceBName
			items = append(items, &DeployItem{
				Type: ItemTypeLink,
				Link: &LinkDeployItem{
					LinkName:      linkName,
					ResourceAName: resourceAName,
					ResourceBName: resourceBName,
					Action:        ActionUpdate,
				},
				ParentChild: i.GetID(),
				Depth:       i.Depth + 1,
			})
		}

		// Removed outbound links
		for _, linkName := range resourceChanges.RemovedOutboundLinks {
			items = append(items, &DeployItem{
				Type: ItemTypeLink,
				Link: &LinkDeployItem{
					LinkName:      linkName,
					ResourceAName: extractResourceAFromLinkName(linkName),
					ResourceBName: extractResourceBFromLinkName(linkName),
					Action:        ActionDelete,
				},
				ParentChild: i.GetID(),
				Depth:       i.Depth + 1,
			})
		}
	}

	// Also check top-level RemovedLinks
	for _, linkName := range childChanges.RemovedLinks {
		items = append(items, &DeployItem{
			Type: ItemTypeLink,
			Link: &LinkDeployItem{
				LinkName:      linkName,
				ResourceAName: extractResourceAFromLinkName(linkName),
				ResourceBName: extractResourceBFromLinkName(linkName),
				Action:        ActionDelete,
			},
			ParentChild: i.GetID(),
			Depth:       i.Depth + 1,
		})
	}

	return items
}

// ToSplitPaneItems converts a slice of DeployItems to splitpane.Items.
func ToSplitPaneItems(items []DeployItem) []splitpane.Item {
	result := make([]splitpane.Item, len(items))
	for idx := range items {
		result[idx] = &items[idx]
	}
	return result
}
