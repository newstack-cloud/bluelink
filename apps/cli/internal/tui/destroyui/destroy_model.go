package destroyui

import (
	"errors"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"go.uber.org/zap"
)

// Type aliases for shared types.
type (
	ItemType   = shared.ItemType
	ActionType = shared.ActionType
)

// Re-export constants.
const (
	ItemTypeResource = shared.ItemTypeResource
	ItemTypeChild    = shared.ItemTypeChild
	ItemTypeLink     = shared.ItemTypeLink

	ActionCreate   = shared.ActionCreate
	ActionUpdate   = shared.ActionUpdate
	ActionDelete   = shared.ActionDelete
	ActionRecreate = shared.ActionRecreate
	ActionNoChange = shared.ActionNoChange
)

// MaxExpandDepth is the maximum nesting depth for expanding child blueprints.
const MaxExpandDepth = 2

// Re-export shared result types for backwards compatibility.
type (
	ElementFailure     = shared.ElementFailure
	InterruptedElement = shared.InterruptedElement
)

// DestroyedElement represents an element that was destroyed successfully.
type DestroyedElement struct {
	ElementName string
	ElementPath string
	ElementType string
}

// ResourceDestroyItem represents a resource being destroyed with real-time status.
type ResourceDestroyItem struct {
	Name           string
	ResourceID     string
	ResourceType   string
	DisplayName    string
	Action         ActionType
	Status         core.ResourceStatus
	PreciseStatus  core.PreciseResourceStatus
	FailureReasons []string
	Attempt        int
	CanRetry       bool
	Group          int
	Durations      *state.ResourceCompletionDurations
	Timestamp      int64
	Skipped        bool
	Changes        *provider.Changes
	ResourceState  *state.ResourceState
}

func (r *ResourceDestroyItem) GetAction() shared.ActionType        { return shared.ActionType(r.Action) }
func (r *ResourceDestroyItem) GetResourceStatus() core.ResourceStatus { return r.Status }
func (r *ResourceDestroyItem) SetSkipped(skipped bool)             { r.Skipped = skipped }

// ChildDestroyItem represents a child blueprint being destroyed.
type ChildDestroyItem struct {
	Name             string
	ParentInstanceID string
	ChildInstanceID  string
	Action           ActionType
	Status           core.InstanceStatus
	FailureReasons   []string
	Group            int
	Durations        *state.InstanceCompletionDuration
	Timestamp        int64
	Depth            int
	Skipped          bool
	Changes          *changes.BlueprintChanges
}

func (c *ChildDestroyItem) GetAction() shared.ActionType       { return shared.ActionType(c.Action) }
func (c *ChildDestroyItem) GetChildStatus() core.InstanceStatus { return c.Status }
func (c *ChildDestroyItem) SetSkipped(skipped bool)            { c.Skipped = skipped }

// LinkDestroyItem represents a link being destroyed.
type LinkDestroyItem struct {
	LinkID               string
	LinkName             string
	ResourceAName        string
	ResourceBName        string
	Action               ActionType
	Status               core.LinkStatus
	PreciseStatus        core.PreciseLinkStatus
	FailureReasons       []string
	CurrentStageAttempt  int
	CanRetryCurrentStage bool
	Durations            *state.LinkCompletionDurations
	Timestamp            int64
	Skipped              bool
}

func (l *LinkDestroyItem) GetAction() shared.ActionType   { return shared.ActionType(l.Action) }
func (l *LinkDestroyItem) GetLinkStatus() core.LinkStatus { return l.Status }
func (l *LinkDestroyItem) SetSkipped(skipped bool)        { l.Skipped = skipped }

// DestroyItem is the unified item type for the split-pane.
type DestroyItem struct {
	Type          ItemType
	Resource      *ResourceDestroyItem
	Child         *ChildDestroyItem
	Link          *LinkDestroyItem
	ParentChild   string
	Depth         int
	Path          string
	Changes       *changes.BlueprintChanges
	InstanceState *state.InstanceState

	childrenByName  map[string]*ChildDestroyItem
	resourcesByName map[string]*ResourceDestroyItem
	linksByName     map[string]*LinkDestroyItem
}

// DestroyModel is the model for the destroy view with real-time split-pane.
type DestroyModel struct {
	splitPane       splitpane.Model
	detailsRenderer *DestroyDetailsRenderer
	sectionGrouper  *DestroySectionGrouper
	footerRenderer  *DestroyFooterRenderer

	driftSplitPane       splitpane.Model
	driftDetailsRenderer *DriftDetailsRenderer
	driftSectionGrouper  *DriftSectionGrouper
	driftFooterRenderer  *DriftFooterRenderer

	driftReviewMode         bool
	driftResult             *container.ReconciliationCheckResult
	driftMessage            string
	driftBlockedChangesetID string
	driftContext            driftui.DriftContext
	driftInstanceState      *state.InstanceState

	width  int
	height int

	items           []DestroyItem
	resourcesByName map[string]*ResourceDestroyItem
	childrenByName  map[string]*ChildDestroyItem
	linksByName     map[string]*LinkDestroyItem

	instanceIDToChildName   map[string]string
	instanceIDToParentID    map[string]string
	childNameToInstancePath map[string]string

	instanceID               string
	instanceName             string
	changesetID              string
	streaming                bool
	fetchingPreDestroyState  bool
	fetchingChangeset        bool
	finished                 bool
	finalStatus              core.InstanceStatus
	failureReasons           []string
	elementFailures          []ElementFailure
	interruptedElements      []InterruptedElement
	destroyedElements        []DestroyedElement
	err                      error
	deployChangesetError     bool
	showingOverview          bool
	showingPreDestroyState   bool
	overviewViewport         viewport.Model
	preDestroyStateViewport  viewport.Model
	preDestroyInstanceState  *state.InstanceState
	postDestroyInstanceState *state.InstanceState

	engine      engine.DeployEngine
	eventStream chan types.BlueprintInstanceEvent
	errStream   chan error

	force bool

	changesetChanges *changes.BlueprintChanges

	headlessMode   bool
	headlessWriter io.Writer
	printer        *headless.Printer
	jsonMode       bool

	styles  *stylespkg.Styles
	logger  *zap.Logger
	spinner spinner.Model
}

// Init initializes the destroy model.
func (m DestroyModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages for the destroy model.
func (m DestroyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case sharedui.SelectBlueprintMsg:
		return m.handleSelectBlueprint(msg)
	case StartDestroyMsg:
		return m.handleStartDestroy()
	case DestroyStartedMsg:
		return m.handleDestroyStarted(msg)
	case DestroyEventMsg:
		return m.handleDestroyEvent(msg)
	case DestroyErrorMsg:
		return m.handleDestroyError(msg)
	case DeployChangesetErrorMsg:
		return m.handleDeployChangesetError()
	case DestroyStreamClosedMsg:
		return m.handleDestroyStreamClosed()
	case PreDestroyInstanceStateFetchedMsg:
		return m.handlePreDestroyInstanceStateFetched(msg)
	case ChangesetFetchedMsg:
		return m.handleChangesetFetched(msg)
	case PostDestroyInstanceStateFetchedMsg:
		return m.handlePostDestroyInstanceStateFetched(msg)
	case driftui.DriftDetectedMsg:
		return m.handleDriftDetected(msg)
	case driftui.ReconciliationCompleteMsg:
		return m.handleReconciliationComplete()
	case driftui.ReconciliationErrorMsg:
		return m.handleReconciliationError(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.footerRenderer.SpinnerView = m.spinner.View()
		return m, cmd
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.MouseMsg:
		return m.handleMouseMsg(msg)
	case splitpane.QuitMsg:
		return m, tea.Quit
	case splitpane.BackMsg:
		if m.driftReviewMode {
			return m, tea.Quit
		}
		return m, nil
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, nil
}

func (m DestroyModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.splitPane, cmd = m.splitPane.Update(msg)
	cmds = append(cmds, cmd)

	m.driftSplitPane, cmd = m.driftSplitPane.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.overviewViewport.Width = msg.Width
	m.overviewViewport.Height = msg.Height - overviewFooterHeight()

	m.preDestroyStateViewport.Width = msg.Width
	m.preDestroyStateViewport.Height = msg.Height - overviewFooterHeight()

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(cmds...)
}

func (m DestroyModel) handleSelectBlueprint(_ sharedui.SelectBlueprintMsg) (tea.Model, tea.Cmd) {
	// Blueprint is not required for destroy execution
	// This message is only relevant when coming from staging
	return m.handleStartDestroy()
}

func (m DestroyModel) handleStartDestroy() (tea.Model, tea.Cmd) {
	if m.streaming || m.fetchingPreDestroyState || m.fetchingChangeset {
		return m, nil
	}

	var cmds []tea.Cmd

	needsInstanceState := m.preDestroyInstanceState == nil && (m.instanceID != "" || m.instanceName != "")
	needsChangesetChanges := m.changesetChanges == nil && m.changesetID != ""

	if needsInstanceState {
		m.fetchingPreDestroyState = true
		cmds = append(cmds, fetchPreDestroyInstanceStateCmd(m))
	}

	if needsChangesetChanges {
		m.fetchingChangeset = true
		cmds = append(cmds, fetchChangesetChangesCmd(m))
	}

	if len(cmds) > 0 {
		m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
		return m, tea.Batch(cmds...)
	}

	m.streaming = true
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(startDestroyCmd(m), checkForDestroyErrCmd(m))
}

func (m DestroyModel) handlePreDestroyInstanceStateFetched(msg PreDestroyInstanceStateFetchedMsg) (tea.Model, tea.Cmd) {
	m.fetchingPreDestroyState = false

	if m.streaming {
		return m, nil
	}

	m.SetPreDestroyInstanceState(msg.InstanceState)

	// Wait for changeset to be fetched if still in progress
	if m.fetchingChangeset {
		return m, nil
	}

	return m.startDestroyAfterFetches()
}

func (m DestroyModel) handleChangesetFetched(msg ChangesetFetchedMsg) (tea.Model, tea.Cmd) {
	m.fetchingChangeset = false

	if m.streaming {
		return m, nil
	}

	if msg.Changes != nil {
		m.changesetChanges = msg.Changes
	}

	// Wait for instance state to be fetched if still in progress
	if m.fetchingPreDestroyState {
		return m, nil
	}

	return m.startDestroyAfterFetches()
}

func (m DestroyModel) startDestroyAfterFetches() (tea.Model, tea.Cmd) {
	if m.changesetChanges != nil {
		m.items = buildItemsFromChangeset(m.changesetChanges, m.resourcesByName, m.childrenByName, m.linksByName, m.preDestroyInstanceState)
		m.splitPane.SetItems(ToSplitPaneItems(m.items))
	}

	m.streaming = true
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(startDestroyCmd(m), checkForDestroyErrCmd(m))
}

func (m DestroyModel) handleDestroyStarted(msg DestroyStartedMsg) (tea.Model, tea.Cmd) {
	m.instanceID = msg.InstanceID
	m.streaming = true
	m.footerRenderer.InstanceID = msg.InstanceID

	if m.headlessMode && !m.jsonMode {
		m.printHeadlessHeader()
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(waitForNextDestroyEventCmd(m), checkForDestroyErrCmd(m))
}

func (m DestroyModel) handleDestroyEvent(msg DestroyEventMsg) (tea.Model, tea.Cmd) {
	event := types.BlueprintInstanceEvent(msg)
	m.processEvent(&event)
	m.splitPane.UpdateItems(ToSplitPaneItems(m.items))
	// Explicitly refresh viewports to ensure details view is updated
	m.splitPane.RefreshViewports()

	cmds := []tea.Cmd{checkForDestroyErrCmd(m)}

	finishData, isFinish := event.AsFinish()
	if !isFinish {
		cmds = append(cmds, waitForNextDestroyEventCmd(m))
		m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
		return m, tea.Batch(cmds...)
	}

	if !finishData.EndOfStream {
		cmds = append(cmds, waitForNextDestroyEventCmd(m))
		m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
		return m, tea.Batch(cmds...)
	}

	m.finished = true
	m.finalStatus = finishData.Status
	m.failureReasons = finishData.FailureReasons
	m.footerRenderer.FinalStatus = finishData.Status
	m.footerRenderer.Finished = true
	m.detailsRenderer.Finished = true

	if IsFailedStatus(finishData.Status) {
		m.markPendingItemsAsSkipped()
		m.markInProgressItemsAsInterrupted()
		m.splitPane.UpdateItems(ToSplitPaneItems(m.items))
	}

	m.collectDestroyResults()
	m.footerRenderer.DestroyedElements = m.destroyedElements
	m.footerRenderer.ElementFailures = m.elementFailures
	m.footerRenderer.InterruptedElements = m.interruptedElements

	cmds = append(cmds, fetchPostDestroyInstanceStateCmd(m))

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(cmds...)
}

func (m DestroyModel) handleDestroyError(msg DestroyErrorMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		return m, nil
	}

	m.err = msg.Err
	if m.headlessMode {
		if m.jsonMode {
			m.outputJSONError(msg.Err)
		} else {
			m.printHeadlessError(msg.Err)
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m DestroyModel) handleDeployChangesetError() (tea.Model, tea.Cmd) {
	m.deployChangesetError = true
	if m.headlessMode {
		if m.jsonMode {
			m.outputJSONError(errors.New("cannot destroy using a deploy changeset"))
		} else {
			m.printHeadlessDeployChangesetError()
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m DestroyModel) handleDestroyStreamClosed() (tea.Model, tea.Cmd) {
	if !m.finished {
		m.finished = true
		m.streaming = false
		m.err = fmt.Errorf("destroy event stream closed unexpectedly (connection timeout or dropped)")
		if m.headlessMode {
			if m.jsonMode {
				m.outputJSONError(m.err)
			} else {
				m.printHeadlessError(m.err)
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m DestroyModel) handlePostDestroyInstanceStateFetched(msg PostDestroyInstanceStateFetchedMsg) (tea.Model, tea.Cmd) {
	m.postDestroyInstanceState = msg.InstanceState
	m.detailsRenderer.PostDestroyInstanceState = msg.InstanceState

	if msg.InstanceState != nil {
		m.footerRenderer.HasInstanceState = true
	}

	if m.headlessMode {
		if m.jsonMode {
			m.outputJSON()
		} else {
			m.printHeadlessSummary()
		}
		return m, tea.Quit
	}

	return m, nil
}

func (m DestroyModel) handleDriftDetected(msg driftui.DriftDetectedMsg) (tea.Model, tea.Cmd) {
	m.driftReviewMode = true
	m.driftResult = msg.ReconciliationResult
	m.driftMessage = msg.Message
	m.driftBlockedChangesetID = msg.ChangesetID
	m.driftContext = driftui.DriftContextDestroy
	m.driftInstanceState = msg.InstanceState
	m.streaming = false

	if m.driftResult != nil {
		driftItems := BuildDriftItems(m.driftResult, m.driftInstanceState)
		m.driftSplitPane.SetItems(driftItems)
	}

	if m.headlessMode {
		if m.jsonMode {
			m.outputJSONDrift()
		} else {
			m.printHeadlessDriftDetected()
		}
		return m, tea.Quit
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, nil
}

func (m DestroyModel) handleReconciliationComplete() (tea.Model, tea.Cmd) {
	m.driftReviewMode = false
	m.driftResult = nil
	m.driftMessage = ""

	if m.driftBlockedChangesetID != "" {
		m.changesetID = m.driftBlockedChangesetID
	}
	m.driftBlockedChangesetID = ""

	m.streaming = true

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, continueDestroyCmd(m)
}

func (m DestroyModel) handleReconciliationError(msg driftui.ReconciliationErrorMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		return m, nil
	}

	m.err = msg.Err
	m.driftReviewMode = false

	if m.headlessMode {
		if m.jsonMode {
			m.outputJSONError(msg.Err)
		} else {
			m.printHeadlessError(msg.Err)
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m DestroyModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.err != nil || m.deployChangesetError {
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	if m.showingOverview {
		return m.handleOverviewKeyMsg(msg)
	}

	if m.showingPreDestroyState {
		return m.handlePreDestroyStateKeyMsg(msg)
	}

	if m.finished && (msg.String() == "o" || msg.String() == "O") {
		m.showingOverview = true
		m.overviewViewport.SetContent(m.renderOverviewContent())
		m.overviewViewport.GotoTop()
		return m, nil
	}

	// Toggle pre-destroy state view when instance state is available
	if (msg.String() == "s" || msg.String() == "S") && m.preDestroyInstanceState != nil {
		m.showingPreDestroyState = true
		m.preDestroyStateViewport.SetContent(m.renderPreDestroyStateContent())
		m.preDestroyStateViewport.GotoTop()
		return m, nil
	}

	if m.driftReviewMode {
		return m.handleDriftReviewKeyMsg(msg)
	}

	var cmd tea.Cmd
	m.splitPane, cmd = m.splitPane.Update(msg)
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, cmd
}

func (m DestroyModel) handleOverviewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result := shared.HandleViewportKeyMsg(msg, m.overviewViewport, "o", "O")
	if result.ShouldQuit {
		return m, tea.Quit
	}
	if result.ShouldClose {
		m.showingOverview = false
		return m, nil
	}
	m.overviewViewport = result.Viewport
	return m, result.Cmd
}

func (m DestroyModel) handlePreDestroyStateKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result := shared.HandleViewportKeyMsg(msg, m.preDestroyStateViewport, "s", "S")
	if result.ShouldQuit {
		return m, tea.Quit
	}
	if result.ShouldClose {
		m.showingPreDestroyState = false
		return m, nil
	}
	m.preDestroyStateViewport = result.Viewport
	return m, result.Cmd
}

func (m DestroyModel) handleDriftReviewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a", "A":
		return m, applyReconciliationCmd(m)
	case "q":
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.driftSplitPane, cmd = m.driftSplitPane.Update(msg)
		m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
		return m, cmd
	}
}

func (m DestroyModel) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.driftReviewMode {
		m.driftSplitPane, cmd = m.driftSplitPane.Update(msg)
	} else {
		m.splitPane, cmd = m.splitPane.Update(msg)
	}
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, cmd
}

// View renders the destroy model.
func (m DestroyModel) View() string {
	if m.headlessMode {
		return ""
	}

	if m.err != nil {
		return m.renderError(m.err)
	}

	if m.deployChangesetError {
		return m.renderDeployChangesetError()
	}

	if m.showingOverview {
		return m.renderOverviewView()
	}

	if m.showingPreDestroyState {
		return m.renderPreDestroyStateView()
	}

	if m.driftReviewMode {
		return m.driftSplitPane.View()
	}

	return m.splitPane.View()
}

// SetChangesetChanges sets the changeset changes and rebuilds items from them.
func (m *DestroyModel) SetChangesetChanges(changesetChanges *changes.BlueprintChanges) {
	if changesetChanges == nil {
		return
	}
	m.changesetChanges = changesetChanges
	m.items = buildItemsFromChangeset(changesetChanges, m.resourcesByName, m.childrenByName, m.linksByName, m.preDestroyInstanceState)
	m.splitPane.SetItems(ToSplitPaneItems(m.items))
}

// SetPreDestroyInstanceState sets the pre-destroy instance state.
func (m *DestroyModel) SetPreDestroyInstanceState(instanceState *state.InstanceState) {
	m.preDestroyInstanceState = instanceState
	m.detailsRenderer.PreDestroyInstanceState = instanceState
	if instanceState != nil {
		m.footerRenderer.HasInstanceState = true
	}
}

// NewDestroyModel creates a new destroy model.
func NewDestroyModel(
	destroyEngine engine.DeployEngine,
	logger *zap.Logger,
	changesetID string,
	instanceID string,
	instanceName string,
	force bool,
	styles *stylespkg.Styles,
	isHeadless bool,
	headlessWriter io.Writer,
	changesetChanges *changes.BlueprintChanges,
	jsonMode bool,
) DestroyModel {
	detailsRenderer, sectionGrouper, footerRenderer := createDestroyRenderers(instanceID, instanceName, changesetID)
	splitPaneConfig := createDestroySplitPaneConfig(styles, detailsRenderer, sectionGrouper, footerRenderer)

	driftDetailsRenderer, driftSectionGrouper, driftFooterRenderer := createDestroyDriftRenderers()
	driftSplitPaneConfig := createDestroyDriftSplitPaneConfig(styles, driftDetailsRenderer, driftSectionGrouper, driftFooterRenderer)

	printer := createDestroyHeadlessPrinter(isHeadless, headlessWriter)

	resourcesByName := make(map[string]*ResourceDestroyItem)
	childrenByName := make(map[string]*ChildDestroyItem)
	linksByName := make(map[string]*LinkDestroyItem)
	items := buildItemsFromChangeset(changesetChanges, resourcesByName, childrenByName, linksByName, nil)

	model := DestroyModel{
		splitPane:               splitpane.New(splitPaneConfig),
		detailsRenderer:         detailsRenderer,
		sectionGrouper:          sectionGrouper,
		footerRenderer:          footerRenderer,
		driftSplitPane:          splitpane.New(driftSplitPaneConfig),
		driftDetailsRenderer:    driftDetailsRenderer,
		driftSectionGrouper:     driftSectionGrouper,
		driftFooterRenderer:     driftFooterRenderer,
		engine:                  destroyEngine,
		logger:                  logger,
		changesetID:             changesetID,
		instanceID:              instanceID,
		instanceName:            instanceName,
		force:                   force,
		changesetChanges:        changesetChanges,
		styles:                  styles,
		headlessMode:            isHeadless,
		headlessWriter:          headlessWriter,
		printer:                 printer,
		jsonMode:                jsonMode,
		spinner:                 createDestroySpinner(styles),
		eventStream:             make(chan types.BlueprintInstanceEvent),
		errStream:               make(chan error),
		resourcesByName:         resourcesByName,
		childrenByName:          childrenByName,
		linksByName:             linksByName,
		instanceIDToChildName:   make(map[string]string),
		instanceIDToParentID:    make(map[string]string),
		childNameToInstancePath: make(map[string]string),
		items:                   items,
	}

	if len(items) > 0 {
		model.splitPane.SetItems(ToSplitPaneItems(items))
	}

	return model
}

func createDestroyRenderers(instanceID, instanceName, changesetID string) (*DestroyDetailsRenderer, *DestroySectionGrouper, *DestroyFooterRenderer) {
	detailsRenderer := &DestroyDetailsRenderer{
		MaxExpandDepth:       MaxExpandDepth,
		NavigationStackDepth: 0,
	}
	sectionGrouper := &DestroySectionGrouper{
		SectionGrouper: shared.SectionGrouper{MaxExpandDepth: MaxExpandDepth},
	}
	footerRenderer := &DestroyFooterRenderer{
		InstanceID:   instanceID,
		InstanceName: instanceName,
		ChangesetID:  changesetID,
	}
	return detailsRenderer, sectionGrouper, footerRenderer
}

func createDestroySplitPaneConfig(
	styles *stylespkg.Styles,
	detailsRenderer *DestroyDetailsRenderer,
	sectionGrouper *DestroySectionGrouper,
	footerRenderer *DestroyFooterRenderer,
) splitpane.Config {
	return splitpane.Config{
		Styles:          styles,
		Title:           "Destroy",
		DetailsRenderer: detailsRenderer,
		LeftPaneRatio:   0.4,
		MaxExpandDepth:  MaxExpandDepth,
		SectionGrouper:  sectionGrouper,
		FooterRenderer:  footerRenderer,
	}
}

func createDestroyDriftRenderers() (*DriftDetailsRenderer, *DriftSectionGrouper, *DriftFooterRenderer) {
	driftDetailsRenderer := &DriftDetailsRenderer{
		MaxExpandDepth:       MaxExpandDepth,
		NavigationStackDepth: 0,
	}
	driftSectionGrouper := &DriftSectionGrouper{
		MaxExpandDepth: MaxExpandDepth,
	}
	driftFooterRenderer := &DriftFooterRenderer{
		Context: driftui.DriftContextDestroy,
	}
	return driftDetailsRenderer, driftSectionGrouper, driftFooterRenderer
}

func createDestroyDriftSplitPaneConfig(
	styles *stylespkg.Styles,
	detailsRenderer *DriftDetailsRenderer,
	sectionGrouper *DriftSectionGrouper,
	footerRenderer *DriftFooterRenderer,
) splitpane.Config {
	return splitpane.Config{
		Styles:          styles,
		DetailsRenderer: detailsRenderer,
		Title:           "⚠ Drift Detected",
		LeftPaneRatio:   0.4,
		MaxExpandDepth:  MaxExpandDepth,
		SectionGrouper:  sectionGrouper,
		FooterRenderer:  footerRenderer,
	}
}

func createDestroyHeadlessPrinter(isHeadless bool, headlessWriter io.Writer) *headless.Printer {
	if !isHeadless || headlessWriter == nil {
		return nil
	}
	prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[destroy] ")
	return headless.NewPrinter(prefixedWriter, 80)
}

func createDestroySpinner(styles *stylespkg.Styles) spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner
	return s
}

func (m *DestroyModel) markPendingItemsAsSkipped() {
	shared.MarkPendingResourcesAsSkipped(m.resourcesByName)
	shared.MarkPendingChildrenAsSkipped(m.childrenByName)
	shared.MarkPendingLinksAsSkipped(m.linksByName)
}

func (m *DestroyModel) markInProgressItemsAsInterrupted() {
	for _, item := range m.resourcesByName {
		if item.Action == ActionNoChange {
			continue
		}
		if IsInProgressResourceStatus(item.Status) {
			item.Status = core.ResourceStatusDestroyInterrupted
			item.PreciseStatus = core.PreciseResourceStatusDestroyInterrupted
		}
	}

	for _, item := range m.childrenByName {
		if item.Action == ActionNoChange {
			continue
		}
		if IsInProgressInstanceStatus(item.Status) {
			item.Status = core.InstanceStatusDestroyInterrupted
		}
	}

	for _, item := range m.linksByName {
		if item.Action == ActionNoChange {
			continue
		}
		if IsInProgressLinkStatus(item.Status) {
			item.Status = core.LinkStatusDestroyInterrupted
			item.PreciseStatus = core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted
		}
	}
}

// Test accessor methods - these provide read-only access for testing purposes.

// Err returns the error stored in the model.
func (m *DestroyModel) Err() error {
	return m.err
}

// FinalStatus returns the final instance status after destroy.
func (m *DestroyModel) FinalStatus() core.InstanceStatus {
	return m.finalStatus
}

// Items returns the destroy items.
func (m *DestroyModel) Items() []DestroyItem {
	return m.items
}

// ResourcesByName returns the resources lookup map.
func (m *DestroyModel) ResourcesByName() map[string]*ResourceDestroyItem {
	return m.resourcesByName
}

// Force returns whether force mode is enabled.
func (m *DestroyModel) Force() bool {
	return m.force
}
