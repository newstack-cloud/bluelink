package stageui

import (
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
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
	stageOptionsInput
	stageView
)

// MainModel is the top-level model for the stage command TUI.
// It manages the session state and delegates to sub-models.
type MainModel struct {
	sessionState     stageSessionState
	blueprintFile    string
	quitting         bool
	selectBlueprint  tea.Model
	stageOptionsForm *StageOptionsFormModel
	stage            tea.Model
	styles           *stylespkg.Styles
	Error            error
	// needsOptionsInput tracks whether we should prompt for stage options
	needsOptionsInput bool
}

func (m MainModel) Init() tea.Cmd {
	bpCmd := m.selectBlueprint.Init()
	stageCmd := m.stage.Init()
	var optionsCmd tea.Cmd
	if m.stageOptionsForm != nil {
		optionsCmd = m.stageOptionsForm.Init()
	}
	return tea.Batch(bpCmd, stageCmd, optionsCmd)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case sharedui.SelectBlueprintMsg:
		m.blueprintFile = sharedui.ToFullBlueprintPath(msg.BlueprintFile, msg.Source)
		// If we need options input, go to that state first
		if m.needsOptionsInput {
			m.sessionState = stageOptionsInput
		} else {
			m.sessionState = stageView
			var cmd tea.Cmd
			m.stage, cmd = m.stage.Update(msg)
			cmds = append(cmds, cmd)
		}
	case StageOptionsSelectedMsg:
		// Options provided, now proceed to staging
		m.sessionState = stageView
		// Update the stage model with the selected options
		stageModel := m.stage.(StageModel)
		stageModel.SetInstanceName(msg.InstanceName)
		stageModel.SetDestroy(msg.Destroy)
		stageModel.SetSkipDriftCheck(msg.SkipDriftCheck)
		m.stage = stageModel
		// Send the blueprint selection to the stage model to start staging
		var cmd tea.Cmd
		m.stage, cmd = m.stage.Update(sharedui.SelectBlueprintMsg{
			BlueprintFile: m.blueprintFile,
			Source:        consts.BlueprintSourceFile,
		})
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
	case stageOptionsInput:
		if m.stageOptionsForm != nil {
			var cmd tea.Cmd
			m.stageOptionsForm, cmd = m.stageOptionsForm.Update(msg)
			cmds = append(cmds, cmd)
		}
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
	if m.sessionState == stageOptionsInput {
		selected := "\n  You selected blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n\n"
		if m.stageOptionsForm != nil {
			return selected + m.stageOptionsForm.View()
		}
		return selected
	}

	// Only show "You selected blueprint" during streaming, not in split-pane views
	// (finished staging view, drift review mode, exports view, overview)
	stageModel, ok := m.stage.(StageModel)
	if ok && (stageModel.finished || stageModel.driftReviewMode || stageModel.showingExportsView || stageModel.showingOverview) {
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
	skipDriftCheck bool,
	bluelinkStyles *stylespkg.Styles,
	headless bool,
	headlessWriter io.Writer,
	jsonMode bool,
) (*MainModel, error) {
	sessionState := stageBlueprintSelect
	// Auto-stage when:
	// 1. A non-default blueprint file is provided, OR
	// 2. An instance identifier is provided (staging for existing instance), OR
	// 3. Running in headless mode
	hasInstanceIdentifier := instanceID != "" || instanceName != ""
	autoStage := (blueprintFile != "" && !isDefaultBlueprintFile) || hasInstanceIdentifier || headless

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
		skipDriftCheck,
		bluelinkStyles,
		headless,
		headlessWriter,
		jsonMode,
	)

	// Determine if we need to prompt for stage options
	// We need options input if:
	// 1. Not headless mode (interactive)
	// 2. No instance ID or instance name provided
	// This allows users to configure instance name, destroy mode, and skip drift check interactively.
	needsOptionsInput := !headless && instanceID == "" && instanceName == ""

	var stageOptionsForm *StageOptionsFormModel
	if needsOptionsInput {
		stageOptionsForm = NewStageOptionsFormModel(bluelinkStyles, StageOptionsFormConfig{
			InitialInstanceName:   instanceName,
			InitialDestroy:        destroy,
			InitialSkipDriftCheck: skipDriftCheck,
			Engine:                deployEngine,
		})
	}

	return &MainModel{
		sessionState:      sessionState,
		blueprintFile:     blueprintFile,
		selectBlueprint:   selectBlueprint,
		stageOptionsForm:  stageOptionsForm,
		stage:             stage,
		styles:            bluelinkStyles,
		needsOptionsInput: needsOptionsInput,
	}, nil
}
