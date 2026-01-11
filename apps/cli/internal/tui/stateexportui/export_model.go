package stateexportui

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

// ExportModelConfig holds configuration for the export model.
type ExportModelConfig struct {
	EngineConfig    *stateio.EngineConfig
	FilePath        string
	InstanceFilters []string
	Styles          *stylespkg.Styles
	Headless        bool
	HeadlessWriter  io.Writer
	JSONMode        bool
}

// ExportModel handles the export progress display.
type ExportModel struct {
	spinner         spinner.Model
	engineConfig    *stateio.EngineConfig
	filePath        string
	instanceFilters []string
	exporting       bool
	result          *stateio.ExportResult
	err             error
	finished        bool
	headless        bool
	headlessWriter  io.Writer
	jsonMode        bool
	styles          *stylespkg.Styles
	width           int
}

// NewExportModel creates a new export model.
func NewExportModel(config ExportModelConfig) *ExportModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = config.Styles.Spinner

	return &ExportModel{
		spinner:         s,
		engineConfig:    config.EngineConfig,
		filePath:        config.FilePath,
		instanceFilters: config.InstanceFilters,
		headless:        config.Headless,
		headlessWriter:  config.HeadlessWriter,
		jsonMode:        config.JSONMode,
		styles:          config.Styles,
		width:           80,
	}
}

func (m *ExportModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *ExportModel) Update(msg tea.Msg) (*ExportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case ExportStartedMsg:
		m.exporting = true
		return m, nil
	case ExportCompleteMsg:
		m.exporting = false
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

func (m *ExportModel) View() string {
	if m.headless {
		return ""
	}

	if m.exporting {
		return fmt.Sprintf("\n  %s Exporting state to %s...\n", m.spinner.View(), m.filePath)
	}

	if m.finished {
		return m.renderResult()
	}

	return ""
}

func (m *ExportModel) renderResult() string {
	if m.err != nil {
		maxWidth := m.width - 6
		if maxWidth < 40 {
			maxWidth = 40
		}

		wrapStyle := lipgloss.NewStyle().Width(maxWidth)
		wrappedError := wrapStyle.Render(m.err.Error())

		return fmt.Sprintf("\n  %s Export failed:\n\n  %s\n\n  Press q to quit\n",
			m.styles.Error.Render("✗"),
			wrappedError,
		)
	}

	if m.result == nil {
		return "\n  Export completed with no result.\n\n  Press q to quit\n"
	}

	return fmt.Sprintf("\n  %s Export complete\n\n    Instances exported: %d\n    Output file: %s\n\n  Press q to quit\n",
		m.styles.Success.Render("✓"),
		m.result.InstancesCount,
		m.result.FilePath,
	)
}

func (m *ExportModel) writeHeadlessOutput() {
	if m.headlessWriter == nil {
		return
	}

	if m.jsonMode {
		m.writeJSONOutput()
		return
	}

	m.writeTextOutput()
}

func (m *ExportModel) writeJSONOutput() {
	if m.err != nil {
		jsonout.WriteJSON(m.headlessWriter, jsonout.NewErrorOutput(m.err))
		return
	}

	if m.result != nil {
		output := jsonout.StateImportOutput{
			Success:        m.result.Success,
			Mode:           "export",
			InstancesCount: m.result.InstancesCount,
			Message:        m.result.Message,
		}
		jsonout.WriteJSON(m.headlessWriter, output)
	}
}

func (m *ExportModel) writeTextOutput() {
	if m.err != nil {
		fmt.Fprintf(m.headlessWriter, "Export failed: %v\n", m.err)
		return
	}

	if m.result != nil {
		fmt.Fprintf(m.headlessWriter, "%s\n", m.result.Message)
	}
}

// SetFilePath sets the file path for the export.
func (m *ExportModel) SetFilePath(filePath string) {
	m.filePath = filePath
}

// SetWidth sets the terminal width for the export model.
func (m *ExportModel) SetWidth(width int) {
	m.width = width
}

// IsFinished returns whether the export has finished.
func (m *ExportModel) IsFinished() bool {
	return m.finished
}

// StartExport returns a command to start the export process.
func (m *ExportModel) StartExport() tea.Cmd {
	return startExportCmd(m.engineConfig, m.filePath, m.instanceFilters)
}
