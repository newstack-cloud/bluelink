package deployui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// ChangeSummary holds counts of different change types.
type ChangeSummary struct {
	Create   int
	Update   int
	Delete   int
	Recreate int
}

// DeployDetailsRenderer implements splitpane.DetailsRenderer for deploy UI.
type DeployDetailsRenderer struct {
	MaxExpandDepth       int
	NavigationStackDepth int
}

// Ensure DeployDetailsRenderer implements splitpane.DetailsRenderer.
var _ splitpane.DetailsRenderer = (*DeployDetailsRenderer)(nil)

// RenderDetails renders the right pane content for a selected item.
func (r *DeployDetailsRenderer) RenderDetails(item splitpane.Item, width int, s *styles.Styles) string {
	deployItem, ok := item.(*DeployItem)
	if !ok {
		return s.Muted.Render("Unknown item type")
	}

	switch deployItem.Type {
	case ItemTypeResource:
		return r.renderResourceDetails(deployItem.Resource, width, s)
	case ItemTypeChild:
		return r.renderChildDetails(deployItem.Child, width, s)
	case ItemTypeLink:
		return r.renderLinkDetails(deployItem.Link, width, s)
	default:
		return s.Muted.Render("Unknown item type")
	}
}

func (r *DeployDetailsRenderer) renderResourceDetails(res *ResourceDeployItem, width int, s *styles.Styles) string {
	if res == nil {
		return s.Muted.Render("No resource data")
	}

	sb := strings.Builder{}

	// Header
	headerText := res.Name
	if res.DisplayName != "" {
		headerText = res.DisplayName
	}
	sb.WriteString(s.Header.Render(headerText))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Metadata
	if res.DisplayName != "" {
		sb.WriteString(s.Muted.Render("Name: "))
		sb.WriteString(res.Name)
		sb.WriteString("\n")
	}
	if res.ResourceType != "" {
		sb.WriteString(s.Muted.Render("Type: "))
		sb.WriteString(res.ResourceType)
		sb.WriteString("\n")
	}
	if res.ResourceID != "" {
		sb.WriteString(s.Muted.Render("ID: "))
		sb.WriteString(res.ResourceID)
		sb.WriteString("\n")
	}

	// Status
	sb.WriteString(s.Muted.Render("Status: "))
	if res.Skipped {
		sb.WriteString(s.Warning.Render("Skipped"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString("Not attempted due to deployment failure")
	} else {
		sb.WriteString(renderResourceStatus(res.Status, s))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString(renderPreciseResourceStatus(res.PreciseStatus))
	}
	sb.WriteString("\n")

	// Action (from changeset)
	if res.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(renderAction(res.Action, s))
		sb.WriteString("\n")
	}

	// Attempt info
	if res.Attempt > 1 {
		sb.WriteString(s.Muted.Render("Attempt: "))
		sb.WriteString(fmt.Sprintf("%d", res.Attempt))
		if res.CanRetry {
			sb.WriteString(s.Info.Render(" (can retry)"))
		}
		sb.WriteString("\n")
	}

	// Failure reasons
	if len(res.FailureReasons) > 0 {
		sb.WriteString("\n")
		sb.WriteString(s.Error.Render("Failure Reasons:"))
		sb.WriteString("\n\n")
		// Use nearly full width, just account for minimal padding
		reasonWidth := ui.SafeWidth(width - 2)
		wrapStyle := lipgloss.NewStyle().Width(reasonWidth)
		for i, reason := range res.FailureReasons {
			wrappedReason := wrapStyle.Render(reason)
			sb.WriteString(s.Error.Render(wrappedReason))
			// Add spacing between multiple reasons
			if i < len(res.FailureReasons)-1 {
				sb.WriteString("\n\n")
			}
		}
		sb.WriteString("\n")
	}

	// Duration info
	if res.Durations != nil {
		sb.WriteString("\n")
		sb.WriteString(s.Category.Render("Timing:\n"))
		sb.WriteString(renderResourceDurations(res.Durations, s))
	}

	return sb.String()
}

func renderResourceStatus(status core.ResourceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.ResourceStatusCreating:
		return s.Info.Render("Creating")
	case core.ResourceStatusCreated:
		return successStyle.Render("Created")
	case core.ResourceStatusCreateFailed:
		return s.Error.Render("Create Failed")
	case core.ResourceStatusUpdating:
		return s.Info.Render("Updating")
	case core.ResourceStatusUpdated:
		return successStyle.Render("Updated")
	case core.ResourceStatusUpdateFailed:
		return s.Error.Render("Update Failed")
	case core.ResourceStatusDestroying:
		return s.Info.Render("Destroying")
	case core.ResourceStatusDestroyed:
		return successStyle.Render("Destroyed")
	case core.ResourceStatusDestroyFailed:
		return s.Error.Render("Destroy Failed")
	case core.ResourceStatusRollingBack:
		return s.Warning.Render("Rolling Back")
	case core.ResourceStatusRollbackFailed:
		return s.Error.Render("Rollback Failed")
	case core.ResourceStatusRollbackComplete:
		return s.Muted.Render("Rolled Back")
	default:
		return s.Muted.Render("Pending")
	}
}

func renderPreciseResourceStatus(status core.PreciseResourceStatus) string {
	switch status {
	case core.PreciseResourceStatusCreating:
		return "Creating resource..."
	case core.PreciseResourceStatusConfigComplete:
		return "Configuration applied, waiting for stability"
	case core.PreciseResourceStatusCreated:
		return "Resource created and stable"
	case core.PreciseResourceStatusCreateFailed:
		return "Failed to create resource"
	case core.PreciseResourceStatusUpdating:
		return "Updating resource..."
	case core.PreciseResourceStatusUpdateConfigComplete:
		return "Update applied, waiting for stability"
	case core.PreciseResourceStatusUpdated:
		return "Resource updated and stable"
	case core.PreciseResourceStatusUpdateFailed:
		return "Failed to update resource"
	case core.PreciseResourceStatusDestroying:
		return "Destroying resource..."
	case core.PreciseResourceStatusDestroyed:
		return "Resource destroyed"
	case core.PreciseResourceStatusDestroyFailed:
		return "Failed to destroy resource"
	case core.PreciseResourceStatusCreateRollingBack:
		return "Rolling back resource creation..."
	case core.PreciseResourceStatusCreateRollbackFailed:
		return "Failed to roll back resource creation"
	case core.PreciseResourceStatusCreateRollbackComplete:
		return "Resource creation rolled back"
	case core.PreciseResourceStatusUpdateRollingBack:
		return "Rolling back resource update..."
	case core.PreciseResourceStatusUpdateRollbackFailed:
		return "Failed to roll back resource update"
	case core.PreciseResourceStatusUpdateRollbackConfigComplete:
		return "Update rollback applied, waiting for stability"
	case core.PreciseResourceStatusUpdateRollbackComplete:
		return "Resource update rolled back"
	case core.PreciseResourceStatusDestroyRollingBack:
		return "Rolling back resource destruction..."
	case core.PreciseResourceStatusDestroyRollbackFailed:
		return "Failed to roll back resource destruction"
	case core.PreciseResourceStatusDestroyRollbackConfigComplete:
		return "Destruction rollback applied, waiting for stability"
	case core.PreciseResourceStatusDestroyRollbackComplete:
		return "Resource destruction rolled back"
	default:
		return "Pending"
	}
}

func renderResourceDurations(durations *state.ResourceCompletionDurations, s *styles.Styles) string {
	if durations == nil {
		return ""
	}
	sb := strings.Builder{}
	if durations.TotalDuration != nil && *durations.TotalDuration > 0 {
		sb.WriteString(s.Muted.Render(fmt.Sprintf("  Total: %.2f milliseconds\n", *durations.TotalDuration)))
	}
	return sb.String()
}

func renderAction(action ActionType, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())
	switch action {
	case ActionCreate:
		return successStyle.Render("CREATE")
	case ActionUpdate:
		return s.Warning.Render("UPDATE")
	case ActionDelete:
		return s.Error.Render("DELETE")
	case ActionRecreate:
		return s.Info.Render("RECREATE")
	default:
		return s.Muted.Render(string(action))
	}
}

func (r *DeployDetailsRenderer) renderChildDetails(child *ChildDeployItem, width int, s *styles.Styles) string {
	if child == nil {
		return s.Muted.Render("No child data")
	}

	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(child.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Instance IDs
	if child.ChildInstanceID != "" {
		sb.WriteString(s.Muted.Render("Instance ID: "))
		sb.WriteString(child.ChildInstanceID)
		sb.WriteString("\n")
	}
	if child.ParentInstanceID != "" {
		sb.WriteString(s.Muted.Render("Parent Instance: "))
		sb.WriteString(child.ParentInstanceID)
		sb.WriteString("\n")
	}

	// Status
	sb.WriteString(s.Muted.Render("Status: "))
	if child.Skipped {
		sb.WriteString(s.Warning.Render("Skipped"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString("Not attempted due to deployment failure")
		sb.WriteString("\n")
	} else {
		sb.WriteString(renderInstanceStatus(child.Status, s))
		sb.WriteString("\n")
	}

	// Action
	if child.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(renderAction(child.Action, s))
		sb.WriteString("\n")
	}

	// Failure reasons (only show if not skipped)
	if !child.Skipped && len(child.FailureReasons) > 0 {
		sb.WriteString("\n")
		sb.WriteString(s.Error.Render("Failure Reasons:"))
		sb.WriteString("\n\n")
		// Use nearly full width, just account for minimal padding
		reasonWidth := ui.SafeWidth(width - 2)
		wrapStyle := lipgloss.NewStyle().Width(reasonWidth)
		for i, reason := range child.FailureReasons {
			wrappedReason := wrapStyle.Render(reason)
			sb.WriteString(s.Error.Render(wrappedReason))
			// Add spacing between multiple reasons
			if i < len(child.FailureReasons)-1 {
				sb.WriteString("\n\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderInstanceStatus(status core.InstanceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.InstanceStatusPreparing:
		return s.Muted.Render("Preparing")
	case core.InstanceStatusDeploying:
		return s.Info.Render("Deploying")
	case core.InstanceStatusDeployed:
		return successStyle.Render("Deployed")
	case core.InstanceStatusDeployFailed:
		return s.Error.Render("Deploy Failed")
	case core.InstanceStatusUpdating:
		return s.Info.Render("Updating")
	case core.InstanceStatusUpdated:
		return successStyle.Render("Updated")
	case core.InstanceStatusUpdateFailed:
		return s.Error.Render("Update Failed")
	case core.InstanceStatusDestroying:
		return s.Info.Render("Destroying")
	case core.InstanceStatusDestroyed:
		return successStyle.Render("Destroyed")
	case core.InstanceStatusDestroyFailed:
		return s.Error.Render("Destroy Failed")
	case core.InstanceStatusDeployRollingBack:
		return s.Warning.Render("Rolling Back Deploy")
	case core.InstanceStatusDeployRollbackFailed:
		return s.Error.Render("Deploy Rollback Failed")
	case core.InstanceStatusDeployRollbackComplete:
		return s.Muted.Render("Deploy Rolled Back")
	case core.InstanceStatusUpdateRollingBack:
		return s.Warning.Render("Rolling Back Update")
	case core.InstanceStatusUpdateRollbackFailed:
		return s.Error.Render("Update Rollback Failed")
	case core.InstanceStatusUpdateRollbackComplete:
		return s.Muted.Render("Update Rolled Back")
	case core.InstanceStatusDestroyRollingBack:
		return s.Warning.Render("Rolling Back Destroy")
	case core.InstanceStatusDestroyRollbackFailed:
		return s.Error.Render("Destroy Rollback Failed")
	case core.InstanceStatusDestroyRollbackComplete:
		return s.Muted.Render("Destroy Rolled Back")
	case core.InstanceStatusNotDeployed:
		return s.Muted.Render("Not Deployed")
	default:
		return s.Muted.Render("Unknown")
	}
}

func (r *DeployDetailsRenderer) renderLinkDetails(link *LinkDeployItem, width int, s *styles.Styles) string {
	if link == nil {
		return s.Muted.Render("No link data")
	}

	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(link.LinkName))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Resources
	sb.WriteString(s.Muted.Render("Resource A: "))
	sb.WriteString(link.ResourceAName)
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render("Resource B: "))
	sb.WriteString(link.ResourceBName)
	sb.WriteString("\n")

	// Link ID
	if link.LinkID != "" {
		sb.WriteString(s.Muted.Render("Link ID: "))
		sb.WriteString(link.LinkID)
		sb.WriteString("\n")
	}

	// Status
	sb.WriteString(s.Muted.Render("Status: "))
	if link.Skipped {
		sb.WriteString(s.Warning.Render("Skipped"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString("Not attempted due to deployment failure")
		sb.WriteString("\n")
	} else {
		sb.WriteString(renderLinkStatus(link.Status, s))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString(renderPreciseLinkStatus(link.PreciseStatus))
		sb.WriteString("\n")
	}

	// Action
	if link.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(renderAction(link.Action, s))
		sb.WriteString("\n")
	}

	// Stage attempt (only show if not skipped)
	if !link.Skipped && link.CurrentStageAttempt > 1 {
		sb.WriteString(s.Muted.Render("Stage Attempt: "))
		sb.WriteString(fmt.Sprintf("%d", link.CurrentStageAttempt))
		if link.CanRetryCurrentStage {
			sb.WriteString(s.Info.Render(" (can retry)"))
		}
		sb.WriteString("\n")
	}

	// Failure reasons (only show if not skipped)
	if !link.Skipped && len(link.FailureReasons) > 0 {
		sb.WriteString("\n")
		sb.WriteString(s.Error.Render("Failure Reasons:"))
		sb.WriteString("\n\n")
		// Use nearly full width, just account for minimal padding
		reasonWidth := ui.SafeWidth(width - 2)
		wrapStyle := lipgloss.NewStyle().Width(reasonWidth)
		for i, reason := range link.FailureReasons {
			wrappedReason := wrapStyle.Render(reason)
			sb.WriteString(s.Error.Render(wrappedReason))
			// Add spacing between multiple reasons
			if i < len(link.FailureReasons)-1 {
				sb.WriteString("\n\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderLinkStatus(status core.LinkStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.LinkStatusCreating:
		return s.Info.Render("Creating")
	case core.LinkStatusCreated:
		return successStyle.Render("Created")
	case core.LinkStatusCreateFailed:
		return s.Error.Render("Create Failed")
	case core.LinkStatusUpdating:
		return s.Info.Render("Updating")
	case core.LinkStatusUpdated:
		return successStyle.Render("Updated")
	case core.LinkStatusUpdateFailed:
		return s.Error.Render("Update Failed")
	case core.LinkStatusDestroying:
		return s.Info.Render("Destroying")
	case core.LinkStatusDestroyed:
		return successStyle.Render("Destroyed")
	case core.LinkStatusDestroyFailed:
		return s.Error.Render("Destroy Failed")
	case core.LinkStatusCreateRollingBack:
		return s.Warning.Render("Rolling Back Create")
	case core.LinkStatusCreateRollbackFailed:
		return s.Error.Render("Create Rollback Failed")
	case core.LinkStatusCreateRollbackComplete:
		return s.Muted.Render("Create Rolled Back")
	case core.LinkStatusUpdateRollingBack:
		return s.Warning.Render("Rolling Back Update")
	case core.LinkStatusUpdateRollbackFailed:
		return s.Error.Render("Update Rollback Failed")
	case core.LinkStatusUpdateRollbackComplete:
		return s.Muted.Render("Update Rolled Back")
	case core.LinkStatusDestroyRollingBack:
		return s.Warning.Render("Rolling Back Destroy")
	case core.LinkStatusDestroyRollbackFailed:
		return s.Error.Render("Destroy Rollback Failed")
	case core.LinkStatusDestroyRollbackComplete:
		return s.Muted.Render("Destroy Rolled Back")
	default:
		return s.Muted.Render("Pending")
	}
}

func renderPreciseLinkStatus(status core.PreciseLinkStatus) string {
	switch status {
	case core.PreciseLinkStatusUpdatingResourceA:
		return "Updating resource A..."
	case core.PreciseLinkStatusResourceAUpdated:
		return "Resource A updated"
	case core.PreciseLinkStatusResourceAUpdateFailed:
		return "Failed to update resource A"
	case core.PreciseLinkStatusResourceAUpdateRollingBack:
		return "Rolling back resource A update..."
	case core.PreciseLinkStatusResourceAUpdateRollbackFailed:
		return "Failed to roll back resource A update"
	case core.PreciseLinkStatusResourceAUpdateRollbackComplete:
		return "Resource A update rolled back"
	case core.PreciseLinkStatusUpdatingResourceB:
		return "Updating resource B..."
	case core.PreciseLinkStatusResourceBUpdated:
		return "Resource B updated"
	case core.PreciseLinkStatusResourceBUpdateFailed:
		return "Failed to update resource B"
	case core.PreciseLinkStatusResourceBUpdateRollingBack:
		return "Rolling back resource B update..."
	case core.PreciseLinkStatusResourceBUpdateRollbackFailed:
		return "Failed to roll back resource B update"
	case core.PreciseLinkStatusResourceBUpdateRollbackComplete:
		return "Resource B update rolled back"
	case core.PreciseLinkStatusUpdatingIntermediaryResources:
		return "Updating intermediary resources..."
	case core.PreciseLinkStatusIntermediaryResourcesUpdated:
		return "Intermediary resources updated"
	case core.PreciseLinkStatusIntermediaryResourceUpdateFailed:
		return "Failed to update intermediary resources"
	case core.PreciseLinkStatusIntermediaryResourceUpdateRollingBack:
		return "Rolling back intermediary resources..."
	case core.PreciseLinkStatusIntermediaryResourceUpdateRollbackFailed:
		return "Failed to roll back intermediary resources"
	case core.PreciseLinkStatusIntermediaryResourceUpdateRollbackComplete:
		return "Intermediary resources rolled back"
	default:
		return "Pending"
	}
}

// DeploySectionGrouper implements splitpane.SectionGrouper for deploy UI.
type DeploySectionGrouper struct {
	MaxExpandDepth int
}

// Ensure DeploySectionGrouper implements splitpane.SectionGrouper.
var _ splitpane.SectionGrouper = (*DeploySectionGrouper)(nil)

// GroupItems organizes items into sections: Resources, Child Blueprints, Links.
// The isExpanded function is provided by the splitpane to query expansion state.
func (g *DeploySectionGrouper) GroupItems(items []splitpane.Item, isExpanded func(id string) bool) []splitpane.Section {
	var resources []splitpane.Item
	var children []splitpane.Item
	var links []splitpane.Item

	for _, item := range items {
		deployItem, ok := item.(*DeployItem)
		if !ok {
			continue
		}

		// Nested items (with ParentChild set) go to children section
		if deployItem.ParentChild != "" {
			children = append(children, item)
			continue
		}

		switch deployItem.Type {
		case ItemTypeResource:
			resources = append(resources, item)
		case ItemTypeChild:
			children = append(children, item)
			// If expanded, recursively add children inline (respecting max depth)
			children = g.appendExpandedChildren(children, item, isExpanded)
		case ItemTypeLink:
			links = append(links, item)
		}
	}

	// Sort each section for consistent ordering
	sortDeployItems(resources)
	sortDeployItems(links)
	// Don't sort children slice as it contains both parents and their expanded children
	// which need to maintain their relative positions

	var sections []splitpane.Section

	if len(resources) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Resources",
			Items: resources,
		})
	}

	if len(children) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Child Blueprints",
			Items: children,
		})
	}

	if len(links) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Links",
			Items: links,
		})
	}

	return sections
}

// appendExpandedChildren recursively appends children of an expanded item.
// It handles nested expansions (grandchildren, etc.) by checking each child's expansion state.
// Expansion stops when the item's depth reaches MaxExpandDepth.
func (g *DeploySectionGrouper) appendExpandedChildren(children []splitpane.Item, item splitpane.Item, isExpanded func(id string) bool) []splitpane.Item {
	if isExpanded == nil || !isExpanded(item.GetID()) {
		return children
	}

	// Check if item is at or beyond max expand depth
	if item.GetDepth() >= g.MaxExpandDepth {
		return children
	}

	childItems := item.GetChildren()
	// Sort child items for consistent ordering
	sortDeployItems(childItems)

	for _, child := range childItems {
		children = append(children, child)
		// Recursively check if this child is also expanded (depth check happens in recursive call)
		if child.IsExpandable() {
			children = g.appendExpandedChildren(children, child, isExpanded)
		}
	}

	return children
}

func sortDeployItems(items []splitpane.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
}

// DeployFooterRenderer implements splitpane.FooterRenderer for deploy UI.
type DeployFooterRenderer struct {
	InstanceID     string
	InstanceName   string
	ChangesetID    string
	CurrentStatus  core.InstanceStatus
	FinalStatus    core.InstanceStatus
	FailureReasons []string
	Finished       bool
}

// Ensure DeployFooterRenderer implements splitpane.FooterRenderer.
var _ splitpane.FooterRenderer = (*DeployFooterRenderer)(nil)

// RenderFooter renders the deploy-specific footer.
func (r *DeployFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	if model.IsInDrillDown() {
		// Show breadcrumb path when viewing a child blueprint
		sb.WriteString(s.Muted.Render("  Viewing: "))
		for i, name := range model.NavigationPath() {
			if i > 0 {
				sb.WriteString(s.Muted.Render(" > "))
			}
			sb.WriteString(s.Selected.Render(name))
		}
		sb.WriteString("\n\n")

		// Navigation help for child view
		sb.WriteString(s.Muted.Render("  "))
		sb.WriteString(s.Key.Render("esc"))
		sb.WriteString(s.Muted.Render(" back  "))
		sb.WriteString(s.Key.Render("↑/↓"))
		sb.WriteString(s.Muted.Render(" navigate  "))
		sb.WriteString(s.Key.Render("tab"))
		sb.WriteString(s.Muted.Render(" switch pane  "))
		if r.Finished {
			sb.WriteString(s.Key.Render("q"))
			sb.WriteString(s.Muted.Render(" quit"))
		}
		sb.WriteString("\n")

		return sb.String()
	}

	if r.Finished {
		// Deployment complete
		sb.WriteString(s.Muted.Render("  Deployment "))
		sb.WriteString(renderFinalStatus(r.FinalStatus, s))
		sb.WriteString("\n")

		sb.WriteString(s.Muted.Render("  Instance ID: "))
		sb.WriteString(s.Selected.Render(r.InstanceID))
		sb.WriteString("\n")

		if r.InstanceName != "" {
			sb.WriteString(s.Muted.Render("  Instance Name: "))
			sb.WriteString(s.Selected.Render(r.InstanceName))
			sb.WriteString("\n")
		}

		// Show failure reasons if present
		if len(r.FailureReasons) > 0 {
			sb.WriteString("\n")
			sb.WriteString(s.Error.Render("  Failure Reasons:"))
			sb.WriteString("\n")
			for _, reason := range r.FailureReasons {
				sb.WriteString(s.Error.Render("  • "))
				sb.WriteString(s.Error.Render(reason))
				sb.WriteString("\n")
			}
		}
	} else {
		// Deployment in progress
		sb.WriteString(s.Info.Render("  Deploying..."))
		sb.WriteString("\n")

		if r.ChangesetID != "" {
			sb.WriteString(s.Muted.Render("  Changeset: "))
			sb.WriteString(s.Selected.Render(r.ChangesetID))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")

	// Navigation help
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" navigate  "))
	sb.WriteString(s.Key.Render("tab"))
	sb.WriteString(s.Muted.Render(" switch pane  "))
	if r.Finished {
		sb.WriteString(s.Key.Render("q"))
		sb.WriteString(s.Muted.Render(" quit"))
	}
	sb.WriteString("\n")

	return sb.String()
}

func renderFinalStatus(status core.InstanceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.InstanceStatusDeployed, core.InstanceStatusUpdated, core.InstanceStatusDestroyed:
		return successStyle.Render("complete")
	case core.InstanceStatusDeployFailed, core.InstanceStatusUpdateFailed, core.InstanceStatusDestroyFailed:
		return s.Error.Render("failed")
	case core.InstanceStatusDeployRollbackComplete, core.InstanceStatusUpdateRollbackComplete, core.InstanceStatusDestroyRollbackComplete:
		return s.Warning.Render("rolled back")
	case core.InstanceStatusDeployRollbackFailed, core.InstanceStatusUpdateRollbackFailed, core.InstanceStatusDestroyRollbackFailed:
		return s.Error.Render("rollback failed")
	default:
		return s.Muted.Render("unknown")
	}
}

// DeployStagingFooterRenderer implements splitpane.FooterRenderer for the staging
// view when used in the deploy command flow. It shows a confirmation prompt instead
// of the standalone staging footer.
type DeployStagingFooterRenderer struct {
	ChangesetID string
	Summary     ChangeSummary
}

// Ensure DeployStagingFooterRenderer implements splitpane.FooterRenderer.
var _ splitpane.FooterRenderer = (*DeployStagingFooterRenderer)(nil)

// RenderFooter renders the footer with staging summary and confirmation prompt.
// The footer height matches the original StageFooterRenderer for consistent split pane layout.
func (r *DeployStagingFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	// Show different footer when viewing a child blueprint
	if model.IsInDrillDown() {
		// Show breadcrumb path
		sb.WriteString(s.Muted.Render("  Viewing: "))
		for i, name := range model.NavigationPath() {
			if i > 0 {
				sb.WriteString(s.Muted.Render(" > "))
			}
			sb.WriteString(s.Selected.Render(name))
		}
		sb.WriteString("\n\n")

		// Navigation help for child view
		sb.WriteString(s.Muted.Render("  "))
		sb.WriteString(s.Key.Render("esc"))
		sb.WriteString(s.Muted.Render(" back  "))
		sb.WriteString(s.Key.Render("↑/↓"))
		sb.WriteString(s.Muted.Render(" navigate  "))
		sb.WriteString(s.Key.Render("enter"))
		sb.WriteString(s.Muted.Render(" expand/inspect  "))
		sb.WriteString(s.Key.Render("tab"))
		sb.WriteString(s.Muted.Render(" switch pane  "))
		sb.WriteString(s.Key.Render("q"))
		sb.WriteString(s.Muted.Render(" quit"))
		sb.WriteString("\n")

		return sb.String()
	}

	// Line 1: Staging summary with changeset ID
	sb.WriteString(s.Muted.Render("  Staging complete. Changeset: "))
	sb.WriteString(s.Selected.Render(r.ChangesetID))
	sb.WriteString("\n\n")

	// Line 2-3: Change summary and confirmation prompt (matches "To apply these changes" section)
	sb.WriteString("  ")
	summaryParts := []string{}
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())
	if r.Summary.Create > 0 {
		summaryParts = append(summaryParts, successStyle.Render(fmt.Sprintf("%d to create", r.Summary.Create)))
	}
	if r.Summary.Update > 0 {
		summaryParts = append(summaryParts, s.Warning.Render(fmt.Sprintf("%d to update", r.Summary.Update)))
	}
	if r.Summary.Delete > 0 {
		summaryParts = append(summaryParts, s.Error.Render(fmt.Sprintf("%d to delete", r.Summary.Delete)))
	}
	if r.Summary.Recreate > 0 {
		summaryParts = append(summaryParts, s.Info.Render(fmt.Sprintf("%d to recreate", r.Summary.Recreate)))
	}
	if len(summaryParts) > 0 {
		sb.WriteString(strings.Join(summaryParts, ", "))
	} else {
		sb.WriteString(s.Muted.Render("No changes"))
	}
	sb.WriteString("\n")

	// Line 4: Confirmation prompt (replaces the deploy command line)
	sb.WriteString(s.Muted.Render("  Apply these changes? "))
	sb.WriteString(s.Key.Render("y"))
	sb.WriteString(s.Muted.Render("/"))
	sb.WriteString(s.Key.Render("n"))
	sb.WriteString("\n\n")

	// Line 5: Navigation help
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" navigate  "))
	sb.WriteString(s.Key.Render("enter"))
	sb.WriteString(s.Muted.Render(" expand/collapse  "))
	sb.WriteString(s.Key.Render("tab"))
	sb.WriteString(s.Muted.Render(" switch pane  "))
	sb.WriteString(s.Key.Render("q"))
	sb.WriteString(s.Muted.Render(" quit"))
	sb.WriteString("\n")

	return sb.String()
}
