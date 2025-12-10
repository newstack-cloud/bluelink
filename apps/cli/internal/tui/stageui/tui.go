package stageui

import (
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

type stageSessionState uint32

const (
	stageBlueprintSelect stageSessionState = iota
	stageView
)

// MainModel is the top-level model for the stage command TUI.
// It manages the session state and delegates to sub-models.
type MainModel struct {
	sessionState    stageSessionState
	blueprintFile   string
	quitting        bool
	selectBlueprint tea.Model
	stage           tea.Model
	styles          *stylespkg.Styles
	Error           error
}

func (m MainModel) Init() tea.Cmd {
	bpCmd := m.selectBlueprint.Init()
	stageCmd := m.stage.Init()
	return tea.Batch(bpCmd, stageCmd)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case sharedui.SelectBlueprintMsg:
		m.sessionState = stageView
		m.blueprintFile = sharedui.ToFullBlueprintPath(msg.BlueprintFile, msg.Source)
		var cmd tea.Cmd
		m.stage, cmd = m.stage.Update(msg)
		cmds = append(cmds, cmd)
	case sharedui.ClearSelectedBlueprintMsg:
		m.sessionState = stageBlueprintSelect
		m.blueprintFile = ""
	case tea.WindowSizeMsg:
		var bpCmd tea.Cmd
		m.selectBlueprint, bpCmd = m.selectBlueprint.Update(msg)
		var stageCmd tea.Cmd
		m.stage, stageCmd = m.stage.Update(msg)
		cmds = append(cmds, bpCmd, stageCmd)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			// Only quit if we're in the stage view and staging is finished
			if m.sessionState == stageView {
				stageModel, ok := m.stage.(StageModel)
				if ok && stageModel.finished {
					m.quitting = true
					return m, tea.Quit
				}
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.stage, cmd = m.stage.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.sessionState {
	case stageBlueprintSelect:
		newSelectBlueprint, newCmd := m.selectBlueprint.Update(msg)
		selectBlueprintModel, ok := newSelectBlueprint.(sharedui.SelectBlueprintModel)
		if !ok {
			panic("failed to perform assertion on select blueprint model in stage")
		}
		m.selectBlueprint = selectBlueprintModel
		cmds = append(cmds, newCmd)
	case stageView:
		newStage, newCmd := m.stage.Update(msg)
		stageModel, ok := newStage.(StageModel)
		if !ok {
			panic("failed to perform assertion on stage model")
		}
		m.stage = stageModel
		cmds = append(cmds, newCmd)
		if stageModel.err != nil {
			m.Error = stageModel.err
		}
	}
	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	if m.quitting {
		return quitTextStyle.Render("See you next time.")
	}
	if m.sessionState == stageBlueprintSelect {
		return m.selectBlueprint.View()
	}

	// Only show "You selected blueprint" during streaming, not in the finished split-pane view
	stageModel, ok := m.stage.(StageModel)
	if ok && stageModel.finished {
		return m.stage.View()
	}

	selected := "\n  You selected blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n"
	return selected + m.stage.View()
}

// NewStageApp creates a new stage application with the given configuration.
func NewStageApp(
	deployEngine engine.DeployEngine,
	logger *zap.Logger,
	blueprintFile string,
	isDefaultBlueprintFile bool,
	instanceID string,
	instanceName string,
	destroy bool,
	bluelinkStyles *stylespkg.Styles,
	headless bool,
	headlessWriter io.Writer,
) (*MainModel, error) {
	sessionState := stageBlueprintSelect
	// In headless mode, use the default blueprint file
	// if no explicit file is provided.
	autoStage := (blueprintFile != "" && !isDefaultBlueprintFile) || headless

	if autoStage {
		sessionState = stageView
	}

	fp, err := sharedui.BlueprintLocalFilePicker(bluelinkStyles)
	if err != nil {
		return nil, err
	}

	selectBlueprint, err := sharedui.NewSelectBlueprint(
		blueprintFile,
		autoStage,
		"stage",
		bluelinkStyles,
		&fp,
	)
	if err != nil {
		return nil, err
	}

	stage := NewStageModel(
		deployEngine,
		logger,
		instanceID,
		instanceName,
		destroy,
		bluelinkStyles,
		headless,
		headlessWriter,
	)

	return &MainModel{
		sessionState:    sessionState,
		blueprintFile:   blueprintFile,
		selectBlueprint: selectBlueprint,
		stage:           stage,
		styles:          bluelinkStyles,
	}, nil
}
