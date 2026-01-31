package pluginlistui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
)

// PluginsLoadedMsg is sent when the plugin list has been loaded from the manifest.
type PluginsLoadedMsg struct {
	Plugins []*plugins.InstalledPlugin
}

// PluginsLoadErrorMsg is sent when loading the plugin list fails.
type PluginsLoadErrorMsg struct {
	Err error
}

func loadPluginsCmd(typeFilter string) tea.Cmd {
	return func() tea.Msg {
		manager := plugins.NewManager(nil, nil)
		allPlugins, err := manager.ListInstalled()
		if err != nil {
			return PluginsLoadErrorMsg{Err: err}
		}

		filtered := filterByType(allPlugins, typeFilter)
		return PluginsLoadedMsg{Plugins: filtered}
	}
}

func filterByType(
	allPlugins []*plugins.InstalledPlugin,
	typeFilter string,
) []*plugins.InstalledPlugin {
	if typeFilter == "all" {
		return allPlugins
	}

	filtered := make([]*plugins.InstalledPlugin, 0, len(allPlugins))
	for _, p := range allPlugins {
		if p.Type == typeFilter {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func filterBySearch(
	pluginList []*plugins.InstalledPlugin,
	search string,
) []*plugins.InstalledPlugin {
	if search == "" {
		return pluginList
	}

	lowerSearch := strings.ToLower(search)
	filtered := make([]*plugins.InstalledPlugin, 0, len(pluginList))
	for _, p := range pluginList {
		if strings.Contains(strings.ToLower(p.ID), lowerSearch) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
