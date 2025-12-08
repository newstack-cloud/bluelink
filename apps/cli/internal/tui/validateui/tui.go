package validateui

import (
	"errors"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"go.uber.org/zap"
)

var (
	quitTextStyle = stylespkg.NewDefaultStyles(
		stylespkg.NewBluelinkPalette(),
	).Muted.Margin(1, 0, 2, 4)
)

// ValidateStage is an enum that represents the different stages
// of the validation process.
type ValidateStage int

const (
	// ValidateStageConfigStructure is the stage where application configuration
	// and project structure is validated.
	ValidateStageConfigStructure ValidateStage = iota
	// ValidateStageBlueprint is the stage where the blueprint is validated.
	ValidateStageBlueprint
	// ValidateStageSourceCode is the stage where the source code of the
	// application is validated.
	ValidateStageSourceCode
)

type validateSessionState uint32

const (
	validateBlueprintSelect validateSessionState = iota
	validateView
)

type MainModel struct {
	sessionState validateSessionState
	// validateStage   ValidateStage
	blueprintFile   string
	quitting        bool
	selectBlueprint tea.Model
	validate        tea.Model
	styles          *stylespkg.Styles
	Error           error
}

func (m MainModel) Init() tea.Cmd {
	bpCmd := m.selectBlueprint.Init()
	validateCmd := m.validate.Init()
	return tea.Batch(bpCmd, validateCmd)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case sharedui.SelectBlueprintMsg:
		m.sessionState = validateView
		m.blueprintFile = sharedui.ToFullBlueprintPath(msg.BlueprintFile, msg.Source)
		var cmd tea.Cmd
		m.validate, cmd = m.validate.Update(msg)
		cmds = append(cmds, cmd)
	case sharedui.ClearSelectedBlueprintMsg:
		m.sessionState = validateBlueprintSelect
		m.blueprintFile = ""
	case tea.WindowSizeMsg:
		var bpCmd tea.Cmd
		m.selectBlueprint, bpCmd = m.selectBlueprint.Update(msg)
		var validateCmd tea.Cmd
		m.validate, validateCmd = m.validate.Update(msg)
		cmds = append(cmds, bpCmd, validateCmd)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.validate, cmd = m.validate.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.sessionState {
	case validateBlueprintSelect:
		newSelectBlueprint, newCmd := m.selectBlueprint.Update(msg)
		selectBlueprintModel, ok := newSelectBlueprint.(sharedui.SelectBlueprintModel)
		if !ok {
			panic("failed to perform assertion on select blueprint model in validate")
		}
		m.selectBlueprint = selectBlueprintModel
		cmds = append(cmds, newCmd)
	case validateView:
		newValidate, newCmd := m.validate.Update(msg)
		validateModel, ok := newValidate.(ValidateModel)
		if !ok {
			panic("failed to perform assertion on validate model")
		}
		m.validate = validateModel
		cmds = append(cmds, newCmd)
		if validateModel.err != nil {
			m.Error = validateModel.err
		}
		if validateModel.validationFailed {
			m.Error = errors.New("validation failed")
		}
	}
	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	if m.quitting {
		return quitTextStyle.Render("Had enough? See you next time.")
	}
	if m.sessionState == validateBlueprintSelect {
		return m.selectBlueprint.View()
	}

	selected := "\n  You selected blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n"
	return selected + m.validate.View()
}

func NewValidateApp(
	engine engine.DeployEngine,
	logger *zap.Logger,
	blueprintFile string,
	isDefaultBlueprintFile bool,
	bluelinkStyles *stylespkg.Styles,
	headless bool,
	headlessWriter io.Writer,
) (*MainModel, error) {
	sessionState := validateBlueprintSelect
	// In headless mode, use the default blueprint file
	// if no explicit file is provided.
	autoValidate := (blueprintFile != "" && !isDefaultBlueprintFile) || headless

	if autoValidate {
		sessionState = validateView
	}

	fp, err := sharedui.BlueprintLocalFilePicker(bluelinkStyles)
	if err != nil {
		return nil, err
	}

	selectBlueprint, err := sharedui.NewSelectBlueprint(
		blueprintFile,
		autoValidate,
		"validate",
		bluelinkStyles,
		&fp,
	)
	if err != nil {
		return nil, err
	}
	validate := NewValidateModel(engine, logger, headless, headlessWriter)
	return &MainModel{
		sessionState:    sessionState,
		blueprintFile:   blueprintFile,
		selectBlueprint: selectBlueprint,
		validate:        validate,
		styles:          bluelinkStyles,
	}, nil
}
