package deployui

import (
	"fmt"
	"io"
	"log"
	"strings"

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
)

// MaxExpandDepth is the maximum nesting depth for expanding child blueprints.
const MaxExpandDepth = 2

// ElementFailure represents a failure for a specific element with its root cause reasons.
type ElementFailure struct {
	ElementName    string
	ElementPath    string // Full path like "children.notifications::resources.notificationQueue"
	ElementType    string // "resource", "child", or "link"
	FailureReasons []string
}

// InterruptedElement represents an element that was interrupted during deployment.
type InterruptedElement struct {
	ElementName string
	ElementPath string // Full path like "children.notifications::resources.notificationQueue"
	ElementType string // "resource", "child", or "link"
}

// SuccessfulElement represents an element that completed successfully.
type SuccessfulElement struct {
	ElementName string
	ElementPath string // Full path like "children.notifications::resources.notificationQueue"
	ElementType string // "resource", "child", or "link"
	Action      string // "created", "updated", "destroyed", etc.
}

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
	instanceID              string
	instanceName            string
	changesetID             string
	streaming               bool
	fetchingPreDeployState  bool // True while fetching pre-deploy instance state
	finished                bool
	finalStatus          core.InstanceStatus
	failureReasons       []string             // Legacy: generic failure messages from backend
	elementFailures      []ElementFailure     // Structured failures with root cause details
	interruptedElements  []InterruptedElement // Elements that were interrupted
	successfulElements   []SuccessfulElement  // Elements that completed successfully
	err                  error
	showingOverview      bool           // When true, show full-screen deployment overview
	overviewViewport     viewport.Model // Scrollable viewport for deployment overview
	showingSpecView      bool           // When true, show full-screen spec view for selected resource
	specViewport         viewport.Model // Scrollable viewport for spec view
	showingExportsView   bool           // When true, show full-screen exports view
	exportsModel         ExportsModel   // Exports view model with split pane
	preDeployInstanceState  *state.InstanceState // Instance state fetched before deployment for unchanged items
	postDeployInstanceState *state.InstanceState // Instance state fetched after deployment completes

	// Streaming channels
	engine      engine.DeployEngine
	eventStream chan types.BlueprintInstanceEvent
	errStream   chan error

	// Config
	blueprintFile   string
	blueprintSource string
	asRollback      bool
	autoRollback    bool
	force           bool

	// Changeset data - used to build item hierarchy
	changesetChanges *changes.BlueprintChanges

	// Headless mode
	headlessMode   bool
	headlessWriter io.Writer
	printer        *headless.Printer

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
	case DeployStreamClosedMsg:
		return m.handleDeployStreamClosed()
	case PreDeployInstanceStateFetchedMsg:
		return m.handlePreDeployInstanceStateFetched(msg)
	case PostDeployInstanceStateFetchedMsg:
		return m.handlePostDeployInstanceStateFetched(msg)
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
		m.items = buildItemsFromChangeset(m.changesetChanges, m.resourcesByName, m.childrenByName, m.linksByName, m.preDeployInstanceState)
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

	if m.headlessMode {
		m.printHeadlessHeader()
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(waitForNextDeployEventCmd(m), checkForErrCmd(m))
}

func (m DeployModel) handleDeployEvent(msg DeployEventMsg) (tea.Model, tea.Cmd) {
	event := types.BlueprintInstanceEvent(msg)
	m.processEvent(&event)
	m.splitPane.SetItems(ToSplitPaneItems(m.items))

	cmds := []tea.Cmd{checkForErrCmd(m)}

	finishData, isFinish := event.AsFinish()
	if !isFinish {
		cmds = append(cmds, waitForNextDeployEventCmd(m))
		m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
		return m, tea.Batch(cmds...)
	}

	m.handleDeployFinish(finishData)

	if m.headlessMode {
		m.printHeadlessSummary()
		cmds = append(cmds, tea.Quit)
	} else {
		// Fetch updated instance state for outputs display (only in interactive mode)
		cmds = append(cmds, fetchPostDeployInstanceStateCmd(m))
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(cmds...)
}

func (m *DeployModel) handleDeployFinish(finishData *container.DeploymentFinishedMessage) {
	m.finished = true
	m.finalStatus = finishData.Status
	m.failureReasons = finishData.FailureReasons
	m.footerRenderer.FinalStatus = finishData.Status
	m.footerRenderer.Finished = true
	m.detailsRenderer.Finished = true

	if isFailedStatus(finishData.Status) {
		m.markPendingItemsAsSkipped()
		m.markInProgressItemsAsInterrupted()
		m.splitPane.SetItems(ToSplitPaneItems(m.items))
	}

	m.collectDeploymentResults()
	m.footerRenderer.SuccessfulElements = m.successfulElements
	m.footerRenderer.ElementFailures = m.elementFailures
	m.footerRenderer.InterruptedElements = m.interruptedElements
}

func (m DeployModel) handleDeployError(msg DeployErrorMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		return m, nil
	}

	m.err = msg.Err
	if m.headlessMode {
		m.printHeadlessError(msg.Err)
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
			m.printHeadlessError(m.err)
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
	// Update footer to show exports hint when instance state is available
	if msg.InstanceState != nil {
		m.footerRenderer.HasInstanceState = true
	}
	// Update exports view if it's currently showing
	if m.showingExportsView {
		m.exportsModel.UpdateInstanceState(msg.InstanceState)
	}
	return m, nil
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
		m.printHeadlessDriftDetected()
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
		m.printHeadlessError(msg.Err)
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

	// Toggle deployment overview when deployment has finished
	if m.finished {
		if msg.String() == "o" || msg.String() == "O" {
			m.showingOverview = true
			m.overviewViewport.SetContent(m.renderOverviewContent())
			m.overviewViewport.GotoTop()
			return m, nil
		}
		// Toggle spec view when a resource is selected
		if msg.String() == "s" || msg.String() == "S" {
			resourceState, resourceName := m.getSelectedResourceState()
			if resourceState != nil && resourceState.SpecData != nil {
				m.showingSpecView = true
				m.specViewport.SetContent(m.renderSpecContent(resourceState, resourceName))
				m.specViewport.GotoTop()
				return m, nil
			}
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

func (m *DeployModel) hasFailuresOrInterruptions() bool {
	return len(m.failureReasons) > 0 || len(m.elementFailures) > 0 || len(m.interruptedElements) > 0
}

func (m DeployModel) handleOverviewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "o", "O":
		m.showingOverview = false
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.overviewViewport, cmd = m.overviewViewport.Update(msg)
		return m, cmd
	}
}

func (m DeployModel) handleSpecViewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "s", "S":
		m.showingSpecView = false
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.specViewport, cmd = m.specViewport.Update(msg)
		return m, cmd
	}
}

func (m DeployModel) handleExportsViewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "e", "E":
		m.showingExportsView = false
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.exportsModel, cmd = m.exportsModel.Update(msg)
		return m, cmd
	}
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
	if resourceData, ok := event.AsResourceUpdate(); ok {
		m.processResourceUpdate(resourceData)
		if m.headlessMode {
			m.printHeadlessResourceEvent(resourceData)
		}
	} else if childData, ok := event.AsChildUpdate(); ok {
		m.processChildUpdate(childData)
		if m.headlessMode {
			m.printHeadlessChildEvent(childData)
		}
	} else if linkData, ok := event.AsLinkUpdate(); ok {
		m.processLinkUpdate(linkData)
		if m.headlessMode {
			m.printHeadlessLinkEvent(linkData)
		}
	} else if instanceData, ok := event.AsInstanceUpdate(); ok {
		m.processInstanceUpdate(instanceData)
	}
}

func (m *DeployModel) processResourceUpdate(data *container.ResourceDeployUpdateMessage) {
	// Check if this resource belongs to a child instance (not root)
	isRootResource := data.InstanceID == "" || data.InstanceID == m.instanceID

	// Build path-based key for unique identification
	resourcePath := m.buildResourcePath(data.InstanceID, data.ResourceName)

	log.Printf("DEBUG processResourceUpdate: resource=%s path=%s instanceID=%s isRoot=%v status=%v\n",
		data.ResourceName, resourcePath, data.InstanceID, isRootResource, data.Status)

	item, exists := m.resourcesByName[resourcePath]
	if !exists {
		// Also try simple name lookup for backwards compatibility with pre-populated items
		item, exists = m.resourcesByName[data.ResourceName]
		if exists {
			// Migrate the item to use the path-based key
			delete(m.resourcesByName, data.ResourceName)
			m.resourcesByName[resourcePath] = item
			log.Printf("DEBUG processResourceUpdate: migrated resource=%s to path=%s\n", data.ResourceName, resourcePath)
		}
	}

	if !exists {
		log.Printf("DEBUG processResourceUpdate: creating new item for resource=%s path=%s\n", data.ResourceName, resourcePath)
		// First time seeing this resource - create it
		item = &ResourceDeployItem{
			Name:       data.ResourceName,
			ResourceID: data.ResourceID,
			Group:      data.Group,
		}
		m.resourcesByName[resourcePath] = item
		// Only add to top-level items if this belongs to the root instance
		if isRootResource {
			m.items = append(m.items, DeployItem{
				Type:     ItemTypeResource,
				Resource: item,
			})
		}
	} else {
		log.Printf("DEBUG processResourceUpdate: updating existing item for resource=%s path=%s oldStatus=%v newStatus=%v\n",
			data.ResourceName, resourcePath, item.Status, data.Status)
	}

	// Update with exact types from the event, but correct interrupted statuses
	// based on the action type since the backend may not have that context
	status, preciseStatus := data.Status, data.PreciseStatus
	if isInterruptedResourceStatus(data.Status) {
		status, preciseStatus = determineResourceInterruptedStatusFromAction(item.Action, data.Status)
	}
	item.Status = status
	item.PreciseStatus = preciseStatus
	item.FailureReasons = data.FailureReasons
	item.Attempt = data.Attempt
	item.CanRetry = data.CanRetry
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

func (m *DeployModel) processChildUpdate(data *container.ChildDeployUpdateMessage) {
	// Track the child instance ID -> child name and parent ID mappings
	if data.ChildInstanceID != "" && data.ChildName != "" {
		m.instanceIDToChildName[data.ChildInstanceID] = data.ChildName
		m.instanceIDToParentID[data.ChildInstanceID] = data.ParentInstanceID
	}

	// Build the full path for this child (e.g., "parentChild/thisChild")
	childPath := m.buildInstancePath(data.ParentInstanceID, data.ChildName)

	log.Printf("DEBUG processChildUpdate: child=%s path=%s parentInstanceID=%s childInstanceID=%s status=%v\n",
		data.ChildName, childPath, data.ParentInstanceID, data.ChildInstanceID, data.Status)

	// Use path-based key for unique identification
	item, exists := m.childrenByName[childPath]
	if !exists {
		// Also try simple name lookup for backwards compatibility with pre-populated items
		item, exists = m.childrenByName[data.ChildName]
		if exists {
			// Migrate the item to use the path-based key
			delete(m.childrenByName, data.ChildName)
			m.childrenByName[childPath] = item
			log.Printf("DEBUG processChildUpdate: migrated child=%s to path=%s\n", data.ChildName, childPath)
		}
	}

	if !exists {
		log.Printf("DEBUG processChildUpdate: creating new item for child=%s path=%s\n", data.ChildName, childPath)
		// Only add to top-level items if the parent is the root instance
		// (i.e., this is a direct child of the root blueprint)
		isDirectChildOfRoot := data.ParentInstanceID == "" || data.ParentInstanceID == m.instanceID

		// Try to get Changes from changeset data for this child
		var childChanges *changes.BlueprintChanges
		if m.changesetChanges != nil {
			if nc, ok := m.changesetChanges.NewChildren[data.ChildName]; ok {
				childChanges = &changes.BlueprintChanges{
					NewResources: nc.NewResources,
					NewChildren:  nc.NewChildren,
				}
			} else if cc, ok := m.changesetChanges.ChildChanges[data.ChildName]; ok {
				ccCopy := cc
				childChanges = &ccCopy
			}
		}

		item = &ChildDeployItem{
			Name:             data.ChildName,
			ParentInstanceID: data.ParentInstanceID,
			ChildInstanceID:  data.ChildInstanceID,
			Group:            data.Group,
			Changes:          childChanges,
		}
		m.childrenByName[childPath] = item
		if isDirectChildOfRoot {
			m.items = append(m.items, DeployItem{
				Type:            ItemTypeChild,
				Child:           item,
				Changes:         childChanges,
				childrenByName:  m.childrenByName,
				resourcesByName: m.resourcesByName,
				linksByName:     m.linksByName,
			})
		}
	} else {
		log.Printf("DEBUG processChildUpdate: updating existing item for child=%s path=%s oldStatus=%v newStatus=%v\n",
			data.ChildName, childPath, item.Status, data.Status)
	}

	// Track the path for this child name (used by resource/link lookups)
	m.childNameToInstancePath[data.ChildName] = childPath

	// Correct interrupted statuses based on the action type since the backend
	// may not have context for nested child blueprint elements
	status := data.Status
	if isInterruptedInstanceStatus(data.Status) {
		status = determineChildInterruptedStatusFromAction(item.Action, data.Status)
	}
	item.Status = status
	item.FailureReasons = data.FailureReasons
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

func (m *DeployModel) processLinkUpdate(data *container.LinkDeployUpdateMessage) {
	// Check if this link belongs to a child instance (not root)
	isRootLink := data.InstanceID == "" || data.InstanceID == m.instanceID

	// Build path-based key for unique identification
	linkPath := m.buildResourcePath(data.InstanceID, data.LinkName)

	item, exists := m.linksByName[linkPath]
	if !exists {
		// Also try simple name lookup for backwards compatibility with pre-populated items
		item, exists = m.linksByName[data.LinkName]
		if exists {
			// Migrate the item to use the path-based key
			delete(m.linksByName, data.LinkName)
			m.linksByName[linkPath] = item
		}
	}

	if !exists {
		item = &LinkDeployItem{
			LinkID:        data.LinkID,
			LinkName:      data.LinkName,
			ResourceAName: extractResourceAFromLinkName(data.LinkName),
			ResourceBName: extractResourceBFromLinkName(data.LinkName),
		}
		m.linksByName[linkPath] = item
		// Only add to top-level items if this belongs to the root instance
		if isRootLink {
			m.items = append(m.items, DeployItem{
				Type: ItemTypeLink,
				Link: item,
			})
		}
	}

	item.Status = data.Status
	item.PreciseStatus = data.PreciseStatus
	item.FailureReasons = data.FailureReasons
	item.CurrentStageAttempt = data.CurrentStageAttempt
	item.CanRetryCurrentStage = data.CanRetryCurrentStage
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

func (m *DeployModel) processInstanceUpdate(data *container.DeploymentUpdateMessage) {
	// Update overall deployment status in footer
	m.footerRenderer.CurrentStatus = data.Status

	// If the deployment is rolling back or has failed, mark pending and in-progress items immediately
	// This provides real-time feedback rather than waiting for the finish event
	if isRollingBackOrFailedStatus(data.Status) && !m.finished {
		m.markPendingItemsAsSkipped()
		m.markInProgressItemsAsInterrupted()
	}
}

// buildInstancePath builds a path from instance ID to the child name.
// For root instance resources, returns just the name.
// For nested children, returns a path like "parentChild/childName".
func (m *DeployModel) buildInstancePath(parentInstanceID, childName string) string {
	// If parent is root or empty, this is a direct child
	if parentInstanceID == "" || parentInstanceID == m.instanceID {
		return childName
	}

	// Build path by traversing parent chain
	var pathParts []string
	currentID := parentInstanceID
	for currentID != "" && currentID != m.instanceID {
		if name, ok := m.instanceIDToChildName[currentID]; ok {
			pathParts = append([]string{name}, pathParts...)
			currentID = m.instanceIDToParentID[currentID]
		} else {
			break
		}
	}

	pathParts = append(pathParts, childName)
	return joinPath(pathParts)
}

// buildResourcePath builds a path for a resource based on its instance ID.
// For root instance resources, returns just the resource name.
// For nested resources, returns a path like "parentChild/childName/resourceName".
func (m *DeployModel) buildResourcePath(instanceID, resourceName string) string {
	// If this is a root resource, just use the name
	if instanceID == "" || instanceID == m.instanceID {
		return resourceName
	}

	// Build path by traversing parent chain from the instance
	var pathParts []string
	currentID := instanceID
	for currentID != "" && currentID != m.instanceID {
		if name, ok := m.instanceIDToChildName[currentID]; ok {
			pathParts = append([]string{name}, pathParts...)
			currentID = m.instanceIDToParentID[currentID]
		} else {
			break
		}
	}

	pathParts = append(pathParts, resourceName)
	return joinPath(pathParts)
}

// joinPath joins path parts with a separator.
func joinPath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "/" + parts[i]
	}
	return result
}

// View renders the deploy model.
func (m DeployModel) View() string {
	if m.headlessMode {
		return ""
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
	m.items = buildItemsFromChangeset(changesetChanges, m.resourcesByName, m.childrenByName, m.linksByName, m.preDeployInstanceState)
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
	asRollback bool,
	autoRollback bool,
	force bool,
	styles *stylespkg.Styles,
	isHeadless bool,
	headlessWriter io.Writer,
	changesetChanges *changes.BlueprintChanges,
) DeployModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	detailsRenderer := &DeployDetailsRenderer{
		MaxExpandDepth:       MaxExpandDepth,
		NavigationStackDepth: 0,
	}

	sectionGrouper := &DeploySectionGrouper{
		MaxExpandDepth: MaxExpandDepth,
	}

	footerRenderer := &DeployFooterRenderer{
		InstanceID:   instanceID,
		InstanceName: instanceName,
		ChangesetID:  changesetID,
	}

	splitPaneConfig := splitpane.Config{
		Styles:          styles,
		Title:           "Deployment",
		DetailsRenderer: detailsRenderer,
		LeftPaneRatio:   0.4,
		MaxExpandDepth:  MaxExpandDepth,
		SectionGrouper:  sectionGrouper,
		FooterRenderer:  footerRenderer,
	}

	// Create drift review renderers
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

	// Create drift splitpane config
	driftSplitPaneConfig := splitpane.Config{
		Styles:          styles,
		DetailsRenderer: driftDetailsRenderer,
		Title:           "⚠ Drift Detected",
		LeftPaneRatio:   0.4,
		MaxExpandDepth:  MaxExpandDepth,
		SectionGrouper:  driftSectionGrouper,
		FooterRenderer:  driftFooterRenderer,
	}

	var printer *headless.Printer
	if isHeadless && headlessWriter != nil {
		prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[deploy] ")
		printer = headless.NewPrinter(prefixedWriter, 80)
	}

	// Pre-populate items from changeset if available
	// Note: At NewDeployModel time, we don't have instance state yet.
	// It will be set via SetPreDeployInstanceState when staging completes.
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)
	items := buildItemsFromChangeset(changesetChanges, resourcesByName, childrenByName, linksByName, nil)

	model := DeployModel{
		splitPane:       splitpane.New(splitPaneConfig),
		detailsRenderer: detailsRenderer,
		sectionGrouper:  sectionGrouper,
		footerRenderer:  footerRenderer,
		driftSplitPane:        splitpane.New(driftSplitPaneConfig),
		driftDetailsRenderer:  driftDetailsRenderer,
		driftSectionGrouper:   driftSectionGrouper,
		driftFooterRenderer:   driftFooterRenderer,
		engine:                deployEngine,
		logger:                logger,
		changesetID:           changesetID,
		instanceID:            instanceID,
		instanceName:          instanceName,
		blueprintFile:         blueprintFile,
		asRollback:            asRollback,
		autoRollback:          autoRollback,
		force:                 force,
		changesetChanges:      changesetChanges,
		styles:                styles,
		headlessMode:          isHeadless,
		headlessWriter:        headlessWriter,
		printer:               printer,
		spinner:               s,
		eventStream:            make(chan types.BlueprintInstanceEvent),
		errStream:              make(chan error),
		resourcesByName:        resourcesByName,
		childrenByName:         childrenByName,
		linksByName:            linksByName,
		instanceIDToChildName:  make(map[string]string),
		instanceIDToParentID:   make(map[string]string),
		childNameToInstancePath: make(map[string]string),
		items:                  items,
	}

	// Initialize split pane with pre-populated items
	if len(items) > 0 {
		model.splitPane.SetItems(ToSplitPaneItems(items))
	}

	return model
}

// rollingBackOrFailedStatuses contains instance statuses that indicate
// a rollback is in progress or has completed/failed.
var rollingBackOrFailedStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployFailed:            true,
	core.InstanceStatusUpdateFailed:            true,
	core.InstanceStatusDestroyFailed:           true,
	core.InstanceStatusDeployRollingBack:       true,
	core.InstanceStatusUpdateRollingBack:       true,
	core.InstanceStatusDestroyRollingBack:      true,
	core.InstanceStatusDeployRollbackFailed:    true,
	core.InstanceStatusUpdateRollbackFailed:    true,
	core.InstanceStatusDestroyRollbackFailed:   true,
	core.InstanceStatusDeployRollbackComplete:  true,
	core.InstanceStatusUpdateRollbackComplete:  true,
	core.InstanceStatusDestroyRollbackComplete: true,
}

// isRollingBackOrFailedStatus returns true if the instance status indicates
// a rollback is in progress or has completed/failed.
// This is used to mark pending items as skipped in real-time.
func isRollingBackOrFailedStatus(status core.InstanceStatus) bool {
	return rollingBackOrFailedStatuses[status]
}

// failedStatuses contains instance statuses that indicate a failure
// or a rollback complete (which means the original operation failed).
var failedStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployFailed:  true,
	core.InstanceStatusUpdateFailed:  true,
	core.InstanceStatusDestroyFailed: true,
	// Rollback failed states
	core.InstanceStatusDeployRollbackFailed:  true,
	core.InstanceStatusUpdateRollbackFailed:  true,
	core.InstanceStatusDestroyRollbackFailed: true,
	// Rollback complete states also indicate the original operation failed
	core.InstanceStatusDeployRollbackComplete:  true,
	core.InstanceStatusUpdateRollbackComplete:  true,
	core.InstanceStatusDestroyRollbackComplete: true,
}

// isFailedStatus returns true if the instance status indicates a failure
// or a rollback complete (which means the original operation failed).
// Used to determine final deployment outcome.
func isFailedStatus(status core.InstanceStatus) bool {
	return failedStatuses[status]
}

// markPendingItemsAsSkipped updates items that were never attempted (still in unknown/pending state)
// to indicate they were skipped due to deployment failure.
// Items with ActionNoChange are excluded since they were never meant to be deployed.
func (m *DeployModel) markPendingItemsAsSkipped() {
	// Mark pending items in the shared maps (includes nested items)
	for _, item := range m.resourcesByName {
		// Skip items that have no changes - they were never meant to be deployed
		if item.Action == ActionNoChange {
			continue
		}
		// ResourceStatusUnknown is iota = 0, the initial/pending state
		if item.Status == core.ResourceStatusUnknown {
			item.Skipped = true
		}
	}

	for _, item := range m.childrenByName {
		// Skip items that have no changes - they were never meant to be deployed
		if item.Action == ActionNoChange {
			continue
		}
		// InstanceStatusPreparing is iota = 0, InstanceStatusNotDeployed is also a pending state
		if item.Status == core.InstanceStatusPreparing || item.Status == core.InstanceStatusNotDeployed {
			item.Skipped = true
		}
	}

	for _, item := range m.linksByName {
		// Skip items that have no changes - they were never meant to be deployed
		if item.Action == ActionNoChange {
			continue
		}
		// LinkStatusUnknown is iota = 0, the initial/pending state
		if item.Status == core.LinkStatusUnknown {
			item.Skipped = true
		}
	}
}

// markInProgressItemsAsInterrupted updates items that are stuck in an in-progress state
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
		if isInProgressResourceStatus(item.Status) {
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
		if isInProgressInstanceStatus(item.Status) {
			item.Status = determineChildInterruptedStatusFromAction(item.Action, item.Status)
		}
	}
	// Mark in-progress links as interrupted
	for _, item := range m.linksByName {
		// Skip items that have no changes - they were never meant to be deployed
		if item.Action == ActionNoChange {
			continue
		}
		if isInProgressLinkStatus(item.Status) {
			status, preciseStatus := determineLinkInterruptedStatusFromAction(item.Action, item.Status)
			item.Status = status
			item.PreciseStatus = preciseStatus
		}
	}
}

// determineResourceInterruptedStatusFromAction returns the appropriate interrupted status
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

// determineChildInterruptedStatusFromAction returns the appropriate interrupted status
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

// determineLinkInterruptedStatusFromAction returns the appropriate interrupted status
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

// isInProgressResourceStatus returns true if the resource status indicates
// the resource is still being processed (not in a terminal state).
func isInProgressResourceStatus(status core.ResourceStatus) bool {
	switch status {
	case core.ResourceStatusCreating,
		core.ResourceStatusUpdating,
		core.ResourceStatusDestroying,
		core.ResourceStatusRollingBack:
		return true
	}
	return false
}

// isInProgressInstanceStatus returns true if the instance status indicates
// the child blueprint is still being processed (not in a terminal state).
func isInProgressInstanceStatus(status core.InstanceStatus) bool {
	switch status {
	case core.InstanceStatusDeploying,
		core.InstanceStatusUpdating,
		core.InstanceStatusDestroying,
		core.InstanceStatusDeployRollingBack,
		core.InstanceStatusUpdateRollingBack,
		core.InstanceStatusDestroyRollingBack:
		return true
	}
	return false
}

// isInProgressLinkStatus returns true if the link status indicates
// the link is still being processed (not in a terminal state).
func isInProgressLinkStatus(status core.LinkStatus) bool {
	switch status {
	case core.LinkStatusCreating,
		core.LinkStatusUpdating,
		core.LinkStatusDestroying,
		core.LinkStatusCreateRollingBack,
		core.LinkStatusUpdateRollingBack,
		core.LinkStatusDestroyRollingBack:
		return true
	}
	return false
}

// collectDeploymentResults scans all items to collect successful operations,
// failures, and interrupted elements. This provides the data for the deployment overview.
// It traverses the hierarchy to build full element paths.
func (m *DeployModel) collectDeploymentResults() {
	var successful []SuccessfulElement
	var failures []ElementFailure
	var interrupted []InterruptedElement

	// Traverse the hierarchy starting from top-level items
	collectFromItems(m.items, "", m.resourcesByName, m.childrenByName, m.linksByName, &successful, &failures, &interrupted)

	m.successfulElements = successful
	m.elementFailures = failures
	m.interruptedElements = interrupted
}

// collectFromItems recursively collects successful operations, failures, and interruptions from items,
// building full paths as it traverses the hierarchy.
func collectFromItems(
	items []DeployItem,
	parentPath string,
	resourcesByName map[string]*ResourceDeployItem,
	childrenByName map[string]*ChildDeployItem,
	linksByName map[string]*LinkDeployItem,
	successful *[]SuccessfulElement,
	failures *[]ElementFailure,
	interrupted *[]InterruptedElement,
) {
	for _, item := range items {
		switch item.Type {
		case ItemTypeResource:
			if item.Resource != nil {
				path := buildElementPath(parentPath, "resources", item.Resource.Name)
				collectResourceResult(item.Resource, path, successful, failures, interrupted)
			}
		case ItemTypeChild:
			if item.Child != nil {
				path := buildElementPath(parentPath, "children", item.Child.Name)
				collectChildResult(item.Child, path, successful, failures, interrupted)

				// Recursively collect from nested items in this child
				// Use the child name as the path prefix for map lookups
				if item.Changes != nil {
					collectFromChanges(item.Changes, path, item.Child.Name, resourcesByName, childrenByName, linksByName, successful, failures, interrupted)
				}
			}
		case ItemTypeLink:
			if item.Link != nil {
				path := buildElementPath(parentPath, "links", item.Link.LinkName)
				collectLinkResult(item.Link, path, successful, failures, interrupted)
			}
		}
	}
}

// collectFromChanges recursively collects results from nested blueprint changes.
// The pathPrefix is used for map key lookups (e.g., "parentChild/childName"),
// while parentPath is used for display (e.g., "children.parentChild::children.childName").
func collectFromChanges(
	blueprintChanges *changes.BlueprintChanges,
	parentPath string,
	pathPrefix string,
	resourcesByName map[string]*ResourceDeployItem,
	childrenByName map[string]*ChildDeployItem,
	linksByName map[string]*LinkDeployItem,
	successful *[]SuccessfulElement,
	failures *[]ElementFailure,
	interrupted *[]InterruptedElement,
) {
	if blueprintChanges == nil {
		return
	}

	// Collect nested resources
	for resourceName := range blueprintChanges.NewResources {
		resourceKey := buildMapKey(pathPrefix, resourceName)
		resource := lookupResource(resourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := buildElementPath(parentPath, "resources", resourceName)
			collectResourceResult(resource, path, successful, failures, interrupted)
		}
	}
	for resourceName := range blueprintChanges.ResourceChanges {
		resourceKey := buildMapKey(pathPrefix, resourceName)
		resource := lookupResource(resourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := buildElementPath(parentPath, "resources", resourceName)
			collectResourceResult(resource, path, successful, failures, interrupted)
		}
	}

	// Collect nested children and recurse
	for childName, nc := range blueprintChanges.NewChildren {
		childKey := buildMapKey(pathPrefix, childName)
		child := lookupChild(childrenByName, childKey, childName)
		if child != nil {
			path := buildElementPath(parentPath, "children", childName)
			collectChildResult(child, path, successful, failures, interrupted)

			// Recurse into nested children with updated path prefix
			childChanges := &changes.BlueprintChanges{
				NewResources: nc.NewResources,
				NewChildren:  nc.NewChildren,
			}
			collectFromChanges(childChanges, path, childKey, resourcesByName, childrenByName, linksByName, successful, failures, interrupted)
		}
	}
	for childName, cc := range blueprintChanges.ChildChanges {
		childKey := buildMapKey(pathPrefix, childName)
		child := lookupChild(childrenByName, childKey, childName)
		if child != nil {
			path := buildElementPath(parentPath, "children", childName)
			collectChildResult(child, path, successful, failures, interrupted)

			// Recurse into changed children with updated path prefix
			ccCopy := cc
			collectFromChanges(&ccCopy, path, childKey, resourcesByName, childrenByName, linksByName, successful, failures, interrupted)
		}
	}
}

// buildMapKey builds a path-based key for map lookups.
func buildMapKey(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "/" + name
}

// lookupResource looks up a resource by path-based key, falling back to simple name.
func lookupResource(m map[string]*ResourceDeployItem, pathKey, name string) *ResourceDeployItem {
	if r, ok := m[pathKey]; ok {
		return r
	}
	if r, ok := m[name]; ok {
		return r
	}
	return nil
}

// lookupChild looks up a child by path-based key, falling back to simple name.
func lookupChild(m map[string]*ChildDeployItem, pathKey, name string) *ChildDeployItem {
	if c, ok := m[pathKey]; ok {
		return c
	}
	if c, ok := m[name]; ok {
		return c
	}
	return nil
}

// buildElementPath constructs a full path like "children.notifications::resources.queue".
func buildElementPath(parentPath, elementType, elementName string) string {
	segment := elementType + "." + elementName
	if parentPath == "" {
		return segment
	}
	return parentPath + "::" + segment
}

func collectResourceResult(
	item *ResourceDeployItem,
	path string,
	successful *[]SuccessfulElement,
	failures *[]ElementFailure,
	interrupted *[]InterruptedElement,
) {
	if isFailedResourceStatus(item.Status) && len(item.FailureReasons) > 0 {
		*failures = append(*failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    "resource",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if isInterruptedResourceStatus(item.Status) {
		*interrupted = append(*interrupted, InterruptedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "resource",
		})
		return
	}
	if isSuccessResourceStatus(item.Status) {
		*successful = append(*successful, SuccessfulElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "resource",
			Action:      resourceStatusToAction(item.Status),
		})
	}
}

func collectChildResult(
	item *ChildDeployItem,
	path string,
	successful *[]SuccessfulElement,
	failures *[]ElementFailure,
	interrupted *[]InterruptedElement,
) {
	if isFailedInstanceStatus(item.Status) && len(item.FailureReasons) > 0 {
		*failures = append(*failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    "child",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if isInterruptedInstanceStatus(item.Status) {
		*interrupted = append(*interrupted, InterruptedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
		})
		return
	}
	if isSuccessInstanceStatus(item.Status) {
		*successful = append(*successful, SuccessfulElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
			Action:      instanceStatusToAction(item.Status),
		})
	}
}

func collectLinkResult(
	item *LinkDeployItem,
	path string,
	successful *[]SuccessfulElement,
	failures *[]ElementFailure,
	interrupted *[]InterruptedElement,
) {
	if isFailedLinkStatus(item.Status) && len(item.FailureReasons) > 0 {
		*failures = append(*failures, ElementFailure{
			ElementName:    item.LinkName,
			ElementPath:    path,
			ElementType:    "link",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if isInterruptedLinkStatus(item.Status) {
		*interrupted = append(*interrupted, InterruptedElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
		})
		return
	}
	if isSuccessLinkStatus(item.Status) {
		*successful = append(*successful, SuccessfulElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
			Action:      linkStatusToAction(item.Status),
		})
	}
}

// Status check helpers using map lookups for cleaner code

var failedResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusCreateFailed:   true,
	core.ResourceStatusUpdateFailed:   true,
	core.ResourceStatusDestroyFailed:  true,
	core.ResourceStatusRollbackFailed: true,
}

func isFailedResourceStatus(status core.ResourceStatus) bool {
	return failedResourceStatuses[status]
}

var failedInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployFailed:          true,
	core.InstanceStatusUpdateFailed:          true,
	core.InstanceStatusDestroyFailed:         true,
	core.InstanceStatusDeployRollbackFailed:  true,
	core.InstanceStatusUpdateRollbackFailed:  true,
	core.InstanceStatusDestroyRollbackFailed: true,
}

func isFailedInstanceStatus(status core.InstanceStatus) bool {
	return failedInstanceStatuses[status]
}

var failedLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusCreateFailed:          true,
	core.LinkStatusUpdateFailed:          true,
	core.LinkStatusDestroyFailed:         true,
	core.LinkStatusCreateRollbackFailed:  true,
	core.LinkStatusUpdateRollbackFailed:  true,
	core.LinkStatusDestroyRollbackFailed: true,
}

func isFailedLinkStatus(status core.LinkStatus) bool {
	return failedLinkStatuses[status]
}

var interruptedResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusCreateInterrupted:  true,
	core.ResourceStatusUpdateInterrupted:  true,
	core.ResourceStatusDestroyInterrupted: true,
}

func isInterruptedResourceStatus(status core.ResourceStatus) bool {
	return interruptedResourceStatuses[status]
}

var interruptedInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployInterrupted:  true,
	core.InstanceStatusUpdateInterrupted:  true,
	core.InstanceStatusDestroyInterrupted: true,
}

func isInterruptedInstanceStatus(status core.InstanceStatus) bool {
	return interruptedInstanceStatuses[status]
}

var interruptedLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusCreateInterrupted:  true,
	core.LinkStatusUpdateInterrupted:  true,
	core.LinkStatusDestroyInterrupted: true,
}

func isInterruptedLinkStatus(status core.LinkStatus) bool {
	return interruptedLinkStatuses[status]
}

// Success status helpers

var successResourceStatuses = map[core.ResourceStatus]bool{
	core.ResourceStatusCreated:          true,
	core.ResourceStatusUpdated:          true,
	core.ResourceStatusDestroyed:        true,
	core.ResourceStatusRollbackComplete: true,
}

func isSuccessResourceStatus(status core.ResourceStatus) bool {
	return successResourceStatuses[status]
}

var successInstanceStatuses = map[core.InstanceStatus]bool{
	core.InstanceStatusDeployed:  true,
	core.InstanceStatusUpdated:   true,
	core.InstanceStatusDestroyed: true,
}

func isSuccessInstanceStatus(status core.InstanceStatus) bool {
	return successInstanceStatuses[status]
}

var successLinkStatuses = map[core.LinkStatus]bool{
	core.LinkStatusCreated:   true,
	core.LinkStatusUpdated:   true,
	core.LinkStatusDestroyed: true,
}

func isSuccessLinkStatus(status core.LinkStatus) bool {
	return successLinkStatuses[status]
}

// Status to action converters

func resourceStatusToAction(status core.ResourceStatus) string {
	switch status {
	case core.ResourceStatusCreated:
		return "created"
	case core.ResourceStatusUpdated:
		return "updated"
	case core.ResourceStatusDestroyed:
		return "destroyed"
	case core.ResourceStatusRollbackComplete:
		return "rolled back"
	default:
		return status.String()
	}
}

func instanceStatusToAction(status core.InstanceStatus) string {
	switch status {
	case core.InstanceStatusDeployed:
		return "deployed"
	case core.InstanceStatusUpdated:
		return "updated"
	case core.InstanceStatusDestroyed:
		return "destroyed"
	default:
		return status.String()
	}
}

func linkStatusToAction(status core.LinkStatus) string {
	switch status {
	case core.LinkStatusCreated:
		return "created"
	case core.LinkStatusUpdated:
		return "updated"
	case core.LinkStatusDestroyed:
		return "destroyed"
	default:
		return status.String()
	}
}

// Helper functions for link name parsing.
func extractResourceAFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func extractResourceBFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// buildItemsFromChangeset creates the initial item list from changeset data.
// This provides the proper hierarchy (resources, children, links) from the start.
// It also includes items with no changes from the instance state.
func buildItemsFromChangeset(
	changesetChanges *changes.BlueprintChanges,
	resourcesByName map[string]*ResourceDeployItem,
	childrenByName map[string]*ChildDeployItem,
	linksByName map[string]*LinkDeployItem,
	instanceState *state.InstanceState,
) []DeployItem {
	// Track which items have been added from changes
	addedResources := make(map[string]bool)
	addedChildren := make(map[string]bool)
	addedLinks := make(map[string]bool)

	var items []DeployItem

	if changesetChanges != nil {
		// First, build the top-level items - this populates the maps with the authoritative instances
		items = appendResourceItemsWithTracking(items, changesetChanges, resourcesByName, instanceState, addedResources)
		items = appendChildItemsWithTracking(items, changesetChanges, childrenByName, resourcesByName, linksByName, instanceState, addedChildren)
		items = appendLinkItemsWithTracking(items, changesetChanges, linksByName, addedLinks)

		// Then, pre-populate NESTED items (children of children, resources in children) into the maps.
		// This ensures events for nested items can find the shared instances.
		// We only process the nested content of children that were just added above.
		for _, nc := range changesetChanges.NewChildren {
			childChanges := &changes.BlueprintChanges{
				NewResources: nc.NewResources,
				NewChildren:  nc.NewChildren,
			}
			populateNestedItemsFromChangeset(childChanges, resourcesByName, childrenByName, linksByName)
		}
		for _, cc := range changesetChanges.ChildChanges {
			ccCopy := cc
			populateNestedItemsFromChangeset(&ccCopy, resourcesByName, childrenByName, linksByName)
		}
	}

	// Add items from instance state that have no changes
	items = appendNoChangeItemsFromState(items, instanceState, resourcesByName, childrenByName, linksByName, addedResources, addedChildren, addedLinks)

	return items
}

// appendNoChangeItemsFromState adds items from instance state that have no changes.
// This ensures all resources and children are visible in the navigation.
func appendNoChangeItemsFromState(
	items []DeployItem,
	instanceState *state.InstanceState,
	resourcesByName map[string]*ResourceDeployItem,
	childrenByName map[string]*ChildDeployItem,
	linksByName map[string]*LinkDeployItem,
	addedResources map[string]bool,
	addedChildren map[string]bool,
	addedLinks map[string]bool,
) []DeployItem {
	if instanceState == nil {
		return items
	}

	// Add resources from instance state that have no changes
	for _, resourceState := range instanceState.Resources {
		if addedResources[resourceState.Name] {
			continue
		}
		item := &ResourceDeployItem{
			Name:          resourceState.Name,
			ResourceID:    resourceState.ResourceID,
			ResourceType:  resourceState.Type,
			Action:        ActionNoChange,
			ResourceState: resourceState,
		}
		resourcesByName[resourceState.Name] = item
		items = append(items, DeployItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: instanceState,
		})
	}

	// Add child blueprints from instance state that have no changes
	for name, childState := range instanceState.ChildBlueprints {
		if addedChildren[name] {
			continue
		}
		item := &ChildDeployItem{
			Name:   name,
			Action: ActionNoChange,
			// Provide empty changes so the child can still be expanded
			Changes: &changes.BlueprintChanges{},
		}
		childrenByName[name] = item
		items = append(items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			Changes:         &changes.BlueprintChanges{},
			InstanceState:   childState,
			childrenByName:  childrenByName,
			resourcesByName: resourcesByName,
			linksByName:     linksByName,
		})
	}

	// Add links from instance state that have no changes
	for linkName, linkState := range instanceState.Links {
		if addedLinks[linkName] {
			continue
		}
		item := &LinkDeployItem{
			LinkID:        linkState.LinkID,
			LinkName:      linkName,
			ResourceAName: extractResourceAFromLinkName(linkName),
			ResourceBName: extractResourceBFromLinkName(linkName),
			Action:        ActionNoChange,
			Status:        linkState.Status,
		}
		linksByName[linkName] = item
		items = append(items, DeployItem{
			Type:          ItemTypeLink,
			Link:          item,
			InstanceState: instanceState,
		})
	}

	return items
}

// findResourceState finds a resource state by name using the instance state's
// ResourceIDs map to look up the resource ID, then retrieves the state from Resources.
func findResourceState(instanceState *state.InstanceState, name string) *state.ResourceState {
	if instanceState == nil || instanceState.ResourceIDs == nil || instanceState.Resources == nil {
		return nil
	}
	resourceID, ok := instanceState.ResourceIDs[name]
	if !ok {
		return nil
	}
	return instanceState.Resources[resourceID]
}

// appendResourceItemsWithTracking adds resource items from changeset and tracks which were added.
func appendResourceItemsWithTracking(
	items []DeployItem,
	changesetChanges *changes.BlueprintChanges,
	resourcesByName map[string]*ResourceDeployItem,
	instanceState *state.InstanceState,
	added map[string]bool,
) []DeployItem {
	// New resources
	for name, rc := range changesetChanges.NewResources {
		rcCopy := rc
		item := &ResourceDeployItem{
			Name:    name,
			Action:  ActionCreate,
			Changes: &rcCopy,
		}
		resourcesByName[name] = item
		items = append(items, DeployItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: instanceState,
		})
		added[name] = true
	}

	// Changed resources - look up resource state from instance state if available
	for name, rc := range changesetChanges.ResourceChanges {
		rcCopy := rc

		// Determine action based on changes
		action := ActionUpdate
		if rc.MustRecreate {
			action = ActionRecreate
		} else if !provider.ChangesHasFieldChanges(&rcCopy) {
			// If only link changes (no field changes), treat as no change for the resource itself
			action = ActionNoChange
		}

		// Look up resource state from instance state if available
		var resourceState *state.ResourceState
		if instanceState != nil {
			resourceState = findResourceState(instanceState, name)
		}

		// Also get resource info from Changes.AppliedResourceInfo if available
		var resourceID string
		var resourceType string
		if rc.AppliedResourceInfo.ResourceID != "" {
			resourceID = rc.AppliedResourceInfo.ResourceID
		}
		if rc.AppliedResourceInfo.CurrentResourceState != nil {
			if resourceState == nil {
				resourceState = rc.AppliedResourceInfo.CurrentResourceState
			}
			if resourceType == "" && rc.AppliedResourceInfo.CurrentResourceState.Type != "" {
				resourceType = rc.AppliedResourceInfo.CurrentResourceState.Type
			}
		}

		item := &ResourceDeployItem{
			Name:          name,
			Action:        action,
			Changes:       &rcCopy,
			ResourceID:    resourceID,
			ResourceType:  resourceType,
			ResourceState: resourceState,
		}
		resourcesByName[name] = item
		items = append(items, DeployItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: instanceState,
		})
		added[name] = true
	}

	// Removed resources - look up resource state for showing ID
	for _, name := range changesetChanges.RemovedResources {
		var resourceState *state.ResourceState
		if instanceState != nil {
			resourceState = findResourceState(instanceState, name)
		}

		item := &ResourceDeployItem{
			Name:          name,
			Action:        ActionDelete,
			ResourceState: resourceState,
		}
		resourcesByName[name] = item
		items = append(items, DeployItem{
			Type:          ItemTypeResource,
			Resource:      item,
			InstanceState: instanceState,
		})
		added[name] = true
	}

	return items
}

// appendChildItemsWithTracking adds child blueprint items from changeset and tracks which were added.
func appendChildItemsWithTracking(
	items []DeployItem,
	changesetChanges *changes.BlueprintChanges,
	childrenByName map[string]*ChildDeployItem,
	resourcesByName map[string]*ResourceDeployItem,
	linksByName map[string]*LinkDeployItem,
	instanceState *state.InstanceState,
	added map[string]bool,
) []DeployItem {
	// New children - convert NewBlueprintDefinition to BlueprintChanges for hierarchy
	for name, nc := range changesetChanges.NewChildren {
		childChanges := &changes.BlueprintChanges{
			NewResources: nc.NewResources,
			NewChildren:  nc.NewChildren,
		}
		item := &ChildDeployItem{
			Name:    name,
			Action:  ActionCreate,
			Changes: childChanges,
		}
		childrenByName[name] = item
		items = append(items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			Changes:         childChanges,
			childrenByName:  childrenByName,
			resourcesByName: resourcesByName,
			linksByName:     linksByName,
		})
		added[name] = true
	}

	// Changed children - get nested instance state if available
	for name, cc := range changesetChanges.ChildChanges {
		ccCopy := cc
		var nestedInstanceState *state.InstanceState
		if instanceState != nil && instanceState.ChildBlueprints != nil {
			nestedInstanceState = instanceState.ChildBlueprints[name]
		}
		item := &ChildDeployItem{
			Name:    name,
			Action:  ActionUpdate,
			Changes: &ccCopy,
		}
		childrenByName[name] = item
		items = append(items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			Changes:         &ccCopy,
			InstanceState:   nestedInstanceState,
			childrenByName:  childrenByName,
			resourcesByName: resourcesByName,
			linksByName:     linksByName,
		})
		added[name] = true
	}

	// Recreate children
	for _, name := range changesetChanges.RecreateChildren {
		item := &ChildDeployItem{
			Name:   name,
			Action: ActionRecreate,
		}
		childrenByName[name] = item
		items = append(items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			childrenByName:  childrenByName,
			resourcesByName: resourcesByName,
			linksByName:     linksByName,
		})
		added[name] = true
	}

	// Removed children
	for _, name := range changesetChanges.RemovedChildren {
		item := &ChildDeployItem{
			Name:   name,
			Action: ActionDelete,
		}
		childrenByName[name] = item
		items = append(items, DeployItem{
			Type:            ItemTypeChild,
			Child:           item,
			childrenByName:  childrenByName,
			resourcesByName: resourcesByName,
			linksByName:     linksByName,
		})
		added[name] = true
	}

	return items
}

// appendLinkItemsWithTracking adds link items from changeset and tracks which were added.
func appendLinkItemsWithTracking(
	items []DeployItem,
	changesetChanges *changes.BlueprintChanges,
	linksByName map[string]*LinkDeployItem,
	added map[string]bool,
) []DeployItem {
	// Extract links from new resources
	for resourceAName, resourceChanges := range changesetChanges.NewResources {
		for resourceBName := range resourceChanges.NewOutboundLinks {
			linkName := resourceAName + "::" + resourceBName
			item := &LinkDeployItem{
				LinkName:      linkName,
				ResourceAName: resourceAName,
				ResourceBName: resourceBName,
				Action:        ActionCreate,
			}
			linksByName[linkName] = item
			items = append(items, DeployItem{
				Type: ItemTypeLink,
				Link: item,
			})
			added[linkName] = true
		}
	}

	// Extract links from changed resources
	for resourceAName, resourceChanges := range changesetChanges.ResourceChanges {
		// New outbound links from changed resources
		for resourceBName := range resourceChanges.NewOutboundLinks {
			linkName := resourceAName + "::" + resourceBName
			item := &LinkDeployItem{
				LinkName:      linkName,
				ResourceAName: resourceAName,
				ResourceBName: resourceBName,
				Action:        ActionCreate,
			}
			linksByName[linkName] = item
			items = append(items, DeployItem{
				Type: ItemTypeLink,
				Link: item,
			})
			added[linkName] = true
		}

		// Changed outbound links
		for resourceBName := range resourceChanges.OutboundLinkChanges {
			linkName := resourceAName + "::" + resourceBName
			item := &LinkDeployItem{
				LinkName:      linkName,
				ResourceAName: resourceAName,
				ResourceBName: resourceBName,
				Action:        ActionUpdate,
			}
			linksByName[linkName] = item
			items = append(items, DeployItem{
				Type: ItemTypeLink,
				Link: item,
			})
			added[linkName] = true
		}

		// Removed outbound links
		for _, linkName := range resourceChanges.RemovedOutboundLinks {
			item := &LinkDeployItem{
				LinkName:      linkName,
				ResourceAName: extractResourceAFromLinkName(linkName),
				ResourceBName: extractResourceBFromLinkName(linkName),
				Action:        ActionDelete,
			}
			linksByName[linkName] = item
			items = append(items, DeployItem{
				Type: ItemTypeLink,
				Link: item,
			})
			added[linkName] = true
		}
	}

	// Also check top-level RemovedLinks
	for _, linkName := range changesetChanges.RemovedLinks {
		if _, exists := linksByName[linkName]; !exists {
			item := &LinkDeployItem{
				LinkName:      linkName,
				ResourceAName: extractResourceAFromLinkName(linkName),
				ResourceBName: extractResourceBFromLinkName(linkName),
				Action:        ActionDelete,
			}
			linksByName[linkName] = item
			items = append(items, DeployItem{
				Type: ItemTypeLink,
				Link: item,
			})
			added[linkName] = true
		}
	}

	return items
}

// populateNestedItemsFromChangeset recursively walks the changeset hierarchy and adds all
// nested children and resources to the shared lookup maps. This ensures that when events
// arrive for nested items, they can find and update the shared instances.
func populateNestedItemsFromChangeset(
	blueprintChanges *changes.BlueprintChanges,
	resourcesByName map[string]*ResourceDeployItem,
	childrenByName map[string]*ChildDeployItem,
	linksByName map[string]*LinkDeployItem,
) {
	if blueprintChanges == nil {
		return
	}

	// Process new children and their nested content
	for name, nc := range blueprintChanges.NewChildren {
		childChanges := &changes.BlueprintChanges{
			NewResources: nc.NewResources,
			NewChildren:  nc.NewChildren,
		}

		// Add child to map if not already there
		if _, exists := childrenByName[name]; !exists {
			childrenByName[name] = &ChildDeployItem{
				Name:    name,
				Action:  ActionCreate,
				Changes: childChanges,
			}
		}

		// Add nested resources to the resource map
		for resourceName, rc := range nc.NewResources {
			if _, exists := resourcesByName[resourceName]; !exists {
				rcCopy := rc
				resourcesByName[resourceName] = &ResourceDeployItem{
					Name:    resourceName,
					Action:  ActionCreate,
					Changes: &rcCopy,
				}
			}
		}

		// Recursively process nested children
		populateNestedItemsFromChangeset(childChanges, resourcesByName, childrenByName, linksByName)
	}

	// Process changed children and their nested content
	for name, cc := range blueprintChanges.ChildChanges {
		ccCopy := cc

		// Add child to map if not already there
		if _, exists := childrenByName[name]; !exists {
			childrenByName[name] = &ChildDeployItem{
				Name:    name,
				Action:  ActionUpdate,
				Changes: &ccCopy,
			}
		}

		// Add nested resources (new, changed, removed) to the resource map
		for resourceName, rc := range cc.NewResources {
			if _, exists := resourcesByName[resourceName]; !exists {
				rcCopy := rc
				resourcesByName[resourceName] = &ResourceDeployItem{
					Name:    resourceName,
					Action:  ActionCreate,
					Changes: &rcCopy,
				}
			}
		}
		for resourceName, rc := range cc.ResourceChanges {
			if _, exists := resourcesByName[resourceName]; !exists {
				rcCopy := rc
				// Determine action based on changes
				action := ActionUpdate
				if rc.MustRecreate {
					action = ActionRecreate
				} else if !provider.ChangesHasFieldChanges(&rcCopy) {
					// If only link changes (no field changes), treat as no change
					action = ActionNoChange
				}

				// Extract resource info from AppliedResourceInfo
				var resourceID string
				var resourceType string
				var resourceState *state.ResourceState
				if rc.AppliedResourceInfo.ResourceID != "" {
					resourceID = rc.AppliedResourceInfo.ResourceID
				}
				if rc.AppliedResourceInfo.CurrentResourceState != nil {
					resourceState = rc.AppliedResourceInfo.CurrentResourceState
					if resourceType == "" && rc.AppliedResourceInfo.CurrentResourceState.Type != "" {
						resourceType = rc.AppliedResourceInfo.CurrentResourceState.Type
					}
				}

				resourcesByName[resourceName] = &ResourceDeployItem{
					Name:          resourceName,
					Action:        action,
					Changes:       &rcCopy,
					ResourceID:    resourceID,
					ResourceType:  resourceType,
					ResourceState: resourceState,
				}
			}
		}
		for _, resourceName := range cc.RemovedResources {
			if _, exists := resourcesByName[resourceName]; !exists {
				resourcesByName[resourceName] = &ResourceDeployItem{
					Name:   resourceName,
					Action: ActionDelete,
					// Note: Removed resources don't have Changes since they're being deleted
				}
			}
		}

		// Recursively process nested children
		populateNestedItemsFromChangeset(&ccCopy, resourcesByName, childrenByName, linksByName)
	}
}
