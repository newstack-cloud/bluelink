package stageui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
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

// ItemType represents the type of item in the stage list.
type ItemType string

const (
	ItemTypeResource ItemType = "resource"
	ItemTypeChild    ItemType = "child"
	ItemTypeLink     ItemType = "link"
)

// ActionType represents the action to be taken on an item.
type ActionType string

const (
	ActionCreate   ActionType = "CREATE"
	ActionUpdate   ActionType = "UPDATE"
	ActionDelete   ActionType = "DELETE"
	ActionRecreate ActionType = "RECREATE"
	ActionNoChange ActionType = "NO CHANGE"
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
}

// StageModel is the model for the stage view.
type StageModel struct {
	// Split pane for finished state navigation and display
	splitPane       splitpane.Model
	detailsRenderer *StageDetailsRenderer
	sectionGrouper  *StageSectionGrouper
	footerRenderer  *StageFooterRenderer

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

	// Headless
	headlessMode   bool
	headlessWriter io.Writer
	printer        *headless.Printer

	styles  *stylespkg.Styles
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
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Pass to splitpane for layout
		var cmd tea.Cmd
		m.splitPane, cmd = m.splitPane.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case sharedui.SelectBlueprintMsg:
		m.blueprintFile = msg.BlueprintFile
		m.blueprintSource = msg.Source
		// SelectBlueprintMsg can be sent multiple times, we need to make sure we aren't collecting
		// duplicate results from the stream by not dispatching commands that will create multiple
		// consumers.
		if !m.streaming {
			cmds = append(cmds, startStagingCmd(m), waitForNextEventCmd(m), checkForErrCmd(m))
		}
		m.streaming = true

	case StageStartedMsg:
		m.changesetID = msg.ChangesetID
		// Update footer renderer with changeset ID
		m.footerRenderer.ChangesetID = msg.ChangesetID
		if m.headlessMode {
			m.printHeadlessHeader()
		}

	case StageEventMsg:
		event := types.ChangeStagingEvent(msg)
		m.processEvent(&event)
		cmds = append(cmds, checkForErrCmd(m))

		if eventData, ok := event.AsCompleteChanges(); ok {
			m.finished = true
			m.completeChanges = eventData.Changes
			// Transfer items to splitpane
			m.splitPane.SetItems(ToSplitPaneItems(m.items))
			if m.headlessMode {
				m.printHeadlessSummary()
				cmds = append(cmds, tea.Quit)
			}
		} else {
			cmds = append(cmds, waitForNextEventCmd(m))
		}

	case StageErrorMsg:
		if msg.Err != nil {
			m.err = msg.Err
			if m.headlessMode {
				m.printHeadlessError(msg.Err)
				return m, tea.Quit
			}
			// In interactive mode, don't quit immediately.
			// Let the user read the error and press 'q' to quit.
			return m, nil
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Allow quit when there's an error
		if m.err != nil {
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		// Ignore keys while streaming (not finished)
		if !m.finished {
			return m, nil
		}

		// Delegate key handling to splitpane when finished
		var cmd tea.Cmd
		m.splitPane, cmd = m.splitPane.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case splitpane.QuitMsg:
		return m, tea.Quit

	case splitpane.BackMsg:
		// At root level - ignore (or could quit)
		return m, nil

	case splitpane.ItemExpandedMsg:
		// The splitpane now handles expansion state internally and passes
		// the isExpanded callback to the section grouper, so no sync needed here.
		// This case is kept for potential future use (e.g., analytics, logging).

	case tea.MouseMsg:
		if m.finished && m.err == nil {
			var cmd tea.Cmd
			m.splitPane, cmd = m.splitPane.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Update details renderer with current navigation depth for hint display
	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())

	return m, tea.Batch(cmds...)
}

func (m *StageModel) processEvent(event *types.ChangeStagingEvent) {
	if resourceData, ok := event.AsResourceChanges(); ok {
		m.processResourceChanges(resourceData)
		if m.headlessMode {
			m.printHeadlessResourceEvent(resourceData)
		}
	} else if childData, ok := event.AsChildChanges(); ok {
		m.processChildChanges(childData)
		if m.headlessMode {
			m.printHeadlessChildEvent(childData)
		}
	} else if linkData, ok := event.AsLinkChanges(); ok {
		m.processLinkChanges(linkData)
		if m.headlessMode {
			m.printHeadlessLinkEvent(linkData)
		}
	}
}

func (m *StageModel) processResourceChanges(data *types.ResourceChangesEventData) {
	action := m.determineResourceAction(data)
	state := &ResourceChangeState{
		Name:      data.ResourceName,
		Action:    action,
		Changes:   &data.Changes,
		New:       data.New,
		Removed:   data.Removed,
		Recreate:  data.Changes.MustRecreate,
		Timestamp: data.Timestamp,
	}
	m.resourceChanges[data.ResourceName] = state

	// Extract resource type and display name from the applied resource info
	resourceType, displayName := extractResourceTypeAndDisplayName(&data.Changes)

	// Add to items list
	m.items = append(m.items, StageItem{
		Type:         ItemTypeResource,
		Name:         data.ResourceName,
		ResourceType: resourceType,
		DisplayName:  displayName,
		Action:       action,
		Changes:      &data.Changes,
		New:          data.New,
		Removed:      data.Removed,
		Recreate:     data.Changes.MustRecreate,
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
	state := &ChildChangeState{
		Name:      data.ChildBlueprintName,
		Action:    action,
		Changes:   &data.Changes,
		New:       data.New,
		Removed:   data.Removed,
		Timestamp: data.Timestamp,
	}
	m.childChanges[data.ChildBlueprintName] = state

	// Add to items list
	m.items = append(m.items, StageItem{
		Type:    ItemTypeChild,
		Name:    data.ChildBlueprintName,
		Action:  action,
		Changes: &data.Changes,
		New:     data.New,
		Removed: data.Removed,
	})
}

func (m *StageModel) processLinkChanges(data *types.LinkChangesEventData) {
	linkName := fmt.Sprintf("%s::%s", data.ResourceAName, data.ResourceBName)
	action := m.determineLinkAction(data)
	state := &LinkChangeState{
		ResourceAName: data.ResourceAName,
		ResourceBName: data.ResourceBName,
		Action:        action,
		Changes:       &data.Changes,
		New:           data.New,
		Removed:       data.Removed,
		Timestamp:     data.Timestamp,
	}
	m.linkChanges[linkName] = state

	// Add to items list
	m.items = append(m.items, StageItem{
		Type:    ItemTypeLink,
		Name:    linkName,
		Action:  action,
		Changes: &data.Changes,
		New:     data.New,
		Removed: data.Removed,
	})
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
	if len(data.Changes.ModifiedFields) > 0 || len(data.Changes.NewFields) > 0 || len(data.Changes.RemovedFields) > 0 {
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
	return ActionUpdate
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
		icon = "~"
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
	sb.WriteString(m.styles.Muted.Render("  Press "))
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

	// Generic error display
	sb.WriteString(m.styles.Error.Render("  ✗ Error during change staging\n\n"))
	sb.WriteString(m.styles.Error.Render(fmt.Sprintf("    %s\n", err.Error())))
	sb.WriteString(m.renderErrorFooter())
	return sb.String()
}

func (m StageModel) renderValidationError(clientErr *engineerrors.ClientError) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Error.Render("  ✗ Failed to create changeset\n\n"))

	sb.WriteString(m.styles.Muted.Render("  The following issues must be resolved in the blueprint before changes can be staged:\n\n"))

	// Render validation errors (input validation)
	if len(clientErr.ValidationErrors) > 0 {
		sb.WriteString(m.styles.Category.Render("  Validation Errors:\n"))
		for _, valErr := range clientErr.ValidationErrors {
			location := valErr.Location
			if location == "" {
				location = "unknown"
			}
			sb.WriteString(m.styles.Error.Render(fmt.Sprintf("    • %s: ", location)))
			sb.WriteString(fmt.Sprintf("%s\n", valErr.Message))
		}
		sb.WriteString("\n")
	}

	// Render validation diagnostics (blueprint issues)
	if len(clientErr.ValidationDiagnostics) > 0 {
		sb.WriteString(m.styles.Category.Render("  Blueprint Diagnostics:\n"))
		for _, diag := range clientErr.ValidationDiagnostics {
			sb.WriteString(m.renderDiagnostic(diag))
		}
		sb.WriteString("\n")
	}

	// If no specific errors, show the general message
	if len(clientErr.ValidationErrors) == 0 && len(clientErr.ValidationDiagnostics) == 0 {
		sb.WriteString(m.styles.Error.Render(fmt.Sprintf("    %s\n", clientErr.Message)))
	}

	sb.WriteString(m.renderErrorFooter())
	return sb.String()
}

func (m StageModel) renderStreamError(streamErr *engineerrors.StreamError) string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Error.Render("  ✗ Error during change staging\n\n"))

	sb.WriteString(m.styles.Muted.Render("  The following issues occurred during change staging:\n\n"))
	sb.WriteString(fmt.Sprintf("    %s\n\n", streamErr.Event.Message))

	// Render diagnostics if present
	if len(streamErr.Event.Diagnostics) > 0 {
		sb.WriteString(m.styles.Category.Render("  Diagnostics:\n"))
		for _, diag := range streamErr.Event.Diagnostics {
			sb.WriteString(m.renderDiagnostic(diag))
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

	// Build the prefix (indent + level + location)
	prefix := "    " + levelName
	if diag.Range != nil && diag.Range.Start.Line > 0 {
		prefix += fmt.Sprintf(" [line %d, col %d]", diag.Range.Start.Line, diag.Range.Start.Column)
	}
	prefix += ": "

	// Calculate available width for message wrapping
	// Use terminal width minus prefix length and some padding
	availableWidth := max(
		m.width-len(prefix)-4,
		// Minimum width for readability
		40,
	)

	// Wrap the message text
	wrappedMessage := sdkstrings.WrapText(diag.Message, availableWidth)
	messageLines := strings.Split(wrappedMessage, "\n")

	// First line includes the prefix with styling
	sb.WriteString("    ")
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
	for i := 1; i < len(messageLines); i += 1 {
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
	styles *stylespkg.Styles,
	isHeadless bool,
	headlessWriter io.Writer,
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
	}

	// Create splitpane config
	splitPaneConfig := splitpane.Config{
		Styles:          styles,
		DetailsRenderer: detailsRenderer,
		Title:           "Staging Changes",
		LeftPaneRatio:   0.4,
		MaxExpandDepth:  MaxExpandDepth,
		SectionGrouper:  sectionGrouper,
		FooterRenderer:  footerRenderer,
	}

	// Create headless printer if in headless mode
	var printer *headless.Printer
	if isHeadless && headlessWriter != nil {
		prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[stage] ")
		printer = headless.NewPrinter(prefixedWriter, 80)
	}

	return StageModel{
		splitPane:       splitpane.New(splitPaneConfig),
		detailsRenderer: detailsRenderer,
		sectionGrouper:  sectionGrouper,
		footerRenderer:  footerRenderer,
		engine:          deployEngine,
		logger:          logger,
		instanceID:      instanceID,
		instanceName:    instanceName,
		destroy:         destroy,
		styles:          styles,
		headlessMode:    isHeadless,
		headlessWriter:  headlessWriter,
		printer:         printer,
		spinner:         s,
		eventStream:     make(chan types.ChangeStagingEvent),
		errStream:       make(chan error),
		resourceChanges: make(map[string]*ResourceChangeState),
		childChanges:    make(map[string]*ChildChangeState),
		linkChanges:     make(map[string]*LinkChangeState),
		items:           []StageItem{},
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
