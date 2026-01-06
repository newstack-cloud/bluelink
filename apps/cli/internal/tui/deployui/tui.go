package deployui

import (
	"errors"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"go.uber.org/zap"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stageui"
)

var (
	quitTextStyle = stylespkg.NewDefaultStyles(
		stylespkg.NewBluelinkPalette(),
	).Muted.Margin(1, 0, 2, 4)
)

type deploySessionState uint32

const (
	// deployBlueprintSelect - select blueprint file if not provided
	deployBlueprintSelect deploySessionState = iota
	// deployConfigInput - combined config form for instance name, rollback, staging options
	deployConfigInput
	// deployStaging - run change staging (when --stage is set)
	deployStaging
	// deployExecute - main deployment view with split-pane from the start
	deployExecute
)

// MainModel is the top-level model for the deploy command TUI.
// It manages the session state and delegates to sub-models.
type MainModel struct {
	sessionState deploySessionState
	quitting     bool

	// Sub-models
	selectBlueprint  tea.Model
	deployConfigForm *DeployConfigFormModel
	staging          *stageui.StageModel
	deploy           tea.Model

	// Config from flags
	changesetID        string
	instanceID         string
	instanceName       string
	blueprintFile      string
	blueprintSource    string
	isDefaultBlueprint bool
	autoRollback       bool
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

// GetInstanceID returns the instance ID for the InstanceResolver interface.
func (m MainModel) GetInstanceID() string { return m.instanceID }

// GetInstanceName returns the instance name for the InstanceResolver interface.
func (m MainModel) GetInstanceName() string { return m.instanceName }

// GetEngine returns the engine for the InstanceResolver interface.
func (m MainModel) GetEngine() shared.InstanceLookup { return m.engine }

// Init initializes the main model.
func (m MainModel) Init() tea.Cmd {
	bpCmd := m.selectBlueprint.Init()
	configFormCmd := m.deployConfigForm.Init()
	deployCmd := m.deploy.Init()
	var stagingCmd tea.Cmd
	if m.staging != nil {
		stagingCmd = m.staging.Init()
	}
	return tea.Batch(bpCmd, configFormCmd, deployCmd, stagingCmd)
}

// Update handles messages for the main model.
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case sharedui.SelectBlueprintMsg:
		m.blueprintFile = sharedui.ToFullBlueprintPath(msg.BlueprintFile, msg.Source)
		m.blueprintSource = msg.Source

		// If we're already in deployExecute state, just pass the message to the deploy model
		// without recalculating state. This prevents switching back to staging when
		// SelectBlueprintMsg is received during deployment.
		if m.sessionState == deployExecute {
			var cmd tea.Cmd
			m.deploy, cmd = m.deploy.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// Determine next state based on whether we need instance name
		nextState := m.determineNextStateAfterBlueprintSelect()
		m.sessionState = nextState

		switch nextState {
		case deployStaging:
			// Resolve instance identifiers before starting staging
			// This handles the case where instance name is provided but instance doesn't exist yet
			m.staging.SetBlueprintFile(m.blueprintFile)
			m.staging.SetBlueprintSource(m.blueprintSource)
			cmds = append(cmds, resolveInstanceIdentifiersCmd(m))
		case deployExecute:
			// Pass the message to the deploy model to start deployment
			var cmd tea.Cmd
			m.deploy, cmd = m.deploy.Update(msg)
			cmds = append(cmds, cmd)
		}

	case DeployConfigMsg:
		// Update config from form
		m.instanceName = msg.InstanceName
		m.instanceID = msg.InstanceID
		m.changesetID = msg.ChangesetID
		m.stageFirst = msg.StageFirst
		m.autoApprove = msg.AutoApprove
		m.autoRollback = msg.AutoRollback

		// Update deploy model with the new values
		deployModel, ok := m.deploy.(DeployModel)
		if ok {
			deployModel.instanceName = msg.InstanceName
			deployModel.instanceID = msg.InstanceID
			deployModel.changesetID = msg.ChangesetID
			deployModel.autoRollback = msg.AutoRollback
			deployModel.footerRenderer.InstanceName = msg.InstanceName
			deployModel.footerRenderer.InstanceID = msg.InstanceID
			deployModel.footerRenderer.ChangesetID = msg.ChangesetID
			m.deploy = deployModel
		}

		// Update staging model
		if m.staging != nil {
			m.staging.SetInstanceName(msg.InstanceName)
			m.staging.SetInstanceID(msg.InstanceID)
		}

		// Determine next state - staging if stageFirst is set, otherwise deploy
		if msg.StageFirst {
			m.sessionState = deployStaging
			m.staging.SetBlueprintFile(m.blueprintFile)
			m.staging.SetBlueprintSource(m.blueprintSource)
			// Start the staging spinner animation
			cmds = append(cmds, m.staging.Init())
			// Resolve instance identifiers before starting staging
			// This handles the case where instance name is provided but instance doesn't exist yet
			cmds = append(cmds, resolveInstanceIdentifiersCmd(m))
		} else {
			m.sessionState = deployExecute
			// Trigger deployment start with the provided changeset ID
			cmds = append(cmds, func() tea.Msg {
				return sharedui.SelectBlueprintMsg{
					BlueprintFile: m.blueprintFile,
					Source:        m.blueprintSource,
				}
			})
		}

	case InstanceResolvedMsg:
		// Instance identifiers have been resolved - set them on staging model and start staging
		m.staging.SetInstanceID(msg.InstanceID)
		m.staging.SetInstanceName(msg.InstanceName)
		cmds = append(cmds, m.staging.StartStaging())

	case stageui.StageCompleteMsg:
		// Guard: If we're already in deployExecute, ignore this message.
		// This can happen if a stale StageCompleteMsg arrives after deployment has started.
		if m.sessionState == deployExecute {
			return m, tea.Batch(cmds...)
		}

		m.changesetID = msg.ChangesetID
		// Update deploy model with changeset ID, changes, instance state, and blueprint info
		deployModel, ok := m.deploy.(DeployModel)
		if ok {
			deployModel.changesetID = msg.ChangesetID
			deployModel.footerRenderer.ChangesetID = msg.ChangesetID
			// Propagate blueprint file and source from main model to ensure remote sources work
			deployModel.blueprintFile = m.blueprintFile
			deployModel.blueprintSource = m.blueprintSource
			// Set instance state first so it's available when building items
			deployModel.SetPreDeployInstanceState(msg.InstanceState)
			// Set changeset changes to build proper item hierarchy (uses instance state)
			deployModel.SetChangesetChanges(msg.Changes)
			m.deploy = deployModel
		}

		if m.autoApprove || m.headless {
			// Skip confirmation, go straight to deploy
			m.sessionState = deployExecute
			cmds = append(cmds, m.triggerDeploymentWithChangeset(msg.ChangesetID, msg.Changes))
		} else {
			// Stay in staging state but show confirmation footer
			// Set custom footer renderer for deploy flow confirmation
			create, update, del, recreate := m.staging.CountChangeSummary()
			m.staging.SetFooterRenderer(&DeployStagingFooterRenderer{
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

	case ConfirmDeployMsg:
		if msg.Confirmed {
			m.sessionState = deployExecute
			cmds = append(cmds, m.triggerDeploymentWithChangeset(m.changesetID, nil))
		} else {
			// User cancelled
			m.quitting = true
			return m, tea.Quit
		}

	case StartDeployMsg:
		// Forward StartDeployMsg to the deploy model to initiate deployment
		var cmd tea.Cmd
		m.deploy, cmd = m.deploy.Update(msg)
		cmds = append(cmds, cmd)

	case sharedui.ClearSelectedBlueprintMsg:
		// Guard: Don't reset state if we're in deployStaging or deployExecute.
		// This prevents unexpected state resets during active operations.
		if m.sessionState == deployStaging || m.sessionState == deployExecute {
			return m, tea.Batch(cmds...)
		}
		m.sessionState = deployBlueprintSelect
		m.blueprintFile = ""
		m.blueprintSource = ""

	case tea.WindowSizeMsg:
		var bpCmd tea.Cmd
		m.selectBlueprint, bpCmd = m.selectBlueprint.Update(msg)
		var configFormCmd tea.Cmd
		var configFormModel tea.Model
		configFormModel, configFormCmd = m.deployConfigForm.Update(msg)
		if cfm, ok := configFormModel.(DeployConfigFormModel); ok {
			m.deployConfigForm = &cfm
		}
		var deployCmd tea.Cmd
		m.deploy, deployCmd = m.deploy.Update(msg)
		cmds = append(cmds, bpCmd, configFormCmd, deployCmd)
		if m.staging != nil {
			var stagingModel tea.Model
			var stagingCmd tea.Cmd
			stagingModel, stagingCmd = m.staging.Update(msg)
			if sm, ok := stagingModel.(stageui.StageModel); ok {
				m.staging = &sm
			}
			cmds = append(cmds, stagingCmd)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "y", "Y":
			// Handle confirmation when staging is finished (deploy flow)
			if m.sessionState == deployStaging && m.staging != nil && m.staging.IsFinished() && !m.autoApprove {
				return m, func() tea.Msg {
					return ConfirmDeployMsg{Confirmed: true}
				}
			}
		case "n", "N":
			// Handle rejection when staging is finished (deploy flow)
			if m.sessionState == deployStaging && m.staging != nil && m.staging.IsFinished() && !m.autoApprove {
				return m, func() tea.Msg {
					return ConfirmDeployMsg{Confirmed: false}
				}
			}
		case "q":
			// Only quit if we're in the deploy view and deployment is finished
			if m.sessionState == deployExecute {
				deployModel, ok := m.deploy.(DeployModel)
				if ok && deployModel.finished {
					m.quitting = true
					return m, tea.Quit
				}
			}
			// Allow quit during staging error
			if m.sessionState == deployStaging && m.staging != nil && m.staging.GetError() != nil {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.deploy, cmd = m.deploy.Update(msg)
		cmds = append(cmds, cmd)
		if m.staging != nil && m.sessionState == deployStaging {
			var stagingModel tea.Model
			var stagingCmd tea.Cmd
			stagingModel, stagingCmd = m.staging.Update(msg)
			if sm, ok := stagingModel.(stageui.StageModel); ok {
				m.staging = &sm
			}
			cmds = append(cmds, stagingCmd)
		}

	case stageui.StageStartedMsg, stageui.StageEventMsg, stageui.StageErrorMsg:
		// Route staging-specific messages to staging model.
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

	case driftui.DriftDetectedMsg, driftui.ReconciliationCompleteMsg, driftui.ReconciliationErrorMsg:
		// Route drift messages based on current session state.
		// During deployment (deployExecute), drift is handled by the deploy model.
		// During staging (deployStaging), drift is handled by the staging model.
		if m.sessionState == deployExecute {
			var cmd tea.Cmd
			m.deploy, cmd = m.deploy.Update(msg)
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
	}

	switch m.sessionState {
	case deployBlueprintSelect:
		newSelectBlueprint, newCmd := m.selectBlueprint.Update(msg)
		selectBlueprintModel, ok := newSelectBlueprint.(sharedui.SelectBlueprintModel)
		if !ok {
			panic("failed to perform assertion on select blueprint model in deploy")
		}
		m.selectBlueprint = selectBlueprintModel
		cmds = append(cmds, newCmd)

	case deployStaging:
		// Route non-staging-specific messages (like KeyMsg, MouseMsg) to staging model.
		// Staging-specific messages, drift messages, and spinner ticks are already handled
		// in the type switch above, so skip them here to avoid duplicate processing.
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

	case deployConfigInput:
		var configFormModel tea.Model
		var configFormCmd tea.Cmd
		configFormModel, configFormCmd = m.deployConfigForm.Update(msg)
		if cfm, ok := configFormModel.(DeployConfigFormModel); ok {
			m.deployConfigForm = &cfm
		}
		cmds = append(cmds, configFormCmd)

	case deployExecute:
		// Skip drift messages since they're already handled in the type switch above.
		switch msg.(type) {
		case driftui.DriftDetectedMsg, driftui.ReconciliationCompleteMsg, driftui.ReconciliationErrorMsg:
			// Already handled above
		default:
			newDeploy, newCmd := m.deploy.Update(msg)
			deployModel, ok := newDeploy.(DeployModel)
			if !ok {
				panic("failed to perform assertion on deploy model")
			}
			m.deploy = deployModel
			cmds = append(cmds, newCmd)
			if deployModel.err != nil {
				m.Error = deployModel.err
			} else if deployModel.finished && IsFailedStatus(deployModel.finalStatus) {
				// Deployment completed with a failed status - set error for non-zero exit code
				m.Error = errors.New("deployment failed with status: " + deployModel.finalStatus.String())
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// triggerDeploymentWithChangeset starts deployment with the given changeset.
func (m *MainModel) triggerDeploymentWithChangeset(_ string, _ *changes.BlueprintChanges) tea.Cmd {
	return func() tea.Msg {
		return StartDeployMsg{}
	}
}

// determineNextStateAfterBlueprintSelect determines the next session state
// after a blueprint has been selected.
func (m *MainModel) determineNextStateAfterBlueprintSelect() deploySessionState {
	// Check if all required values are provided for skipping prompts
	instanceIdentified := m.instanceID != "" || m.instanceName != ""
	hasDeployPath := m.stageFirst || m.changesetID != ""
	canSkipForm := m.skipPrompts && instanceIdentified && hasDeployPath

	if m.headless || canSkipForm {
		// Skip config form - go straight to staging or deploy
		if m.stageFirst {
			return deployStaging
		}
		return deployExecute
	}

	// In interactive mode, show the config form so users can review settings
	return deployConfigInput
}

// View renders the main model.
func (m MainModel) View() string {
	if m.quitting {
		return quitTextStyle.Render("See you next time.")
	}

	switch m.sessionState {
	case deployBlueprintSelect:
		return m.selectBlueprint.View()
	case deployConfigInput:
		selected := "\n  Blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n"
		return selected + m.deployConfigForm.View()
	case deployStaging:
		selected := "\n  Blueprint: " + m.styles.Selected.Render(m.blueprintFile) + "\n"
		if m.instanceName != "" {
			selected += "  Instance: " + m.styles.Selected.Render(m.instanceName) + "\n"
		}
		if m.staging != nil {
			return selected + m.staging.View()
		}
		return selected
	default:
		// In deploy view, always show the split-pane (even during streaming)
		return m.deploy.View()
	}
}

// NewDeployApp creates a new deploy application with the given configuration.
func NewDeployApp(
	deployEngine engine.DeployEngine,
	logger *zap.Logger,
	changesetID string,
	instanceID string,
	instanceName string,
	blueprintFile string,
	isDefaultBlueprintFile bool,
	autoRollback bool,
	force bool,
	stageFirst bool,
	autoApprove bool,
	skipPrompts bool,
	bluelinkStyles *stylespkg.Styles,
	headless bool,
	headlessWriter io.Writer,
	jsonMode bool,
) (*MainModel, error) {
	sessionState := deployBlueprintSelect
	// In headless mode or with --skip-prompts, use the default blueprint file if no explicit file is provided.
	// Also auto-start if blueprint file is explicitly provided.
	autoDeploy := (blueprintFile != "" && !isDefaultBlueprintFile) || headless || skipPrompts

	if autoDeploy {
		// Check if all required values are provided for skipping prompts
		instanceIdentified := instanceID != "" || instanceName != ""
		hasDeployPath := stageFirst || changesetID != ""
		canSkipForm := skipPrompts && instanceIdentified && hasDeployPath

		// Flag validation for headless mode is now done at the command level
		// using headless.Validate() in deploy.go

		if headless || canSkipForm {
			// Skip config form - go straight to staging or deploy
			if stageFirst {
				sessionState = deployStaging
			} else {
				sessionState = deployExecute
			}
		} else {
			// In interactive mode, show the config form so users can review settings
			sessionState = deployConfigInput
		}
	}

	fp, err := sharedui.BlueprintLocalFilePicker(bluelinkStyles)
	if err != nil {
		return nil, err
	}

	selectBlueprint, err := sharedui.NewSelectBlueprint(
		blueprintFile,
		autoDeploy,
		"deploy",
		bluelinkStyles,
		&fp,
	)
	if err != nil {
		return nil, err
	}

	deployConfigForm := NewDeployConfigFormModel(
		DeployConfigFormInitialValues{
			InstanceName: instanceName,
			InstanceID:   instanceID,
			ChangesetID:  changesetID,
			StageFirst:   stageFirst,
			AutoApprove:  autoApprove,
			AutoRollback: autoRollback,
		},
		bluelinkStyles,
	)

	// Create staging model for --stage flow (reusing stageui.StageModel)
	stagingModel := stageui.NewStageModel(
		deployEngine,
		logger,
		instanceID,
		instanceName,
		false, // destroy - not applicable for deploy staging
		force, // skipDriftCheck - use force flag to skip drift detection during staging
		bluelinkStyles,
		headless,
		headlessWriter,
		jsonMode,
	)
	staging := &stagingModel
	// Pre-populate blueprint info if available
	staging.SetBlueprintFile(blueprintFile)
	// Mark as deploy flow mode so staging doesn't print apply hint or quit
	staging.SetDeployFlowMode(true)

	blueprintSource := shared.BlueprintSourceFromPath(blueprintFile)
	deploy := NewDeployModel(
		deployEngine,
		logger,
		changesetID,
		instanceID,
		instanceName,
		blueprintFile,
		blueprintSource,
		autoRollback,
		force,
		bluelinkStyles,
		headless,
		headlessWriter,
		nil, // changesetChanges - will be set when staging completes
		jsonMode,
	)

	return &MainModel{
		sessionState:       sessionState,
		selectBlueprint:    selectBlueprint,
		deployConfigForm:   deployConfigForm,
		staging:            staging,
		deploy:             deploy,
		changesetID:        changesetID,
		instanceID:         instanceID,
		instanceName:       instanceName,
		blueprintFile:      blueprintFile,
		blueprintSource:    blueprintSource,
		isDefaultBlueprint: isDefaultBlueprintFile,
		autoRollback:       autoRollback,
		force:              force,
		stageFirst:         stageFirst,
		autoApprove:        autoApprove,
		skipPrompts:        skipPrompts,
		headless:           headless,
		jsonMode:           jsonMode,
		engine:             deployEngine,
		logger:             logger,
		styles:             bluelinkStyles,
	}, nil
}

