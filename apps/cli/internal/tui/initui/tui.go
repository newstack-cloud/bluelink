package initui

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/git"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/project"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

type initStage int

const (
	// Stage where the user selects the template for the project.
	selectTemplateStage initStage = iota

	// Stage where the user inputs the project name,
	// selects blueprint format and whether to use git.
	inputFormStage

	// Stage where the user enters the directory for the project.
	inputDirectoryStage

	// Stage where the repo for the selected template is being downloaded.
	downloadRepoStage

	// Stage where the project is being prepared in the requested directory.
	prepareProjectStage

	// Stage where the user has completed the init process.
	initCompleteStage
)

type InitModel struct {
	stage                   initStage
	selectTemplate          tea.Model
	inputForm               tea.Model
	inputDirectory          tea.Model
	downloadRepo            tea.Model
	prepareProject          tea.Model
	template                string
	selectedTemplate        bool
	projectName             string
	selectedProjectName     bool
	blueprintFormat         string
	selectedBlueprintFormat bool
	noGit                   *bool
	selectedNoGit           bool
	directory               string
	quitting                bool
	Error                   error
	styles                  *stylespkg.Styles
	gitService              git.Git
	preparer                project.Preparer
	headless                bool
	headlessWriter          io.Writer
}

func (m InitModel) Init() tea.Cmd {
	switch m.stage {
	case selectTemplateStage:
		return m.selectTemplate.Init()
	case inputFormStage:
		return m.inputForm.Init()
	case inputDirectoryStage:
		return m.inputDirectory.Init()
	case downloadRepoStage:
		return m.downloadRepo.Init()
	case prepareProjectStage:
		return m.prepareProject.Init()
	}
	return nil
}

func (m InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
		// Any key press on the completion screen exits
		if m.stage == initCompleteStage {
			return m, tea.Quit
		}
	case SelectTemplateMsg:
		if len(strings.TrimSpace(msg.Template)) > 0 {
			m.template = msg.Template
			m.selectedTemplate = true
			m.stage = inputFormStage
			// Initialize the input form when transitioning to it
			return m, m.inputForm.Init()
		}
	case InputFormCompleteMsg:
		m.projectName = msg.ProjectName
		m.selectedProjectName = true
		m.blueprintFormat = msg.BlueprintFormat
		m.selectedBlueprintFormat = true
		noGit := msg.NoGit
		m.noGit = &noGit
		m.selectedNoGit = true
		m.stage = inputDirectoryStage

		// Recreate the input directory model with the new project name
		m.inputDirectory = NewInputDirectoryModel(
			InputDirectoryInitialValues{
				Directory: m.directory,
			},
			m.projectName,
			m.styles,
		)

		// Initialize directory input stage when implemented
		return m, m.inputDirectory.Init()
	case InputDirectoryCompleteMsg:
		m.directory = msg.Directory
		m.stage = downloadRepoStage

		if m.headless {
			fmt.Fprintln(m.headlessWriter, "Downloading template...")
		}

		// Recreate the download repo model with the provided directory
		m.downloadRepo = NewDownloadRepoModel(
			m.template,
			m.directory,
			m.styles,
			m.gitService,
		)

		return m, m.downloadRepo.Init()
	case DownloadCompleteMsg:
		m.stage = prepareProjectStage

		if m.headless {
			fmt.Fprintln(m.headlessWriter, "Preparing project...")
		}

		// Determine noGit value (default to false if not set)
		noGit := false
		if m.noGit != nil {
			noGit = *m.noGit
		}

		// Recreate the prepare project model with all required values
		m.prepareProject = NewPrepareProjectModel(
			m.projectName,
			m.blueprintFormat,
			noGit,
			m.directory,
			m.styles,
			m.gitService,
			m.preparer,
		)

		return m, m.prepareProject.Init()
	case DownloadErrorMsg:
		if m.headless {
			fmt.Fprintf(m.headlessWriter, "Error: %v\n", msg.Err)
		}
		m.Error = msg.Err
		return m, tea.Quit
	case PrepareErrorMsg:
		if m.headless {
			fmt.Fprintf(m.headlessWriter, "Error: %v\n", msg.Err)
		}
		m.Error = msg.Err
		return m, tea.Quit
	case PrepareCompleteMsg:
		m.stage = initCompleteStage
		if m.headless {
			m.writeHeadlessCompletion()
			return m, tea.Quit
		}
		return m, nil
	}

	// Route updates to the current stage's model
	var cmd tea.Cmd
	switch m.stage {
	case selectTemplateStage:
		m.selectTemplate, cmd = m.selectTemplate.Update(msg)
	case inputFormStage:
		m.inputForm, cmd = m.inputForm.Update(msg)
	case inputDirectoryStage:
		m.inputDirectory, cmd = m.inputDirectory.Update(msg)
	case downloadRepoStage:
		m.downloadRepo, cmd = m.downloadRepo.Update(msg)
	case prepareProjectStage:
		m.prepareProject, cmd = m.prepareProject.Update(msg)
	}
	return m, cmd
}

func (m InitModel) View() string {
	if m.headless {
		return ""
	}

	if m.quitting {
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("Not hungry? That's cool.")
	}

	switch m.stage {
	case selectTemplateStage:
		return "\n" + m.selectTemplate.View()
	case inputFormStage:
		return "\n" + m.inputForm.View()
	case inputDirectoryStage:
		return "\n" + m.inputDirectory.View()
	case downloadRepoStage:
		return "\n" + m.downloadRepo.View()
	case prepareProjectStage:
		return "\n" + m.prepareProject.View()
	case initCompleteStage:
		return m.renderCompletionMessage()
	}

	return "\n"
}

func (m InitModel) renderCompletionMessage() string {
	var sb strings.Builder

	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())

	// Success header
	sb.WriteString("\n")
	sb.WriteString(successStyle.Render("  ✓ Project initialized successfully!"))
	sb.WriteString("\n\n")

	// Project details
	sb.WriteString(m.styles.Muted.Render("  Project: "))
	sb.WriteString(m.styles.Selected.Render(m.projectName))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  Location: "))
	sb.WriteString(m.styles.Selected.Render(m.directory))
	sb.WriteString("\n\n")

	// Next steps
	sb.WriteString("  Get started:\n\n")
	sb.WriteString(m.styles.Muted.Render("    $ "))
	sb.WriteString(fmt.Sprintf("cd %s\n", m.directory))
	sb.WriteString(m.styles.Muted.Render("    $ "))
	sb.WriteString("cat README.md\n\n")

	sb.WriteString(m.styles.Muted.Render("  Follow the instructions in the README to configure and deploy your project."))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Press any key to exit."))
	sb.WriteString("\n\n")

	return sb.String()
}

func (m InitModel) writeHeadlessCompletion() {
	fmt.Fprintln(m.headlessWriter, "Project initialized successfully!")
	fmt.Fprintln(m.headlessWriter)
	fmt.Fprintf(m.headlessWriter, "Project: %s\n", m.projectName)
	fmt.Fprintf(m.headlessWriter, "Location: %s\n", m.directory)
	fmt.Fprintln(m.headlessWriter)
	fmt.Fprintln(m.headlessWriter, "Get started:")
	fmt.Fprintf(m.headlessWriter, "  cd %s\n", m.directory)
	fmt.Fprintln(m.headlessWriter, "  cat README.md")
}

// InitialState is the initial state for the init TUI app
// to initialise a new project.
type InitialState struct {
	Template                 string
	IsDefaultTemplate        bool
	ProjectName              string
	BlueprintFormat          string
	IsDefaultBlueprintFormat bool
	NoGit                    *bool
	IsDefaultNoGit           bool
	Directory                string
}

func NewInitApp(
	initialState InitialState,
	bluelinkStyles *stylespkg.Styles,
	gitService git.Git,
	preparer project.Preparer,
	headless bool,
	headlessWriter io.Writer,
) (*InitModel, error) {
	selectTemplate, err := NewSelectTemplateModel(
		bluelinkStyles,
		initialState.IsDefaultTemplate,
	)
	if err != nil {
		return nil, err
	}

	// Create input form model
	inputForm := NewInputFormModel(
		InputFormInitialValues{
			ProjectName:              initialState.ProjectName,
			BlueprintFormat:          initialState.BlueprintFormat,
			IsDefaultBlueprintFormat: initialState.IsDefaultBlueprintFormat,
			NoGit:                    initialState.NoGit,
			IsDefaultNoGit:           initialState.IsDefaultNoGit,
		},
		bluelinkStyles,
	)

	inputDirectory := NewInputDirectoryModel(
		InputDirectoryInitialValues{
			Directory: initialState.Directory,
		},
		initialState.ProjectName,
		bluelinkStyles,
	)

	downloadRepo := NewDownloadRepoModel(
		initialState.Template,
		initialState.Directory,
		bluelinkStyles,
		gitService,
	)

	stage := stageFromInitialState(initialState, headless)
	return &InitModel{
		stage:                   stage,
		selectTemplate:          selectTemplate,
		inputForm:               inputForm,
		inputDirectory:          inputDirectory,
		downloadRepo:            downloadRepo,
		template:                initialState.Template,
		selectedTemplate:        initialState.IsDefaultTemplate,
		projectName:             initialState.ProjectName,
		blueprintFormat:         initialState.BlueprintFormat,
		selectedBlueprintFormat: initialState.IsDefaultBlueprintFormat,
		noGit:                   initialState.NoGit,
		selectedNoGit:           initialState.IsDefaultNoGit,
		directory:               initialState.Directory,
		styles:                  bluelinkStyles,
		gitService:              gitService,
		preparer:                preparer,
		headless:                headless,
		headlessWriter:          headlessWriter,
	}, nil
}

func stageFromInitialState(initialState InitialState, headless bool) initStage {
	isTemplateSelected := (initialState.Template != "" && !initialState.IsDefaultTemplate)

	if !isTemplateSelected {
		return selectTemplateStage
	}

	isProjectNameSelected := strings.TrimSpace(initialState.ProjectName) != ""

	// In headless mode, accept default values for blueprint format and noGit
	// since they have sensible defaults (yaml and false respectively)
	isBlueprintFormatSelected := initialState.BlueprintFormat != "" && (!initialState.IsDefaultBlueprintFormat || headless)

	isNoGitSelected := initialState.NoGit != nil && (!initialState.IsDefaultNoGit || headless)

	if !isProjectNameSelected || !isBlueprintFormatSelected || !isNoGitSelected {
		return inputFormStage
	}

	if strings.TrimSpace(initialState.Directory) == "" {
		return inputDirectoryStage
	}

	// All inputs provided, start the download stage to actually initialize the project
	return downloadRepoStage
}
