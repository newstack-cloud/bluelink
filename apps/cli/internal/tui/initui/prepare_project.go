package initui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/git"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/project"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

type PrepareProjectModel struct {
	git             git.Git
	preparer        project.Preparer
	directory       string
	spinner         spinner.Model
	err             error
	projectName     string
	blueprintFormat string
	noGit           bool
	currentStep     string
	styles          *stylespkg.Styles
}

func (m PrepareProjectModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		removeGitHistoryCmd(m.directory, m.preparer),
	)
}

func (m PrepareProjectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case RemoveGitHistoryCompleteMsg:
		m.currentStep = "Cleaning up template files..."
		return m, removeMaintainerFilesCmd(m.directory, m.preparer)
	case RemoveMaintainerFilesCompleteMsg:
		if m.noGit {
			// Skip git init, go straight to blueprint selection
			m.currentStep = "Setting up blueprint files..."
			return m, selectBlueprintFormatCmd(m.directory, m.blueprintFormat, m.preparer)
		}
		m.currentStep = "Initializing git repository..."
		return m, initGitRepoCmd(m.directory, m.git)
	case InitGitRepoCompleteMsg:
		m.currentStep = "Setting up blueprint files..."
		return m, selectBlueprintFormatCmd(m.directory, m.blueprintFormat, m.preparer)
	case SelectBlueprintFormatCompleteMsg:
		m.currentStep = "Configuring project..."
		return m, substitutePlaceholdersCmd(m.directory, m.projectName, m.blueprintFormat, m.preparer)
	case SubstitutePlaceholdersCompleteMsg:
		return m, prepareProjectCompleteCmd()
	case PrepareErrorMsg:
		m.err = msg.Err
		return m, tea.Quit
	}

	return m, nil
}

func (m PrepareProjectModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n\n  Error: %v\n\n", m.err)
	}

	step := m.currentStep
	if step == "" {
		step = "Removing template git history..."
	}

	return fmt.Sprintf("\n\n %s %s\n\n", m.spinner.View(), step)
}

func NewPrepareProjectModel(
	projectName string,
	blueprintFormat string,
	noGit bool,
	directory string,
	bluelinkStyles *stylespkg.Styles,
	git git.Git,
	preparer project.Preparer,
) *PrepareProjectModel {
	spinnerModel := spinner.New()
	spinnerModel.Spinner = spinner.Dot
	spinnerModel.Style = bluelinkStyles.Spinner

	return &PrepareProjectModel{
		git:             git,
		preparer:        preparer,
		spinner:         spinnerModel,
		projectName:     projectName,
		blueprintFormat: blueprintFormat,
		noGit:           noGit,
		directory:       directory,
		styles:          bluelinkStyles,
	}
}
