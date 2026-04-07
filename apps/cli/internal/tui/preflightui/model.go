package preflightui

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/enginectl"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/plugininstallui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/preflight"
)

type preflightStage int

const (
	preflightChecking   preflightStage = iota
	preflightInstalling
	preflightComplete
	preflightError
)

// Type aliases so existing references (tests, validate command) continue to compile
// while the actual message types come from the SDK's preflight package.
type PreflightSatisfiedMsg = preflight.SatisfiedMsg
type PreflightInstalledMsg = preflight.InstalledMsg
type PreflightErrorMsg = preflight.ErrorMsg

// PreflightOptions contains options for creating a new preflight model.
type PreflightOptions struct {
	ConfProvider   *config.Provider
	CommandName    string
	Styles         *stylespkg.Styles
	Headless       bool
	HeadlessWriter io.Writer
	JsonMode       bool
}

// PreflightModel is a TUI sub-model that checks for missing plugin
// dependencies and installs them before the main command runs.
type PreflightModel struct {
	stage        preflightStage
	confProvider *config.Provider

	installModel *plugininstallui.MainModel

	commandName         string
	unsatisfied         []*plugins.PluginID
	allToInstall        []*plugins.PluginID
	installedCount      int
	restartInstructions string

	headless       bool
	headlessWriter io.Writer
	styles         *stylespkg.Styles
	spinner        spinner.Model
	jsonMode       bool

	Error error
}

// NewPreflightModel creates a new preflight check sub-model.
func NewPreflightModel(opts PreflightOptions) *PreflightModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	if opts.Styles != nil {
		s.Style = opts.Styles.Selected
	}

	return &PreflightModel{
		stage:          preflightChecking,
		confProvider:   opts.ConfProvider,
		commandName:    opts.CommandName,
		headless:       opts.Headless,
		headlessWriter: opts.HeadlessWriter,
		styles:         opts.Styles,
		spinner:        s,
		jsonMode:       opts.JsonMode,
	}
}

func (m PreflightModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick, checkPluginsCmd(m.confProvider)}

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "Checking plugin dependencies...\n")
	}

	return tea.Batch(cmds...)
}

func (m PreflightModel) Update(msg tea.Msg) (PreflightModel, tea.Cmd) {
	switch msg := msg.(type) {
	case preflightCheckResultMsg:
		return m.handleCheckResult(msg)

	case plugininstallui.InstallCompleteMsg:
		return m.handleInstallComplete(msg)

	case preflight.SatisfiedMsg:
		return m, nil

	case preflight.InstalledMsg:
		return m, nil

	case preflight.ErrorMsg:
		m.stage = preflightError
		m.Error = msg.Err
		return m, nil

	case tea.KeyMsg:
		if m.stage == preflightComplete && msg.String() == "q" {
			return m, m.preflightInstalledCmd()
		}
	}

	if m.stage == preflightInstalling && m.installModel != nil {
		updated, cmd := m.installModel.Update(msg)
		switch model := updated.(type) {
		case *plugininstallui.MainModel:
			m.installModel = model
		case plugininstallui.MainModel:
			m.installModel = &model
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m PreflightModel) View() string {
	if m.headless {
		return ""
	}

	switch m.stage {
	case preflightChecking:
		return m.renderChecking()
	case preflightInstalling:
		if m.installModel != nil {
			return m.installModel.View()
		}
	case preflightComplete:
		return m.renderComplete()
	}

	return ""
}

func (m *PreflightModel) handleCheckResult(msg preflightCheckResultMsg) (PreflightModel, tea.Cmd) {
	m.unsatisfied = msg.unsatisfied
	m.allToInstall = msg.allToInstall
	m.stage = preflightInstalling

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "Found %d unsatisfied plugin(s), installing...\n",
			len(msg.unsatisfied))
	}

	installModel, err := plugininstallui.NewInstallApp(context.TODO(), plugininstallui.InstallAppOptions{
		PluginIDs:        msg.allToInstall,
		UserRequestedIDs: msg.unsatisfied,
		Styles:           m.styles,
		Headless:         m.headless,
		HeadlessWriter:   m.headlessWriter,
		Manager:          msg.manager,
	})
	if err != nil {
		m.stage = preflightError
		m.Error = fmt.Errorf("failed to create plugin installer: %w", err)
		return *m, func() tea.Msg {
			return preflight.ErrorMsg{Err: m.Error}
		}
	}

	installModel.SetEmbeddedMode(true)
	m.installModel = installModel

	return *m, m.installModel.Init()
}

func (m *PreflightModel) handleInstallComplete(msg plugininstallui.InstallCompleteMsg) (PreflightModel, tea.Cmd) {
	if msg.Error != nil {
		m.stage = preflightError
		m.Error = msg.Error
		return *m, func() tea.Msg {
			return preflight.ErrorMsg{Err: msg.Error}
		}
	}

	m.stage = preflightComplete
	m.installedCount = msg.InstalledCount
	m.restartInstructions = enginectl.RestartInstructions()

	if m.headless {
		fmt.Fprintf(m.headlessWriter,
			"The deploy configuration requires plugin(s) that were not installed.\n")
		fmt.Fprintf(m.headlessWriter, "%d missing plugin(s) installed:\n", m.installedCount)
		for _, p := range m.allToInstall {
			fmt.Fprintf(m.headlessWriter, "  • %s\n", p.String())
		}
		fmt.Fprintf(m.headlessWriter, "\n%s\n", m.restartInstructions)
		if m.commandName != "" {
			fmt.Fprintf(m.headlessWriter,
				"Re-run the `%s` command after restarting the engine.\n", m.commandName)
		}
		return *m, m.preflightInstalledCmd()
	}

	// In interactive mode, stay in the TUI so the user can read the
	// restart instructions. PreflightInstalledMsg is sent when the
	// user presses "q" to quit.
	return *m, nil
}

func (m PreflightModel) preflightInstalledCmd() tea.Cmd {
	pluginNames := make([]string, len(m.allToInstall))
	for i, p := range m.allToInstall {
		pluginNames[i] = p.String()
	}
	msg := preflight.InstalledMsg{
		CommandName:         m.commandName,
		RestartInstructions: m.restartInstructions,
		InstalledPlugins:    pluginNames,
		InstalledCount:      m.installedCount,
	}
	return func() tea.Msg { return msg }
}
