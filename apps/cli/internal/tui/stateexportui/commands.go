package stateexportui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/stateio"
	"github.com/spf13/afero"
)

// ExportStartedMsg indicates that exporting has started.
type ExportStartedMsg struct{}

// ExportCompleteMsg indicates that exporting has completed.
type ExportCompleteMsg struct {
	Result *stateio.ExportResult
	Err    error
}

func startExportCmd(
	engineConfig *stateio.EngineConfig,
	filePath string,
	instanceFilters []string,
) tea.Cmd {
	return func() tea.Msg {
		result, err := stateio.Export(stateio.ExportParams{
			FilePath:        filePath,
			InstanceFilters: instanceFilters,
			EngineConfig:    engineConfig,
			FileSystem:      afero.NewOsFs(),
		})
		return ExportCompleteMsg{Result: result, Err: err}
	}
}
