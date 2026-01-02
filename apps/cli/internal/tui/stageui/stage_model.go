package stageui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
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

// StageItem represents an item in the staging list.
type StageItem struct {
	Type         ItemType
	Name         string
	ResourceType string // For resources: the resource type (e.g., "aws/sqs/queue")
	DisplayName  string // For resources: the metadata display name if provided
	Action       ActionType
	Changes      any // *provider.Changes, *changes.BlueprintChanges, or *provider.LinkChanges
	New          bool
	Removed      bool
	Recreate     bool
	// For child blueprints: the parent child name (empty for top-level items)
	ParentChild string
	// For child blueprints: indicates nesting depth for indentation
	Depth int
	// For child blueprints: the instance state for this child (if it exists)
	// This is used to show resources with NO CHANGE status that aren't in the changeset.
	InstanceState *state.InstanceState
	// For resources: the resource state (if it exists)
	// This is used to show Resource ID and Outputs in the details pane
	// when the resource has state but Changes doesn't include CurrentResourceState.
	ResourceState *state.ResourceState
	// For links: the link state (if it exists)
	// This is used to show the Link ID in the details pane.
	LinkState *state.LinkState
}

// StageModel is the model for the stage view.
type StageModel struct {
	// Split pane for finished state navigation and display
	splitPane       splitpane.Model
	detailsRenderer *StageDetailsRenderer
	sectionGrouper  *StageSectionGrouper
	footerRenderer  *StageFooterRenderer

	// Split pane for drift review mode
	driftSplitPane       splitpane.Model
	driftDetailsRenderer *DriftDetailsRenderer
	driftSectionGrouper  *DriftSectionGrouper
	driftFooterRenderer  *DriftFooterRenderer

	// Layout - only used during streaming
	width  int
	height int

	// Items collected during streaming
	items []StageItem

	// State
	changesetID string
	streaming   bool
	finished    bool
	err         error

	// Event data
	resourceChanges map[string]*ResourceChangeState
	childChanges    map[string]*ChildChangeState
	linkChanges     map[string]*LinkChangeState
	completeChanges *changes.BlueprintChanges

	// Instance state (fetched at start for existing deployments)
	// This is nil for new deployments.
	instanceState *state.InstanceState

	// Streaming
	engine      engine.DeployEngine
	eventStream chan types.ChangeStagingEvent
	errStream   chan error

	// Config
	blueprintFile   string
	blueprintSource string
	instanceID      string
	instanceName    string
	destroy         bool
	skipDriftCheck  bool

	// Headless
	headlessMode   bool
	headlessWriter io.Writer
	printer        *headless.Printer

	// JSON output mode
	jsonMode bool

	// Drift review state
	driftReviewMode bool
	driftResult     *container.ReconciliationCheckResult
	driftMessage    string
	driftContext    driftui.DriftContext

	// Overview state
	showingOverview  bool           // When true, show full-screen staging overview
	overviewViewport viewport.Model // Scrollable viewport for staging overview

	// Exports view state
	showingExportsView bool              // When true, show exports overlay
	exportsModel       StageExportsModel // Exports view model

	styles *stylespkg.Styles
	logger  *zap.Logger
	spinner spinner.Model
}

// ResourceChangeState tracks the state of a resource's changes.
type ResourceChangeState struct {
	Name      string
	Action    ActionType
	Changes   *provider.Changes
	New       bool
	Removed   bool
	Recreate  bool
	Timestamp int64
}

// ChildChangeState tracks the state of a child blueprint's changes.
type ChildChangeState struct {
	Name      string
	Action    ActionType
	Changes   *changes.BlueprintChanges
	New       bool
	Removed   bool
	Timestamp int64
}

// LinkChangeState tracks the state of a link's changes.
type LinkChangeState struct {
	ResourceAName string
	ResourceBName string
	Action        ActionType
	Changes       *provider.LinkChanges
	New           bool
	Removed       bool
	Timestamp     int64
}

func (m StageModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m StageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		var windowCmds []tea.Cmd
		m, windowCmds = m.handleWindowSizeMsg(msg)
		cmds = append(cmds, windowCmds...)

	case sharedui.SelectBlueprintMsg:
		var selectCmds []tea.Cmd
		m, selectCmds = m.handleSelectBlueprintMsg(msg)
		cmds = append(cmds, selectCmds...)

	case StageStartedMsg:
		m = m.handleStageStartedMsg(msg)

	case StageStartedWithStateMsg:
		m = m.handleStageStartedWithStateMsg(msg)

	case StageEventMsg:
		var eventCmds []tea.Cmd
		m, eventCmds = m.handleStageEventMsg(msg)
		cmds = append(cmds, eventCmds...)

	case StageErrorMsg:
		var cmd tea.Cmd
		m, cmd = m.handleStageErrorMsg(msg)
		if cmd != nil {
			return m, cmd
		}

	case StageStreamClosedMsg:
		var cmd tea.Cmd
		m, cmd = m.handleStageStreamClosedMsg()
		if cmd != nil {
			return m, cmd
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case driftui.ReconciliationCompleteMsg:
		var reconcileCmds []tea.Cmd
		m, reconcileCmds = m.handleReconciliationCompleteMsg()
		cmds = append(cmds, reconcileCmds...)

	case driftui.ReconciliationErrorMsg:
		var cmd tea.Cmd
		m, cmd = m.handleReconciliationErrorMsg(msg)
		if cmd != nil {
			return m, cmd
		}

	case splitpane.QuitMsg, splitpane.BackMsg, splitpane.ItemExpandedMsg:
		var cmd tea.Cmd
		m, cmd = m.handleSplitpaneMsg(msg)
		if cmd != nil {
			return m, cmd
		}

	case tea.MouseMsg:
		var cmd tea.Cmd
		m, cmd = m.handleMouseMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())

	return m, tea.Batch(cmds...)
}

func (m *StageModel) processEvent(event *types.ChangeStagingEvent) {
	if resourceData, ok := event.AsResourceChanges(); ok {
		m.processResourceChanges(resourceData)
		if m.headlessMode && !m.jsonMode {
			m.printHeadlessResourceEvent(resourceData)
		}
	} else if childData, ok := event.AsChildChanges(); ok {
		m.processChildChanges(childData)
		if m.headlessMode && !m.jsonMode {
			m.printHeadlessChildEvent(childData)
		}
	} else if linkData, ok := event.AsLinkChanges(); ok {
		m.processLinkChanges(linkData)
		if m.headlessMode && !m.jsonMode {
			m.printHeadlessLinkEvent(linkData)
		}
	} else if driftData, ok := event.AsDriftDetected(); ok {
		m.processDriftDetected(driftData)
	}
}

func (m *StageModel) processDriftDetected(data *types.DriftDetectedEventData) {
	m.driftResult = data.ReconciliationResult
	m.driftMessage = data.Message
	m.driftContext = driftui.DriftContextStage
}

func (m *StageModel) processResourceChanges(data *types.ResourceChangesEventData) {
	action := m.determineResourceAction(data)
	changeState := &ResourceChangeState{
		Name:      data.ResourceName,
		Action:    action,
		Changes:   &data.Changes,
		New:       data.New,
		Removed:   data.Removed,
		Recreate:  data.Changes.MustRecreate,
		Timestamp: data.Timestamp,
	}
	m.resourceChanges[data.ResourceName] = changeState

	// Extract resource type and display name from the applied resource info
	resourceType, displayName := extractResourceTypeAndDisplayName(&data.Changes)

	// Look up resource state from instance state if available
	var resourceState *state.ResourceState
	if m.instanceState != nil {
		resourceState = findResourceState(m.instanceState, data.ResourceName)
	}

	// Add to items list
	m.items = append(m.items, StageItem{
		Type:          ItemTypeResource,
		Name:          data.ResourceName,
		ResourceType:  resourceType,
		DisplayName:   displayName,
		Action:        action,
		Changes:       &data.Changes,
		New:           data.New,
		Removed:       data.Removed,
		Recreate:      data.Changes.MustRecreate,
		ResourceState: resourceState,
	})
}

// extractResourceTypeAndDisplayName extracts the resource type and display name
// from the AppliedResourceInfo in the provider.Changes struct.
func extractResourceTypeAndDisplayName(changes *provider.Changes) (resourceType, displayName string) {
	if changes == nil {
		return "", ""
	}

	resolvedResource := changes.AppliedResourceInfo.ResourceWithResolvedSubs
	if resolvedResource == nil {
		return "", ""
	}

	// Extract resource type
	if resolvedResource.Type != nil {
		resourceType = resolvedResource.Type.Value
	}

	// Extract display name from metadata
	if resolvedResource.Metadata != nil && resolvedResource.Metadata.DisplayName != nil {
		if resolvedResource.Metadata.DisplayName.Scalar != nil &&
			resolvedResource.Metadata.DisplayName.Scalar.StringValue != nil {
			displayName = *resolvedResource.Metadata.DisplayName.Scalar.StringValue
		}
	}

	return resourceType, displayName
}

func (m *StageModel) processChildChanges(data *types.ChildChangesEventData) {
	action := m.determineChildAction(data)
	changeState := &ChildChangeState{
		Name:      data.ChildBlueprintName,
		Action:    action,
		Changes:   &data.Changes,
		New:       data.New,
		Removed:   data.Removed,
		Timestamp: data.Timestamp,
	}
	m.childChanges[data.ChildBlueprintName] = changeState

	// Get the child's instance state if available (for existing deployments)
	var childInstanceState *state.InstanceState
	if m.instanceState != nil && m.instanceState.ChildBlueprints != nil {
		childInstanceState = m.instanceState.ChildBlueprints[data.ChildBlueprintName]
	}

	// Add to items list
	m.items = append(m.items, StageItem{
		Type:          ItemTypeChild,
		Name:          data.ChildBlueprintName,
		Action:        action,
		Changes:       &data.Changes,
		New:           data.New,
		Removed:       data.Removed,
		InstanceState: childInstanceState,
	})
}

func (m *StageModel) processLinkChanges(data *types.LinkChangesEventData) {
	linkName := fmt.Sprintf("%s::%s", data.ResourceAName, data.ResourceBName)
	action := m.determineLinkAction(data)
	linkChangeState := &LinkChangeState{
		ResourceAName: data.ResourceAName,
		ResourceBName: data.ResourceBName,
		Action:        action,
		Changes:       &data.Changes,
		New:           data.New,
		Removed:       data.Removed,
		Timestamp:     data.Timestamp,
	}
	m.linkChanges[linkName] = linkChangeState

	// Get the link state from instance state if available (for existing deployments)
	var linkState *state.LinkState
	if m.instanceState != nil && m.instanceState.Links != nil {
		linkState = m.instanceState.Links[linkName]
	}

	// Add to items list
	m.items = append(m.items, StageItem{
		Type:      ItemTypeLink,
		Name:      linkName,
		Action:    action,
		Changes:   &data.Changes,
		New:       data.New,
		Removed:   data.Removed,
		LinkState: linkState,
	})

	// Update the source resource's outbound link changes and re-evaluate its action
	m.updateResourceWithLinkChanges(data)
}

// updateResourceWithLinkChanges updates the source resource (ResourceAName) to include
// the outbound link changes information, which allows the resource to show as UPDATE
// when it has link changes but no field changes.
func (m *StageModel) updateResourceWithLinkChanges(data *types.LinkChangesEventData) {
	resourceState, exists := m.resourceChanges[data.ResourceAName]
	if !exists {
		return
	}

	// Update the resource's Changes struct with the outbound link information
	if resourceState.Changes == nil {
		return
	}

	if data.New {
		if resourceState.Changes.NewOutboundLinks == nil {
			resourceState.Changes.NewOutboundLinks = make(map[string]provider.LinkChanges)
		}
		resourceState.Changes.NewOutboundLinks[data.ResourceBName] = data.Changes
	} else if data.Removed {
		resourceState.Changes.RemovedOutboundLinks = append(
			resourceState.Changes.RemovedOutboundLinks,
			data.ResourceBName,
		)
	} else if provider.LinkChangesHasFieldChanges(&data.Changes) {
		if resourceState.Changes.OutboundLinkChanges == nil {
			resourceState.Changes.OutboundLinkChanges = make(map[string]provider.LinkChanges)
		}
		resourceState.Changes.OutboundLinkChanges[data.ResourceBName] = data.Changes
	}

	// Re-evaluate the resource action now that it has link changes
	newAction := m.determineResourceActionFromState(resourceState)
	if newAction != resourceState.Action {
		resourceState.Action = newAction
		// Update the corresponding item in the items list
		for i := range m.items {
			if m.items[i].Type == ItemTypeResource && m.items[i].Name == data.ResourceAName {
				m.items[i].Action = newAction
				break
			}
		}
	}
}

// determineResourceActionFromState determines the action for a resource from its state.
func (m *StageModel) determineResourceActionFromState(state *ResourceChangeState) ActionType {
	if state.New {
		return ActionCreate
	}
	if state.Removed {
		return ActionDelete
	}
	if state.Recreate {
		return ActionRecreate
	}
	if provider.HasAnyChanges(state.Changes) {
		return ActionUpdate
	}
	return ActionNoChange
}

// populateItemsFromCompleteChanges populates m.items from the complete changes.
// This is used for destroy operations where individual resource/child/link events
// are not streamed - only the complete changes are sent.
func (m *StageModel) populateItemsFromCompleteChanges(
	bc *changes.BlueprintChanges,
	instanceState *state.InstanceState,
) {
	if bc == nil {
		return
	}

	// Add removed resources
	// Note: Use findResourceState to look up by name, as instanceState.Resources is keyed by resource ID
	for _, resourceName := range bc.RemovedResources {
		resourceState := findResourceState(instanceState, resourceName)
		var resourceType string
		if resourceState != nil {
			resourceType = resourceState.Type
		}
		m.items = append(m.items, StageItem{
			Type:          ItemTypeResource,
			Name:          resourceName,
			ResourceType:  resourceType,
			Action:        ActionDelete,
			Removed:       true,
			ResourceState: resourceState,
		})
	}

	// Add removed children (recursively handle nested children)
	m.populateRemovedChildren(bc.RemovedChildren, instanceState, "", 0)

	// Add removed links
	for _, linkName := range bc.RemovedLinks {
		var linkState *state.LinkState
		if instanceState != nil && instanceState.Links != nil {
			linkState = instanceState.Links[linkName]
		}
		m.items = append(m.items, StageItem{
			Type:      ItemTypeLink,
			Name:      linkName,
			Action:    ActionDelete,
			Removed:   true,
			LinkState: linkState,
		})
	}
}

// populateRemovedChildren adds removed child blueprints and their contents to items.
// The childPath parameter is the full path to this level (e.g., "parent::child" for nested children).
func (m *StageModel) populateRemovedChildren(
	removedChildren []string,
	instanceState *state.InstanceState,
	childPath string,
	depth int,
) {
	for _, childName := range removedChildren {
		var childInstanceState *state.InstanceState
		if instanceState != nil && instanceState.ChildBlueprints != nil {
			childInstanceState = instanceState.ChildBlueprints[childName]
		}

		// Build the full path for this child
		fullChildPath := buildChildPath(childPath, childName)

		m.items = append(m.items, StageItem{
			Type:          ItemTypeChild,
			Name:          childName,
			Action:        ActionDelete,
			Removed:       true,
			ParentChild:   childPath,
			Depth:         depth,
			InstanceState: childInstanceState,
		})

		// Recursively add resources, links, and nested children from the child instance
		if childInstanceState != nil {
			m.populateChildContents(childInstanceState, fullChildPath, depth+1)
		}
	}
}

// populateChildContents adds resources, links, and nested children from a child instance.
func (m *StageModel) populateChildContents(
	childInstanceState *state.InstanceState,
	childPath string,
	depth int,
) {
	// Add child's resources
	// Note: The map key is resource ID, but we want the logical name from ResourceState.Name
	for _, resourceState := range childInstanceState.Resources {
		m.items = append(m.items, StageItem{
			Type:          ItemTypeResource,
			Name:          resourceState.Name,
			ResourceType:  resourceState.Type,
			Action:        ActionDelete,
			Removed:       true,
			ParentChild:   childPath,
			Depth:         depth,
			ResourceState: resourceState,
		})
	}

	// Add child's links
	// Note: The map key is link ID, but we want the logical name from LinkState.Name
	for _, linkState := range childInstanceState.Links {
		m.items = append(m.items, StageItem{
			Type:        ItemTypeLink,
			Name:        linkState.Name,
			Action:      ActionDelete,
			Removed:     true,
			ParentChild: childPath,
			Depth:       depth,
			LinkState:   linkState,
		})
	}

	// Recursively add nested children
	if len(childInstanceState.ChildBlueprints) > 0 {
		nestedChildNames := make([]string, 0, len(childInstanceState.ChildBlueprints))
		for nestedName := range childInstanceState.ChildBlueprints {
			nestedChildNames = append(nestedChildNames, nestedName)
		}
		m.populateRemovedChildren(nestedChildNames, childInstanceState, childPath, depth)
	}
}

// buildChildPath builds a child path by appending a child name to an existing path.
// If the current path is empty, the child name is returned.
// Otherwise, the path is built as "parent::child".
func buildChildPath(currentPath, childName string) string {
	if currentPath == "" {
		return childName
	}
	return currentPath + "::" + childName
}

func (m *StageModel) determineResourceAction(data *types.ResourceChangesEventData) ActionType {
	if data.New {
		return ActionCreate
	}
	if data.Removed {
		return ActionDelete
	}
	if data.Changes.MustRecreate {
		return ActionRecreate
	}
	// Check for both field changes and outbound link changes
	if provider.HasAnyChanges(&data.Changes) {
		return ActionUpdate
	}
	return ActionNoChange
}

func (m *StageModel) determineChildAction(data *types.ChildChangesEventData) ActionType {
	if data.New {
		return ActionCreate
	}
	if data.Removed {
		return ActionDelete
	}
	if blueprintChangesHasAnyChanges(&data.Changes) {
		return ActionUpdate
	}
	return ActionNoChange
}

// blueprintChangesHasAnyChanges checks if a BlueprintChanges has any actual changes.
// This includes new/modified/removed resources, children, links, exports, and metadata.
// Export changes where the value will be resolved on deploy (newValue is nil and path is in ResolveOnDeploy)
// are not considered actual changes.
func blueprintChangesHasAnyChanges(bc *changes.BlueprintChanges) bool {
	if bc == nil {
		return false
	}

	// Check for resource changes
	if len(bc.NewResources) > 0 || len(bc.RemovedResources) > 0 {
		return true
	}

	// Check if any ResourceChanges have actual field changes
	for _, resourceChanges := range bc.ResourceChanges {
		if provider.HasAnyChanges(&resourceChanges) {
			return true
		}
	}

	// Check for child blueprint changes
	if len(bc.NewChildren) > 0 || len(bc.RemovedChildren) > 0 || len(bc.RecreateChildren) > 0 {
		return true
	}

	// Recursively check child blueprint changes
	for _, childChanges := range bc.ChildChanges {
		if blueprintChangesHasAnyChanges(&childChanges) {
			return true
		}
	}

	// Check for link changes
	if len(bc.RemovedLinks) > 0 {
		return true
	}

	// Check for export changes
	// New exports and removed exports are always real changes
	if len(bc.NewExports) > 0 || len(bc.RemovedExports) > 0 {
		return true
	}

	// For ExportChanges, check if any have actual value changes
	// (not just "resolve on deploy" placeholders where newValue is nil)
	if hasRealExportChanges(bc) {
		return true
	}

	// Check for metadata changes
	mc := &bc.MetadataChanges
	if len(mc.NewFields) > 0 || len(mc.ModifiedFields) > 0 || len(mc.RemovedFields) > 0 {
		return true
	}

	return false
}

// hasRealExportChanges checks if any export changes represent actual value changes,
// rather than just placeholders for values that will be resolved on deploy.
// An export change is considered "not real" if:
// - newValue is nil AND
// - the export path is in ResolveOnDeploy
func hasRealExportChanges(bc *changes.BlueprintChanges) bool {
	for exportName, change := range bc.ExportChanges {
		// Build the path as stored in ResolveOnDeploy (e.g., "exports.exportName")
		exportPath := "exports." + exportName

		// If newValue is nil and this is a resolve-on-deploy field, it's not a real change
		if change.NewValue == nil && isInResolveOnDeploy(exportPath, bc.ResolveOnDeploy) {
			continue
		}

		// This is a real export change
		return true
	}
	return false
}

// isInResolveOnDeploy checks if a path is in the ResolveOnDeploy list.
func isInResolveOnDeploy(path string, resolveOnDeploy []string) bool {
	for _, rod := range resolveOnDeploy {
		if rod == path {
			return true
		}
	}
	return false
}

func (m *StageModel) determineLinkAction(data *types.LinkChangesEventData) ActionType {
	if data.New {
		return ActionCreate
	}
	if data.Removed {
		return ActionDelete
	}
	if len(data.Changes.ModifiedFields) > 0 || len(data.Changes.NewFields) > 0 || len(data.Changes.RemovedFields) > 0 {
		return ActionUpdate
	}
	return ActionNoChange
}

func (m StageModel) View() string {
	if m.headlessMode {
		// In headless mode, output is printed directly to the writer
		return ""
	}

	if m.err != nil {
		return m.renderError(m.err)
	}

	// Show staging overview view
	if m.showingOverview {
		return m.renderOverviewView()
	}

	// Show exports view
	if m.showingExportsView {
		return m.exportsModel.View()
	}

	// Show drift review mode
	if m.driftReviewMode {
		return m.driftSplitPane.View()
	}

	if !m.finished {
		return m.renderStreamingView()
	}

	// Use splitpane for finished view
	return m.splitPane.View()
}

func (m StageModel) renderStreamingView() string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s Staging changes...\n\n", m.spinner.View()))

	if m.changesetID != "" {
		sb.WriteString(fmt.Sprintf("  Changeset: %s\n\n", m.styles.Selected.Render(m.changesetID)))
	}

	// Show items received so far
	if len(m.items) > 0 {
		sb.WriteString("  Progress:\n")
		for _, item := range m.items {
			icon := m.getStatusIcon(item.Action)
			sb.WriteString(fmt.Sprintf("    %s %s: %s - %s\n", icon, item.Type, item.Name, m.renderActionBadge(item.Action)))
		}
	}

	return sb.String()
}

// getStatusIcon returns the status icon for an action with color styling.
func (m StageModel) getStatusIcon(action ActionType) string {
	var icon string
	var style lipgloss.Style

	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())

	switch action {
	case ActionCreate:
		icon = "✓"
		style = successStyle
	case ActionUpdate:
		icon = "±"
		style = m.styles.Warning
	case ActionDelete:
		icon = "-"
		style = m.styles.Error
	case ActionRecreate:
		icon = "↻"
		style = m.styles.Info
	default:
		icon = "○"
		style = m.styles.Muted
	}

	return style.Render(icon)
}

func (m StageModel) renderActionBadge(action ActionType) string {
	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())
	switch action {
	case ActionCreate:
		return successStyle.Render(string(action))
	case ActionUpdate:
		return m.styles.Warning.Render(string(action))
	case ActionDelete:
		return m.styles.Error.Render(string(action))
	case ActionRecreate:
		return m.styles.Info.Render(string(action))
	default:
		return m.styles.Muted.Render(string(action))
	}
}

func (m StageModel) renderErrorFooter() string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	keyStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Primary()).Bold(true)
	sb.WriteString("  ")
	sb.WriteString(m.styles.Muted.Render("Press "))
	sb.WriteString(keyStyle.Render("q"))
	sb.WriteString(m.styles.Muted.Render(" to quit"))
	sb.WriteString("\n")
	return sb.String()
}

func (m StageModel) renderError(err error) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	// Check for validation errors (ClientError with ValidationErrors or ValidationDiagnostics)
	if clientErr, isValidation := engineerrors.IsValidationError(err); isValidation {
		return m.renderValidationError(clientErr)
	}

	// Check for stream errors with diagnostics
	if streamErr, ok := err.(*engineerrors.StreamError); ok {
		return m.renderStreamError(streamErr)
	}

	// Generic error display with text wrapping
	sb.WriteString("  " + m.styles.Error.Render("✗ Error during change staging") + "\n\n")

	maxWidth := m.width - 6 // Account for 2-space indent + margin
	if maxWidth < 40 {
		maxWidth = 40
	}
	messageStyle := lipgloss.NewStyle().Width(maxWidth)
	sb.WriteString("  " + messageStyle.Render(err.Error()) + "\n")
	sb.WriteString(m.renderErrorFooter())
	return sb.String()
}

func (m StageModel) renderValidationError(clientErr *engineerrors.ClientError) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString("  " + m.styles.Error.Render("✗ Failed to create changeset") + "\n\n")

	sb.WriteString("  " + m.styles.Muted.Render("The following issues must be resolved in the blueprint before changes can be staged:") + "\n\n")

	maxWidth := m.width - 6 // Account for 2-space indent + margin
	if maxWidth < 40 {
		maxWidth = 40
	}
	messageStyle := lipgloss.NewStyle().Width(maxWidth)

	// Render validation errors (input validation)
	if len(clientErr.ValidationErrors) > 0 {
		sb.WriteString("  " + m.styles.Category.Render("Validation Errors:") + "\n\n")
		for _, valErr := range clientErr.ValidationErrors {
			location := valErr.Location
			if location == "" {
				location = "unknown"
			}
			sb.WriteString("  " + m.styles.Error.Render(fmt.Sprintf("• %s: ", location)))
			sb.WriteString(messageStyle.Render(valErr.Message) + "\n")
		}
		sb.WriteString("\n")
	}

	// Render validation diagnostics (blueprint issues)
	if len(clientErr.ValidationDiagnostics) > 0 {
		sb.WriteString("  " + m.styles.Category.Render("Blueprint Diagnostics:") + "\n\n")
		for _, diag := range clientErr.ValidationDiagnostics {
			sb.WriteString("  " + m.renderDiagnostic(diag))
		}
		sb.WriteString("\n")
	}

	// If no specific errors, show the general message
	if len(clientErr.ValidationErrors) == 0 && len(clientErr.ValidationDiagnostics) == 0 {
		sb.WriteString("  " + messageStyle.Render(clientErr.Message) + "\n")
	}

	sb.WriteString(m.renderErrorFooter())
	return sb.String()
}

func (m StageModel) renderStreamError(streamErr *engineerrors.StreamError) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString("  " + m.styles.Error.Render("✗ Error during change staging") + "\n\n")

	sb.WriteString("  " + m.styles.Muted.Render("The following issues occurred during change staging:") + "\n\n")

	// Wrap the error message to fit the terminal width
	maxWidth := m.width - 6 // Account for 2-space indent + margin
	if maxWidth < 40 {
		maxWidth = 40
	}
	messageStyle := lipgloss.NewStyle().Width(maxWidth)
	sb.WriteString("  " + messageStyle.Render(streamErr.Event.Message) + "\n\n")

	// Render diagnostics if present
	if len(streamErr.Event.Diagnostics) > 0 {
		sb.WriteString("  " + m.styles.Category.Render("Diagnostics:") + "\n\n")
		for _, diag := range streamErr.Event.Diagnostics {
			sb.WriteString("  " + m.renderDiagnostic(diag))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(m.renderErrorFooter())
	return sb.String()
}

func (m StageModel) renderDiagnostic(diag *core.Diagnostic) string {
	sb := strings.Builder{}

	// Determine the level style
	var levelStyle lipgloss.Style
	levelName := "unknown"
	switch diag.Level {
	case core.DiagnosticLevelError:
		levelStyle = m.styles.Error
		levelName = "ERROR"
	case core.DiagnosticLevelWarning:
		levelStyle = m.styles.Warning
		levelName = "WARNING"
	case core.DiagnosticLevelInfo:
		levelStyle = m.styles.Info
		levelName = "INFO"
	default:
		levelStyle = m.styles.Muted
	}

	// Build the prefix (level + location)
	prefix := levelName
	if diag.Range != nil && diag.Range.Start.Line > 0 {
		prefix += fmt.Sprintf(" [line %d, col %d]", diag.Range.Start.Line, diag.Range.Start.Column)
	}
	prefix += ": "

	// Calculate available width for message wrapping
	// Use terminal width minus prefix length and some padding
	availableWidth := max(
		m.width-len(prefix)-2,
		// Minimum width for readability
		40,
	)

	// Wrap the message text
	wrappedMessage := sdkstrings.WrapText(diag.Message, availableWidth)
	messageLines := strings.Split(wrappedMessage, "\n")

	// First line includes the prefix with styling
	sb.WriteString(levelStyle.Render(levelName))
	if diag.Range != nil && diag.Range.Start.Line > 0 {
		sb.WriteString(m.styles.Muted.Render(fmt.Sprintf(" [line %d, col %d]", diag.Range.Start.Line, diag.Range.Start.Column)))
	}
	sb.WriteString(": ")
	if len(messageLines) > 0 {
		sb.WriteString(messageLines[0])
	}
	sb.WriteString("\n")

	// Continuation lines are indented to align with the message
	indent := strings.Repeat(" ", len(prefix))
	for i := 1; i < len(messageLines); i++ {
		sb.WriteString(indent)
		sb.WriteString(messageLines[i])
		sb.WriteString("\n")
	}

	return sb.String()
}

// NewStageModel creates a new stage model with the given configuration.
func NewStageModel(
	deployEngine engine.DeployEngine,
	logger *zap.Logger,
	instanceID string,
	instanceName string,
	destroy bool,
	skipDriftCheck bool,
	styles *stylespkg.Styles,
	isHeadless bool,
	headlessWriter io.Writer,
	jsonMode bool,
) StageModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	// Create renderers
	detailsRenderer := &StageDetailsRenderer{
		MaxExpandDepth:       MaxExpandDepth,
		NavigationStackDepth: 0,
	}

	sectionGrouper := &StageSectionGrouper{
		MaxExpandDepth: MaxExpandDepth,
	}

	footerRenderer := &StageFooterRenderer{
		ChangesetID:  "",
		InstanceID:   instanceID,
		InstanceName: instanceName,
		Destroy:      destroy,
	}

	// Create splitpane config
	splitPaneConfig := splitpane.Config{
		Styles:          styles,
		DetailsRenderer: detailsRenderer,
		Title:           "Change Staging",
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
		Context: driftui.DriftContextStage,
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

	// Create headless printer if in headless mode
	var printer *headless.Printer
	if isHeadless && headlessWriter != nil {
		prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[stage] ")
		printer = headless.NewPrinter(prefixedWriter, 80)
	}

	return StageModel{
		splitPane:            splitpane.New(splitPaneConfig),
		detailsRenderer:      detailsRenderer,
		sectionGrouper:       sectionGrouper,
		footerRenderer:       footerRenderer,
		driftSplitPane:       splitpane.New(driftSplitPaneConfig),
		driftDetailsRenderer: driftDetailsRenderer,
		driftSectionGrouper:  driftSectionGrouper,
		driftFooterRenderer:  driftFooterRenderer,
		engine:               deployEngine,
		logger:               logger,
		instanceID:           instanceID,
		instanceName:         instanceName,
		destroy:              destroy,
		skipDriftCheck:       skipDriftCheck,
		styles:               styles,
		headlessMode:         isHeadless,
		headlessWriter:       headlessWriter,
		printer:              printer,
		jsonMode:             jsonMode,
		spinner:              s,
		eventStream:          make(chan types.ChangeStagingEvent),
		errStream:            make(chan error),
		resourceChanges:      make(map[string]*ResourceChangeState),
		childChanges:         make(map[string]*ChildChangeState),
		linkChanges:          make(map[string]*LinkChangeState),
		items:                []StageItem{},
	}
}

// countChangeSummary returns counts of create, update, delete, recreate actions
func (m *StageModel) countChangeSummary() (create, update, delete, recreate int) {
	for _, item := range m.items {
		switch item.Action {
		case ActionCreate:
			create += 1
		case ActionUpdate:
			update += 1
		case ActionDelete:
			delete += 1
		case ActionRecreate:
			recreate += 1
		}
	}
	return
}

// updateFooterCounts updates the footer renderer with the current change summary counts.
func (m *StageModel) updateFooterCounts() {
	create, update, delete, recreate := m.countChangeSummary()
	m.footerRenderer.CreateCount = create
	m.footerRenderer.UpdateCount = update
	m.footerRenderer.DeleteCount = delete
	m.footerRenderer.RecreateCount = recreate
	// Check if there are export changes to show in the footer
	m.footerRenderer.HasExportChanges = HasAnyExportChanges(m.completeChanges)
}

// countByType returns counts of resources, children, and links
func (m *StageModel) countByType() (resources, children, links int) {
	for _, item := range m.items {
		switch item.Type {
		case ItemTypeResource:
			resources += 1
		case ItemTypeChild:
			children += 1
		case ItemTypeLink:
			links += 1
		}
	}
	return
}

// ---- Exported accessor methods for integration with deploy flow ----

// IsFinished returns true if staging has completed.
func (m *StageModel) IsFinished() bool {
	return m.finished
}

// GetChangesetID returns the changeset ID created during staging.
func (m *StageModel) GetChangesetID() string {
	return m.changesetID
}

// GetChanges returns the complete blueprint changes from staging.
func (m *StageModel) GetChanges() *changes.BlueprintChanges {
	return m.completeChanges
}

// GetItems returns the staged items collected during streaming.
func (m *StageModel) GetItems() []StageItem {
	return m.items
}

// GetError returns any error that occurred during staging.
func (m *StageModel) GetError() error {
	return m.err
}

// CountChangeSummary returns counts of create, update, delete, recreate actions.
// This is the exported version of countChangeSummary.
func (m *StageModel) CountChangeSummary() (create, update, delete, recreate int) {
	return m.countChangeSummary()
}

// ---- Exported setter methods for configuration ----

// SetBlueprintFile sets the blueprint file path.
func (m *StageModel) SetBlueprintFile(file string) {
	m.blueprintFile = file
}

// SetBlueprintSource sets the blueprint source type.
func (m *StageModel) SetBlueprintSource(source string) {
	m.blueprintSource = source
}

// SetInstanceName sets the instance name.
func (m *StageModel) SetInstanceName(name string) {
	m.instanceName = name
	if m.footerRenderer != nil {
		m.footerRenderer.InstanceName = name
	}
}

// SetInstanceID sets the instance ID.
func (m *StageModel) SetInstanceID(id string) {
	m.instanceID = id
	if m.footerRenderer != nil {
		m.footerRenderer.InstanceID = id
	}
}

// SetFooterRenderer sets a custom footer renderer that overrides the default stage footer.
// This allows parent models (like deploy) to inject their own footer rendering.
func (m *StageModel) SetFooterRenderer(renderer splitpane.FooterRenderer) {
	if m.footerRenderer != nil {
		m.footerRenderer.Delegate = renderer
	}
}

// StartStaging initiates the staging process.
// Returns a tea.Cmd that starts the staging workflow.
func (m *StageModel) StartStaging() tea.Cmd {
	if m.streaming {
		return nil
	}
	m.streaming = true
	return tea.Batch(startStagingCmd(*m), waitForNextEventCmd(*m), checkForErrCmd(*m))
}
