package destroyui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
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
		sb.WriteString(shared.RenderResourceStatus(res.Status, s))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString(shared.FormatPreciseResourceStatus(res.PreciseStatus))
	}
	sb.WriteString("\n")

	if res.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(shared.RenderActionBadge(res.Action, s))
		sb.WriteString("\n")
	}

	shared.RenderFailureReasons(&sb, res.FailureReasons, width, s)

	return sb.String()
}

func (r *DestroyDetailsRenderer) renderChildDetails(item *DestroyItem, width int, s *styles.Styles) string {
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
		sb.WriteString(shared.RenderInstanceStatus(child.Status, s))
	}
	sb.WriteString("\n")

	if child.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(shared.RenderActionBadge(child.Action, s))
		sb.WriteString("\n")
	}

	shared.RenderFailureReasons(&sb, child.FailureReasons, width, s)

	return sb.String()
}

func (r *DestroyDetailsRenderer) renderLinkDetails(item *DestroyItem, width int, s *styles.Styles) string {
	link := item.Link
	if link == nil {
		return s.Muted.Render("No link data")
	}

	sb := strings.Builder{}

	linkID := r.getLinkID(item.Path, link.LinkName, link.LinkID)

	// Common header and metadata
	shared.RenderLinkDetailsBase(&sb, shared.LinkDetailsBase{
		LinkName:      link.LinkName,
		ResourceAName: link.ResourceAName,
		ResourceBName: link.ResourceBName,
		LinkID:        linkID,
	}, width, s)

	// Status
	sb.WriteString(s.Muted.Render("Status: "))
	if link.Skipped {
		sb.WriteString(s.Warning.Render("Skipped"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString("Not attempted due to destroy failure")
		sb.WriteString("\n")
	} else {
		sb.WriteString(shared.RenderLinkStatus(link.Status, s))
		sb.WriteString("\n")
	}

	// Action
	shared.RenderLinkAction(&sb, string(link.Action), s)

	shared.RenderFailureReasons(&sb, link.FailureReasons, width, s)

	return sb.String()
}

// DestroySectionGrouper groups items into sections for the destroy UI.
type DestroySectionGrouper struct {
	shared.SectionGrouper
}

var _ splitpane.SectionGrouper = (*DestroySectionGrouper)(nil)

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
	shared.RenderFooterNavigation(&sb, s)

	return sb.String()
}

func (r *DestroyFooterRenderer) renderDrillDownFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	shared.RenderBreadcrumb(&sb, model.NavigationPath(), s)
	shared.RenderFooterNavigation(&sb, s, shared.KeyHint{Key: "esc", Desc: "back"})
	return sb.String()
}

func (r *DestroyFooterRenderer) renderStreamingFooter(s *styles.Styles) string {
	sb := strings.Builder{}

	shared.RenderStreamingFooter(&sb, shared.StreamingFooterParams{
		SpinnerView:      r.SpinnerView,
		ActionVerb:       "Destroying",
		InstanceName:     r.InstanceName,
		ChangesetID:      r.ChangesetID,
		HasInstanceState: r.HasInstanceState,
		StateHintKey:     "s",
		StateHintLabel:   "pre-destroy state",
	}, s)

	if IsRollingBackStatus(r.CurrentStatus) {
		sb.WriteString(s.Warning.Render("  Rolling back..."))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *DestroyFooterRenderer) renderFinishedFooter(s *styles.Styles) string {
	sb := strings.Builder{}

	// Destroy complete - compact format
	sb.WriteString(s.Muted.Render("  Destroy "))
	sb.WriteString(r.renderFinalStatus(s))
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

	shared.RenderElementSummary(&sb, shared.ElementSummary{
		SuccessCount:     len(r.DestroyedElements),
		SuccessLabel:     "destroyed",
		FailureCount:     len(r.ElementFailures),
		InterruptedCount: len(r.InterruptedElements),
	}, s)

	return sb.String()
}

func (r *DestroyFooterRenderer) renderFinalStatus(s *styles.Styles) string {
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())
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

	if shared.RenderStagingDrillDownFooter(&sb, model, s) {
		return sb.String()
	}

	shared.RenderStagingCompleteHeader(&sb, r.ChangesetID, s)

	// Delete summary
	sb.WriteString("  ")
	sb.WriteString(s.Error.Render(fmt.Sprintf("%d to delete", r.Summary.Delete)))
	sb.WriteString("\n")

	shared.RenderConfirmationPrompt(&sb, "Destroy these resources?", s)

	shared.RenderFooterNavigation(&sb, s, shared.KeyHint{Key: "enter", Desc: "expand/collapse"})

	return sb.String()
}

// Type aliases for drift UI components from driftui package
type (
	DriftDetailsRenderer = driftui.DriftDetailsRenderer
	DriftSectionGrouper  = driftui.DriftSectionGrouper
	DriftFooterRenderer  = driftui.DriftFooterRenderer
)


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
