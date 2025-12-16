package stageui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// StageDetailsRenderer implements splitpane.DetailsRenderer for stage UI.
type StageDetailsRenderer struct {
	// MaxExpandDepth is used to determine when to show the drill-down hint
	MaxExpandDepth int
	// NavigationStackDepth is the current depth of navigation stack
	NavigationStackDepth int
}

// Ensure StageDetailsRenderer implements splitpane.DetailsRenderer
var _ splitpane.DetailsRenderer = (*StageDetailsRenderer)(nil)

// RenderDetails renders the right pane content for a selected item.
func (r *StageDetailsRenderer) RenderDetails(item splitpane.Item, width int, s *styles.Styles) string {
	stageItem, ok := item.(*StageItem)
	if !ok {
		return s.Muted.Render("Unknown item type")
	}

	switch stageItem.Type {
	case ItemTypeResource:
		return r.renderResourceDetails(stageItem, width, s)
	case ItemTypeChild:
		return r.renderChildDetails(stageItem, width, s)
	case ItemTypeLink:
		return r.renderLinkDetails(stageItem, width, s)
	default:
		return s.Muted.Render("Unknown item type")
	}
}

func (r *StageDetailsRenderer) renderResourceDetails(item *StageItem, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	// Header - use display name if available, otherwise resource name
	headerText := item.Name
	if item.DisplayName != "" {
		headerText = item.DisplayName
	}
	sb.WriteString(s.Header.Render(headerText))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Resource name (if display name was shown as header)
	if item.DisplayName != "" {
		sb.WriteString(s.Muted.Render("Name: "))
		sb.WriteString(item.Name)
		sb.WriteString("\n")
	}

	// Resource type
	if item.ResourceType != "" {
		sb.WriteString(s.Muted.Render("Type: "))
		sb.WriteString(item.ResourceType)
		sb.WriteString("\n")
	}

	// Action
	sb.WriteString(s.Muted.Render("Action: "))
	sb.WriteString(renderActionBadge(item.Action, s))
	sb.WriteString("\n\n")

	// Changes
	if resourceChanges, ok := item.Changes.(*provider.Changes); ok && resourceChanges != nil {
		sb.WriteString(r.renderResourceChanges(resourceChanges, s))
	}

	return sb.String()
}

func (r *StageDetailsRenderer) renderResourceChanges(resourceChanges *provider.Changes, s *styles.Styles) string {
	sb := strings.Builder{}

	hasChanges := len(resourceChanges.NewFields) > 0 || len(resourceChanges.ModifiedFields) > 0 || len(resourceChanges.RemovedFields) > 0

	if !hasChanges {
		sb.WriteString(s.Muted.Render("No field changes"))
		return sb.String()
	}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	sb.WriteString(s.Category.Render("Changes:"))
	sb.WriteString("\n")

	// New fields (additions)
	for _, field := range resourceChanges.NewFields {
		line := fmt.Sprintf("  + %s: %s", field.FieldPath, headless.FormatMappingNode(field.NewValue))
		sb.WriteString(successStyle.Render(line))
		sb.WriteString("\n")
	}

	// Modified fields
	for _, field := range resourceChanges.ModifiedFields {
		prevValue := headless.FormatMappingNode(field.PrevValue)
		newValue := headless.FormatMappingNode(field.NewValue)
		line := fmt.Sprintf("  ~ %s: %s -> %s", field.FieldPath, prevValue, newValue)
		sb.WriteString(s.Warning.Render(line))
		sb.WriteString("\n")
	}

	// Removed fields
	for _, fieldPath := range resourceChanges.RemovedFields {
		line := fmt.Sprintf("  - %s", fieldPath)
		sb.WriteString(s.Error.Render(line))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *StageDetailsRenderer) renderChildDetails(item *StageItem, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(item.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Status and action
	sb.WriteString(s.Muted.Render("Status: "))
	sb.WriteString("Changes computed")
	sb.WriteString("\n")

	sb.WriteString(s.Muted.Render("Action: "))
	sb.WriteString(renderActionBadge(item.Action, s))
	sb.WriteString("\n\n")

	// Show inspect hint for children at max expand depth
	effectiveDepth := item.Depth + r.NavigationStackDepth
	if effectiveDepth >= r.MaxExpandDepth && item.Changes != nil {
		sb.WriteString(s.Hint.Render("Press enter to inspect this child blueprint"))
		sb.WriteString("\n\n")
	}

	// Child changes summary
	if childChanges, ok := item.Changes.(*changes.BlueprintChanges); ok && childChanges != nil {
		sb.WriteString(r.renderChildChangesSummary(childChanges, s))
	}

	return sb.String()
}

func (r *StageDetailsRenderer) renderChildChangesSummary(childChanges *changes.BlueprintChanges, s *styles.Styles) string {
	sb := strings.Builder{}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	newCount := len(childChanges.NewResources)
	updateCount := len(childChanges.ResourceChanges)
	removeCount := len(childChanges.RemovedResources)

	sb.WriteString(s.Category.Render("Summary:"))
	sb.WriteString("\n")

	if newCount > 0 {
		sb.WriteString(successStyle.Render(fmt.Sprintf("  %d %s to be created", newCount, sdkstrings.Pluralize(newCount, "resource", "resources"))))
		sb.WriteString("\n")
	}
	if updateCount > 0 {
		sb.WriteString(s.Warning.Render(fmt.Sprintf("  %d %s to be updated", updateCount, sdkstrings.Pluralize(updateCount, "resource", "resources"))))
		sb.WriteString("\n")
	}
	if removeCount > 0 {
		sb.WriteString(s.Error.Render(fmt.Sprintf("  %d %s to be removed", removeCount, sdkstrings.Pluralize(removeCount, "resource", "resources"))))
		sb.WriteString("\n")
	}

	// Child blueprints
	newChildren := len(childChanges.NewChildren)
	childUpdates := len(childChanges.ChildChanges)
	removedChildren := len(childChanges.RemovedChildren)

	if newChildren > 0 || childUpdates > 0 || removedChildren > 0 {
		sb.WriteString("\n")
		if newChildren > 0 {
			sb.WriteString(successStyle.Render(fmt.Sprintf("  %d child %s to be created", newChildren, sdkstrings.Pluralize(newChildren, "blueprint", "blueprints"))))
			sb.WriteString("\n")
		}
		if childUpdates > 0 {
			sb.WriteString(s.Warning.Render(fmt.Sprintf("  %d child %s to be updated", childUpdates, sdkstrings.Pluralize(childUpdates, "blueprint", "blueprints"))))
			sb.WriteString("\n")
		}
		if removedChildren > 0 {
			sb.WriteString(s.Error.Render(fmt.Sprintf("  %d child %s to be removed", removedChildren, sdkstrings.Pluralize(removedChildren, "blueprint", "blueprints"))))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (r *StageDetailsRenderer) renderLinkDetails(item *StageItem, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(item.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Status and action
	sb.WriteString(s.Muted.Render("Status: "))
	sb.WriteString("Changes computed")
	sb.WriteString("\n")

	sb.WriteString(s.Muted.Render("Action: "))
	sb.WriteString(renderActionBadge(item.Action, s))
	sb.WriteString("\n\n")

	// Link changes
	if linkChanges, ok := item.Changes.(*provider.LinkChanges); ok && linkChanges != nil {
		sb.WriteString(r.renderLinkChanges(linkChanges, s))
	}

	return sb.String()
}

func (r *StageDetailsRenderer) renderLinkChanges(linkChanges *provider.LinkChanges, s *styles.Styles) string {
	sb := strings.Builder{}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	hasChanges := len(linkChanges.NewFields) > 0 || len(linkChanges.ModifiedFields) > 0 || len(linkChanges.RemovedFields) > 0

	if !hasChanges {
		sb.WriteString(s.Muted.Render("No field changes"))
		return sb.String()
	}

	sb.WriteString(s.Category.Render("Changes:"))
	sb.WriteString("\n")

	// New fields (additions)
	for _, field := range linkChanges.NewFields {
		line := fmt.Sprintf("  + %s: %s", field.FieldPath, headless.FormatMappingNode(field.NewValue))
		sb.WriteString(successStyle.Render(line))
		sb.WriteString("\n")
	}

	// Modified fields
	for _, field := range linkChanges.ModifiedFields {
		prevValue := headless.FormatMappingNode(field.PrevValue)
		newValue := headless.FormatMappingNode(field.NewValue)
		line := fmt.Sprintf("  ~ %s: %s -> %s", field.FieldPath, prevValue, newValue)
		sb.WriteString(s.Warning.Render(line))
		sb.WriteString("\n")
	}

	// Removed fields
	for _, fieldPath := range linkChanges.RemovedFields {
		line := fmt.Sprintf("  - %s", fieldPath)
		sb.WriteString(s.Error.Render(line))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderActionBadge renders an action badge with appropriate styling.
func renderActionBadge(action ActionType, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())
	switch action {
	case ActionCreate:
		return successStyle.Render(string(action))
	case ActionUpdate:
		return s.Warning.Render(string(action))
	case ActionDelete:
		return s.Error.Render(string(action))
	case ActionRecreate:
		return s.Info.Render(string(action))
	default:
		return s.Muted.Render(string(action))
	}
}

// StageSectionGrouper implements splitpane.SectionGrouper for stage UI.
type StageSectionGrouper struct {
	// MaxExpandDepth is the maximum depth for inline expansion.
	// Children at or beyond this depth will not be expanded inline.
	MaxExpandDepth int
}

// Ensure StageSectionGrouper implements splitpane.SectionGrouper
var _ splitpane.SectionGrouper = (*StageSectionGrouper)(nil)

// GroupItems organizes items into sections: Resources, Child Blueprints, Links.
// The isExpanded function is provided by the splitpane to query expansion state.
func (g *StageSectionGrouper) GroupItems(items []splitpane.Item, isExpanded func(id string) bool) []splitpane.Section {
	var resources []splitpane.Item
	var children []splitpane.Item
	var links []splitpane.Item

	for _, item := range items {
		stageItem, ok := item.(*StageItem)
		if !ok {
			continue
		}

		// Nested items (with ParentChild set) go to children section
		if stageItem.ParentChild != "" {
			children = append(children, item)
			continue
		}

		switch stageItem.Type {
		case ItemTypeResource:
			resources = append(resources, item)
		case ItemTypeChild:
			children = append(children, item)
			// If expanded, recursively add children inline (respecting max depth)
			children = g.appendExpandedChildren(children, item, isExpanded)
		case ItemTypeLink:
			links = append(links, item)
		}
	}

	// Sort each section for consistent ordering
	sortStageItems(resources)
	sortStageItems(links)
	// Don't sort children slice as it contains both parents and their expanded children
	// which need to maintain their relative positions

	var sections []splitpane.Section

	if len(resources) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Resources",
			Items: resources,
		})
	}

	if len(children) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Child Blueprints",
			Items: children,
		})
	}

	if len(links) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Links",
			Items: links,
		})
	}

	return sections
}

// appendExpandedChildren recursively appends children of an expanded item.
// It handles nested expansions (grandchildren, etc.) by checking each child's expansion state.
// Expansion stops when the item's depth reaches MaxExpandDepth.
func (g *StageSectionGrouper) appendExpandedChildren(children []splitpane.Item, item splitpane.Item, isExpanded func(id string) bool) []splitpane.Item {
	if isExpanded == nil || !isExpanded(item.GetID()) {
		return children
	}

	// Check if item is at or beyond max expand depth
	if item.GetDepth() >= g.MaxExpandDepth {
		return children
	}

	childItems := item.GetChildren()
	// Sort child items for consistent ordering
	sortStageItems(childItems)

	for _, child := range childItems {
		children = append(children, child)
		// Recursively check if this child is also expanded (depth check happens in recursive call)
		if child.IsExpandable() {
			children = g.appendExpandedChildren(children, child, isExpanded)
		}
	}

	return children
}

// sortStageItems sorts items alphabetically by name for consistent display order.
func sortStageItems(items []splitpane.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
}

// StageFooterRenderer implements splitpane.FooterRenderer for stage UI.
// It supports a delegate pattern to allow custom footer rendering (e.g., for deploy flow).
type StageFooterRenderer struct {
	ChangesetID  string
	InstanceID   string
	InstanceName string
	// Delegate is an optional custom footer renderer that takes precedence when set.
	// This allows the deploy flow to inject its own footer (e.g., confirmation form).
	Delegate splitpane.FooterRenderer
}

// Ensure StageFooterRenderer implements splitpane.FooterRenderer
var _ splitpane.FooterRenderer = (*StageFooterRenderer)(nil)

// RenderFooter renders the stage-specific footer with changeset and deploy instructions.
// If a Delegate is set, it defers to the delegate for rendering.
func (r *StageFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	// If a delegate is set, use it for rendering
	if r.Delegate != nil {
		return r.Delegate.RenderFooter(model, s)
	}
	sb := strings.Builder{}
	sb.WriteString("\n")

	// Show different footer when viewing a child blueprint
	if model.IsInDrillDown() {
		// Show breadcrumb path
		sb.WriteString(s.Muted.Render("  Viewing: "))
		for i, name := range model.NavigationPath() {
			if i > 0 {
				sb.WriteString(s.Muted.Render(" > "))
			}
			sb.WriteString(s.Selected.Render(name))
		}
		sb.WriteString("\n\n")

		// Navigation help for child view
		sb.WriteString(s.Muted.Render("  "))
		sb.WriteString(s.Key.Render("esc"))
		sb.WriteString(s.Muted.Render(" back  "))
		sb.WriteString(s.Key.Render("↑/↓"))
		sb.WriteString(s.Muted.Render(" navigate  "))
		sb.WriteString(s.Key.Render("enter"))
		sb.WriteString(s.Muted.Render(" expand/inspect  "))
		sb.WriteString(s.Key.Render("tab"))
		sb.WriteString(s.Muted.Render(" switch pane  "))
		sb.WriteString(s.Key.Render("q"))
		sb.WriteString(s.Muted.Render(" quit"))
		sb.WriteString("\n")

		return sb.String()
	}

	sb.WriteString(s.Muted.Render("  Staging complete. Changeset ID: "))
	sb.WriteString(s.Selected.Render(r.ChangesetID))
	sb.WriteString("\n\n")

	// Deploy instructions
	sb.WriteString(s.Muted.Render("  To apply these changes, run:\n"))
	deployCmd := fmt.Sprintf("    bluelink deploy --changeset-id %s", r.ChangesetID)
	if r.InstanceName != "" {
		deployCmd += fmt.Sprintf(" --instance-name %s", r.InstanceName)
	} else if r.InstanceID != "" {
		deployCmd += fmt.Sprintf(" --instance-id %s", r.InstanceID)
	} else {
		deployCmd += " --instance-name <name>"
	}
	sb.WriteString(s.Command.Render(deployCmd))
	sb.WriteString("\n\n")

	// Navigation help
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" navigate  "))
	sb.WriteString(s.Key.Render("enter"))
	sb.WriteString(s.Muted.Render(" expand/collapse  "))
	sb.WriteString(s.Key.Render("tab"))
	sb.WriteString(s.Muted.Render(" switch pane  "))
	sb.WriteString(s.Key.Render("q"))
	sb.WriteString(s.Muted.Render(" quit"))
	sb.WriteString("\n")

	return sb.String()
}
