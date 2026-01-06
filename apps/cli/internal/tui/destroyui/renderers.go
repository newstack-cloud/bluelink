package destroyui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
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

// DestroyDetailsRenderer implements splitpane.DetailsRenderer for destroy UI.
type DestroyDetailsRenderer struct {
	MaxExpandDepth           int
	NavigationStackDepth     int
	PreDestroyInstanceState  *state.InstanceState
	PostDestroyInstanceState *state.InstanceState
	Finished                 bool
}

var _ splitpane.DetailsRenderer = (*DestroyDetailsRenderer)(nil)

// getResourceID returns the resource ID by checking multiple sources:
// 1. The item's ResourceID field (from events)
// 2. Pre-destroy instance state
func (r *DestroyDetailsRenderer) getResourceID(path, resourceName, itemResourceID string) string {
	if itemResourceID != "" {
		return itemResourceID
	}

	if r.PreDestroyInstanceState != nil {
		if resourceID := findResourceIDByPath(r.PreDestroyInstanceState, path, resourceName); resourceID != "" {
			return resourceID
		}
	}

	return ""
}

// getChildInstanceID returns the instance ID for a child blueprint by checking instance state.
func (r *DestroyDetailsRenderer) getChildInstanceID(path, itemInstanceID string) string {
	if itemInstanceID != "" {
		return itemInstanceID
	}

	if r.PreDestroyInstanceState != nil {
		if instanceID := findChildInstanceIDByPath(r.PreDestroyInstanceState, path); instanceID != "" {
			return instanceID
		}
	}

	return ""
}

// getLinkID returns the link ID by checking instance state.
func (r *DestroyDetailsRenderer) getLinkID(path, linkName, itemLinkID string) string {
	if itemLinkID != "" {
		return itemLinkID
	}

	if r.PreDestroyInstanceState != nil {
		if linkID := findLinkIDByPath(r.PreDestroyInstanceState, path, linkName); linkID != "" {
			return linkID
		}
	}

	return ""
}

// Re-export shared helper functions for internal use.
var (
	findResourceIDByPath     = shared.FindResourceIDByPath
	findChildInstanceIDByPath = shared.FindChildInstanceIDByPath
	findLinkIDByPath         = shared.FindLinkIDByPath
)

// RenderDetails renders the right pane content for a selected item.
func (r *DestroyDetailsRenderer) RenderDetails(item splitpane.Item, width int, s *styles.Styles) string {
	destroyItem, ok := item.(*DestroyItem)
	if !ok {
		return s.Muted.Render("Unknown item type")
	}

	switch destroyItem.Type {
	case ItemTypeResource:
		return r.renderResourceDetails(destroyItem, width, s)
	case ItemTypeChild:
		return r.renderChildDetails(destroyItem, width, s)
	case ItemTypeLink:
		return r.renderLinkDetails(destroyItem, width, s)
	default:
		return s.Muted.Render("Unknown item type")
	}
}

func (r *DestroyDetailsRenderer) renderResourceDetails(item *DestroyItem, width int, s *styles.Styles) string {
	res := item.Resource
	if res == nil {
		return s.Muted.Render("No resource data")
	}

	sb := strings.Builder{}

	headerText := res.Name
	if res.DisplayName != "" {
		headerText = res.DisplayName
	}
	sb.WriteString(s.Header.Render(headerText))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	resourceID := r.getResourceID(item.Path, res.Name, res.ResourceID)
	if resourceID != "" {
		sb.WriteString(s.Muted.Render("Resource ID: "))
		sb.WriteString(resourceID)
		sb.WriteString("\n")
	}
	if res.ResourceType != "" {
		sb.WriteString(s.Muted.Render("Type: "))
		sb.WriteString(res.ResourceType)
		sb.WriteString("\n")
	}

	sb.WriteString(s.Muted.Render("Status: "))
	if res.Skipped {
		sb.WriteString(s.Warning.Render("Skipped"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString("Not attempted due to destroy failure")
	} else {
		sb.WriteString(renderResourceStatus(res.Status, s))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString(renderPreciseResourceStatus(res.PreciseStatus))
	}
	sb.WriteString("\n")

	if res.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(renderAction(res.Action, s))
		sb.WriteString("\n")
	}

	if len(res.FailureReasons) > 0 {
		sb.WriteString("\n")
		sb.WriteString(s.Error.Render("Failure Reasons:"))
		sb.WriteString("\n\n")
		reasonWidth := ui.SafeWidth(width - 2)
		wrapStyle := lipgloss.NewStyle().Width(reasonWidth)
		for i, reason := range res.FailureReasons {
			wrappedReason := wrapStyle.Render(reason)
			sb.WriteString(s.Error.Render(wrappedReason))
			if i < len(res.FailureReasons)-1 {
				sb.WriteString("\n\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *DestroyDetailsRenderer) renderChildDetails(item *DestroyItem, _ int, s *styles.Styles) string {
	child := item.Child
	if child == nil {
		return s.Muted.Render("No child data")
	}

	sb := strings.Builder{}

	sb.WriteString(s.Header.Render(child.Name))
	sb.WriteString("\n\n")

	childPath := item.Path
	if childPath == "" {
		childPath = child.Name
	}
	instanceID := r.getChildInstanceID(childPath, child.ChildInstanceID)
	if instanceID != "" {
		sb.WriteString(s.Muted.Render("Instance ID: "))
		sb.WriteString(instanceID)
		sb.WriteString("\n")
	}

	sb.WriteString(s.Muted.Render("Status: "))
	if child.Skipped {
		sb.WriteString(s.Warning.Render("Skipped"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString("Not attempted due to destroy failure")
	} else {
		sb.WriteString(renderInstanceStatus(child.Status, s))
	}
	sb.WriteString("\n")

	if child.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(renderAction(child.Action, s))
		sb.WriteString("\n")
	}

	if len(child.FailureReasons) > 0 {
		sb.WriteString("\n")
		sb.WriteString(s.Error.Render("Failure Reasons:"))
		sb.WriteString("\n")
		for _, reason := range child.FailureReasons {
			sb.WriteString(s.Error.Render("  • " + reason))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (r *DestroyDetailsRenderer) renderLinkDetails(item *DestroyItem, width int, s *styles.Styles) string {
	link := item.Link
	if link == nil {
		return s.Muted.Render("No link data")
	}

	sb := strings.Builder{}

	sb.WriteString(s.Header.Render(link.LinkName))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	sb.WriteString(s.Muted.Render("Resource A: "))
	sb.WriteString(link.ResourceAName)
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render("Resource B: "))
	sb.WriteString(link.ResourceBName)
	sb.WriteString("\n")

	linkID := r.getLinkID(item.Path, link.LinkName, link.LinkID)
	if linkID != "" {
		sb.WriteString(s.Muted.Render("Link ID: "))
		sb.WriteString(linkID)
		sb.WriteString("\n")
	}

	sb.WriteString(s.Muted.Render("Status: "))
	if link.Skipped {
		sb.WriteString(s.Warning.Render("Skipped"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString("Not attempted due to destroy failure")
		sb.WriteString("\n")
	} else {
		sb.WriteString(renderLinkStatus(link.Status, s))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString(renderPreciseLinkStatus(link.PreciseStatus))
		sb.WriteString("\n")
	}

	if link.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(renderAction(link.Action, s))
		sb.WriteString("\n")
	}

	if len(link.FailureReasons) > 0 {
		sb.WriteString("\n")
		sb.WriteString(s.Error.Render("Failure Reasons:"))
		sb.WriteString("\n")
		for _, reason := range link.FailureReasons {
			sb.WriteString(s.Error.Render("  • " + reason))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// DestroySectionGrouper groups items into sections for the destroy UI.
type DestroySectionGrouper struct {
	MaxExpandDepth int
}

var _ splitpane.SectionGrouper = (*DestroySectionGrouper)(nil)

// GroupItems groups items into resources, children, and links sections.
func (g *DestroySectionGrouper) GroupItems(items []splitpane.Item, isExpanded func(id string) bool) []splitpane.Section {
	var resources, children, links []splitpane.Item

	for _, item := range items {
		destroyItem, ok := item.(*DestroyItem)
		if !ok {
			continue
		}

		// Nested items (with ParentChild set) go to children section
		if destroyItem.ParentChild != "" {
			children = append(children, item)
			continue
		}

		switch destroyItem.Type {
		case ItemTypeResource:
			resources = append(resources, item)
		case ItemTypeChild:
			children = append(children, item)
			// If expanded, recursively add children inline
			children = g.appendExpandedChildren(children, item, isExpanded)
		case ItemTypeLink:
			links = append(links, item)
		}
	}

	// Sort each section for consistent ordering
	sortDestroyItems(resources)
	sortDestroyItems(links)

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
func (g *DestroySectionGrouper) appendExpandedChildren(
	children []splitpane.Item,
	item splitpane.Item,
	isExpanded func(id string) bool,
) []splitpane.Item {
	if isExpanded == nil || !isExpanded(item.GetID()) {
		return children
	}

	if item.GetDepth() >= g.MaxExpandDepth {
		return children
	}

	childItems := item.GetChildren()
	sortDestroyItems(childItems)

	for _, child := range childItems {
		children = append(children, child)
		if child.IsExpandable() {
			children = g.appendExpandedChildren(children, child, isExpanded)
		}
	}

	return children
}

func sortDestroyItems(items []splitpane.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
}

// DestroyFooterRenderer renders the footer for the destroy split-pane.
type DestroyFooterRenderer struct {
	InstanceID          string
	InstanceName        string
	ChangesetID         string
	CurrentStatus       core.InstanceStatus
	FinalStatus         core.InstanceStatus
	Finished            bool
	SpinnerView         string
	HasInstanceState    bool
	DestroyedElements   []DestroyedElement
	ElementFailures     []ElementFailure
	InterruptedElements []InterruptedElement
}

var _ splitpane.FooterRenderer = (*DestroyFooterRenderer)(nil)

// RenderFooter renders the footer content.
func (r *DestroyFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	if model.IsInDrillDown() {
		return r.renderDrillDownFooter(model, s)
	}

	if r.Finished {
		sb.WriteString(r.renderFinishedFooter(s))
	} else {
		sb.WriteString(r.renderStreamingFooter(s))
	}

	sb.WriteString("\n")

	// Navigation help
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" navigate  "))
	sb.WriteString(s.Key.Render("tab"))
	sb.WriteString(s.Muted.Render(" switch pane  "))
	sb.WriteString(s.Key.Render("q"))
	sb.WriteString(s.Muted.Render(" quit"))
	sb.WriteString("\n")

	return sb.String()
}

func (r *DestroyFooterRenderer) renderDrillDownFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}

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
	sb.WriteString(s.Key.Render("q"))
	sb.WriteString(s.Muted.Render(" quit"))
	sb.WriteString("\n")

	return sb.String()
}

func (r *DestroyFooterRenderer) renderStreamingFooter(s *styles.Styles) string {
	sb := strings.Builder{}

	sb.WriteString("  ")
	if r.SpinnerView != "" {
		sb.WriteString(r.SpinnerView)
		sb.WriteString(" ")
	}
	sb.WriteString(s.Info.Render("Destroying "))
	if r.InstanceName != "" {
		italicStyle := lipgloss.NewStyle().Italic(true)
		sb.WriteString(italicStyle.Render(r.InstanceName))
	}
	sb.WriteString("\n")

	if r.ChangesetID != "" {
		sb.WriteString(s.Muted.Render("  Changeset: "))
		sb.WriteString(s.Selected.Render(r.ChangesetID))
		sb.WriteString("\n")
	}

	if IsRollingBackStatus(r.CurrentStatus) {
		sb.WriteString(s.Warning.Render("  Rolling back..."))
		sb.WriteString("\n")
	}

	// Show pre-destroy state hint when available
	if r.HasInstanceState {
		sb.WriteString(s.Muted.Render("  press "))
		sb.WriteString(s.Key.Render("s"))
		sb.WriteString(s.Muted.Render(" for pre-destroy state"))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *DestroyFooterRenderer) renderFinishedFooter(s *styles.Styles) string {
	sb := strings.Builder{}
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	// Destroy complete - compact format
	sb.WriteString(s.Muted.Render("  Destroy "))
	sb.WriteString(r.renderFinalStatus(s, successStyle))
	if r.InstanceName != "" {
		sb.WriteString(s.Muted.Render(" • "))
		sb.WriteString(s.Selected.Render(r.InstanceName))
	}
	sb.WriteString(s.Muted.Render(" - press "))
	sb.WriteString(s.Key.Render("o"))
	sb.WriteString(s.Muted.Render(" for overview"))
	if r.HasInstanceState {
		sb.WriteString(s.Muted.Render(", "))
		sb.WriteString(s.Key.Render("s"))
		sb.WriteString(s.Muted.Render(" for pre-destroy state"))
	}
	sb.WriteString("\n")

	// Show summary of destroyed, failed, and interrupted elements
	hasSummary := len(r.DestroyedElements) > 0 || len(r.ElementFailures) > 0 || len(r.InterruptedElements) > 0
	if hasSummary {
		sb.WriteString("  ")
		needsComma := false
		if len(r.DestroyedElements) > 0 {
			sb.WriteString(successStyle.Render(fmt.Sprintf("%d destroyed", len(r.DestroyedElements))))
			needsComma = true
		}
		if len(r.ElementFailures) > 0 {
			if needsComma {
				sb.WriteString(s.Muted.Render(", "))
			}
			sb.WriteString(s.Error.Render(fmt.Sprintf("%d %s", len(r.ElementFailures), sdkstrings.Pluralize(len(r.ElementFailures), "failure", "failures"))))
			needsComma = true
		}
		if len(r.InterruptedElements) > 0 {
			if needsComma {
				sb.WriteString(s.Muted.Render(", "))
			}
			sb.WriteString(s.Warning.Render(fmt.Sprintf("%d interrupted", len(r.InterruptedElements))))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *DestroyFooterRenderer) renderFinalStatus(s *styles.Styles, successStyle lipgloss.Style) string {
	switch r.FinalStatus {
	case core.InstanceStatusDestroyed:
		return successStyle.Render("complete")
	case core.InstanceStatusDestroyFailed:
		return s.Error.Render("failed")
	case core.InstanceStatusDestroyRollbackComplete:
		return s.Warning.Render("rolled back")
	case core.InstanceStatusDestroyRollbackFailed:
		return s.Error.Render("rollback failed")
	default:
		return s.Muted.Render("unknown")
	}
}

// DestroyStagingFooterRenderer renders the footer during staging in destroy flow.
type DestroyStagingFooterRenderer struct {
	ChangesetID      string
	Summary          ChangeSummary
	HasExportChanges bool
}

var _ splitpane.FooterRenderer = (*DestroyStagingFooterRenderer)(nil)

// RenderFooter renders the staging confirmation footer.
func (r *DestroyStagingFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	// Show breadcrumb when in drill-down
	if model.IsInDrillDown() {
		sb.WriteString(s.Muted.Render("  Viewing: "))
		for i, name := range model.NavigationPath() {
			if i > 0 {
				sb.WriteString(s.Muted.Render(" > "))
			}
			sb.WriteString(s.Selected.Render(name))
		}
		sb.WriteString("\n\n")

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

	// Staging summary with changeset ID
	sb.WriteString(s.Muted.Render("  Staging complete. Changeset: "))
	sb.WriteString(s.Selected.Render(r.ChangesetID))
	sb.WriteString(s.Muted.Render(" - press "))
	sb.WriteString(s.Key.Render("o"))
	sb.WriteString(s.Muted.Render(" for overview"))
	sb.WriteString("\n")

	// Delete summary
	sb.WriteString("  ")
	sb.WriteString(s.Error.Render(fmt.Sprintf("%d to delete", r.Summary.Delete)))
	sb.WriteString("\n")

	// Confirmation prompt
	sb.WriteString(s.Muted.Render("  Destroy these resources? "))
	sb.WriteString(s.Key.Render("y"))
	sb.WriteString(s.Muted.Render("/"))
	sb.WriteString(s.Key.Render("n"))
	sb.WriteString("\n\n")

	// Navigation help
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

// Type aliases for drift UI components from driftui package
type (
	DriftDetailsRenderer = driftui.DriftDetailsRenderer
	DriftSectionGrouper  = driftui.DriftSectionGrouper
	DriftFooterRenderer  = driftui.DriftFooterRenderer
)

// Status helper functions

func renderResourceStatus(status core.ResourceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.ResourceStatusDestroying:
		return s.Info.Render("Destroying")
	case core.ResourceStatusDestroyed:
		return successStyle.Render("Destroyed")
	case core.ResourceStatusDestroyFailed:
		return s.Error.Render("Destroy Failed")
	case core.ResourceStatusRollingBack:
		return s.Warning.Render("Rolling Back")
	case core.ResourceStatusRollbackComplete:
		return s.Muted.Render("Rolled Back")
	case core.ResourceStatusRollbackFailed:
		return s.Error.Render("Rollback Failed")
	case core.ResourceStatusDestroyInterrupted:
		return s.Warning.Render("Interrupted")
	default:
		return s.Muted.Render("Pending")
	}
}

func renderPreciseResourceStatus(status core.PreciseResourceStatus) string {
	switch status {
	case core.PreciseResourceStatusDestroying:
		return "Deleting resource..."
	case core.PreciseResourceStatusDestroyed:
		return "Resource deleted"
	case core.PreciseResourceStatusDestroyFailed:
		return "Failed to delete resource"
	case core.PreciseResourceStatusDestroyRollingBack:
		return "Rolling back deletion..."
	case core.PreciseResourceStatusDestroyRollbackComplete:
		return "Deletion rolled back"
	case core.PreciseResourceStatusDestroyRollbackFailed:
		return "Failed to rollback deletion"
	case core.PreciseResourceStatusDestroyInterrupted:
		return "Deletion interrupted"
	default:
		return "Pending"
	}
}

func renderInstanceStatus(status core.InstanceStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.InstanceStatusDestroying:
		return s.Info.Render("Destroying")
	case core.InstanceStatusDestroyed:
		return successStyle.Render("Destroyed")
	case core.InstanceStatusDestroyFailed:
		return s.Error.Render("Destroy Failed")
	case core.InstanceStatusDestroyRollingBack:
		return s.Warning.Render("Rolling Back")
	case core.InstanceStatusDestroyRollbackComplete:
		return s.Muted.Render("Rolled Back")
	case core.InstanceStatusDestroyRollbackFailed:
		return s.Error.Render("Rollback Failed")
	case core.InstanceStatusDestroyInterrupted:
		return s.Warning.Render("Interrupted")
	default:
		return s.Muted.Render("Pending")
	}
}

func renderLinkStatus(status core.LinkStatus, s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	switch status {
	case core.LinkStatusDestroying:
		return s.Info.Render("Destroying")
	case core.LinkStatusDestroyed:
		return successStyle.Render("Destroyed")
	case core.LinkStatusDestroyFailed:
		return s.Error.Render("Destroy Failed")
	case core.LinkStatusDestroyRollingBack:
		return s.Warning.Render("Rolling Back")
	case core.LinkStatusDestroyRollbackComplete:
		return s.Muted.Render("Rolled Back")
	case core.LinkStatusDestroyRollbackFailed:
		return s.Error.Render("Rollback Failed")
	case core.LinkStatusDestroyInterrupted:
		return s.Warning.Render("Interrupted")
	default:
		return s.Muted.Render("Pending")
	}
}

func renderPreciseLinkStatus(status core.PreciseLinkStatus) string {
	// Links don't have destroy-specific precise statuses, just return the generic string
	if status == core.PreciseLinkStatusUnknown {
		return "Pending"
	}
	return status.String()
}

func renderAction(action ActionType, s *styles.Styles) string {
	switch action {
	case ActionCreate:
		return s.Info.Render("Create")
	case ActionUpdate:
		return s.Info.Render("Update")
	case ActionDelete:
		return s.Error.Render("Delete")
	case ActionRecreate:
		return s.Warning.Render("Recreate")
	case ActionNoChange:
		return s.Muted.Render("No Change")
	default:
		return s.Muted.Render("Unknown")
	}
}

// BuildDriftItems builds split-pane items from drift reconciliation results.
func BuildDriftItems(result *container.ReconciliationCheckResult, instanceState *state.InstanceState) []splitpane.Item {
	return driftui.BuildDriftItems(result, instanceState)
}

// extractResourceAFromLinkName extracts resource A name from a link name.
func extractResourceAFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 1 {
		return parts[0]
	}
	return linkName
}

// extractResourceBFromLinkName extracts resource B name from a link name.
func extractResourceBFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// ToSplitPaneItems converts DestroyItems to splitpane.Items.
func ToSplitPaneItems(items []DestroyItem) []splitpane.Item {
	result := make([]splitpane.Item, len(items))
	for i := range items {
		result[i] = &items[i]
	}
	return result
}
