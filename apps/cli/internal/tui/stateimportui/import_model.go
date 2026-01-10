package stateimportui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// ImportModelConfig holds configuration for the import model.
type ImportModelConfig struct {
	EngineConfig   *stateio.EngineConfig
	FilePath       string
	Styles         *stylespkg.Styles
	Headless       bool
	HeadlessWriter io.Writer
	JSONMode       bool
}

// ImportModel handles the import progress display.
type ImportModel struct {
	spinner        spinner.Model
	engineConfig   *stateio.EngineConfig
	filePath       string
	downloading    bool
	importing      bool
	result         *stateio.ImportResult
	err            error
	finished       bool
	headless       bool
	headlessWriter io.Writer
	jsonMode       bool
	styles         *stylespkg.Styles
	width          int
}

// NewImportModel creates a new import model.
func NewImportModel(config ImportModelConfig) *ImportModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = config.Styles.Spinner

	return &ImportModel{
		spinner:        s,
		engineConfig:   config.EngineConfig,
		filePath:       config.FilePath,
		headless:       config.Headless,
		headlessWriter: config.HeadlessWriter,
		jsonMode:       config.JSONMode,
		styles:         config.Styles,
		width:          80, // Default width, will be updated on first WindowSizeMsg
	}
}

func (m *ImportModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *ImportModel) Update(msg tea.Msg) (*ImportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case DownloadStartedMsg:
		m.downloading = true
		return m, nil
	case DownloadCompleteMsg:
		m.downloading = false
		if msg.Err != nil {
			m.err = msg.Err
			m.finished = true
			return m, nil
		}
		m.importing = true
		return m, startImportWithDataCmd(m.engineConfig, msg.Data)
	case ImportStartedMsg:
		m.importing = true
		return m, nil
	case ImportCompleteMsg:
		m.importing = false
		m.finished = true
		m.result = msg.Result
		m.err = msg.Err
		if m.headless {
			m.writeHeadlessOutput()
		}
		return m, nil
	}

	return m, nil
}

func (m *ImportModel) View() string {
	if m.headless {
		return ""
	}

	if m.downloading {
		return fmt.Sprintf("\n  %s Downloading from %s...\n", m.spinner.View(), m.filePath)
	}

	if m.importing {
		return fmt.Sprintf("\n  %s Importing state...\n", m.spinner.View())
	}

	if m.finished {
		return m.renderResult()
	}

	return ""
}

func (m *ImportModel) renderResult() string {
	if m.err != nil {
		// Calculate maximum width for error message wrapping
		maxWidth := m.width - 6 // Account for indent and margins
		if maxWidth < 40 {
			maxWidth = 40
		}

		// Use lipgloss to wrap the error message
		wrapStyle := lipgloss.NewStyle().Width(maxWidth)
		wrappedError := wrapStyle.Render(m.err.Error())

		return fmt.Sprintf("\n  %s Import failed:\n\n  %s\n\n  Press q to quit\n",
			m.styles.Error.Render("✗"),
			wrappedError,
		)
	}

	if m.result == nil {
		return "\n  Import completed with no result.\n\n  Press q to quit\n"
	}

	return fmt.Sprintf("\n  %s Import complete\n\n    Instances imported: %d\n\n  Press q to quit\n",
		m.styles.Success.Render("✓"),
		m.result.InstancesCount,
	)
}

func (m *ImportModel) writeHeadlessOutput() {
	if m.headlessWriter == nil {
		return
	}

	if m.jsonMode {
		m.writeJSONOutput()
		return
	}

	m.writeTextOutput()
}

func (m *ImportModel) writeJSONOutput() {
	if m.err != nil {
		jsonout.WriteJSON(m.headlessWriter, jsonout.NewErrorOutput(m.err))
		return
	}

	if m.result != nil {
		output := jsonout.StateImportOutput{
			Success:        m.result.Success,
			Mode:           "import",
			InstancesCount: m.result.InstancesCount,
			Message:        m.result.Message,
		}
		jsonout.WriteJSON(m.headlessWriter, output)
	}
}

func (m *ImportModel) writeTextOutput() {
	if m.err != nil {
		fmt.Fprintf(m.headlessWriter, "Import failed: %v\n", m.err)
		return
	}

	if m.result != nil {
		fmt.Fprintf(m.headlessWriter, "%s\n", m.result.Message)
	}
}

// SetFilePath sets the file path for the import.
func (m *ImportModel) SetFilePath(filePath string) {
	m.filePath = filePath
}

// SetWidth sets the terminal width for the import model.
func (m *ImportModel) SetWidth(width int) {
	m.width = width
}

// IsFinished returns whether the import has finished.
func (m *ImportModel) IsFinished() bool {
	return m.finished
}

// StartImport returns a command to start the import process.
func (m *ImportModel) StartImport() tea.Cmd {
	if stateio.IsRemoteFile(m.filePath) {
		return startDownloadCmd(m.filePath)
	}
	return startImportCmd(m.engineConfig, m.filePath)
}
