package stateimportui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/stateio"
	"github.com/spf13/afero"
)

// DownloadStartedMsg indicates that downloading has started.
type DownloadStartedMsg struct{}

// DownloadCompleteMsg indicates that downloading has completed.
type DownloadCompleteMsg struct {
	Data []byte
	Err  error
}

// ImportStartedMsg indicates that importing has started.
type ImportStartedMsg struct{}

// ImportCompleteMsg indicates that importing has completed.
type ImportCompleteMsg struct {
	Result *stateio.ImportResult
	Err    error
}

func startDownloadCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		data, err := stateio.DownloadRemoteFile(context.Background(), filePath, nil)
		return DownloadCompleteMsg{Data: data, Err: err}
	}
}

func startImportCmd(
	engineConfig *stateio.EngineConfig,
	filePath string,
) tea.Cmd {
	return func() tea.Msg {
		result, err := stateio.Import(stateio.ImportParams{
			FilePath:     filePath,
			EngineConfig: engineConfig,
			FileSystem:   afero.NewOsFs(),
		})
		return ImportCompleteMsg{Result: result, Err: err}
	}
}

func startImportWithDataCmd(
	engineConfig *stateio.EngineConfig,
	data []byte,
) tea.Cmd {
	return func() tea.Msg {
		result, err := stateio.Import(stateio.ImportParams{
			EngineConfig: engineConfig,
			FileSystem:   afero.NewOsFs(),
			FileData:     data,
		})
		return ImportCompleteMsg{Result: result, Err: err}
	}
}
