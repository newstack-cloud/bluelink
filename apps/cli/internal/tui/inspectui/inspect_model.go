package inspectui

import (
	"errors"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/deployui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/stateutil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"go.uber.org/zap"
)

var errStreamClosedUnexpectedly = errors.New("event stream closed unexpectedly")

// InspectModel is the model for the inspect view.
type InspectModel struct {
	splitPane       splitpane.Model
	detailsRenderer *InspectDetailsRenderer
	sectionGrouper  *InspectSectionGrouper
	footerRenderer  *InspectFooterRenderer

	// Layout
	width  int
	height int

	// Items - indexed for fast updates
	items           []deployui.DeployItem
	resourcesByName map[string]*deployui.ResourceDeployItem
	childrenByName  map[string]*deployui.ChildDeployItem
	linksByName     map[string]*deployui.LinkDeployItem

	// Instance tracking for nested children
	pathBuilder             *shared.PathBuilder
	childNameToInstancePath map[string]string

	// State
	instanceID    string
	instanceName  string
	instanceState *state.InstanceState
	streaming     bool
	finished      bool
	err           error

	// Views
	showingOverview  bool
	overviewViewport viewport.Model
	showingSpecView  bool
	specViewport     viewport.Model
	showingExports   bool
	exportsModel     deployui.ExportsModel

	// Streaming channels
	engine      engine.DeployEngine
	eventStream chan types.BlueprintInstanceEvent
	errStream   chan error

	// Output modes
	headlessMode   bool
	headlessWriter io.Writer
	printer        *headless.Printer
	jsonMode       bool

	styles  *stylespkg.Styles
	logger  *zap.Logger
	spinner spinner.Model
}

// Init initializes the inspect model.
func (m InspectModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages for the inspect model.
func (m InspectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.MouseMsg:
		return m.handleMouseMsg(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.footerRenderer.SpinnerView = m.spinner.View()
		return m, cmd
	case splitpane.QuitMsg:
		return m, tea.Quit
	case splitpane.BackMsg:
		return m, nil
	case InstanceStateFetchedMsg:
		return m.handleInstanceStateFetched(msg)
	case InstanceNotFoundMsg:
		return m.handleInstanceNotFound(msg)
	case InspectStreamStartedMsg:
		return m, waitForNextEventCmd(m)
	case InspectEventMsg:
		return m.handleInspectEvent(msg)
	case InspectStreamClosedMsg:
		return m.handleStreamClosed()
	case InspectErrorMsg:
		return m.handleInspectError(msg)
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, nil
}

func (m InspectModel) handleInstanceStateFetched(msg InstanceStateFetchedMsg) (tea.Model, tea.Cmd) {
	m.SetInstanceState(msg.InstanceState)

	if msg.IsInProgress {
		m.streaming = true
		m.footerRenderer.Streaming = true
		return m, startStreamingCmd(m)
	}

	m.finished = true
	m.footerRenderer.Finished = true
	m.detailsRenderer.Finished = true

	if m.headlessMode {
		if m.jsonMode {
			m.outputJSON()
		} else {
			m.printHeadlessInstanceState()
		}
		return m, tea.Quit
	}

	return m, nil
}

func (m InspectModel) handleInstanceNotFound(msg InstanceNotFoundMsg) (tea.Model, tea.Cmd) {
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

func (m InspectModel) handleInspectEvent(msg InspectEventMsg) (tea.Model, tea.Cmd) {
	m.processEvent(&msg)
	m.splitPane.UpdateItems(ToSplitPaneItems(m.items))

	finishData, isFinish := msg.AsFinish()
	if isFinish && finishData.EndOfStream {
		m.finished = true
		m.streaming = false
		m.footerRenderer.Streaming = false
		m.footerRenderer.Finished = true
		m.footerRenderer.CurrentStatus = finishData.Status
		m.detailsRenderer.Finished = true

		if m.headlessMode {
			if m.jsonMode {
				m.outputJSON()
			} else {
				m.printHeadlessInstanceState()
			}
			return m, tea.Quit
		}

		return m, nil
	}

	return m, waitForNextEventCmd(m)
}

func (m InspectModel) handleStreamClosed() (tea.Model, tea.Cmd) {
	m.streaming = false
	m.footerRenderer.Streaming = false
	if !m.finished {
		m.err = errStreamClosedUnexpectedly
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

func (m InspectModel) handleInspectError(msg InspectErrorMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.err = msg.Err
		if m.headlessMode {
			if m.jsonMode {
				m.outputJSONError(msg.Err)
			} else {
				m.printHeadlessError(msg.Err)
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m InspectModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.splitPane, cmd = m.splitPane.Update(msg)
	cmds = append(cmds, cmd)

	m.overviewViewport.Width = msg.Width
	m.overviewViewport.Height = msg.Height - 4

	m.specViewport.Width = msg.Width
	m.specViewport.Height = msg.Height - 4

	if m.showingExports {
		m.exportsModel, cmd = m.exportsModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, tea.Batch(cmds...)
}

func (m InspectModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.err != nil {
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	// Handle overview view
	if m.showingOverview {
		return m.handleOverviewKeyMsg(msg)
	}

	// Handle spec view
	if m.showingSpecView {
		return m.handleSpecViewKeyMsg(msg)
	}

	// Handle exports view
	if m.showingExports {
		return m.handleExportsKeyMsg(msg)
	}

	// Toggle exports view
	if msg.String() == "e" || msg.String() == "E" {
		if m.instanceState != nil {
			m.showingExports = true
			m.exportsModel = deployui.NewExportsModel(
				m.instanceState,
				m.instanceName,
				m.width, m.height,
				m.styles,
			)
			m.exportsModel, _ = m.exportsModel.Update(tea.WindowSizeMsg{
				Width:  m.width,
				Height: m.height,
			})
			return m, nil
		}
	}

	// Toggle spec view when a resource with spec data is selected
	// (available during streaming once the resource has completed)
	if msg.String() == "s" || msg.String() == "S" {
		resourceState, resourceName := m.getSelectedResourceState()
		if resourceState != nil && resourceState.SpecData != nil {
			m.showingSpecView = true
			m.specViewport.SetContent(m.renderSpecContent(resourceState, resourceName))
			m.specViewport.GotoTop()
			return m, nil
		}
	}

	// Toggle overview when finished (only makes sense with complete state)
	if m.finished {
		if msg.String() == "o" || msg.String() == "O" {
			m.showingOverview = true
			m.overviewViewport.SetContent(m.renderOverviewContent())
			m.overviewViewport.GotoTop()
			return m, nil
		}
	}

	// Delegate to split-pane
	var cmd tea.Cmd
	m.splitPane, cmd = m.splitPane.Update(msg)
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, cmd
}

func (m InspectModel) handleOverviewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m InspectModel) handleSpecViewKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m InspectModel) handleExportsKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch shared.CheckExportsKeyMsg(msg) {
	case shared.ExportsKeyActionQuit:
		return m, tea.Quit
	case shared.ExportsKeyActionClose:
		m.showingExports = false
		return m, nil
	default:
		var cmd tea.Cmd
		m.exportsModel, cmd = m.exportsModel.Update(msg)
		return m, cmd
	}
}

func (m InspectModel) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.splitPane, cmd = m.splitPane.Update(msg)
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())
	return m, cmd
}

// View renders the inspect model.
func (m InspectModel) View() string {
	if m.headlessMode {
		return ""
	}

	if m.err != nil {
		return m.renderError(m.err)
	}

	if m.showingOverview {
		return m.renderOverviewView()
	}

	if m.showingSpecView {
		return m.renderSpecView()
	}

	if m.showingExports {
		return m.exportsModel.View()
	}

	return m.splitPane.View()
}

// SetInstanceState sets the instance state and rebuilds items from it.
func (m *InspectModel) SetInstanceState(instanceState *state.InstanceState) {
	m.instanceState = instanceState
	if instanceState == nil {
		return
	}

	m.instanceID = instanceState.InstanceID
	m.instanceName = instanceState.InstanceName
	m.footerRenderer.InstanceID = instanceState.InstanceID
	m.footerRenderer.InstanceName = instanceState.InstanceName
	m.footerRenderer.CurrentStatus = instanceState.Status
	m.footerRenderer.HasInstanceState = true

	m.detailsRenderer.InstanceState = instanceState

	// Build items from the instance state
	m.items = buildItemsFromInstanceState(
		instanceState,
		m.resourcesByName,
		m.childrenByName,
		m.linksByName,
	)
	m.splitPane.SetItems(ToSplitPaneItems(m.items))
}

// RefreshInstanceState updates the instance state and hydrates existing items
// with the latest ResourceState data. This preserves the current item list
// and selection while updating output/spec data for completed resources.
func (m *InspectModel) RefreshInstanceState(instanceState *state.InstanceState) {
	if instanceState == nil {
		return
	}

	m.instanceState = instanceState
	m.footerRenderer.CurrentStatus = instanceState.Status
	m.detailsRenderer.InstanceState = instanceState

	// Hydrate existing resource items with updated ResourceState
	// Use index-based iteration to modify actual slice elements
	for i := range m.items {
		m.hydrateItemFromState(&m.items[i], instanceState)
	}

	// Update items in split pane (preserves selection)
	m.splitPane.UpdateItems(ToSplitPaneItems(m.items))
}

// hydrateItemFromState updates a deploy item with the latest state data.
func (m *InspectModel) hydrateItemFromState(item *deployui.DeployItem, instanceState *state.InstanceState) {
	switch item.Type {
	case deployui.ItemTypeResource:
		m.hydrateResourceFromState(item, instanceState)
	case deployui.ItemTypeChild:
		m.hydrateChildFromState(item, instanceState)
	case deployui.ItemTypeLink:
		m.hydrateLinkFromState(item, instanceState)
	}
}

func (m *InspectModel) hydrateResourceFromState(item *deployui.DeployItem, instanceState *state.InstanceState) {
	if item.Resource == nil {
		return
	}

	// Find the resource state in the instance
	resourceState := stateutil.FindResourceState(instanceState, item.Resource.Name)
	if resourceState != nil {
		item.Resource.ResourceState = resourceState
		item.Resource.ResourceID = resourceState.ResourceID
		item.Resource.ResourceType = resourceState.Type
		item.Resource.Status = resourceState.Status
	}

	// Update the item's instance state reference
	item.InstanceState = instanceState
}

func (m *InspectModel) hydrateChildFromState(item *deployui.DeployItem, instanceState *state.InstanceState) {
	if item.Child == nil {
		return
	}

	// Find the child instance state
	childState := stateutil.FindChildInstanceState(instanceState, item.Child.Name)
	if childState != nil {
		item.Child.ChildInstanceID = childState.InstanceID
		item.Child.Status = childState.Status
		item.InstanceState = childState
	}
}

func (m *InspectModel) hydrateLinkFromState(item *deployui.DeployItem, instanceState *state.InstanceState) {
	if item.Link == nil {
		return
	}

	// Find the link state
	linkState := stateutil.FindLinkState(instanceState, item.Link.LinkName)
	if linkState != nil {
		item.Link.LinkID = linkState.LinkID
		item.Link.Status = linkState.Status
	}

	item.InstanceState = instanceState
}

// processEvent processes a streaming event and updates the model.
func (m *InspectModel) processEvent(event *InspectEventMsg) {
	instanceEvent := types.BlueprintInstanceEvent(*event)

	if resourceData, ok := instanceEvent.AsResourceUpdate(); ok {
		m.processResourceUpdate(resourceData)
		if m.headlessMode && !m.jsonMode {
			m.printHeadlessResourceEvent(resourceData)
		}
	} else if childData, ok := instanceEvent.AsChildUpdate(); ok {
		m.processChildUpdate(childData)
		if m.headlessMode && !m.jsonMode {
			m.printHeadlessChildEvent(childData)
		}
	} else if linkData, ok := instanceEvent.AsLinkUpdate(); ok {
		m.processLinkUpdate(linkData)
		if m.headlessMode && !m.jsonMode {
			m.printHeadlessLinkEvent(linkData)
		}
	} else if instanceData, ok := instanceEvent.AsInstanceUpdate(); ok {
		m.processInstanceUpdate(instanceData)
	}
}

// NewInspectModel creates a new inspect model.
func NewInspectModel(
	deployEngine engine.DeployEngine,
	logger *zap.Logger,
	instanceID string,
	instanceName string,
	styles *stylespkg.Styles,
	isHeadless bool,
	headlessWriter io.Writer,
	jsonMode bool,
) *InspectModel {
	detailsRenderer, sectionGrouper, footerRenderer := createInspectRenderers(instanceID, instanceName)
	splitPaneConfig := createInspectSplitPaneConfig(styles, detailsRenderer, sectionGrouper, footerRenderer)

	printer := createInspectHeadlessPrinter(isHeadless, headlessWriter)

	resourcesByName := make(map[string]*deployui.ResourceDeployItem)
	childrenByName := make(map[string]*deployui.ChildDeployItem)
	linksByName := make(map[string]*deployui.LinkDeployItem)

	model := &InspectModel{
		splitPane:               splitpane.New(splitPaneConfig),
		detailsRenderer:         detailsRenderer,
		sectionGrouper:          sectionGrouper,
		footerRenderer:          footerRenderer,
		engine:                  deployEngine,
		logger:                  logger,
		instanceID:              instanceID,
		instanceName:            instanceName,
		styles:                  styles,
		headlessMode:            isHeadless,
		headlessWriter:          headlessWriter,
		printer:                 printer,
		jsonMode:                jsonMode,
		spinner:                 createInspectSpinner(styles),
		eventStream:             make(chan types.BlueprintInstanceEvent),
		errStream:               make(chan error),
		resourcesByName:         resourcesByName,
		childrenByName:          childrenByName,
		linksByName:             linksByName,
		pathBuilder:             shared.NewPathBuilder(instanceID),
		childNameToInstancePath: make(map[string]string),
	}

	return model
}

func createInspectRenderers(instanceID, instanceName string) (*InspectDetailsRenderer, *InspectSectionGrouper, *InspectFooterRenderer) {
	detailsRenderer := &InspectDetailsRenderer{
		MaxExpandDepth:       deployui.MaxExpandDepth,
		NavigationStackDepth: 0,
	}
	sectionGrouper := &InspectSectionGrouper{
		SectionGrouper: shared.SectionGrouper{MaxExpandDepth: deployui.MaxExpandDepth},
	}
	footerRenderer := &InspectFooterRenderer{
		InstanceID:   instanceID,
		InstanceName: instanceName,
	}
	return detailsRenderer, sectionGrouper, footerRenderer
}

func createInspectSplitPaneConfig(
	styles *stylespkg.Styles,
	detailsRenderer *InspectDetailsRenderer,
	sectionGrouper *InspectSectionGrouper,
	footerRenderer *InspectFooterRenderer,
) splitpane.Config {
	return splitpane.Config{
		Styles:          styles,
		Title:           "Instance Inspector",
		DetailsRenderer: detailsRenderer,
		LeftPaneRatio:   0.4,
		MaxExpandDepth:  deployui.MaxExpandDepth,
		SectionGrouper:  sectionGrouper,
		FooterRenderer:  footerRenderer,
	}
}

func createInspectHeadlessPrinter(isHeadless bool, headlessWriter io.Writer) *headless.Printer {
	if !isHeadless || headlessWriter == nil {
		return nil
	}
	prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[inspect] ")
	return headless.NewPrinter(prefixedWriter, 80)
}

func createInspectSpinner(styles *stylespkg.Styles) spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner
	return s
}

// getSelectedResourceState returns the resource state for the currently selected item.
func (m *InspectModel) getSelectedResourceState() (*state.ResourceState, string) {
	selected := m.splitPane.SelectedItem()
	if selected == nil {
		return nil, ""
	}

	item, ok := selected.(*deployui.DeployItem)
	if !ok || item.Type != deployui.ItemTypeResource || item.Resource == nil {
		return nil, ""
	}

	// First check if ResourceState is already set on the item
	if item.Resource.ResourceState != nil {
		return item.Resource.ResourceState, item.Resource.Name
	}

	// Look up resource state from the item's instance state (handles nested blueprints)
	instanceState := item.InstanceState
	if instanceState == nil {
		instanceState = m.instanceState
	}
	if instanceState == nil {
		return nil, ""
	}

	resourceID, ok := instanceState.ResourceIDs[item.Resource.Name]
	if !ok {
		return nil, ""
	}

	resourceState := instanceState.Resources[resourceID]
	return resourceState, item.Resource.Name
}

// isInProgressStatus returns true if the status indicates an operation is in progress.
func isInProgressStatus(status core.InstanceStatus) bool {
	switch status {
	case core.InstanceStatusPreparing,
		core.InstanceStatusDeploying,
		core.InstanceStatusDestroying,
		core.InstanceStatusDeployRollingBack,
		core.InstanceStatusDestroyRollingBack,
		core.InstanceStatusUpdating,
		core.InstanceStatusUpdateRollingBack:
		return true
	default:
		return false
	}
}

// Test accessor methods - these provide read-only access for testing purposes.

// Err returns the current error from the inspect model.
func (m *InspectModel) Err() error {
	return m.err
}

// GetError returns the current error from the inspect model.
// Deprecated: Use Err() for consistency with other models.
func (m *InspectModel) GetError() error {
	return m.err
}

// IsFinished returns whether the inspect operation has finished.
func (m *InspectModel) IsFinished() bool {
	return m.finished
}

// InstanceState returns the current instance state.
func (m *InspectModel) InstanceState() *state.InstanceState {
	return m.instanceState
}

// CurrentStatus returns the current status from the footer renderer.
func (m *InspectModel) CurrentStatus() core.InstanceStatus {
	return m.footerRenderer.CurrentStatus
}

// SetEmbeddedInList marks the inspect model as embedded within the list UI.
// This changes the footer to show "esc back to list" instead of just "q quit".
func (m *InspectModel) SetEmbeddedInList(embedded bool) {
	m.footerRenderer.EmbeddedInList = embedded
}

// IsInSubView returns true if the inspect model is showing a sub-view
// (overview, spec, exports, or drill-down) where esc should close the sub-view
// rather than navigate back to the list.
func (m *InspectModel) IsInSubView() bool {
	return m.showingOverview || m.showingSpecView || m.showingExports || m.splitPane.IsInDrillDown()
}
