package shared

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// ItemType represents the type of item in a stage or deploy list.
type ItemType string

const (
	ItemTypeResource ItemType = "resource"
	ItemTypeChild    ItemType = "child"
	ItemTypeLink     ItemType = "link"
)

// ActionType represents the action to be taken on an item.
type ActionType string

const (
	ActionCreate   ActionType = "CREATE"
	ActionUpdate   ActionType = "UPDATE"
	ActionDelete   ActionType = "DELETE"
	ActionRecreate ActionType = "RECREATE"
	ActionNoChange ActionType = "NO CHANGE"
	// ActionInspect indicates the item is being viewed in inspect mode.
	// The icon will be rendered based on the item's actual status rather than its action.
	ActionInspect ActionType = ""
)

// DetermineResourceAction determines the appropriate action based on item state and changes.
func DetermineResourceAction(isNew, isRemoved, mustRecreate bool, changes *provider.Changes) ActionType {
	if isNew {
		return ActionCreate
	}
	if isRemoved {
		return ActionDelete
	}
	if mustRecreate {
		return ActionRecreate
	}
	if changes != nil && provider.HasAnyChanges(changes) {
		return ActionUpdate
	}
	return ActionNoChange
}

// ActionIcon returns the icon character for an action type.
func ActionIcon(action ActionType) string {
	switch action {
	case ActionCreate:
		return "✓"
	case ActionUpdate:
		return "±"
	case ActionDelete:
		return "-"
	case ActionRecreate:
		return "↻"
	default:
		return "○"
	}
}

// StyledActionIcon returns a styled icon for an action type.
func StyledActionIcon(action ActionType, s *styles.Styles, applyStyle bool) string {
	icon := ActionIcon(action)
	if !applyStyle {
		return icon
	}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch action {
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

// RenderActionBadge renders an action badge with appropriate styling.
func RenderActionBadge(action ActionType, s *styles.Styles) string {
	successBadge := lipgloss.NewStyle().
		Foreground(s.Palette.Success()).
		Bold(true)

	switch action {
	case ActionCreate:
		return successBadge.Render(string(action))
	case ActionUpdate:
		return s.Warning.Bold(true).Render(string(action))
	case ActionDelete:
		return s.Error.Bold(true).Render(string(action))
	case ActionRecreate:
		return s.Info.Bold(true).Render(string(action))
	case ActionNoChange:
		return s.Muted.Render(string(action))
	default:
		return string(action)
	}
}
