package pluginlistui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

func (m MainModel) renderSearchView() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(m.styles.Title.MarginLeft(2).Render("Search Plugins"))
	sb.WriteString("\n\n")

	sb.WriteString("  ")
	sb.WriteString(m.styles.Key.Render("/"))
	sb.WriteString(" ")
	sb.WriteString(m.searchInput.View())
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.MarginLeft(2).Render("enter to search • esc to cancel"))
	sb.WriteString("\n")

	return sb.String()
}

func (m MainModel) renderList() string {
	var sb strings.Builder

	sb.WriteString("\n")
	title := m.buildTitle()
	sb.WriteString(m.styles.Title.MarginLeft(2).Render(title))
	sb.WriteString("\n\n")

	if len(m.filteredPlugins) == 0 {
		sb.WriteString(m.styles.Muted.MarginLeft(4).Render("No plugins found."))
		sb.WriteString("\n")
	} else {
		providers := filterGroupByType(m.filteredPlugins, "provider")
		transformers := filterGroupByType(m.filteredPlugins, "transformer")
		unknown := filterGroupByType(m.filteredPlugins, "")

		offset := 0
		offset = renderPluginGroup(&sb, "Providers", providers, m.filteredPlugins, m.styles, m.cursor, offset)
		offset = renderPluginGroup(&sb, "Transformers", transformers, m.filteredPlugins, m.styles, m.cursor, offset)
		renderPluginGroup(&sb, "Other", unknown, m.filteredPlugins, m.styles, m.cursor, offset)
	}

	sb.WriteString("\n")
	sb.WriteString(m.renderFooter())
	sb.WriteString("\n")

	return sb.String()
}

func (m MainModel) buildTitle() string {
	title := "Installed Plugins"
	var filters []string
	if m.typeFilter != "all" {
		filters = append(filters, fmt.Sprintf("type: %s", m.typeFilter))
	}
	if m.searchTerm != "" {
		filters = append(filters, fmt.Sprintf("search: %q", m.searchTerm))
	}
	if len(filters) > 0 {
		title += " (" + strings.Join(filters, ", ") + ")"
	}
	return title
}

func renderPluginGroup(
	sb *strings.Builder,
	groupTitle string,
	groupPlugins []*plugins.InstalledPlugin,
	allFiltered []*plugins.InstalledPlugin,
	styles *stylespkg.Styles,
	cursor int,
	offset int,
) int {
	if len(groupPlugins) == 0 {
		return offset
	}

	sb.WriteString("  ")
	sb.WriteString(styles.Selected.Render(groupTitle))
	sb.WriteString("\n\n")

	for _, p := range groupPlugins {
		isSelected := offset == cursor
		renderPluginEntry(sb, p, allFiltered, styles, isSelected)
		sb.WriteString("\n")
		offset++
	}

	return offset
}

func renderPluginEntry(
	sb *strings.Builder,
	p *plugins.InstalledPlugin,
	allPlugins []*plugins.InstalledPlugin,
	styles *stylespkg.Styles,
	isSelected bool,
) {
	cursorStr := "  "
	if isSelected {
		cursorStr = styles.Selected.Render("> ")
	}

	name := p.ID
	if isSelected {
		name = styles.Selected.Render(name)
	}

	typeLabel := renderTypeLabel(p.Type, styles)

	sb.WriteString("  ")
	sb.WriteString(cursorStr)
	sb.WriteString(name)
	if typeLabel != "" {
		sb.WriteString("  ")
		sb.WriteString(typeLabel)
	}
	sb.WriteString("\n")

	// Version and install date
	installedStr := formatRelativeTime(p.InstalledAt)
	sb.WriteString("      ")
	sb.WriteString(styles.Muted.Render(
		fmt.Sprintf("Version: %s | Installed: %s", p.Version, installedStr),
	))
	sb.WriteString("\n")

	// Dependencies
	renderDependencyTree(sb, p, allPlugins, styles)
}

func renderTypeLabel(pluginType string, styles *stylespkg.Styles) string {
	switch pluginType {
	case "provider":
		return styles.Muted.Render("[provider]")
	case "transformer":
		return styles.Muted.Render("[transformer]")
	default:
		return ""
	}
}

func renderDependencyTree(
	sb *strings.Builder,
	p *plugins.InstalledPlugin,
	allPlugins []*plugins.InstalledPlugin,
	styles *stylespkg.Styles,
) {
	if len(p.Dependencies) == 0 {
		return
	}

	sb.WriteString("      ")
	sb.WriteString(styles.Muted.Render("Dependencies:"))
	sb.WriteString("\n")

	depIDs := make([]string, 0, len(p.Dependencies))
	for depID := range p.Dependencies {
		depIDs = append(depIDs, depID)
	}
	sort.Strings(depIDs)

	for i, depID := range depIDs {
		depVersion := p.Dependencies[depID]
		isLast := i == len(depIDs)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		depDisplay := depID
		if depVersion != "" {
			depDisplay += "@" + depVersion
		}

		statusLabel := styles.Warning.Render("(not installed)")
		if isPluginInstalled(allPlugins, depID) {
			statusLabel = styles.Success.Render("(installed)")
		}

		sb.WriteString("      ")
		sb.WriteString(styles.Muted.Render(connector))
		sb.WriteString(depDisplay)
		sb.WriteString(" ")
		sb.WriteString(statusLabel)
		sb.WriteString("\n")
	}
}

func (m MainModel) renderFooter() string {
	var sb strings.Builder

	sb.WriteString("  ")
	sb.WriteString(m.styles.Muted.Render(
		fmt.Sprintf("Total: %d plugin(s)", len(m.filteredPlugins)),
	))
	sb.WriteString("\n")

	sb.WriteString("  ")
	sb.WriteString(m.styles.Key.Render("↑/↓"))
	sb.WriteString(m.styles.Muted.Render(" navigate  "))
	sb.WriteString(m.styles.Key.Render("/"))
	sb.WriteString(m.styles.Muted.Render(" search  "))
	if m.searchTerm != "" {
		sb.WriteString(m.styles.Key.Render("esc"))
		sb.WriteString(m.styles.Muted.Render(" clear  "))
	}
	sb.WriteString(m.styles.Key.Render("q"))
	sb.WriteString(m.styles.Muted.Render(" quit"))

	return sb.String()
}

func filterGroupByType(
	pluginList []*plugins.InstalledPlugin,
	pluginType string,
) []*plugins.InstalledPlugin {
	filtered := make([]*plugins.InstalledPlugin, 0)
	for _, p := range pluginList {
		if p.Type == pluginType {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func isPluginInstalled(allPlugins []*plugins.InstalledPlugin, depID string) bool {
	for _, p := range allPlugins {
		// Strip version suffix to get the plugin path (e.g. "host/ns/name").
		pluginPath := p.ID
		if idx := strings.LastIndex(pluginPath, "@"); idx >= 0 {
			pluginPath = pluginPath[:idx]
		}

		// Exact match (default registry: "bluelink/test-provider")
		if pluginPath == depID {
			return true
		}

		// Suffix match for non-default registries where the installed ID
		// includes the host (e.g. "localhost:8080/bluelink/test-provider").
		if strings.HasSuffix(pluginPath, "/"+depID) {
			return true
		}
	}
	return false
}

func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "Just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 48*time.Hour:
		return "Yesterday"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}
