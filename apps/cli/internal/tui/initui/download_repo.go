package initui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/git"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

type DownloadRepoModel struct {
	git       git.Git
	directory string
	spinner   spinner.Model
	err       error
	template  string
	styles    *stylespkg.Styles
}

func (m DownloadRepoModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		startDownloadCmd(&m),
	)
}

func (m DownloadRepoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case DownloadCompleteMsg:
		return m, downloadRepoCompleteCmd(msg.RepoPath)
	case DownloadErrorMsg:
		m.err = msg.Err
		return m, tea.Quit
	}

	return m, nil
}

func (m DownloadRepoModel) View() string {
	return fmt.Sprintf("\n\n %s Downloading template repository...\n\n", m.spinner.View())
}

func NewDownloadRepoModel(
	template string,
	directory string,
	bluelinkStyles *stylespkg.Styles,
	git git.Git,
) *DownloadRepoModel {
	spinnerModel := spinner.New()
	spinnerModel.Spinner = spinner.Dot
	spinnerModel.Style = bluelinkStyles.Spinner

	return &DownloadRepoModel{
		git:       git,
		spinner:   spinnerModel,
		template:  template,
		directory: directory,
		styles:    bluelinkStyles,
	}
}
