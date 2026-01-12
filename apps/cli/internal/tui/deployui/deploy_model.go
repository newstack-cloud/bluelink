package deployui

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

// Type aliases for backwards compatibility with shared types.
type (
	ItemType   = shared.ItemType
	ActionType = shared.ActionType
)

// Re-export constants for backwards compatibility.
const (
	ItemTypeResource = shared.ItemTypeResource
	ItemTypeChild    = shared.ItemTypeChild
	ItemTypeLink     = shared.ItemTypeLink

	ActionCreate   = shared.ActionCreate
	ActionUpdate   = shared.ActionUpdate
	ActionDelete   = shared.ActionDelete
	ActionRecreate = shared.ActionRecreate
	ActionNoChange = shared.ActionNoChange
	ActionInspect  = shared.ActionInspect
)

// MaxExpandDepth is the maximum nesting depth for expanding child blueprints.
const MaxExpandDepth = 2

// Re-export shared result types for backwards compatibility.
type (
	ElementFailure     = shared.ElementFailure
	InterruptedElement = shared.InterruptedElement
	SuccessfulElement  = shared.SuccessfulElement
)

// ResourceDeployItem represents a resource being deployed with real-time status.
type ResourceDeployItem struct {
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
	Skipped        bool // Set to true when deployment failed before this resource was attempted
	// Changes holds the provider.Changes data from the changeset, providing access to
	// AppliedResourceInfo.CurrentResourceState for pre-deployment outputs and spec data.
	Changes *provider.Changes
	// ResourceState holds the pre-deployment resource state from the instance.
	// Used for displaying outputs and spec data for items with no changes or before deployment completes.
	ResourceState *state.ResourceState
}

func (r *ResourceDeployItem) GetAction() shared.ActionType      { return shared.ActionType(r.Action) }
func (r *ResourceDeployItem) GetResourceStatus() core.ResourceStatus { return r.Status }
func (r *ResourceDeployItem) SetSkipped(skipped bool)           { r.Skipped = skipped }

// ChildDeployItem represents a child blueprint being deployed.
type ChildDeployItem struct {
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
	Skipped          bool // Set to true when deployment failed before this child was attempted
	// Changes holds the blueprint changes for this child (from staging)
	// Used to provide hierarchy for GetChildren()
	Changes *changes.BlueprintChanges
}

func (c *ChildDeployItem) GetAction() shared.ActionType     { return shared.ActionType(c.Action) }
func (c *ChildDeployItem) GetChildStatus() core.InstanceStatus { return c.Status }
func (c *ChildDeployItem) SetSkipped(skipped bool)          { c.Skipped = skipped }

// LinkDeployItem represents a link being deployed.
type LinkDeployItem struct {
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
	Skipped              bool // Set to true when deployment failed before this link was attempted
}

func (l *LinkDeployItem) GetAction() shared.ActionType { return shared.ActionType(l.Action) }
func (l *LinkDeployItem) GetLinkStatus() core.LinkStatus  { return l.Status }
func (l *LinkDeployItem) SetSkipped(skipped bool)      { l.Skipped = skipped }

// DeployItem is the unified item type for the split-pane.
type DeployItem struct {
	Type        ItemType
	Resource    *ResourceDeployItem
	Child       *ChildDeployItem
	Link        *LinkDeployItem
	ParentChild string // For nested items
	Depth       int
	// Path is the full path to this item (e.g., "childA/childB/resourceName")
	// Used for unique keying in the shared lookup maps.
	Path string
	// Changes holds the blueprint changes for this item (for children)
	// Used to provide hierarchy for GetChildren()
	Changes *changes.BlueprintChanges
	// InstanceState holds the instance state for this level of the hierarchy.
	// Used to provide resource state data for items with no changes and
	// to populate the navigation tree with all instance elements.
	InstanceState *state.InstanceState

	// Lookup maps for sharing state with dynamically created nested items.
	// These are set on top-level items and passed down to nested items.
	childrenByName  map[string]*ChildDeployItem
	resourcesByName map[string]*ResourceDeployItem
	linksByName     map[string]*LinkDeployItem
}

// DeployModel is the model for the deploy view with real-time split-pane.
type DeployModel struct {
	// Split pane shown from the START of deployment
	splitPane       splitpane.Model
	detailsRenderer *DeployDetailsRenderer
	sectionGrouper  *DeploySectionGrouper
	footerRenderer  *DeployFooterRenderer

	// Split pane for drift review mode
	driftSplitPane       splitpane.Model
	driftDetailsRenderer *DriftDetailsRenderer
	driftSectionGrouper  *DriftSectionGrouper
	driftFooterRenderer  *DriftFooterRenderer

	// Drift review state
	driftReviewMode         bool
	driftResult             *container.ReconciliationCheckResult
	driftMessage            string
	driftBlockedChangesetID string // Changeset ID to use after reconciliation
	driftContext            driftui.DriftContext
	driftInstanceState      *state.InstanceState // Instance state for displaying computed fields in drift UI

	// Layout
	width  int
	height int

	// Items - indexed for fast updates
	items           []DeployItem
	resourcesByName map[string]*ResourceDeployItem
	childrenByName  map[string]*ChildDeployItem
	linksByName     map[string]*LinkDeployItem

	// Instance tracking - maps child instance IDs to child names and parent instance IDs
	// This allows us to route resource/link updates to the correct hierarchy level
	instanceIDToChildName   map[string]string
	instanceIDToParentID    map[string]string
	childNameToInstancePath map[string]string // Maps child name to its full path (e.g., "childA/childB")

	// State
	instanceID               string
	instanceName             string
	changesetID              string
	streaming                bool
	fetchingPreDeployState   bool // True while fetching pre-deploy instance state
	finished                 bool
	finalStatus              core.InstanceStatus
	failureReasons           []string             // generic failure messages from backend
	elementFailures          []ElementFailure     // Structured failures with root cause details
	interruptedElements      []InterruptedElement // Elements that were interrupted
	successfulElements       []SuccessfulElement  // Elements that completed successfully
	err                      error
	destroyChangesetError    bool                               // True when deployment failed due to destroy changeset
	showingOverview          bool                               // When true, show full-screen deployment overview
	overviewViewport         viewport.Model                     // Scrollable viewport for deployment overview
	showingSpecView          bool                               // When true, show full-screen spec view for selected resource
	specViewport             viewport.Model                     // Scrollable viewport for spec view
	showingExportsView       bool                               // When true, show full-screen exports view
	exportsModel             ExportsModel                       // Exports view model with split pane
	preDeployInstanceState   *state.InstanceState               // Instance state fetched before deployment for unchanged items
	postDeployInstanceState  *state.InstanceState               // Instance state fetched after deployment completes
	preRollbackState         *container.PreRollbackStateMessage // Captured state before auto-rollback
	showingPreRollbackState  bool                               // When true, show full-screen pre-rollback state view
	preRollbackStateViewport viewport.Model                     // Scrollable viewport for pre-rollback state view
	skippedRollbackItems     []container.SkippedRollbackItem    // Items skipped during rollback due to unsafe state

	// Streaming channels
	engine      engine.DeployEngine
	eventStream chan types.BlueprintInstanceEvent
	errStream   chan error

	// Config
	blueprintFile   string
	blueprintSource string
	autoRollback    bool
	force           bool

	// Changeset data - used to build item hierarchy
	changesetChanges *changes.BlueprintChanges

	// Headless mode
	headlessMode   bool
	headlessWriter io.Writer
	printer        *headless.Printer
	jsonMode       bool

	styles  *stylespkg.Styles
	logger  *zap.Logger
	spinner spinner.Model
}

// Init initializes the deploy model.
func (m DeployModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages for the deploy model.
func (m DeployModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case sharedui.SelectBlueprintMsg:
		return m.handleSelectBlueprint(msg)
	case StartDeployMsg:
		return m.handleStartDeploy()
	case DeployStartedMsg:
		return m.handleDeployStarted(msg)
	case DeployEventMsg:
		return m.handleDeployEvent(msg)
	case DeployErrorMsg:
		return m.handleDeployError(msg)
	case DestroyChangesetErrorMsg:
		return m.handleDestroyChangesetError()
	case DeployStreamClosedMsg:
		return m.handleDeployStreamClosed()
	case PreDeployInstanceStateFetchedMsg:
		return m.handlePreDeployInstanceStateFetched(msg)
	case PostDeployInstanceStateFetchedMsg:
		return m.handlePostDeployInstanceStateFetched(msg)
	case DeployStateRefreshTickMsg:
		return m.handleDeployStateRefreshTick()
	case DeployStateRefreshedMsg:
		return m.handleDeployStateRefreshed(msg)
	case driftui.DriftDetectedMsg:
		return m.handleDriftDetected(msg)
	case driftui.ReconciliationCompleteMsg:
		return m.handleReconciliationComplete()
	case driftui.ReconciliationErrorMsg:
		return m.handleReconciliationError(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		// Update footer renderer with current spinner frame
		m.footerRenderer.SpinnerView = m.spinner.View()
		return m, cmd
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.MouseMsg:
		return m.handleMouseMsg(msg)
	case splitpane.QuitMsg:
		return m, tea.Quit
	case splitpane.BackMsg:
		// At root level of split pane - quit if in drift review mode
		if m.driftReviewMode {
			return m, tea.Quit
		}
		return m, nil
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, nil
}

func (m DeployModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
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

	m.specViewport.Width = msg.Width
	m.specViewport.Height = msg.Height - specViewFooterHeight()

	m.preRollbackStateViewport.Width = msg.Width
	m.preRollbackStateViewport.Height = msg.Height - preRollbackStateFooterHeight()

	// Update exports model if showing
	if m.showingExportsView {
		m.exportsModel, cmd = m.exportsModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(cmds...)
}

func (m DeployModel) handleSelectBlueprint(msg sharedui.SelectBlueprintMsg) (tea.Model, tea.Cmd) {
	m.blueprintFile = msg.BlueprintFile
	m.blueprintSource = msg.Source
	return m.handleStartDeploy()
}

func (m DeployModel) handleStartDeploy() (tea.Model, tea.Cmd) {
	if m.streaming || m.fetchingPreDeployState {
		return m, nil
	}

	// If we don't have pre-deploy instance state and we have an instance ID/name,
	// fetch it first to populate unchanged items
	if m.preDeployInstanceState == nil && (m.instanceID != "" || m.instanceName != "") {
		m.fetchingPreDeployState = true
		m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
		return m, fetchPreDeployInstanceStateCmd(m)
	}

	m.streaming = true
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(startDeploymentCmd(m), checkForErrCmd(m))
}

func (m DeployModel) handlePreDeployInstanceStateFetched(msg PreDeployInstanceStateFetchedMsg) (tea.Model, tea.Cmd) {
	// Clear the fetching flag
	m.fetchingPreDeployState = false

	// Guard: Don't start deployment if already streaming
	if m.streaming {
		return m, nil
	}

	// Store the pre-deploy instance state
	m.SetPreDeployInstanceState(msg.InstanceState)

	// Rebuild items with the instance state to include unchanged items
	if m.changesetChanges != nil {
		m.items = BuildItemsFromChangeset(m.changesetChanges, m.resourcesByName, m.childrenByName, m.linksByName, m.preDeployInstanceState)
		m.splitPane.SetItems(ToSplitPaneItems(m.items))
	}

	// Now start deployment
	m.streaming = true
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(startDeploymentCmd(m), checkForErrCmd(m))
}

func (m DeployModel) handleDeployStarted(msg DeployStartedMsg) (tea.Model, tea.Cmd) {
	m.instanceID = msg.InstanceID
	m.streaming = true
	m.footerRenderer.InstanceID = msg.InstanceID

	if m.headlessMode && !m.jsonMode {
		m.printHeadlessHeader()
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(
		waitForNextDeployEventCmd(m),
		checkForErrCmd(m),
		startDeployStateRefreshTickerCmd(),
	)
}

func (m DeployModel) handleDeployEvent(msg DeployEventMsg) (tea.Model, tea.Cmd) {
	event := types.BlueprintInstanceEvent(msg)
	m.processEvent(&event)
	m.splitPane.UpdateItems(ToSplitPaneItems(m.items))
	// Explicitly refresh viewports to ensure details view is updated
	m.splitPane.RefreshViewports()

	cmds := []tea.Cmd{checkForErrCmd(m)}

	finishData, isFinish := event.AsFinish()
	if !isFinish {
		cmds = append(cmds, waitForNextDeployEventCmd(m))
		m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
		return m, tea.Batch(cmds...)
	}

	// Check if more events will follow (e.g., auto-rollback after failure)
	if !finishData.EndOfStream {
		cmds = append(cmds, waitForNextDeployEventCmd(m))
		m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
		return m, tea.Batch(cmds...)
	}

	// Inline finish handling to ensure state is preserved in the returned model
	m.finished = true
	m.finalStatus = finishData.Status
	m.failureReasons = finishData.FailureReasons
	m.skippedRollbackItems = finishData.SkippedRollbackItems
	m.footerRenderer.FinalStatus = finishData.Status
	m.footerRenderer.Finished = true
	m.detailsRenderer.Finished = true

	if IsFailedStatus(finishData.Status) {
		m.markPendingItemsAsSkipped()
		m.markInProgressItemsAsInterrupted()
		m.splitPane.UpdateItems(ToSplitPaneItems(m.items))
	}

	m.collectDeploymentResults()
	m.footerRenderer.SuccessfulElements = m.successfulElements
	m.footerRenderer.ElementFailures = m.elementFailures
	m.footerRenderer.InterruptedElements = m.interruptedElements

	// Fetch updated instance state for outputs display
	// In headless mode, the state fetch handler will print the summary and quit
	cmds = append(cmds, fetchPostDeployInstanceStateCmd(m))

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(cmds...)
}

func (m DeployModel) handleDeployError(msg DeployErrorMsg) (tea.Model, tea.Cmd) {
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

func (m DeployModel) handleDestroyChangesetError() (tea.Model, tea.Cmd) {
	m.destroyChangesetError = true
	if m.headlessMode {
		if m.jsonMode {
			m.outputJSONError(errors.New("cannot deploy using a destroy changeset"))
		} else {
			m.printHeadlessDestroyChangesetError()
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m DeployModel) handleDeployStreamClosed() (tea.Model, tea.Cmd) {
	// The deploy event stream was closed (typically due to timeout).
	// If deployment hasn't finished, mark it as interrupted.
	if !m.finished {
		m.finished = true
		m.streaming = false
		m.err = fmt.Errorf("deployment event stream closed unexpectedly (connection timeout or dropped)")
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

func (m DeployModel) handlePostDeployInstanceStateFetched(msg PostDeployInstanceStateFetchedMsg) (tea.Model, tea.Cmd) {
	// Store the fetched instance state for use in rendering outputs
	m.postDeployInstanceState = msg.InstanceState
	// Pass state to details renderer for output display
	m.detailsRenderer.PostDeployInstanceState = msg.InstanceState

	if msg.InstanceState != nil {
		m.footerRenderer.HasInstanceState = true
	}

	if m.showingExportsView {
		m.exportsModel.UpdateInstanceState(msg.InstanceState)
	}

	// In headless mode, now that we have the state, print summary and quit
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

func (m DeployModel) handleDeployStateRefreshTick() (tea.Model, tea.Cmd) {
	// Only refresh if still streaming and not finished
	if !m.streaming || m.finished {
		return m, nil
	}
	return m, tea.Batch(
		refreshDeployInstanceStateCmd(m),
		startDeployStateRefreshTickerCmd(),
	)
}

func (m DeployModel) handleDeployStateRefreshed(msg DeployStateRefreshedMsg) (tea.Model, tea.Cmd) {
	if msg.InstanceState == nil || !m.streaming {
		return m, nil
	}

	// Hydrate existing items with updated ResourceState
	m.refreshInstanceState(msg.InstanceState)

	// Update the split pane items (preserves selection)
	m.splitPane.UpdateItems(ToSplitPaneItems(m.items))
	// Explicitly refresh viewports to ensure details view is updated
	m.splitPane.RefreshViewports()

	return m, nil
}

// refreshInstanceState hydrates existing items with the latest state data.
func (m *DeployModel) refreshInstanceState(instanceState *state.InstanceState) {
	// Update renderer's reference to instance state
	m.detailsRenderer.PostDeployInstanceState = instanceState

	// Hydrate existing resource items with updated ResourceState
	for i := range m.items {
		m.hydrateItemFromState(&m.items[i], instanceState)
	}
}

// hydrateItemFromState updates a deploy item with the latest state data.
func (m *DeployModel) hydrateItemFromState(item *DeployItem, instanceState *state.InstanceState) {
	switch item.Type {
	case ItemTypeResource:
		m.hydrateResourceFromState(item, instanceState)
	case ItemTypeChild:
		m.hydrateChildFromState(item, instanceState)
	case ItemTypeLink:
		m.hydrateLinkFromState(item, instanceState)
	}
}

func (m *DeployModel) hydrateResourceFromState(item *DeployItem, instanceState *state.InstanceState) {
	if item.Resource == nil || instanceState == nil {
		return
	}

	// Find the resource state in the instance
	if instanceState.ResourceIDs == nil || instanceState.Resources == nil {
		return
	}
	resourceID, ok := instanceState.ResourceIDs[item.Resource.Name]
	if !ok {
		return
	}
	resourceState := instanceState.Resources[resourceID]
	if resourceState != nil {
		item.Resource.ResourceState = resourceState
		if item.Resource.ResourceID == "" {
			item.Resource.ResourceID = resourceState.ResourceID
		}
		if item.Resource.ResourceType == "" {
			item.Resource.ResourceType = resourceState.Type
		}
	}

	// Update the item's instance state reference
	item.InstanceState = instanceState
}

func (m *DeployModel) hydrateChildFromState(item *DeployItem, instanceState *state.InstanceState) {
	if item.Child == nil || instanceState == nil {
		return
	}

	// Find the child instance state
	if instanceState.ChildBlueprints == nil {
		return
	}
	childState := instanceState.ChildBlueprints[item.Child.Name]
	if childState != nil {
		if item.Child.ChildInstanceID == "" {
			item.Child.ChildInstanceID = childState.InstanceID
		}
		item.InstanceState = childState
	}
}

func (m *DeployModel) hydrateLinkFromState(item *DeployItem, instanceState *state.InstanceState) {
	if item.Link == nil || instanceState == nil {
		return
	}

	// Find the link state
	if instanceState.Links == nil {
		return
	}
	linkState := instanceState.Links[item.Link.LinkName]
	if linkState != nil {
		if item.Link.LinkID == "" {
			item.Link.LinkID = linkState.LinkID
		}
	}

	item.InstanceState = instanceState
}

func (m DeployModel) handleDriftDetected(msg driftui.DriftDetectedMsg) (tea.Model, tea.Cmd) {
	m.driftReviewMode = true
	m.driftResult = msg.ReconciliationResult
	m.driftMessage = msg.Message
	m.driftBlockedChangesetID = msg.ChangesetID
	m.driftContext = driftui.DriftContextDeploy
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

func (m DeployModel) handleReconciliationComplete() (tea.Model, tea.Cmd) {
	m.driftReviewMode = false
	m.driftResult = nil
	m.driftMessage = ""

	if m.driftBlockedChangesetID != "" {
		m.changesetID = m.driftBlockedChangesetID
	}
	m.driftBlockedChangesetID = ""

	m.streaming = true

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, continueDeploymentCmd(m)
}

func (m DeployModel) handleReconciliationError(msg driftui.ReconciliationErrorMsg) (tea.Model, tea.Cmd) {
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

func (m DeployModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle error state
	if m.err != nil {
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	// Handle deployment overview view
	if m.showingOverview {
		return m.handleOverviewKeyMsg(msg)
	}

	// Handle spec view
	if m.showingSpecView {
		return m.handleSpecViewKeyMsg(msg)
	}

	// Handle exports view
	if m.showingExportsView {
		return m.handleExportsViewKeyMsg(msg)
	}

	// Handle pre-rollback state view
	if m.showingPreRollbackState {
		return m.handlePreRollbackStateViewKeyMsg(msg)
	}

	// Toggle exports view - available when instance state is available
	if msg.String() == "e" || msg.String() == "E" {
		instanceState := m.postDeployInstanceState
		if instanceState == nil {
			instanceState = m.preDeployInstanceState
		}
		if instanceState != nil {
			m.showingExportsView = true
			m.exportsModel = NewExportsModel(
				instanceState,
				m.instanceName,
				m.width, m.height,
				m.styles,
			)
			// Initialize the split pane with current window size
			m.exportsModel, _ = m.exportsModel.Update(tea.WindowSizeMsg{
				Width:  m.width,
				Height: m.height,
			})
			return m, nil
		}
	}

	// Toggle pre-rollback state view - available when pre-rollback state was captured
	if msg.String() == "r" || msg.String() == "R" {
		if m.preRollbackState != nil {
			m.showingPreRollbackState = true
			m.preRollbackStateViewport.SetContent(m.renderPreRollbackStateContent())
			m.preRollbackStateViewport.GotoTop()
			return m, nil
		}
	}

	// Toggle spec view when a resource with spec data is selected
	// (available during deployment once the resource has completed and state is refreshed)
	if msg.String() == "s" || msg.String() == "S" {
		resourceState, resourceName := m.getSelectedResourceState()
		if resourceState != nil && resourceState.SpecData != nil {
			m.showingSpecView = true
			m.specViewport.SetContent(m.renderSpecContent(resourceState, resourceName))
			m.specViewport.GotoTop()
			return m, nil
		}
	}

	// Toggle deployment overview when deployment has finished
	if m.finished {
		if msg.String() == "o" || msg.String() == "O" {
			m.showingOverview = true
			m.overviewViewport.SetContent(m.renderOverviewContent())
			m.overviewViewport.GotoTop()
			return m, nil
		}
	}

	// Handle drift review mode
	if m.driftReviewMode {
		return m.handleDriftReviewKeyMsg(msg)
	}

	// Delegate to split-pane
	var cmd tea.Cmd
	m.splitPane, cmd = m.splitPane.Update(msg)
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, cmd
}

func (m DeployModel) handleOverviewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m DeployModel) handleSpecViewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result := shared.HandleViewportKeyMsg(msg, m.specViewport, "s", "S")
	if result.ShouldQuit {
		return m, tea.Quit
	}
	if result.ShouldClose {
		m.showingSpecView = false
		return m, nil
	}
	m.specViewport = result.Viewport
	return m, result.Cmd
}

func (m DeployModel) handleExportsViewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch shared.CheckExportsKeyMsg(msg) {
	case shared.ExportsKeyActionQuit:
		return m, tea.Quit
	case shared.ExportsKeyActionClose:
		m.showingExportsView = false
		return m, nil
	default:
		var cmd tea.Cmd
		m.exportsModel, cmd = m.exportsModel.Update(msg)
		return m, cmd
	}
}

func (m DeployModel) handlePreRollbackStateViewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result := shared.HandleViewportKeyMsg(msg, m.preRollbackStateViewport, "r", "R")
	if result.ShouldQuit {
		return m, tea.Quit
	}
	if result.ShouldClose {
		m.showingPreRollbackState = false
		return m, nil
	}
	m.preRollbackStateViewport = result.Viewport
	return m, result.Cmd
}

func (m DeployModel) handleDriftReviewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a", "A":
		return m, applyReconciliationCmd(m)
	case "q":
		return m, tea.Quit
	default:
		// Delegate other keys (including esc) to drift split pane
		// The split pane handles esc for back navigation when in nested views
		var cmd tea.Cmd
		m.driftSplitPane, cmd = m.driftSplitPane.Update(msg)
		m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
		return m, cmd
	}
}

func (m DeployModel) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.driftReviewMode {
		m.driftSplitPane, cmd = m.driftSplitPane.Update(msg)
	} else {
		m.splitPane, cmd = m.splitPane.Update(msg)
	}
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, cmd
}

func (m *DeployModel) processEvent(event *types.BlueprintInstanceEvent) {
	printHeadless := m.headlessMode && !m.jsonMode
	shared.DispatchBlueprintEvent(event, shared.BlueprintEventHandlers{
		OnResourceUpdate: func(data *container.ResourceDeployUpdateMessage) {
			m.processResourceUpdate(data)
			if printHeadless {
				m.printHeadlessResourceEvent(data)
			}
		},
		OnChildUpdate: func(data *container.ChildDeployUpdateMessage) {
			m.processChildUpdate(data)
			if printHeadless {
				m.printHeadlessChildEvent(data)
			}
		},
		OnLinkUpdate: func(data *container.LinkDeployUpdateMessage) {
			m.processLinkUpdate(data)
			if printHeadless {
				m.printHeadlessLinkEvent(data)
			}
		},
		OnInstanceUpdate: m.processInstanceUpdate,
		OnPreRollbackState: func(data *container.PreRollbackStateMessage) {
			m.processPreRollbackState(data)
			if printHeadless {
				m.printHeadlessPreRollbackState(data)
			}
		},
	})
}

// View renders the deploy model.
func (m DeployModel) View() string {
	if m.headlessMode {
		return ""
	}

	if m.destroyChangesetError {
		return m.renderDestroyChangesetError()
	}

	if m.err != nil {
		return m.renderError(m.err)
	}

	// Show full-screen deployment overview
	if m.showingOverview {
		return m.renderOverviewView()
	}

	// Show full-screen spec view
	if m.showingSpecView {
		return m.renderSpecView()
	}

	// Show full-screen exports view
	if m.showingExportsView {
		return m.exportsModel.View()
	}

	// Show full-screen pre-rollback state view
	if m.showingPreRollbackState {
		return m.renderPreRollbackStateView()
	}

	// Show drift review mode
	if m.driftReviewMode {
		return m.driftSplitPane.View()
	}

	// Always show split-pane (even during streaming)
	return m.splitPane.View()
}

// SetChangesetChanges sets the changeset changes and rebuilds items from them.
// This is called when staging completes with the full changeset data.
func (m *DeployModel) SetChangesetChanges(changesetChanges *changes.BlueprintChanges) {
	if changesetChanges == nil {
		return
	}
	m.changesetChanges = changesetChanges
	m.items = BuildItemsFromChangeset(changesetChanges, m.resourcesByName, m.childrenByName, m.linksByName, m.preDeployInstanceState)
	m.splitPane.SetItems(ToSplitPaneItems(m.items))
}

// SetPreDeployInstanceState sets the pre-deployment instance state.
// This is called when staging completes with the instance state for displaying unchanged items.
func (m *DeployModel) SetPreDeployInstanceState(instanceState *state.InstanceState) {
	m.preDeployInstanceState = instanceState
	// Also pass to details renderer for pre-deploy lookups
	m.detailsRenderer.PreDeployInstanceState = instanceState
	// Update footer to show exports hint when instance state is available
	if instanceState != nil {
		m.footerRenderer.HasInstanceState = true
	}
}

// NewDeployModel creates a new deploy model.
func NewDeployModel(
	deployEngine engine.DeployEngine,
	logger *zap.Logger,
	changesetID string,
	instanceID string,
	instanceName string,
	blueprintFile string,
	blueprintSource string,
	autoRollback bool,
	force bool,
	styles *stylespkg.Styles,
	isHeadless bool,
	headlessWriter io.Writer,
	changesetChanges *changes.BlueprintChanges,
	jsonMode bool,
) DeployModel {
	detailsRenderer, sectionGrouper, footerRenderer := createDeployRenderers(instanceID, instanceName, changesetID)
	splitPaneConfig := createDeploySplitPaneConfig(styles, detailsRenderer, sectionGrouper, footerRenderer)

	driftDetailsRenderer, driftSectionGrouper, driftFooterRenderer := createDriftRenderers()
	driftSplitPaneConfig := createDriftSplitPaneConfig(styles, driftDetailsRenderer, driftSectionGrouper, driftFooterRenderer)

	printer := createHeadlessPrinter(isHeadless, headlessWriter)

	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)
	items := BuildItemsFromChangeset(changesetChanges, resourcesByName, childrenByName, linksByName, nil)

	model := DeployModel{
		splitPane:               splitpane.New(splitPaneConfig),
		detailsRenderer:         detailsRenderer,
		sectionGrouper:          sectionGrouper,
		footerRenderer:          footerRenderer,
		driftSplitPane:          splitpane.New(driftSplitPaneConfig),
		driftDetailsRenderer:    driftDetailsRenderer,
		driftSectionGrouper:     driftSectionGrouper,
		driftFooterRenderer:     driftFooterRenderer,
		engine:                  deployEngine,
		logger:                  logger,
		changesetID:             changesetID,
		instanceID:              instanceID,
		instanceName:            instanceName,
		blueprintFile:           blueprintFile,
		blueprintSource:         blueprintSource,
		autoRollback:            autoRollback,
		force:                   force,
		changesetChanges:        changesetChanges,
		styles:                  styles,
		headlessMode:            isHeadless,
		headlessWriter:          headlessWriter,
		printer:                 printer,
		jsonMode:                jsonMode,
		spinner:                 createDeploySpinner(styles),
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

func createDeployRenderers(instanceID, instanceName, changesetID string) (*DeployDetailsRenderer, *DeploySectionGrouper, *DeployFooterRenderer) {
	detailsRenderer := &DeployDetailsRenderer{
		MaxExpandDepth:       MaxExpandDepth,
		NavigationStackDepth: 0,
	}
	sectionGrouper := &DeploySectionGrouper{
		SectionGrouper: shared.SectionGrouper{MaxExpandDepth: MaxExpandDepth},
	}
	footerRenderer := &DeployFooterRenderer{
		InstanceID:   instanceID,
		InstanceName: instanceName,
		ChangesetID:  changesetID,
	}
	return detailsRenderer, sectionGrouper, footerRenderer
}

func createDeploySplitPaneConfig(
	styles *stylespkg.Styles,
	detailsRenderer *DeployDetailsRenderer,
	sectionGrouper *DeploySectionGrouper,
	footerRenderer *DeployFooterRenderer,
) splitpane.Config {
	return splitpane.Config{
		Styles:          styles,
		Title:           "Deployment",
		DetailsRenderer: detailsRenderer,
		LeftPaneRatio:   0.4,
		MaxExpandDepth:  MaxExpandDepth,
		SectionGrouper:  sectionGrouper,
		FooterRenderer:  footerRenderer,
	}
}

func createDriftRenderers() (*DriftDetailsRenderer, *DriftSectionGrouper, *DriftFooterRenderer) {
	driftDetailsRenderer := &DriftDetailsRenderer{
		MaxExpandDepth:       MaxExpandDepth,
		NavigationStackDepth: 0,
	}
	driftSectionGrouper := &DriftSectionGrouper{
		MaxExpandDepth: MaxExpandDepth,
	}
	driftFooterRenderer := &DriftFooterRenderer{
		Context: driftui.DriftContextDeploy,
	}
	return driftDetailsRenderer, driftSectionGrouper, driftFooterRenderer
}

func createDriftSplitPaneConfig(
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

func createHeadlessPrinter(isHeadless bool, headlessWriter io.Writer) *headless.Printer {
	if !isHeadless || headlessWriter == nil {
		return nil
	}
	prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[deploy] ")
	return headless.NewPrinter(prefixedWriter, 80)
}

func createDeploySpinner(styles *stylespkg.Styles) spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner
	return s
}

// Updates items that were never attempted (still in unknown/pending state)
// to indicate they were skipped due to deployment failure.
// Items with ActionNoChange are excluded since they were never meant to be deployed.
func (m *DeployModel) markPendingItemsAsSkipped() {
	shared.MarkPendingResourcesAsSkipped(m.resourcesByName)
	shared.MarkPendingChildrenAsSkipped(m.childrenByName)
	shared.MarkPendingLinksAsSkipped(m.linksByName)
}

// Updates items that are stuck in an in-progress state
// (e.g., CREATING, DEPLOYING, UPDATING) to indicate they were interrupted.
// This handles the case where nested child blueprint resources never receive a terminal
// status because the drain logic only operates on the root blueprint's deployment state.
// Items with ActionNoChange are excluded since they were never meant to be deployed.
func (m *DeployModel) markInProgressItemsAsInterrupted() {
	// Mark in-progress resources as interrupted
	for _, item := range m.resourcesByName {
		// Skip items that have no changes - they were never meant to be deployed
		if item.Action == ActionNoChange {
			continue
		}
		if IsInProgressResourceStatus(item.Status) {
			status, preciseStatus := determineResourceInterruptedStatusFromAction(item.Action, item.Status)
			item.Status = status
			item.PreciseStatus = preciseStatus
		}
	}
	// Mark in-progress children as interrupted
	for _, item := range m.childrenByName {
		// Skip items that have no changes - they were never meant to be deployed
		if item.Action == ActionNoChange {
			continue
		}
		if IsInProgressInstanceStatus(item.Status) {
			item.Status = determineChildInterruptedStatusFromAction(item.Action, item.Status)
		}
	}
	// Mark in-progress links as interrupted
	for _, item := range m.linksByName {
		// Skip items that have no changes - they were never meant to be deployed
		if item.Action == ActionNoChange {
			continue
		}
		if IsInProgressLinkStatus(item.Status) {
			status, preciseStatus := determineLinkInterruptedStatusFromAction(item.Action, item.Status)
			item.Status = status
			item.PreciseStatus = preciseStatus
		}
	}
}

// Returns the appropriate interrupted status
// based on the action type and current in-progress status.
func determineResourceInterruptedStatusFromAction(
	action ActionType,
	currentStatus core.ResourceStatus,
) (core.ResourceStatus, core.PreciseResourceStatus) {
	// If destroying, it's a destroy interruption
	if currentStatus == core.ResourceStatusDestroying {
		return core.ResourceStatusDestroyInterrupted, core.PreciseResourceStatusDestroyInterrupted
	}

	// For CREATE actions (new elements), use CreateInterrupted
	if action == ActionCreate {
		return core.ResourceStatusCreateInterrupted, core.PreciseResourceStatusCreateInterrupted
	}

	// For RECREATE, if we're in the creating phase, use CreateInterrupted
	if action == ActionRecreate && currentStatus == core.ResourceStatusCreating {
		return core.ResourceStatusCreateInterrupted, core.PreciseResourceStatusCreateInterrupted
	}

	// For UPDATE, RECREATE (update phase), or unknown actions, use UpdateInterrupted
	return core.ResourceStatusUpdateInterrupted, core.PreciseResourceStatusUpdateInterrupted
}

// Returns the appropriate interrupted status
// for a child blueprint based on the action type and current status.
func determineChildInterruptedStatusFromAction(
	action ActionType,
	currentStatus core.InstanceStatus,
) core.InstanceStatus {
	// If destroying, it's a destroy interruption
	if currentStatus == core.InstanceStatusDestroying ||
		currentStatus == core.InstanceStatusDestroyRollingBack {
		return core.InstanceStatusDestroyInterrupted
	}

	// For CREATE actions (new child blueprints), use DeployInterrupted
	if action == ActionCreate {
		return core.InstanceStatusDeployInterrupted
	}

	// For RECREATE, if we're in the deploying phase, use DeployInterrupted
	if action == ActionRecreate && currentStatus == core.InstanceStatusDeploying {
		return core.InstanceStatusDeployInterrupted
	}

	// For UPDATE, RECREATE (update phase), or unknown actions, use UpdateInterrupted
	return core.InstanceStatusUpdateInterrupted
}

// Returns the appropriate interrupted status
// for a link based on the action type and current status.
func determineLinkInterruptedStatusFromAction(
	action ActionType,
	currentStatus core.LinkStatus,
) (core.LinkStatus, core.PreciseLinkStatus) {
	// If destroying, it's a destroy interruption
	if currentStatus == core.LinkStatusDestroying {
		return core.LinkStatusDestroyInterrupted, core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted
	}

	// For CREATE actions (new links), use CreateInterrupted
	if action == ActionCreate {
		return core.LinkStatusCreateInterrupted, core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted
	}

	// For RECREATE, if we're in the creating phase, use CreateInterrupted
	if action == ActionRecreate && currentStatus == core.LinkStatusCreating {
		return core.LinkStatusCreateInterrupted, core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted
	}

	// For UPDATE, RECREATE (update phase), or unknown actions, use UpdateInterrupted
	return core.LinkStatusUpdateInterrupted, core.PreciseLinkStatusResourceAUpdateInterrupted
}

// Test accessor methods - these provide read-only access for testing purposes.

// Err returns the error stored in the model.
func (m *DeployModel) Err() error {
	return m.err
}

// FinalStatus returns the final instance status after deployment.
func (m *DeployModel) FinalStatus() core.InstanceStatus {
	return m.finalStatus
}

// Items returns the deployment items.
func (m *DeployModel) Items() []DeployItem {
	return m.items
}

// ResourcesByName returns the resources lookup map.
func (m *DeployModel) ResourcesByName() map[string]*ResourceDeployItem {
	return m.resourcesByName
}
