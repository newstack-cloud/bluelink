package stageui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// Ensure StageItem implements splitpane.Item
var _ splitpane.Item = (*StageItem)(nil)

// GetID returns a unique identifier for the item.
func (i *StageItem) GetID() string {
	return i.Name
}

// GetName returns the display name for the item.
func (i *StageItem) GetName() string {
	return i.Name
}

// GetIcon returns a status icon for the item.
// When selected is false, the icon is styled; when true, it's unstyled
// so the selection style can apply uniformly.
func (i *StageItem) GetIcon(selected bool) string {
	// This will be called with the styles from context
	// For now, return the plain icon - styling handled by IconWithStyles
	return i.getIconChar()
}

// getIconChar returns the icon character without styling.
func (i *StageItem) getIconChar() string {
	switch i.Action {
	case ActionCreate:
		return "✓"
	case ActionUpdate:
		return "~"
	case ActionDelete:
		return "-"
	case ActionRecreate:
		return "↻"
	default:
		return "○"
	}
}

// GetIconStyled returns a styled icon for the item.
func (i *StageItem) GetIconStyled(s *styles.Styles, styled bool) string {
	icon := i.getIconChar()
	if !styled {
		return icon
	}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch i.Action {
	case ActionCreate:
		return successStyle.Render(icon)
	case ActionUpdate:
		return s.Warning.Render(icon)
	case ActionDelete:
		return s.Error.Render(icon)
	case ActionRecreate:
		return s.Info.Render(icon)
	default:
		return s.Muted.Render(icon)
	}
}

// GetAction returns the action badge text.
func (i *StageItem) GetAction() string {
	return string(i.Action)
}

// GetDepth returns the nesting depth for indentation.
func (i *StageItem) GetDepth() int {
	return i.Depth
}

// GetParentID returns the parent item ID.
func (i *StageItem) GetParentID() string {
	return i.ParentChild
}

// GetItemType returns the type for section grouping.
func (i *StageItem) GetItemType() string {
	return string(i.Type)
}

// IsExpandable returns true if the item can be expanded in-place.
func (i *StageItem) IsExpandable() bool {
	return i.Type == ItemTypeChild && i.Changes != nil
}

// CanDrillDown returns true if the item can be drilled into.
func (i *StageItem) CanDrillDown() bool {
	if i.Type != ItemTypeChild {
		return false
	}
	_, ok := i.Changes.(*changes.BlueprintChanges)
	return ok
}

// GetChildren returns child items when expanded.
func (i *StageItem) GetChildren() []splitpane.Item {
	if i.Type != ItemTypeChild {
		return nil
	}
	childChanges, ok := i.Changes.(*changes.BlueprintChanges)
	if !ok || childChanges == nil {
		return nil
	}

	// Extract items from the child changes
	var items []splitpane.Item

	// New resources
	for name, rc := range childChanges.NewResources {
		rcCopy := rc
		resourceType, displayName := extractResourceTypeAndDisplayName(&rcCopy)
		items = append(items, &StageItem{
			Type:         ItemTypeResource,
			Name:         name,
			ResourceType: resourceType,
			DisplayName:  displayName,
			Action:       ActionCreate,
			Changes:      &rcCopy,
			New:          true,
			ParentChild:  i.Name,
			Depth:        i.Depth + 1,
		})
	}

	// Changed resources
	for name, rc := range childChanges.ResourceChanges {
		rcCopy := rc
		resourceType, displayName := extractResourceTypeAndDisplayName(&rcCopy)
		action := ActionUpdate
		if rcCopy.MustRecreate {
			action = ActionRecreate
		}
		items = append(items, &StageItem{
			Type:         ItemTypeResource,
			Name:         name,
			ResourceType: resourceType,
			DisplayName:  displayName,
			Action:       action,
			Changes:      &rcCopy,
			Recreate:     rcCopy.MustRecreate,
			ParentChild:  i.Name,
			Depth:        i.Depth + 1,
		})
	}

	// Removed resources
	for _, name := range childChanges.RemovedResources {
		items = append(items, &StageItem{
			Type:        ItemTypeResource,
			Name:        name,
			Action:      ActionDelete,
			Removed:     true,
			ParentChild: i.Name,
			Depth:       i.Depth + 1,
		})
	}

	// New children
	for name, nc := range childChanges.NewChildren {
		ncCopy := nc
		items = append(items, &StageItem{
			Type:   ItemTypeChild,
			Name:   name,
			Action: ActionCreate,
			Changes: &changes.BlueprintChanges{
				NewResources: ncCopy.NewResources,
				NewChildren:  ncCopy.NewChildren,
				NewExports:   ncCopy.NewExports,
			},
			New:         true,
			ParentChild: i.Name,
			Depth:       i.Depth + 1,
		})
	}

	// Changed children
	for name, cc := range childChanges.ChildChanges {
		ccCopy := cc
		items = append(items, &StageItem{
			Type:        ItemTypeChild,
			Name:        name,
			Action:      ActionUpdate,
			Changes:     &ccCopy,
			ParentChild: i.Name,
			Depth:       i.Depth + 1,
		})
	}

	// Removed children
	for _, name := range childChanges.RemovedChildren {
		items = append(items, &StageItem{
			Type:        ItemTypeChild,
			Name:        name,
			Action:      ActionDelete,
			Removed:     true,
			ParentChild: i.Name,
			Depth:       i.Depth + 1,
		})
	}

	return items
}

// ToSplitPaneItems converts a slice of StageItems to splitpane.Items.
func ToSplitPaneItems(items []StageItem) []splitpane.Item {
	result := make([]splitpane.Item, len(items))
	for i := range items {
		result[i] = &items[i]
	}
	return result
}
