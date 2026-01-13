package plugininstallui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
)

// PluginProgressMsg reports progress during plugin installation.
type PluginProgressMsg struct {
	Index      int
	Status     PluginStatus
	StatusText string
	Downloaded int64
	Total      int64
}

// PluginCompleteMsg signals completion of a plugin installation.
type PluginCompleteMsg struct {
	Index  int
	Result *plugins.InstallResult
}

// PluginErrorMsg signals an error during plugin installation.
type PluginErrorMsg struct {
	Index int
	Error error
}

func installPluginCmd(
	ctx context.Context,
	manager *plugins.Manager,
	index int,
	pluginID *plugins.PluginID,
) tea.Cmd {
	return func() tea.Msg {
		// Create a progress channel to send updates
		progressChan := make(chan PluginProgressMsg, 10)
		resultChan := make(chan *plugins.InstallResult, 1)
		errChan := make(chan error, 1)

		// Run installation in background
		go func() {
			result, err := manager.Install(ctx, pluginID, func(
				pid *plugins.PluginID,
				stage plugins.InstallStage,
				downloaded, total int64,
			) {
				var status PluginStatus
				var statusText string

				switch stage {
				case plugins.StageResolving:
					status = PluginResolving
					statusText = "Resolving..."
				case plugins.StageDownloading:
					status = PluginDownloading
					statusText = "Downloading..."
				case plugins.StageVerifying:
					status = PluginVerifying
					statusText = "Verifying..."
				case plugins.StageExtracting:
					status = PluginExtracting
					statusText = "Extracting..."
				case plugins.StageComplete:
					status = PluginComplete
					statusText = "Complete"
				}

				select {
				case progressChan <- PluginProgressMsg{
					Index:      index,
					Status:     status,
					StatusText: statusText,
					Downloaded: downloaded,
					Total:      total,
				}:
				default:
					// Channel full, skip this update
				}
			})

			if err != nil {
				errChan <- err
				return
			}
			resultChan <- result
		}()

		// Wait for result
		select {
		case result := <-resultChan:
			return PluginCompleteMsg{Index: index, Result: result}
		case err := <-errChan:
			return PluginErrorMsg{Index: index, Error: err}
		case <-ctx.Done():
			return PluginErrorMsg{Index: index, Error: ctx.Err()}
		}
	}
}

// InstallAllPluginsCmd starts installation of all plugins.
type InstallAllPluginsCmd struct {
	Results []*plugins.InstallResult
}
