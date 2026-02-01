package stageui

import (
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/preflightui"
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
	stagePreflight       stageSessionState = iota
	stageBlueprintSelect
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
	preflight        *preflightui.PreflightModel
	styles           *stylespkg.Styles
	Error            error
	// needsOptionsInput tracks whether we should prompt for stage options
	needsOptionsInput bool
	// autoStage tracks the original auto-stage state for post-preflight transition
	autoStage bool
	// restartInstructions stores engine restart info after plugin installation
	restartInstructions    string
	installedPlugins       []string
	preflightCommandName   string
}

func (m MainModel) Init() tea.Cmd {
	if m.sessionState == stagePreflight && m.preflight != nil {
		return m.preflight.Init()
	}
	cmds := []tea.Cmd{m.selectBlueprint.Init(), m.stage.Init()}
	if m.stageOptionsForm != nil {
		cmds = append(cmds, m.stageOptionsForm.Init())
	}
	return tea.Batch(cmds...)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.sessionState == stagePreflight {
		return m.updatePreflight(msg)
	}
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

func (m MainModel) updatePreflight(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case preflightui.PreflightSatisfiedMsg:
		if m.autoStage {
			m.sessionState = stageView
		} else {
			m.sessionState = stageBlueprintSelect
		}
		cmds := []tea.Cmd{m.selectBlueprint.Init(), m.stage.Init()}
		if m.stageOptionsForm != nil {
			cmds = append(cmds, m.stageOptionsForm.Init())
		}
		return m, tea.Batch(cmds...)
	case preflightui.PreflightInstalledMsg:
		m.restartInstructions = msg.RestartInstructions
		m.installedPlugins = msg.InstalledPlugins
		m.preflightCommandName = msg.CommandName
		m.quitting = true
		return m, tea.Quit
	case preflightui.PreflightErrorMsg:
		m.Error = msg.Err
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	if m.preflight != nil {
		updated, cmd := m.preflight.Update(msg)
		m.preflight = &updated
		return m, cmd
	}
	return m, nil
}

func (m MainModel) View() string {
	if m.quitting {
		if m.restartInstructions != "" {
			return preflightui.RenderInstallSummary(
				m.styles, m.installedPlugins, len(m.installedPlugins),
				m.restartInstructions, m.preflightCommandName,
			)
		}
		return quitTextStyle.Render("See you next time.")
	}
	if m.sessionState == stagePreflight {
		if m.preflight != nil {
			return m.preflight.View()
		}
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
	preflight *preflightui.PreflightModel,
) (*MainModel, error) {
	// Auto-stage when:
	// 1. A non-default blueprint file is provided, OR
	// 2. An instance identifier is provided (staging for existing instance), OR
	// 3. Running in headless mode
	hasInstanceIdentifier := instanceID != "" || instanceName != ""
	autoStage := (blueprintFile != "" && !isDefaultBlueprintFile) || hasInstanceIdentifier || headless

	sessionState := stageBlueprintSelect
	if autoStage {
		sessionState = stageView
	}
	if preflight != nil {
		sessionState = stagePreflight
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
		preflight:         preflight,
		styles:            bluelinkStyles,
		needsOptionsInput: needsOptionsInput,
		autoStage:         autoStage,
	}, nil
}

// Test accessor methods - these provide read-only access for testing purposes.

// Stage returns the stage model (as tea.Model interface).
func (m *MainModel) Stage() tea.Model {
	return m.stage
}

// StageOptionsForm returns the stage options form model.
func (m *MainModel) StageOptionsForm() *StageOptionsFormModel {
	return m.stageOptionsForm
}

// NeedsOptionsInput returns whether the model requires options input.
func (m *MainModel) NeedsOptionsInput() bool {
	return m.needsOptionsInput
}
