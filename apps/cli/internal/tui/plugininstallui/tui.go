package plugininstallui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/registries"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

type installStage int

const (
	preparingStage installStage = iota
	installingStage
	completeStage
	errorStage
)

// PluginStatus represents the current status of a plugin installation.
type PluginStatus int

const (
	PluginPending PluginStatus = iota
	PluginResolving
	PluginDownloading
	PluginVerifying
	PluginExtracting
	PluginComplete
	PluginSkipped
	PluginFailed
)

// PluginInstallState tracks the state of a single plugin installation.
type PluginInstallState struct {
	PluginID        *plugins.PluginID
	Status          PluginStatus
	StatusText      string
	DownloadedBytes int64
	TotalBytes      int64
	Error           error
	IsDependency    bool
}

// MainModel is the main model for the plugin install TUI.
type MainModel struct {
	ctx            context.Context
	stage          installStage
	pluginStates   []*PluginInstallState
	currentPlugin  int
	spinner        spinner.Model
	progress       progress.Model
	styles         *stylespkg.Styles
	headless       bool
	headlessWriter io.Writer
	quitting       bool
	width          int
	Error          error

	// Results
	installedCount int
	skippedCount   int
	failedCount    int

	// Dependencies
	manager *plugins.Manager
}

func (m MainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "Installing %d plugin(s)...\n", len(m.pluginStates))
	}

	// Start installing the first plugin
	if len(m.pluginStates) > 0 {
		m.pluginStates[0].Status = PluginResolving
		m.pluginStates[0].StatusText = "Resolving..."
		cmds = append(cmds, installPluginCmd(m.ctx, m.manager, 0, m.pluginStates[0].PluginID))
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
		m.progress.Width = msg.Width - 20
		if m.progress.Width > 60 {
			m.progress.Width = 60
		}
		return m, nil

	case PluginProgressMsg:
		return m.handlePluginProgress(msg)

	case PluginCompleteMsg:
		return m.handlePluginComplete(msg)

	case PluginErrorMsg:
		return m.handlePluginError(msg)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
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
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("Install cancelled.")
	}

	switch m.stage {
	case preparingStage, installingStage:
		return m.renderInstalling()
	case completeStage:
		return m.renderComplete()
	case errorStage:
		return m.renderError()
	}

	return "\n"
}

func (m *MainModel) handlePluginProgress(msg PluginProgressMsg) (tea.Model, tea.Cmd) {
	if msg.Index >= len(m.pluginStates) {
		return m, nil
	}

	state := m.pluginStates[msg.Index]
	state.Status = msg.Status
	state.StatusText = msg.StatusText
	state.DownloadedBytes = msg.Downloaded
	state.TotalBytes = msg.Total

	if m.headless && msg.Status == PluginDownloading && msg.Total > 0 {
		percent := float64(msg.Downloaded) / float64(msg.Total) * 100
		fmt.Fprintf(m.headlessWriter, "  %s: downloading %.0f%%\n",
			state.PluginID.String(), percent)
	}

	var cmds []tea.Cmd
	if msg.Total > 0 && state.Status == PluginDownloading {
		percent := float64(msg.Downloaded) / float64(msg.Total)
		cmds = append(cmds, m.progress.SetPercent(percent))
	}

	return m, tea.Batch(cmds...)
}

func (m *MainModel) handlePluginComplete(msg PluginCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Index >= len(m.pluginStates) {
		return m, nil
	}

	state := m.pluginStates[msg.Index]
	result := msg.Result

	switch result.Status {
	case plugins.StatusInstalled:
		state.Status = PluginComplete
		state.StatusText = "Installed"
		m.installedCount += 1
		if m.headless {
			label := "installed"
			if state.IsDependency {
				label = "installed (dependency)"
			}
			fmt.Fprintf(m.headlessWriter, "  %s: %s\n", state.PluginID.String(), label)
		}

	case plugins.StatusSkipped:
		state.Status = PluginSkipped
		state.StatusText = "Already installed"
		m.skippedCount += 1
		if m.headless {
			fmt.Fprintf(m.headlessWriter, "  %s: skipped (already installed)\n", state.PluginID.String())
		}

	case plugins.StatusFailed:
		state.Status = PluginFailed
		state.StatusText = "Failed"
		state.Error = result.Error
		m.failedCount += 1
		if m.headless {
			fmt.Fprintf(m.headlessWriter, "  %s: failed - %v\n", state.PluginID.String(), result.Error)
		}
	}

	// Move to next plugin or complete
	m.currentPlugin += 1
	if m.currentPlugin >= len(m.pluginStates) {
		return m.finishInstallation()
	}

	// Start next plugin
	nextState := m.pluginStates[m.currentPlugin]
	nextState.Status = PluginResolving
	nextState.StatusText = "Resolving..."

	return m, installPluginCmd(m.ctx, m.manager, m.currentPlugin, nextState.PluginID)
}

func (m *MainModel) handlePluginError(msg PluginErrorMsg) (tea.Model, tea.Cmd) {
	if msg.Index >= len(m.pluginStates) {
		return m, nil
	}

	state := m.pluginStates[msg.Index]
	state.Status = PluginFailed
	state.StatusText = "Failed"
	state.Error = msg.Error
	m.failedCount += 1

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "  %s: failed - %v\n", state.PluginID.String(), msg.Error)
	}

	// Move to next plugin or complete
	m.currentPlugin += 1
	if m.currentPlugin >= len(m.pluginStates) {
		return m.finishInstallation()
	}

	// Start next plugin
	nextState := m.pluginStates[m.currentPlugin]
	nextState.Status = PluginResolving
	nextState.StatusText = "Resolving..."

	return m, installPluginCmd(m.ctx, m.manager, m.currentPlugin, nextState.PluginID)
}

func (m *MainModel) finishInstallation() (tea.Model, tea.Cmd) {
	if m.failedCount > 0 {
		m.stage = errorStage
		m.Error = fmt.Errorf("%d plugin(s) failed to install", m.failedCount)
	} else {
		m.stage = completeStage
	}

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "\nInstalled: %d, Skipped: %d, Failed: %d\n",
			m.installedCount, m.skippedCount, m.failedCount)
		return m, tea.Quit
	}

	return m, nil
}

func (m MainModel) renderInstalling() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("  ")
	sb.WriteString(m.styles.Selected.Render("Installing plugins"))
	sb.WriteString("\n\n")

	for i, state := range m.pluginStates {
		sb.WriteString("  ")

		// Status icon
		switch state.Status {
		case PluginPending:
			sb.WriteString(m.styles.Muted.Render("○"))
		case PluginResolving, PluginDownloading, PluginVerifying, PluginExtracting:
			sb.WriteString(m.spinner.View())
		case PluginComplete:
			sb.WriteString(lipgloss.NewStyle().Foreground(m.styles.Palette.Success()).Render("✓"))
		case PluginSkipped:
			sb.WriteString(m.styles.Muted.Render("–"))
		case PluginFailed:
			sb.WriteString(lipgloss.NewStyle().Foreground(m.styles.Palette.Error()).Render("✗"))
		}

		sb.WriteString(" ")
		sb.WriteString(state.PluginID.String())
		if state.IsDependency {
			sb.WriteString(" ")
			sb.WriteString(m.styles.Muted.Render("(dependency)"))
		}

		if state.StatusText != "" {
			sb.WriteString(" ")
			sb.WriteString(m.styles.Muted.Render("(" + state.StatusText + ")"))
		}

		if state.Status == PluginDownloading && state.TotalBytes > 0 && i == m.currentPlugin {
			sb.WriteString("\n    ")
			sb.WriteString(m.progress.View())
		}

		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

func (m MainModel) renderComplete() string {
	var sb strings.Builder

	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())

	sb.WriteString("\n")
	sb.WriteString(successStyle.Render("  ✓ Installation complete!"))
	sb.WriteString("\n\n")

	// Show summary
	sb.WriteString(m.styles.Muted.Render("  Summary:"))
	sb.WriteString("\n")

	if m.installedCount > 0 {
		sb.WriteString(fmt.Sprintf("    Installed: %d\n", m.installedCount))
	}
	if m.skippedCount > 0 {
		sb.WriteString(fmt.Sprintf("    Skipped:   %d (already installed)\n", m.skippedCount))
	}

	sb.WriteString("\n")

	for _, state := range m.pluginStates {
		sb.WriteString("  ")
		switch state.Status {
		case PluginComplete:
			sb.WriteString(successStyle.Render("✓"))
		case PluginSkipped:
			sb.WriteString(m.styles.Muted.Render("–"))
		}
		sb.WriteString(" ")
		sb.WriteString(state.PluginID.String())
		if state.IsDependency {
			sb.WriteString(m.styles.Muted.Render(" (dependency)"))
		}
		if state.Status == PluginSkipped {
			sb.WriteString(m.styles.Muted.Render(" (already installed)"))
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
	sb.WriteString(errorStyle.Render("  ✗ Installation completed with errors"))
	sb.WriteString("\n\n")

	// Show summary
	sb.WriteString(m.styles.Muted.Render("  Summary:"))
	sb.WriteString("\n")
	if m.installedCount > 0 {
		sb.WriteString(fmt.Sprintf("    Installed: %d\n", m.installedCount))
	}
	if m.skippedCount > 0 {
		sb.WriteString(fmt.Sprintf("    Skipped:   %d\n", m.skippedCount))
	}
	if m.failedCount > 0 {
		sb.WriteString(fmt.Sprintf("    Failed:    %d\n", m.failedCount))
	}
	sb.WriteString("\n")

	// Calculate max width for error text wrapping
	// Account for 4-space indent on error lines
	maxWidth := m.width - 8
	if maxWidth < 40 {
		maxWidth = 40
	}
	if maxWidth > 100 {
		maxWidth = 100
	}
	errorWrapStyle := errorStyle.Width(maxWidth)

	for _, state := range m.pluginStates {
		sb.WriteString("  ")
		switch state.Status {
		case PluginComplete:
			sb.WriteString(successStyle.Render("✓"))
		case PluginSkipped:
			sb.WriteString(m.styles.Muted.Render("–"))
		case PluginFailed:
			sb.WriteString(errorStyle.Render("✗"))
		}
		sb.WriteString(" ")
		sb.WriteString(state.PluginID.String())
		if state.IsDependency {
			sb.WriteString(m.styles.Muted.Render(" (dependency)"))
		}

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

// InstallAppOptions contains options for creating a new install app.
type InstallAppOptions struct {
	PluginIDs        []*plugins.PluginID
	UserRequestedIDs []*plugins.PluginID
	Styles           *stylespkg.Styles
	Headless         bool
	HeadlessWriter   io.Writer
	Manager          *plugins.Manager
}

// NewInstallApp creates a new plugin install TUI application.
func NewInstallApp(ctx context.Context, opts InstallAppOptions) (*MainModel, error) {
	if len(opts.PluginIDs) == 0 {
		return nil, fmt.Errorf("no plugins to install")
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = opts.Styles.Selected

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	states := buildPluginStates(opts.PluginIDs, opts.UserRequestedIDs)

	manager := opts.Manager
	if manager == nil {
		authStore := registries.NewAuthConfigStore()
		tokenStore := registries.NewTokenStore()
		discoveryClient := registries.NewServiceDiscoveryClient()
		registryClient := registries.NewRegistryClient(authStore, tokenStore, discoveryClient)
		manager = plugins.NewManager(registryClient, discoveryClient)
	}

	return &MainModel{
		ctx:            ctx,
		stage:          preparingStage,
		pluginStates:   states,
		currentPlugin:  0,
		spinner:        s,
		progress:       p,
		styles:         opts.Styles,
		headless:       opts.Headless,
		headlessWriter: opts.HeadlessWriter,
		width:          80, // Default width until WindowSizeMsg is received
		manager:        manager,
	}, nil
}

func buildPluginStates(
	pluginIDs []*plugins.PluginID,
	userRequestedIDs []*plugins.PluginID,
) []*PluginInstallState {
	userRequested := make(map[string]bool, len(userRequestedIDs))
	for _, id := range userRequestedIDs {
		userRequested[id.FullyQualified()] = true
	}

	states := make([]*PluginInstallState, len(pluginIDs))
	for i, id := range pluginIDs {
		states[i] = &PluginInstallState{
			PluginID:     id,
			Status:       PluginPending,
			IsDependency: len(userRequested) > 0 && !userRequested[id.FullyQualified()],
		}
	}
	return states
}
