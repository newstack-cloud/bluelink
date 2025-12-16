package deployui

import (
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"go.uber.org/zap"
)

// ItemType represents the type of item being deployed.
type ItemType string

const (
	ItemTypeResource ItemType = "resource"
	ItemTypeChild    ItemType = "child"
	ItemTypeLink     ItemType = "link"
)

// ActionType represents the planned action from the changeset.
type ActionType string

const (
	ActionCreate   ActionType = "CREATE"
	ActionUpdate   ActionType = "UPDATE"
	ActionDelete   ActionType = "DELETE"
	ActionRecreate ActionType = "RECREATE"
	ActionNoChange ActionType = "NO CHANGE"
)

// MaxExpandDepth is the maximum nesting depth for expanding child blueprints.
const MaxExpandDepth = 2

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
	// Changes holds the blueprint changes for this item (for children)
	// Used to provide hierarchy for GetChildren()
	Changes *changes.BlueprintChanges
}

// DeployModel is the model for the deploy view with real-time split-pane.
type DeployModel struct {
	// Split pane shown from the START of deployment
	splitPane       splitpane.Model
	detailsRenderer *DeployDetailsRenderer
	sectionGrouper  *DeploySectionGrouper
	footerRenderer  *DeployFooterRenderer

	// Layout
	width  int
	height int

	// Items - indexed for fast updates
	items           []DeployItem
	resourcesByName map[string]*ResourceDeployItem
	childrenByName  map[string]*ChildDeployItem
	linksByName     map[string]*LinkDeployItem

	// Instance tracking - maps child instance IDs to child names
	// This allows us to route resource/link updates to the correct hierarchy level
	instanceIDToChildName map[string]string

	// State
	instanceID     string
	instanceName   string
	changesetID    string
	streaming      bool
	finished       bool
	finalStatus    core.InstanceStatus
	failureReasons []string
	err            error

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
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		var cmd tea.Cmd
		m.splitPane, cmd = m.splitPane.Update(msg)
		cmds = append(cmds, cmd)

	case sharedui.SelectBlueprintMsg:
		m.blueprintFile = msg.BlueprintFile
		m.blueprintSource = msg.Source
		// SelectBlueprintMsg can be sent multiple times, we need to make sure we aren't collecting
		// duplicate results from the stream by not dispatching commands that will create multiple
		// consumers.
		if !m.streaming {
			cmds = append(cmds, startDeploymentCmd(m), checkForErrCmd(m))
		}
		m.streaming = true

	case StartDeployMsg:
		// StartDeployMsg is used to initiate deployment after staging confirmation.
		// The model should already have all required fields set (blueprintFile, changesetID, etc.)
		if !m.streaming {
			cmds = append(cmds, startDeploymentCmd(m), checkForErrCmd(m))
		}
		m.streaming = true

	case DeployStartedMsg:
		m.instanceID = msg.InstanceID
		m.streaming = true
		m.footerRenderer.InstanceID = msg.InstanceID
		if m.headlessMode {
			m.printHeadlessHeader()
		}
		cmds = append(cmds, waitForNextDeployEventCmd(m), checkForErrCmd(m))

	case DeployEventMsg:
		event := types.BlueprintInstanceEvent(msg)
		m.processEvent(&event)

		// Refresh split-pane with updated items
		m.splitPane.SetItems(ToSplitPaneItems(m.items))

		cmds = append(cmds, checkForErrCmd(m))

		if finishData, ok := event.AsFinish(); ok {
			m.finished = true
			m.finalStatus = finishData.Status
			m.failureReasons = finishData.FailureReasons
			m.footerRenderer.FinalStatus = finishData.Status
			m.footerRenderer.FailureReasons = finishData.FailureReasons
			m.footerRenderer.Finished = true

			// Mark pending items as skipped if deployment failed
			if isFailedStatus(finishData.Status) {
				m.markPendingItemsAsSkipped()
				m.splitPane.SetItems(ToSplitPaneItems(m.items))
			}

			if m.headlessMode {
				m.printHeadlessSummary()
				cmds = append(cmds, tea.Quit)
			}
		} else {
			cmds = append(cmds, waitForNextDeployEventCmd(m))
		}

	case DeployErrorMsg:
		if msg.Err != nil {
			m.err = msg.Err
			if m.headlessMode {
				m.printHeadlessError(msg.Err)
				return m, tea.Quit
			}
			return m, nil
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.err != nil {
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		// Delegate to split-pane (works during streaming too!)
		var cmd tea.Cmd
		m.splitPane, cmd = m.splitPane.Update(msg)
		cmds = append(cmds, cmd)

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.splitPane, cmd = m.splitPane.Update(msg)
		cmds = append(cmds, cmd)

	case splitpane.QuitMsg:
		return m, tea.Quit
	}

	m.detailsRenderer.NavigationStackDepth = len(m.splitPane.NavigationStack())

	return m, tea.Batch(cmds...)
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

	item, exists := m.resourcesByName[data.ResourceName]
	if !exists {
		// First time seeing this resource - create it
		item = &ResourceDeployItem{
			Name:       data.ResourceName,
			ResourceID: data.ResourceID,
			Group:      data.Group,
		}
		m.resourcesByName[data.ResourceName] = item
		// Only add to top-level items if this belongs to the root instance
		if isRootResource {
			m.items = append(m.items, DeployItem{
				Type:     ItemTypeResource,
				Resource: item,
			})
		}
	}

	// Update with exact types from the event
	item.Status = data.Status
	item.PreciseStatus = data.PreciseStatus
	item.FailureReasons = data.FailureReasons
	item.Attempt = data.Attempt
	item.CanRetry = data.CanRetry
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

func (m *DeployModel) processChildUpdate(data *container.ChildDeployUpdateMessage) {
	// Track the child instance ID -> child name mapping for routing resource/link updates
	if data.ChildInstanceID != "" && data.ChildName != "" {
		m.instanceIDToChildName[data.ChildInstanceID] = data.ChildName
	}

	item, exists := m.childrenByName[data.ChildName]
	if !exists {
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
		m.childrenByName[data.ChildName] = item
		if isDirectChildOfRoot {
			m.items = append(m.items, DeployItem{
				Type:    ItemTypeChild,
				Child:   item,
				Changes: childChanges,
			})
		}
	}

	item.Status = data.Status
	item.FailureReasons = data.FailureReasons
	item.Timestamp = data.UpdateTimestamp
	if data.Durations != nil {
		item.Durations = data.Durations
	}
}

func (m *DeployModel) processLinkUpdate(data *container.LinkDeployUpdateMessage) {
	// Check if this link belongs to a child instance (not root)
	isRootLink := data.InstanceID == "" || data.InstanceID == m.instanceID

	item, exists := m.linksByName[data.LinkName]
	if !exists {
		item = &LinkDeployItem{
			LinkID:        data.LinkID,
			LinkName:      data.LinkName,
			ResourceAName: extractResourceAFromLinkName(data.LinkName),
			ResourceBName: extractResourceBFromLinkName(data.LinkName),
		}
		m.linksByName[data.LinkName] = item
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

	// If the deployment is rolling back or has failed, mark pending items as skipped immediately
	// This provides real-time feedback rather than waiting for the finish event
	if isRollingBackOrFailedStatus(data.Status) && !m.finished {
		m.markPendingItemsAsSkipped()
	}
}

// View renders the deploy model.
func (m DeployModel) View() string {
	if m.headlessMode {
		return ""
	}

	if m.err != nil {
		return m.renderError(m.err)
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
	m.items = buildItemsFromChangeset(changesetChanges, m.resourcesByName, m.childrenByName, m.linksByName)
	m.splitPane.SetItems(ToSplitPaneItems(m.items))
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
		DetailsRenderer: detailsRenderer,
		Title:           "Deploying",
		LeftPaneRatio:   0.4,
		MaxExpandDepth:  MaxExpandDepth,
		SectionGrouper:  sectionGrouper,
		FooterRenderer:  footerRenderer,
	}

	var printer *headless.Printer
	if isHeadless && headlessWriter != nil {
		prefixedWriter := headless.NewPrefixedWriter(headlessWriter, "[deploy] ")
		printer = headless.NewPrinter(prefixedWriter, 80)
	}

	// Pre-populate items from changeset if available
	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)
	items := buildItemsFromChangeset(changesetChanges, resourcesByName, childrenByName, linksByName)

	model := DeployModel{
		splitPane:             splitpane.New(splitPaneConfig),
		detailsRenderer:       detailsRenderer,
		sectionGrouper:        sectionGrouper,
		footerRenderer:        footerRenderer,
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
		eventStream:           make(chan types.BlueprintInstanceEvent),
		errStream:             make(chan error),
		resourcesByName:       resourcesByName,
		childrenByName:        childrenByName,
		linksByName:           linksByName,
		instanceIDToChildName: make(map[string]string),
		items:                 items,
	}

	// Initialize split pane with pre-populated items
	if len(items) > 0 {
		model.splitPane.SetItems(ToSplitPaneItems(items))
	}

	return model
}

// isRollingBackOrFailedStatus returns true if the instance status indicates
// a rollback is in progress or has completed/failed.
// This is used to mark pending items as skipped in real-time.
func isRollingBackOrFailedStatus(status core.InstanceStatus) bool {
	switch status {
	case core.InstanceStatusDeployFailed,
		core.InstanceStatusUpdateFailed,
		core.InstanceStatusDestroyFailed,
		core.InstanceStatusDeployRollingBack,
		core.InstanceStatusUpdateRollingBack,
		core.InstanceStatusDestroyRollingBack,
		core.InstanceStatusDeployRollbackFailed,
		core.InstanceStatusUpdateRollbackFailed,
		core.InstanceStatusDestroyRollbackFailed,
		core.InstanceStatusDeployRollbackComplete,
		core.InstanceStatusUpdateRollbackComplete,
		core.InstanceStatusDestroyRollbackComplete:
		return true
	default:
		return false
	}
}

// isFailedStatus returns true if the instance status indicates a failure
// or a rollback complete (which means the original operation failed).
// Used to determine final deployment outcome.
func isFailedStatus(status core.InstanceStatus) bool {
	switch status {
	case core.InstanceStatusDeployFailed,
		core.InstanceStatusUpdateFailed,
		core.InstanceStatusDestroyFailed,
		core.InstanceStatusDeployRollbackFailed,
		core.InstanceStatusUpdateRollbackFailed,
		core.InstanceStatusDestroyRollbackFailed,
		// Rollback complete states also indicate the original operation failed
		core.InstanceStatusDeployRollbackComplete,
		core.InstanceStatusUpdateRollbackComplete,
		core.InstanceStatusDestroyRollbackComplete:
		return true
	default:
		return false
	}
}

// markPendingItemsAsSkipped updates items that were never attempted (still in unknown/pending state)
// to indicate they were skipped due to deployment failure.
func (m *DeployModel) markPendingItemsAsSkipped() {
	for i := range m.items {
		item := &m.items[i]
		switch item.Type {
		case ItemTypeResource:
			// ResourceStatusUnknown is iota = 0, the initial/pending state
			if item.Resource != nil && item.Resource.Status == core.ResourceStatusUnknown {
				item.Resource.Skipped = true
			}
		case ItemTypeChild:
			// InstanceStatusPreparing is iota = 0, InstanceStatusNotDeployed is also a pending state
			if item.Child != nil && (item.Child.Status == core.InstanceStatusPreparing || item.Child.Status == core.InstanceStatusNotDeployed) {
				item.Child.Skipped = true
			}
		case ItemTypeLink:
			// LinkStatusUnknown is iota = 0, the initial/pending state
			if item.Link != nil && item.Link.Status == core.LinkStatusUnknown {
				item.Link.Skipped = true
			}
		}
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
func buildItemsFromChangeset(
	changesetChanges *changes.BlueprintChanges,
	resourcesByName map[string]*ResourceDeployItem,
	childrenByName map[string]*ChildDeployItem,
	linksByName map[string]*LinkDeployItem,
) []DeployItem {
	if changesetChanges == nil {
		return []DeployItem{}
	}

	var items []DeployItem
	items = appendResourceItems(items, changesetChanges, resourcesByName)
	items = appendChildItems(items, changesetChanges, childrenByName)
	items = appendLinkItems(items, changesetChanges, linksByName)
	return items
}

// appendResourceItems adds resource items from changeset to the items slice.
func appendResourceItems(
	items []DeployItem,
	changesetChanges *changes.BlueprintChanges,
	resourcesByName map[string]*ResourceDeployItem,
) []DeployItem {
	// New resources
	for name := range changesetChanges.NewResources {
		item := &ResourceDeployItem{
			Name:   name,
			Action: ActionCreate,
		}
		resourcesByName[name] = item
		items = append(items, DeployItem{
			Type:     ItemTypeResource,
			Resource: item,
		})
	}

	// Changed resources
	for name, rc := range changesetChanges.ResourceChanges {
		action := ActionUpdate
		if rc.MustRecreate {
			action = ActionRecreate
		}
		item := &ResourceDeployItem{
			Name:   name,
			Action: action,
		}
		resourcesByName[name] = item
		items = append(items, DeployItem{
			Type:     ItemTypeResource,
			Resource: item,
		})
	}

	// Removed resources
	for _, name := range changesetChanges.RemovedResources {
		item := &ResourceDeployItem{
			Name:   name,
			Action: ActionDelete,
		}
		resourcesByName[name] = item
		items = append(items, DeployItem{
			Type:     ItemTypeResource,
			Resource: item,
		})
	}

	return items
}

// appendChildItems adds child blueprint items from changeset to the items slice.
func appendChildItems(
	items []DeployItem,
	changesetChanges *changes.BlueprintChanges,
	childrenByName map[string]*ChildDeployItem,
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
			Type:    ItemTypeChild,
			Child:   item,
			Changes: childChanges,
		})
	}

	// Changed children
	for name, cc := range changesetChanges.ChildChanges {
		ccCopy := cc
		item := &ChildDeployItem{
			Name:    name,
			Action:  ActionUpdate,
			Changes: &ccCopy,
		}
		childrenByName[name] = item
		items = append(items, DeployItem{
			Type:    ItemTypeChild,
			Child:   item,
			Changes: &ccCopy,
		})
	}

	// Recreate children
	for _, name := range changesetChanges.RecreateChildren {
		item := &ChildDeployItem{
			Name:   name,
			Action: ActionRecreate,
		}
		childrenByName[name] = item
		items = append(items, DeployItem{
			Type:  ItemTypeChild,
			Child: item,
		})
	}

	// Removed children
	for _, name := range changesetChanges.RemovedChildren {
		item := &ChildDeployItem{
			Name:   name,
			Action: ActionDelete,
		}
		childrenByName[name] = item
		items = append(items, DeployItem{
			Type:  ItemTypeChild,
			Child: item,
		})
	}

	return items
}

// appendLinkItems adds link items from changeset to the items slice.
// Links are found within resource changes as NewOutboundLinks, OutboundLinkChanges, and RemovedOutboundLinks.
func appendLinkItems(
	items []DeployItem,
	changesetChanges *changes.BlueprintChanges,
	linksByName map[string]*LinkDeployItem,
) []DeployItem {
	// Extract links from new resources
	for resourceAName, resourceChanges := range changesetChanges.NewResources {
		// New outbound links from new resources
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
		}
	}

	return items
}
