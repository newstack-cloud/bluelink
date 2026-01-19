package pluginuninstallui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

type uninstallStage int

const (
	preparingStage uninstallStage = iota
	uninstallingStage
	completeStage
	errorStage
)

// PluginStatus represents the current status of a plugin uninstallation.
type PluginStatus int

const (
	PluginPending PluginStatus = iota
	PluginRemoving
	PluginRemoved
	PluginNotFound
	PluginFailed
)

// PluginUninstallState tracks the state of a single plugin uninstallation.
type PluginUninstallState struct {
	PluginID   *plugins.PluginID
	Status     PluginStatus
	StatusText string
	Error      error
}

// MainModel is the main model for the plugin uninstall TUI.
type MainModel struct {
	stage          uninstallStage
	pluginStates   []*PluginUninstallState
	currentIndex   int
	spinner        spinner.Model
	styles         *stylespkg.Styles
	headless       bool
	headlessWriter io.Writer
	quitting       bool
	width          int
	Error          error

	// Results
	removedCount  int
	notFoundCount int
	failedCount   int

	// Dependencies
	manager *plugins.Manager
}

func (m MainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "Uninstalling %d plugin(s)...\n", len(m.pluginStates))
	}

	// Start uninstalling the first plugin
	if len(m.pluginStates) > 0 {
		m.pluginStates[0].Status = PluginRemoving
		m.pluginStates[0].StatusText = "Removing..."
		cmds = append(cmds, uninstallPluginCmd(m.manager, 0, m.pluginStates[0].PluginID))
	}

	return tea.Batch(cmds...)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			if m.stage == completeStage || m.stage == errorStage {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case PluginCompleteMsg:
		return m.handlePluginComplete(msg)

	case PluginErrorMsg:
		return m.handlePluginError(msg)
	}

	// Update spinner
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m MainModel) View() string {
	if m.headless {
		return ""
	}

	if m.quitting {
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("Uninstall cancelled.")
	}

	switch m.stage {
	case preparingStage, uninstallingStage:
		return m.renderUninstalling()
	case completeStage:
		return m.renderComplete()
	case errorStage:
		return m.renderError()
	}

	return "\n"
}

func (m *MainModel) handlePluginComplete(msg PluginCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Index >= len(m.pluginStates) {
		return m, nil
	}

	state := m.pluginStates[msg.Index]
	result := msg.Result

	switch result.Status {
	case plugins.UninstallStatusRemoved:
		state.Status = PluginRemoved
		state.StatusText = "Removed"
		m.removedCount++
		if m.headless {
			fmt.Fprintf(m.headlessWriter, "  %s: removed\n", state.PluginID.String())
		}

	case plugins.UninstallStatusNotFound:
		state.Status = PluginNotFound
		state.StatusText = "Not found"
		m.notFoundCount++
		if m.headless {
			fmt.Fprintf(m.headlessWriter, "  %s: not found\n", state.PluginID.String())
		}

	case plugins.UninstallStatusFailed:
		state.Status = PluginFailed
		state.StatusText = "Failed"
		state.Error = result.Error
		m.failedCount++
		if m.headless {
			fmt.Fprintf(m.headlessWriter, "  %s: failed - %v\n", state.PluginID.String(), result.Error)
		}
	}

	return m.moveToNextOrFinish()
}

func (m *MainModel) handlePluginError(msg PluginErrorMsg) (tea.Model, tea.Cmd) {
	if msg.Index >= len(m.pluginStates) {
		return m, nil
	}

	state := m.pluginStates[msg.Index]
	state.Status = PluginFailed
	state.StatusText = "Failed"
	state.Error = msg.Error
	m.failedCount++

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "  %s: failed - %v\n", state.PluginID.String(), msg.Error)
	}

	return m.moveToNextOrFinish()
}

func (m *MainModel) moveToNextOrFinish() (tea.Model, tea.Cmd) {
	m.currentIndex += 1
	if m.currentIndex >= len(m.pluginStates) {
		return m.finishUninstalling()
	}

	// Start next plugin
	nextState := m.pluginStates[m.currentIndex]
	nextState.Status = PluginRemoving
	nextState.StatusText = "Removing..."

	return m, uninstallPluginCmd(m.manager, m.currentIndex, nextState.PluginID)
}

func (m *MainModel) finishUninstalling() (tea.Model, tea.Cmd) {
	if m.failedCount > 0 {
		m.stage = errorStage
		m.Error = fmt.Errorf("%d plugin(s) failed to uninstall", m.failedCount)
	} else {
		m.stage = completeStage
	}

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "\nRemoved: %d, Not found: %d, Failed: %d\n",
			m.removedCount, m.notFoundCount, m.failedCount)
		return m, tea.Quit
	}

	return m, nil
}

func (m MainModel) renderUninstalling() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("  ")
	sb.WriteString(m.styles.Selected.Render("Uninstalling plugins"))
	sb.WriteString("\n\n")

	for _, state := range m.pluginStates {
		sb.WriteString("  ")
		sb.WriteString(m.renderStatusIcon(state.Status))
		sb.WriteString(" ")
		sb.WriteString(state.PluginID.String())

		if state.StatusText != "" {
			sb.WriteString(" ")
			sb.WriteString(m.styles.Muted.Render("(" + state.StatusText + ")"))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

func (m MainModel) renderStatusIcon(status PluginStatus) string {
	switch status {
	case PluginPending:
		return m.styles.Muted.Render("○")
	case PluginRemoving:
		return m.spinner.View()
	case PluginRemoved:
		return lipgloss.NewStyle().Foreground(m.styles.Palette.Success()).Render("✓")
	case PluginNotFound:
		return m.styles.Muted.Render("–")
	case PluginFailed:
		return lipgloss.NewStyle().Foreground(m.styles.Palette.Error()).Render("✗")
	}
	return m.styles.Muted.Render("○")
}

func (m MainModel) renderComplete() string {
	var sb strings.Builder

	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())

	sb.WriteString("\n")
	sb.WriteString(successStyle.Render("  ✓ Uninstall complete!"))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Summary:"))
	sb.WriteString("\n")

	if m.removedCount > 0 {
		sb.WriteString(fmt.Sprintf("    Removed:   %d\n", m.removedCount))
	}
	if m.notFoundCount > 0 {
		sb.WriteString(fmt.Sprintf("    Not found: %d\n", m.notFoundCount))
	}

	sb.WriteString("\n")

	for _, state := range m.pluginStates {
		sb.WriteString("  ")
		sb.WriteString(m.renderStatusIcon(state.Status))
		sb.WriteString(" ")
		sb.WriteString(state.PluginID.String())
		if state.Status == PluginNotFound {
			sb.WriteString(m.styles.Muted.Render(" (not found)"))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  Press q to quit."))
	sb.WriteString("\n\n")

	return sb.String()
}

func (m MainModel) renderError() string {
	var sb strings.Builder

	errorStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Error())
	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())

	sb.WriteString("\n")
	sb.WriteString(errorStyle.Render("  ✗ Uninstall completed with errors"))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Summary:"))
	sb.WriteString("\n")
	if m.removedCount > 0 {
		sb.WriteString(fmt.Sprintf("    Removed:   %d\n", m.removedCount))
	}
	if m.notFoundCount > 0 {
		sb.WriteString(fmt.Sprintf("    Not found: %d\n", m.notFoundCount))
	}
	if m.failedCount > 0 {
		sb.WriteString(fmt.Sprintf("    Failed:    %d\n", m.failedCount))
	}
	sb.WriteString("\n")

	maxWidth := min(max(m.width-8, 40), 100)
	errorWrapStyle := errorStyle.Width(maxWidth)

	for _, state := range m.pluginStates {
		sb.WriteString("  ")
		switch state.Status {
		case PluginRemoved:
			sb.WriteString(successStyle.Render("✓"))
		case PluginNotFound:
			sb.WriteString(m.styles.Muted.Render("–"))
		case PluginFailed:
			sb.WriteString(errorStyle.Render("✗"))
		}
		sb.WriteString(" ")
		sb.WriteString(state.PluginID.String())

		if state.Status == PluginFailed && state.Error != nil {
			sb.WriteString("\n    ")
			sb.WriteString(errorWrapStyle.Render(state.Error.Error()))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  Press q to quit."))
	sb.WriteString("\n\n")

	return sb.String()
}

// UninstallAppOptions contains options for creating a new uninstall app.
type UninstallAppOptions struct {
	PluginIDs      []*plugins.PluginID
	Styles         *stylespkg.Styles
	Headless       bool
	HeadlessWriter io.Writer
	Manager        *plugins.Manager
}

// NewUninstallApp creates a new plugin uninstall TUI application.
func NewUninstallApp(opts UninstallAppOptions) (*MainModel, error) {
	if len(opts.PluginIDs) == 0 {
		return nil, fmt.Errorf("no plugins to uninstall")
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = opts.Styles.Selected

	// Create plugin states
	states := make([]*PluginUninstallState, len(opts.PluginIDs))
	for i, id := range opts.PluginIDs {
		states[i] = &PluginUninstallState{
			PluginID: id,
			Status:   PluginPending,
		}
	}

	// Create manager if not provided
	manager := opts.Manager
	if manager == nil {
		manager = plugins.NewManager(nil, nil)
	}

	return &MainModel{
		stage:          preparingStage,
		pluginStates:   states,
		currentIndex:   0,
		spinner:        s,
		styles:         opts.Styles,
		headless:       opts.Headless,
		headlessWriter: opts.HeadlessWriter,
		width:          80,
		manager:        manager,
	}, nil
}
