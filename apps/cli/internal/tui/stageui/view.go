package stageui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
)

// MaxExpandDepth is the maximum nesting depth for expanding child blueprints.
// Children at this depth will be shown but cannot be expanded further in the left pane.
// Their details are still viewable in the right pane when selected.
const MaxExpandDepth = 2

func stageOverviewFooterHeight() int {
	return 3
}

// OverviewItem holds a stage item with its full element path for overview display.
type OverviewItem struct {
	Item        StageItem
	ElementPath string
}

func buildElementPath(parentPath, elementType, elementName string) string {
	segment := elementType + "." + elementName
	if parentPath == "" {
		return segment
	}
	return parentPath + "::" + segment
}

func (m StageModel) handleOverviewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "o", "O":
		m.showingOverview = false
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.overviewViewport, cmd = m.overviewViewport.Update(msg)
		return m, cmd
	}
}

func (m StageModel) renderOverviewView() string {
	sb := strings.Builder{}
	sb.WriteString(m.overviewViewport.View())
	sb.WriteString("\n")
	shared.RenderViewportOverlayFooter(&sb, "o", m.styles)
	return sb.String()
}

func (m StageModel) renderOverviewContent() string {
	sb := strings.Builder{}

	sb.WriteString("\n")
	sb.WriteString(m.styles.Header.Render("  Change Staging Summary"))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  " + strings.Repeat("─", 60)))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Changeset ID: "))
	sb.WriteString(m.styles.Selected.Render(m.changesetID))
	sb.WriteString("\n")
	if m.instanceID != "" {
		sb.WriteString(m.styles.Muted.Render("  Instance ID: "))
		sb.WriteString(m.styles.Selected.Render(m.instanceID))
		sb.WriteString("\n")
	}
	if m.instanceName != "" {
		sb.WriteString(m.styles.Muted.Render("  Instance Name: "))
		sb.WriteString(m.styles.Selected.Render(m.instanceName))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	creates, updates, recreates, deletes, noChanges := m.categorizeItems()

	m.renderCategoryItems(&sb, "To Be Created", creates, ActionCreate)
	m.renderCategoryItems(&sb, "To Be Updated", updates, ActionUpdate)
	m.renderCategoryItems(&sb, "To Be Recreated", recreates, ActionRecreate)
	m.renderCategoryItems(&sb, "To Be Removed", deletes, ActionDelete)
	m.renderCategoryItems(&sb, "With No Changes", noChanges, ActionNoChange)

	return sb.String()
}

func (m *StageModel) categorizeItems() (creates, updates, recreates, deletes, noChanges []OverviewItem) {
	allItems := m.collectAllItemsWithPaths()

	for _, item := range allItems {
		switch item.Item.Action {
		case ActionCreate:
			creates = append(creates, item)
		case ActionUpdate:
			updates = append(updates, item)
		case ActionRecreate:
			recreates = append(recreates, item)
		case ActionDelete:
			deletes = append(deletes, item)
		case ActionNoChange:
			noChanges = append(noChanges, item)
		}
	}
	return
}

func (m *StageModel) collectAllItemsWithPaths() []OverviewItem {
	var items []OverviewItem

	for _, item := range m.items {
		parentPath := convertParentChildToElementPath(item.ParentChild)
		path := buildItemPath(parentPath, &item)
		items = append(items, OverviewItem{
			Item:        item,
			ElementPath: path,
		})

		if item.Type == ItemTypeChild && item.Changes != nil {
			items = m.collectChildItemsWithPaths(items, &item, path)
		}
	}

	return items
}

func convertParentChildToElementPath(parentChild string) string {
	if parentChild == "" {
		return ""
	}

	parts := strings.Split(parentChild, "::")
	var pathParts []string
	for _, part := range parts {
		pathParts = append(pathParts, "children."+part)
	}
	return strings.Join(pathParts, "::")
}

func (m *StageModel) collectChildItemsWithPaths(
	items []OverviewItem,
	parent *StageItem,
	parentPath string,
) []OverviewItem {
	children := parent.GetChildren()
	for _, child := range children {
		stageItem, ok := child.(*StageItem)
		if !ok {
			continue
		}

		path := buildItemPath(parentPath, stageItem)
		items = append(items, OverviewItem{
			Item:        *stageItem,
			ElementPath: path,
		})

		if stageItem.Type == ItemTypeChild {
			items = m.collectChildItemsWithPaths(items, stageItem, path)
		}
	}

	return items
}

func buildItemPath(parentPath string, item *StageItem) string {
	switch item.Type {
	case ItemTypeResource:
		return buildElementPath(parentPath, "resources", item.Name)
	case ItemTypeChild:
		return buildElementPath(parentPath, "children", item.Name)
	case ItemTypeLink:
		return buildElementPath(parentPath, "links", item.Name)
	default:
		return buildElementPath(parentPath, "unknown", item.Name)
	}
}

func (m *StageModel) renderCategoryItems(
	sb *strings.Builder,
	title string,
	items []OverviewItem,
	action ActionType,
) {
	if len(items) == 0 {
		return
	}

	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())

	var titleStyle, iconStyle lipgloss.Style
	var icon string
	switch action {
	case ActionCreate:
		titleStyle = successStyle
		iconStyle = successStyle
		icon = "✓"
	case ActionUpdate:
		titleStyle = m.styles.Warning
		iconStyle = m.styles.Warning
		icon = "±"
	case ActionRecreate:
		titleStyle = m.styles.Info
		iconStyle = m.styles.Info
		icon = "↻"
	case ActionDelete:
		titleStyle = m.styles.Error
		iconStyle = m.styles.Error
		icon = "-"
	default:
		titleStyle = m.styles.Muted
		iconStyle = m.styles.Muted
		icon = "○"
	}

	elementLabel := sdkstrings.Pluralize(len(items), "Element", "Elements")
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %d %s %s:", len(items), elementLabel, title)))
	sb.WriteString("\n\n")

	for _, overviewItem := range items {
		item := overviewItem.Item
		sb.WriteString("  ")
		sb.WriteString(iconStyle.Render(icon + " "))
		sb.WriteString(m.styles.Selected.Render(overviewItem.ElementPath))
		if item.ResourceType != "" {
			sb.WriteString(m.styles.Muted.Render(" (" + item.ResourceType + ")"))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}
