package destroyui

import (
	"errors"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"go.uber.org/zap"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stageui"
)

var (
	quitTextStyle = stylespkg.NewDefaultStyles(
		stylespkg.NewBluelinkPalette(),
	).Muted.Margin(1, 0, 2, 4)
)

type destroySessionState uint32

const (
	// destroyBlueprintSelect - select blueprint file (only when --stage is set)
	destroyBlueprintSelect destroySessionState = iota
	// destroyConfigInput - combined config form for instance name, staging options
	destroyConfigInput
	// destroyStaging - run change staging with destroy=true (when --stage is set)
	destroyStaging
	// destroyExecute - main destroy view with split-pane from the start
	destroyExecute
)

// MainModel is the top-level model for the destroy command TUI.
// It manages the session state and delegates to sub-models.
type MainModel struct {
	sessionState destroySessionState
	quitting     bool

	// Sub-models
	selectBlueprint   tea.Model
	destroyConfigForm *DestroyConfigFormModel
	staging           *stageui.StageModel
	destroy           tea.Model

	// Config from flags
	changesetID        string
	instanceID         string
	instanceName       string
	blueprintFile      string
	blueprintSource    string
	isDefaultBlueprint bool
	force              bool
	stageFirst         bool
	autoApprove        bool
	skipPrompts        bool

	// Runtime state
	headless bool
	jsonMode bool
	engine   engine.DeployEngine
	logger   *zap.Logger

	styles *stylespkg.Styles
	Error  error
}

// Init initializes the main model.
func (m MainModel) Init() tea.Cmd {
	bpCmd := m.selectBlueprint.Init()
	configFormCmd := m.destroyConfigForm.Init()
	destroyCmd := m.destroy.Init()
	var stagingCmd tea.Cmd
	if m.staging != nil {
		stagingCmd = m.staging.Init()
	}

	cmds := []tea.Cmd{bpCmd, configFormCmd, destroyCmd, stagingCmd}

	// If starting in destroyExecute state (direct destroy with changeset), trigger destroy start
	if m.sessionState == destroyExecute {
		cmds = append(cmds, func() tea.Msg {
			return StartDestroyMsg{}
		})
	}

	return tea.Batch(cmds...)
}

// Update handles messages for the main model.
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case sharedui.SelectBlueprintMsg:
		return m.handleSelectBlueprint(msg)

	case DestroyConfigMsg:
		return m.handleDestroyConfig(msg)

	case InstanceResolvedMsg:
		m.staging.SetInstanceID(msg.InstanceID)
		m.staging.SetInstanceName(msg.InstanceName)
		cmds = append(cmds, m.staging.StartStaging())

	case stageui.StageCompleteMsg:
		return m.handleStageComplete(msg)

	case ConfirmDestroyMsg:
		return m.handleConfirmDestroy(msg)

	case StartDestroyMsg:
		var cmd tea.Cmd
		m.destroy, cmd = m.destroy.Update(msg)
		cmds = append(cmds, cmd)

	case sharedui.ClearSelectedBlueprintMsg:
		if m.sessionState == destroyStaging || m.sessionState == destroyExecute {
			return m, tea.Batch(cmds...)
		}
		m.sessionState = destroyBlueprintSelect
		m.blueprintFile = ""
		m.blueprintSource = ""

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.KeyMsg:
		newModel, cmd, handled := m.handleKeyMsg(msg)
		if handled {
			return newModel, cmd
		}
		cmds = append(cmds, cmd)

	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)

	case stageui.StageStartedMsg, stageui.StageEventMsg, stageui.StageErrorMsg:
		return m.handleStagingMessage(msg)

	case driftui.DriftDetectedMsg, driftui.ReconciliationCompleteMsg, driftui.ReconciliationErrorMsg:
		return m.handleDriftMessage(msg)
	}

	return m.handleSessionStateUpdate(msg, cmds)
}

func (m MainModel) handleSelectBlueprint(msg sharedui.SelectBlueprintMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	m.blueprintFile = sharedui.ToFullBlueprintPath(msg.BlueprintFile, msg.Source)
	m.blueprintSource = msg.Source

	if m.sessionState == destroyExecute {
		var cmd tea.Cmd
		m.destroy, cmd = m.destroy.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	nextState := m.determineNextStateAfterBlueprintSelect()
	m.sessionState = nextState

	switch nextState {
	case destroyStaging:
		m.staging.SetBlueprintFile(m.blueprintFile)
		m.staging.SetBlueprintSource(m.blueprintSource)
		cmds = append(cmds, resolveInstanceIdentifiersCmd(m))
	case destroyExecute:
		var cmd tea.Cmd
		m.destroy, cmd = m.destroy.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleDestroyConfig(msg DestroyConfigMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	m.instanceName = msg.InstanceName
	m.instanceID = msg.InstanceID
	m.changesetID = msg.ChangesetID
	m.stageFirst = msg.StageFirst
	m.autoApprove = msg.AutoApprove

	destroyModel, ok := m.destroy.(DestroyModel)
	if ok {
		destroyModel.instanceName = msg.InstanceName
		destroyModel.instanceID = msg.InstanceID
		destroyModel.changesetID = msg.ChangesetID
		destroyModel.footerRenderer.InstanceName = msg.InstanceName
		destroyModel.footerRenderer.InstanceID = msg.InstanceID
		destroyModel.footerRenderer.ChangesetID = msg.ChangesetID
		m.destroy = destroyModel
	}

	if m.staging != nil {
		m.staging.SetInstanceName(msg.InstanceName)
		m.staging.SetInstanceID(msg.InstanceID)
	}

	if msg.StageFirst {
		m.sessionState = destroyStaging
		m.staging.SetBlueprintFile(m.blueprintFile)
		m.staging.SetBlueprintSource(m.blueprintSource)
		cmds = append(cmds, m.staging.Init())
		cmds = append(cmds, resolveInstanceIdentifiersCmd(m))
	} else {
		m.sessionState = destroyExecute
		cmds = append(cmds, func() tea.Msg {
			return StartDestroyMsg{}
		})
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleStageComplete(msg stageui.StageCompleteMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	if m.sessionState == destroyExecute {
		return m, tea.Batch(cmds...)
	}

	m.changesetID = msg.ChangesetID

	destroyModel, ok := m.destroy.(DestroyModel)
	if ok {
		destroyModel.changesetID = msg.ChangesetID
		destroyModel.footerRenderer.ChangesetID = msg.ChangesetID
		destroyModel.SetPreDestroyInstanceState(msg.InstanceState)
		destroyModel.SetChangesetChanges(msg.Changes)
		m.destroy = destroyModel
	}

	if m.autoApprove || m.headless {
		m.sessionState = destroyExecute
		cmds = append(cmds, m.triggerDestroyWithChangeset(msg.ChangesetID, msg.Changes))
	} else {
		create, update, del, recreate := m.staging.CountChangeSummary()
		m.staging.SetFooterRenderer(&DestroyStagingFooterRenderer{
			ChangesetID: msg.ChangesetID,
			Summary: ChangeSummary{
				Create:   create,
				Update:   update,
				Delete:   del,
				Recreate: recreate,
			},
			HasExportChanges: stageui.HasAnyExportChanges(msg.Changes),
		})
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleConfirmDestroy(msg ConfirmDestroyMsg) (tea.Model, tea.Cmd) {
	if msg.Confirmed {
		m.sessionState = destroyExecute
		return m, m.triggerDestroyWithChangeset(m.changesetID, nil)
	}
	m.quitting = true
	return m, tea.Quit
}

func (m MainModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	var bpCmd tea.Cmd
	m.selectBlueprint, bpCmd = m.selectBlueprint.Update(msg)
	cmds = append(cmds, bpCmd)

	var configFormCmd tea.Cmd
	var configFormModel tea.Model
	configFormModel, configFormCmd = m.destroyConfigForm.Update(msg)
	if cfm, ok := configFormModel.(DestroyConfigFormModel); ok {
		m.destroyConfigForm = &cfm
	}
	cmds = append(cmds, configFormCmd)

	var destroyCmd tea.Cmd
	m.destroy, destroyCmd = m.destroy.Update(msg)
	cmds = append(cmds, destroyCmd)

	if m.staging != nil {
		var stagingModel tea.Model
		var stagingCmd tea.Cmd
		stagingModel, stagingCmd = m.staging.Update(msg)
		if sm, ok := stagingModel.(stageui.StageModel); ok {
			m.staging = &sm
		}
		cmds = append(cmds, stagingCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit, true
	case "y", "Y":
		if m.sessionState == destroyStaging && m.staging != nil && m.staging.IsFinished() && !m.autoApprove {
			return m, func() tea.Msg {
				return ConfirmDestroyMsg{Confirmed: true}
			}, true
		}
	case "n", "N":
		if m.sessionState == destroyStaging && m.staging != nil && m.staging.IsFinished() && !m.autoApprove {
			return m, func() tea.Msg {
				return ConfirmDestroyMsg{Confirmed: false}
			}, true
		}
	case "q":
		if m.sessionState == destroyExecute {
			destroyModel, ok := m.destroy.(DestroyModel)
			if ok && destroyModel.finished {
				m.quitting = true
				return m, tea.Quit, true
			}
		}
		if m.sessionState == destroyStaging && m.staging != nil && m.staging.GetError() != nil {
			m.quitting = true
			return m, tea.Quit, true
		}
	}

	return m, nil, false
}

func (m MainModel) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	var cmd tea.Cmd
	m.destroy, cmd = m.destroy.Update(msg)
	cmds = append(cmds, cmd)

	if m.staging != nil && m.sessionState == destroyStaging {
		var stagingModel tea.Model
		var stagingCmd tea.Cmd
		stagingModel, stagingCmd = m.staging.Update(msg)
		if sm, ok := stagingModel.(stageui.StageModel); ok {
			m.staging = &sm
		}
		cmds = append(cmds, stagingCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleStagingMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	if m.staging != nil {
		var stagingModel tea.Model
		var stagingCmd tea.Cmd
		stagingModel, stagingCmd = m.staging.Update(msg)
		if sm, ok := stagingModel.(stageui.StageModel); ok {
			m.staging = &sm
		}
		cmds = append(cmds, stagingCmd)
		if m.staging.GetError() != nil {
			m.Error = m.staging.GetError()
		}
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleDriftMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	if m.sessionState == destroyExecute {
		var cmd tea.Cmd
		m.destroy, cmd = m.destroy.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.staging != nil {
		var stagingModel tea.Model
		var stagingCmd tea.Cmd
		stagingModel, stagingCmd = m.staging.Update(msg)
		if sm, ok := stagingModel.(stageui.StageModel); ok {
			m.staging = &sm
		}
		cmds = append(cmds, stagingCmd)
		if m.staging.GetError() != nil {
			m.Error = m.staging.GetError()
		}
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) handleSessionStateUpdate(msg tea.Msg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch m.sessionState {
	case destroyBlueprintSelect:
		newSelectBlueprint, newCmd := m.selectBlueprint.Update(msg)
		selectBlueprintModel, ok := newSelectBlueprint.(sharedui.SelectBlueprintModel)
		if !ok {
			panic("failed to perform assertion on select blueprint model in destroy")
		}
		m.selectBlueprint = selectBlueprintModel
		cmds = append(cmds, newCmd)

	case destroyStaging:
		switch msg.(type) {
		case stageui.StageStartedMsg, stageui.StageEventMsg, stageui.StageErrorMsg, stageui.StageCompleteMsg,
			driftui.DriftDetectedMsg, driftui.ReconciliationCompleteMsg, driftui.ReconciliationErrorMsg,
			spinner.TickMsg:
			// Already handled above
		default:
			if m.staging != nil {
				var stagingModel tea.Model
				var stagingCmd tea.Cmd
				stagingModel, stagingCmd = m.staging.Update(msg)
				if sm, ok := stagingModel.(stageui.StageModel); ok {
					m.staging = &sm
				}
				cmds = append(cmds, stagingCmd)
			}
		}

	case destroyConfigInput:
		var configFormModel tea.Model
		var configFormCmd tea.Cmd
		configFormModel, configFormCmd = m.destroyConfigForm.Update(msg)
		if cfm, ok := configFormModel.(DestroyConfigFormModel); ok {
			m.destroyConfigForm = &cfm
		}
		cmds = append(cmds, configFormCmd)

	case destroyExecute:
		switch msg.(type) {
		case driftui.DriftDetectedMsg, driftui.ReconciliationCompleteMsg, driftui.ReconciliationErrorMsg:
			// Already handled above
		default:
			newDestroy, newCmd := m.destroy.Update(msg)
			destroyModel, ok := newDestroy.(DestroyModel)
			if !ok {
				panic("failed to perform assertion on destroy model")
			}
			m.destroy = destroyModel
			cmds = append(cmds, newCmd)
			if destroyModel.err != nil {
				m.Error = destroyModel.err
			} else if destroyModel.finished && IsFailedStatus(destroyModel.finalStatus) {
				m.Error = errors.New("destroy failed with status: " + destroyModel.finalStatus.String())
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// triggerDestroyWithChangeset starts destroy with the given changeset.
func (m *MainModel) triggerDestroyWithChangeset(_ string, _ *changes.BlueprintChanges) tea.Cmd {
	return func() tea.Msg {
		return StartDestroyMsg{}
	}
}

// determineNextStateAfterBlueprintSelect determines the next session state
// after a blueprint has been selected.
func (m *MainModel) determineNextStateAfterBlueprintSelect() destroySessionState {
	instanceIdentified := m.instanceID != "" || m.instanceName != ""
	hasDestroyPath := m.stageFirst || m.changesetID != ""
	canSkipForm := m.skipPrompts && instanceIdentified && hasDestroyPath

	if m.headless || canSkipForm {
		if m.stageFirst {
			return destroyStaging
		}
		return destroyExecute
	}

	return destroyConfigInput
}

// View renders the main model.
func (m MainModel) View() string {
	if m.quitting {
		return quitTextStyle.Render("See you next time.")
	}

	switch m.sessionState {
	case destroyBlueprintSelect:
		return m.selectBlueprint.View()
	case destroyConfigInput:
		selected := "\n  Blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n"
		return selected + m.destroyConfigForm.View()
	case destroyStaging:
		selected := "\n  Blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n"
		if m.instanceName != "" {
			selected += "  Instance: " + m.styles.Selected.Render(m.instanceName) + "\n"
		}
		if m.staging != nil {
			return selected + m.staging.View()
		}
		return selected
	default:
		return m.destroy.View()
	}
}

// NewDestroyApp creates a new destroy application with the given configuration.
func NewDestroyApp(
	destroyEngine engine.DeployEngine,
	logger *zap.Logger,
	changesetID string,
	instanceID string,
	instanceName string,
	blueprintFile string,
	isDefaultBlueprintFile bool,
	force bool,
	stageFirst bool,
	autoApprove bool,
	skipPrompts bool,
	bluelinkStyles *stylespkg.Styles,
	headless bool,
	headlessWriter io.Writer,
	jsonMode bool,
) (*MainModel, error) {
	sessionState := determineInitialSessionState(
		blueprintFile, isDefaultBlueprintFile, instanceID, instanceName,
		changesetID, stageFirst, skipPrompts, headless,
	)

	autoSelect := shouldAutoSelect(blueprintFile, isDefaultBlueprintFile, headless, skipPrompts, stageFirst)

	fp, err := sharedui.BlueprintLocalFilePicker(bluelinkStyles)
	if err != nil {
		return nil, err
	}

	selectBlueprint, err := sharedui.NewSelectBlueprint(
		blueprintFile,
		autoSelect,
		"destroy",
		bluelinkStyles,
		&fp,
	)
	if err != nil {
		return nil, err
	}

	destroyConfigForm := NewDestroyConfigFormModel(
		DestroyConfigFormInitialValues{
			InstanceName: instanceName,
			InstanceID:   instanceID,
			ChangesetID:  changesetID,
			StageFirst:   stageFirst,
			AutoApprove:  autoApprove,
		},
		bluelinkStyles,
	)

	stagingModel := stageui.NewStageModel(
		destroyEngine,
		logger,
		instanceID,
		instanceName,
		true,  // destroy = true for staging destroy changes
		force, // skipDriftCheck - use force flag to skip drift detection during staging
		bluelinkStyles,
		headless,
		headlessWriter,
		jsonMode,
	)
	staging := &stagingModel
	staging.SetBlueprintFile(blueprintFile)
	blueprintSource := blueprintSourceFromPath(blueprintFile)
	staging.SetBlueprintSource(blueprintSource)
	staging.SetDeployFlowMode(true)

	destroy := NewDestroyModel(
		destroyEngine,
		logger,
		changesetID,
		instanceID,
		instanceName,
		force,
		bluelinkStyles,
		headless,
		headlessWriter,
		nil,
		jsonMode,
	)

	return &MainModel{
		sessionState:       sessionState,
		selectBlueprint:    selectBlueprint,
		destroyConfigForm:  destroyConfigForm,
		staging:            staging,
		destroy:            destroy,
		changesetID:        changesetID,
		instanceID:         instanceID,
		instanceName:       instanceName,
		blueprintFile:      blueprintFile,
		blueprintSource:    blueprintSource,
		isDefaultBlueprint: isDefaultBlueprintFile,
		force:              force,
		stageFirst:         stageFirst,
		autoApprove:        autoApprove,
		skipPrompts:        skipPrompts,
		headless:           headless,
		jsonMode:           jsonMode,
		engine:             destroyEngine,
		logger:             logger,
		styles:             bluelinkStyles,
	}, nil
}

func determineInitialSessionState(
	blueprintFile string,
	isDefaultBlueprintFile bool,
	instanceID, instanceName, changesetID string,
	stageFirst, skipPrompts, headless bool,
) destroySessionState {
	instanceIdentified := instanceID != "" || instanceName != ""
	hasDestroyPath := stageFirst || changesetID != ""
	canSkipForm := skipPrompts && instanceIdentified && hasDestroyPath

	// If staging is required, we need blueprint selection first
	if stageFirst {
		autoSelect := (blueprintFile != "" && !isDefaultBlueprintFile) || headless || skipPrompts
		if !autoSelect {
			return destroyBlueprintSelect
		}
		if headless || canSkipForm {
			return destroyStaging
		}
		return destroyConfigInput
	}

	// Not staging - using existing changeset
	if headless || canSkipForm {
		return destroyExecute
	}
	return destroyConfigInput
}

func shouldAutoSelect(
	blueprintFile string,
	isDefaultBlueprintFile bool,
	headless, skipPrompts, stageFirst bool,
) bool {
	// Blueprint file is only relevant when staging
	if !stageFirst {
		return true // Skip blueprint selection when not staging
	}
	return (blueprintFile != "" && !isDefaultBlueprintFile) || headless || skipPrompts
}

func blueprintSourceFromPath(blueprintFile string) string {
	if strings.HasPrefix(blueprintFile, "https://") {
		return consts.BlueprintSourceHTTPS
	}
	if strings.HasPrefix(blueprintFile, "s3://") {
		return consts.BlueprintSourceS3
	}
	if strings.HasPrefix(blueprintFile, "gcs://") {
		return consts.BlueprintSourceGCS
	}
	if strings.HasPrefix(blueprintFile, "azureblob://") {
		return consts.BlueprintSourceAzureBlob
	}
	return consts.BlueprintSourceFile
}
