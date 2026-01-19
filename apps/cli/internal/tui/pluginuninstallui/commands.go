package pluginuninstallui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
)

// PluginCompleteMsg is sent when a plugin uninstallation completes.
type PluginCompleteMsg struct {
	Index  int
	Result *plugins.UninstallResult
}

// PluginErrorMsg is sent when a plugin uninstallation encounters an error.
type PluginErrorMsg struct {
	Index int
	Error error
}

// uninstallPluginCmd creates a command that uninstalls a plugin.
func uninstallPluginCmd(manager *plugins.Manager, index int, pluginID *plugins.PluginID) tea.Cmd {
	return func() tea.Msg {
		result := manager.Uninstall(pluginID)
		return PluginCompleteMsg{
			Index:  index,
			Result: result,
		}
	}
}
