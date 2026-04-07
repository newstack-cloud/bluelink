package preflight

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/preflightui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// BluelinkPreflightFactory implements commands.PreflightFactory for the
// Bluelink CLI. It creates a preflight model that checks plugin
// dependencies and installs missing plugins before commands run.
type BluelinkPreflightFactory struct{}

func (f *BluelinkPreflightFactory) CreatePreflight(
	confProvider *config.Provider,
	commandName string,
	s *styles.Styles,
	headless bool,
	writer io.Writer,
	jsonMode bool,
) tea.Model {
	inner := preflightui.NewPreflightModel(preflightui.PreflightOptions{
		ConfProvider:   confProvider,
		CommandName:    commandName,
		Styles:         s,
		Headless:       headless,
		HeadlessWriter: writer,
		JsonMode:       jsonMode,
	})
	return &modelAdapter{inner: inner}
}

// modelAdapter wraps PreflightModel so it satisfies tea.Model.
// PreflightModel.Update returns (PreflightModel, tea.Cmd) rather than
// (tea.Model, tea.Cmd), so this adapter bridges the gap.
type modelAdapter struct {
	inner *preflightui.PreflightModel
}

func (a *modelAdapter) Init() tea.Cmd {
	return a.inner.Init()
}

func (a *modelAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := a.inner.Update(msg)
	a.inner = &updated
	return a, cmd
}

func (a *modelAdapter) View() string {
	return a.inner.View()
}
