package initui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	selectDirectoryStage

	// Stage where the user has completed the init process.
	initCompleteStage
)

type InitModel struct {
	stage                   initStage
	selectTemplate          tea.Model
	inputForm               tea.Model
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
}

func (m InitModel) Init() tea.Cmd {
	switch m.stage {
	case selectTemplateStage:
		return m.selectTemplate.Init()
	case inputFormStage:
		return m.inputForm.Init()
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
		m.stage = selectDirectoryStage
		// TODO: Initialize directory selection stage when implemented
		return m, nil
	}

	// Route updates to the current stage's model
	var cmd tea.Cmd
	switch m.stage {
	case selectTemplateStage:
		m.selectTemplate, cmd = m.selectTemplate.Update(msg)
	case inputFormStage:
		m.inputForm, cmd = m.inputForm.Update(msg)
	}
	return m, cmd
}

func (m InitModel) View() string {
	if m.quitting {
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("Not hungry? That's cool.")
	}

	switch m.stage {
	case selectTemplateStage:
		return "\n" + m.selectTemplate.View()
	case inputFormStage:
		return "\n" + m.inputForm.View()
	case selectDirectoryStage:
		// TODO: Implement directory selection view
		return "\n  Directory selection stage (not yet implemented)\n"
	case initCompleteStage:
		return "\n  Project initialization complete!\n"
	}

	return "\n"
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

	stage := stageFromInitialState(initialState)
	return &InitModel{
		stage:                   stage,
		selectTemplate:          selectTemplate,
		inputForm:               inputForm,
		template:                initialState.Template,
		selectedTemplate:        initialState.IsDefaultTemplate,
		projectName:             initialState.ProjectName,
		blueprintFormat:         initialState.BlueprintFormat,
		selectedBlueprintFormat: initialState.IsDefaultBlueprintFormat,
		noGit:                   initialState.NoGit,
		selectedNoGit:           initialState.IsDefaultNoGit,
		directory:               initialState.Directory,
	}, nil
}

func stageFromInitialState(initialState InitialState) initStage {
	isTemplateSelected := (initialState.Template != "" && !initialState.IsDefaultTemplate)

	if !isTemplateSelected {
		return selectTemplateStage
	}

	isProjectNameSelected := strings.TrimSpace(initialState.ProjectName) != ""

	isBlueprintFormatSelected := initialState.BlueprintFormat != "" && !initialState.IsDefaultBlueprintFormat

	isNoGitSelected := initialState.NoGit != nil && !initialState.IsDefaultNoGit

	if !isProjectNameSelected || !isBlueprintFormatSelected || !isNoGitSelected {
		return inputFormStage
	}

	if strings.TrimSpace(initialState.Directory) == "" {
		return selectDirectoryStage
	}

	return initCompleteStage
}
