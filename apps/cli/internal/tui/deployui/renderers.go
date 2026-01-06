package deployui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
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
	MaxExpandDepth          int
	NavigationStackDepth    int
	PreDeployInstanceState  *state.InstanceState // Instance state fetched before deployment for unchanged items
	PostDeployInstanceState *state.InstanceState // Instance state fetched after deployment completes
	Finished                bool                 // True when deployment has finished (enables spec view shortcut)
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
		return r.renderResourceDetails(deployItem, width, s)
	case ItemTypeChild:
		return r.renderChildDetails(deployItem, width, s)
	case ItemTypeLink:
		return r.renderLinkDetails(deployItem, width, s)
	default:
		return s.Muted.Render("Unknown item type")
	}
}

func (r *DeployDetailsRenderer) renderResourceDetails(item *DeployItem, width int, s *styles.Styles) string {
	res := item.Resource
	if res == nil {
		return s.Muted.Render("No resource data")
	}
	parentChild := item.ParentChild

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

	// Get resource state which may have updated resource ID and type after deployment
	resourceState := r.getResourceState(res, item.Path)

	// Resource metadata (ID, name, type)
	shared.RenderResourceMetadata(&sb, shared.ResourceMetadata{
		ResourceID:   res.ResourceID,
		DisplayName:  res.DisplayName,
		Name:         res.Name,
		ResourceType: res.ResourceType,
	}, resourceState, s)

	// Status - only show for items that will be/were deployed
	if res.Action != ActionNoChange {
		sb.WriteString(s.Muted.Render("Status: "))
		if res.Skipped {
			sb.WriteString(s.Warning.Render("Skipped"))
			sb.WriteString("\n")
			sb.WriteString(s.Muted.Render("Details: "))
			sb.WriteString("Not attempted due to deployment failure")
		} else {
			sb.WriteString(shared.RenderResourceStatus(res.Status, s))
			sb.WriteString("\n")
			sb.WriteString(s.Muted.Render("Details: "))
			sb.WriteString(shared.FormatPreciseResourceStatus(res.PreciseStatus))
		}
		sb.WriteString("\n")
	}

	// Action (from changeset)
	if res.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(shared.RenderActionBadge(res.Action, s))
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

	shared.RenderFailureReasons(&sb, res.FailureReasons, width, s)

	// Duration info (only show if there's actual duration data)
	if durationContent := renderResourceDurations(res.Durations, s); durationContent != "" {
		sb.WriteString("\n")
		sb.WriteString(s.Category.Render("Timing:"))
		sb.WriteString("\n")
		sb.WriteString(durationContent)
	}

	// Outputs section - use resource state fetched earlier
	if resourceState != nil {
		outputsContent := r.renderOutputsSection(resourceState, width, s)
		if outputsContent != "" {
			sb.WriteString("\n")
			sb.WriteString(outputsContent)
		}

		// Spec hint - show when resource has spec data (available for completed resources)
		specHint := r.renderSpecHint(resourceState, s)
		if specHint != "" {
			sb.WriteString("\n")
			sb.WriteString(specHint)
			sb.WriteString("\n")
		}
	}

	// Outbound links section
	outboundLinks := r.renderOutboundLinksSection(res.Name, parentChild, s)
	if outboundLinks != "" {
		sb.WriteString("\n")
		sb.WriteString(outboundLinks)
	}

	return sb.String()
}

// getResourceState returns the resource state for a resource item.
// It checks multiple sources in order of freshness:
// 1. Post-deploy instance state (freshest data after deployment completes)
// 2. Pre-deploy instance state (for no-change items)
// 3. ResourceState field on the item (populated from instance state)
// 4. Changeset's CurrentResourceState (pre-deploy data)
// The path parameter contains the full path to the resource (e.g., "childA/childB/resourceName")
// which is used to traverse nested child blueprints in the instance state.
func (r *DeployDetailsRenderer) getResourceState(res *ResourceDeployItem, path string) *state.ResourceState {
	// First try post-deploy state (most up-to-date after deployment completes)
	if r.PostDeployInstanceState != nil {
		if resourceState := findResourceStateByPath(r.PostDeployInstanceState, path, res.Name); resourceState != nil {
			return resourceState
		}
	}

	// Try pre-deploy state for items with no changes
	if r.PreDeployInstanceState != nil {
		if resourceState := findResourceStateByPath(r.PreDeployInstanceState, path, res.Name); resourceState != nil {
			return resourceState
		}
	}

	// Try the resource state field directly (populated when building items)
	if res.ResourceState != nil {
		return res.ResourceState
	}

	// Fall back to pre-deploy state from changeset
	if res.Changes != nil && res.Changes.AppliedResourceInfo.CurrentResourceState != nil {
		return res.Changes.AppliedResourceInfo.CurrentResourceState
	}

	return nil
}

// findResourceStateByPath finds a resource state by traversing the instance state hierarchy
// using the path. The path format is "childA/childB/resourceName" where the last segment
// is the resource name and the preceding segments are child blueprint names.
func findResourceStateByPath(instanceState *state.InstanceState, path string, resourceName string) *state.ResourceState {
	if instanceState == nil {
		return nil
	}

	// Parse the path to extract child blueprint names
	// Path format: "childA/childB/resourceName" or just "resourceName" for top-level
	segments := strings.Split(path, "/")

	// Navigate to the correct child blueprint
	currentState := instanceState
	for i := 0; i < len(segments)-1; i++ {
		childName := segments[i]
		if currentState.ChildBlueprints == nil {
			return nil
		}
		childState, ok := currentState.ChildBlueprints[childName]
		if !ok || childState == nil {
			return nil
		}
		currentState = childState
	}

	// Now look up the resource in the target instance state
	return findResourceStateByName(currentState, resourceName)
}

// Re-export shared helper functions for internal use.
var (
	findResourceStateByName   = shared.FindResourceStateByName
	findChildInstanceIDByPath = shared.FindChildInstanceIDByPath
	findLinkIDByPath          = shared.FindLinkIDByPath
)

// getChildInstanceID returns the instance ID for a child blueprint by traversing
// the instance state hierarchy using the path.
func (r *DeployDetailsRenderer) getChildInstanceID(path string, childName string) string {
	// Try post-deploy state first
	if r.PostDeployInstanceState != nil {
		if instanceID := findChildInstanceIDByPath(r.PostDeployInstanceState, path); instanceID != "" {
			return instanceID
		}
	}

	// Try pre-deploy state
	if r.PreDeployInstanceState != nil {
		if instanceID := findChildInstanceIDByPath(r.PreDeployInstanceState, path); instanceID != "" {
			return instanceID
		}
	}

	return ""
}

// getLinkID returns the link ID for a link by traversing the instance state hierarchy using the path.
func (r *DeployDetailsRenderer) getLinkID(path string, linkName string) string {
	// Try post-deploy state first
	if r.PostDeployInstanceState != nil {
		if linkID := findLinkIDByPath(r.PostDeployInstanceState, path, linkName); linkID != "" {
			return linkID
		}
	}

	// Try pre-deploy state
	if r.PreDeployInstanceState != nil {
		if linkID := findLinkIDByPath(r.PreDeployInstanceState, path, linkName); linkID != "" {
			return linkID
		}
	}

	return ""
}

// renderOutputsSection renders the outputs (computed fields) section.
func (r *DeployDetailsRenderer) renderOutputsSection(resourceState *state.ResourceState, width int, s *styles.Styles) string {
	if resourceState == nil || resourceState.SpecData == nil {
		return ""
	}

	fields := outpututil.CollectOutputFields(resourceState.SpecData, resourceState.ComputedFields)
	if len(fields) == 0 {
		return ""
	}

	return outpututil.RenderOutputFieldsWithLabel(fields, "Outputs:", width, s)
}

// renderSpecHint renders the spec hint line showing field count and shortcut.
func (r *DeployDetailsRenderer) renderSpecHint(resourceState *state.ResourceState, s *styles.Styles) string {
	if resourceState == nil || resourceState.SpecData == nil {
		return ""
	}

	return outpututil.RenderSpecHint(resourceState.SpecData, resourceState.ComputedFields, s)
}

// renderOutboundLinksSection renders the outbound links from this resource.
func (r *DeployDetailsRenderer) renderOutboundLinksSection(resourceName, parentChild string, s *styles.Styles) string {
	if r.PostDeployInstanceState == nil {
		return ""
	}
	return shared.RenderOutboundLinksSection(resourceName, r.PostDeployInstanceState.Links, s)
}

func renderResourceDurations(durations *state.ResourceCompletionDurations, s *styles.Styles) string {
	if durations == nil {
		return ""
	}
	sb := strings.Builder{}
	if durations.ConfigCompleteDuration != nil &&
		*durations.ConfigCompleteDuration > 0 {
		sb.WriteString(s.Muted.Render(fmt.Sprintf(
			"  Config Complete: %s",
			outpututil.FormatDuration(*durations.ConfigCompleteDuration),
		)))
		sb.WriteString("\n")
	}
	if durations.TotalDuration != nil && *durations.TotalDuration > 0 {
		sb.WriteString(s.Muted.Render(fmt.Sprintf(
			"  Total: %s",
			outpututil.FormatDuration(*durations.TotalDuration),
		)))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (r *DeployDetailsRenderer) renderChildDetails(item *DeployItem, width int, s *styles.Styles) string {
	child := item.Child
	if child == nil {
		return s.Muted.Render("No child data")
	}

	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(child.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Instance IDs - try from child item first, then fall back to instance state
	childInstanceID := child.ChildInstanceID
	if childInstanceID == "" {
		// Try to get from post-deploy or pre-deploy instance state
		// For top-level children, item.Path is empty, so use the child name as the path
		childPath := item.Path
		if childPath == "" {
			childPath = child.Name
		}
		childInstanceID = r.getChildInstanceID(childPath, child.Name)
	}
	if childInstanceID != "" {
		sb.WriteString(s.Muted.Render("Instance ID: "))
		sb.WriteString(childInstanceID)
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
		sb.WriteString(shared.RenderInstanceStatus(child.Status, s))
		sb.WriteString("\n")
	}

	// Action
	if child.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(shared.RenderActionBadge(child.Action, s))
		sb.WriteString("\n")
	}

	// Show inspect hint for children at max expand depth with nested content
	effectiveDepth := item.Depth + r.NavigationStackDepth
	if effectiveDepth >= r.MaxExpandDepth && item.Changes != nil {
		sb.WriteString("\n")
		sb.WriteString(s.Hint.Render("Press enter to inspect this child blueprint"))
		sb.WriteString("\n")
	}

	// Failure reasons (only show if not skipped)
	if !child.Skipped {
		shared.RenderFailureReasons(&sb, child.FailureReasons, width, s)
	}

	return sb.String()
}

func (r *DeployDetailsRenderer) renderLinkDetails(item *DeployItem, width int, s *styles.Styles) string {
	link := item.Link
	if link == nil {
		return s.Muted.Render("No link data")
	}

	sb := strings.Builder{}

	// Link ID - try item first, then fall back to post-deploy state
	linkID := link.LinkID
	if linkID == "" {
		linkID = r.getLinkID(item.Path, link.LinkName)
	}

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
		sb.WriteString("Not attempted due to deployment failure")
		sb.WriteString("\n")
	} else {
		sb.WriteString(shared.RenderLinkStatus(link.Status, s))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("Details: "))
		sb.WriteString(shared.FormatPreciseLinkStatus(link.PreciseStatus))
		sb.WriteString("\n")
	}

	// Action
	shared.RenderLinkAction(&sb, string(link.Action), s)

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
	if !link.Skipped {
		shared.RenderFailureReasons(&sb, link.FailureReasons, width, s)
	}

	return sb.String()
}

// DeploySectionGrouper implements splitpane.SectionGrouper for deploy UI.
type DeploySectionGrouper struct {
	shared.SectionGrouper
}

// Ensure DeploySectionGrouper implements splitpane.SectionGrouper.
var _ splitpane.SectionGrouper = (*DeploySectionGrouper)(nil)

// DeployFooterRenderer implements splitpane.FooterRenderer for deploy UI.
type DeployFooterRenderer struct {
	InstanceID          string
	InstanceName        string
	ChangesetID         string
	CurrentStatus       core.InstanceStatus
	FinalStatus         core.InstanceStatus
	FailureReasons      []string             // Legacy: kept for backwards compatibility
	ElementFailures     []ElementFailure     // Structured failures with root cause details
	InterruptedElements []InterruptedElement // Elements that were interrupted
	SuccessfulElements  []SuccessfulElement  // Elements that completed successfully
	Finished            bool
	SpinnerView         string // Current spinner frame for animated "Deploying" state
	HasInstanceState    bool   // Whether instance state is available (enables exports view)
	HasPreRollbackState bool   // Whether pre-rollback state is available (enables pre-rollback view)
}

// Ensure DeployFooterRenderer implements splitpane.FooterRenderer.
var _ splitpane.FooterRenderer = (*DeployFooterRenderer)(nil)

// RenderFooter renders the deploy-specific footer.
func (r *DeployFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	if model.IsInDrillDown() {
		shared.RenderBreadcrumb(&sb, model.NavigationPath(), s)
		shared.RenderFooterNavigation(&sb, s, shared.KeyHint{Key: "esc", Desc: "back"})
		return sb.String()
	}

	if r.Finished {
		// Deployment complete - compact format to fit within footer budget
		sb.WriteString(s.Muted.Render("  Deployment "))
		sb.WriteString(renderFinalStatus(r.FinalStatus, s))
		if r.InstanceName != "" {
			sb.WriteString(s.Muted.Render(" • "))
			sb.WriteString(s.Selected.Render(r.InstanceName))
		}
		sb.WriteString(s.Muted.Render(" - press "))
		sb.WriteString(s.Key.Render("o"))
		sb.WriteString(s.Muted.Render(" for overview"))
		if r.HasInstanceState {
			sb.WriteString(s.Muted.Render(", "))
			sb.WriteString(s.Key.Render("e"))
			sb.WriteString(s.Muted.Render(" for exports"))
		}
		if r.HasPreRollbackState {
			sb.WriteString(s.Muted.Render(", "))
			sb.WriteString(s.Key.Render("r"))
			sb.WriteString(s.Muted.Render(" for pre-rollback state"))
		}
		sb.WriteString("\n")

		shared.RenderElementSummary(&sb, shared.ElementSummary{
			SuccessCount:     len(r.SuccessfulElements),
			SuccessLabel:     "successful",
			FailureCount:     len(r.ElementFailures),
			InterruptedCount: len(r.InterruptedElements),
		}, s)
	} else {
		shared.RenderStreamingFooter(&sb, shared.StreamingFooterParams{
			SpinnerView:      r.SpinnerView,
			ActionVerb:       "Deploying",
			InstanceName:     r.InstanceName,
			ChangesetID:      r.ChangesetID,
			HasInstanceState: r.HasInstanceState,
			StateHintKey:     "e",
			StateHintLabel:   "exports",
		}, s)
	}

	sb.WriteString("\n")

	// Navigation help
	shared.RenderFooterNavigation(&sb, s)

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
	ChangesetID      string
	Summary          ChangeSummary
	HasExportChanges bool
}

// Ensure DeployStagingFooterRenderer implements splitpane.FooterRenderer.
var _ splitpane.FooterRenderer = (*DeployStagingFooterRenderer)(nil)

// RenderFooter renders the footer with staging summary and confirmation prompt.
// The footer height matches the original StageFooterRenderer for consistent split pane layout.
func (r *DeployStagingFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	if shared.RenderStagingDrillDownFooter(&sb, model, s) {
		return sb.String()
	}

	shared.RenderStagingCompleteHeader(&sb, r.ChangesetID, s)

	// Change summary
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

	shared.RenderConfirmationPrompt(&sb, "Apply these changes?", s)

	// Navigation help
	extraKeys := []shared.KeyHint{{Key: "enter", Desc: "expand/collapse"}}
	if r.HasExportChanges {
		extraKeys = append(extraKeys, shared.KeyHint{Key: "e", Desc: "exports"})
	}
	shared.RenderFooterNavigation(&sb, s, extraKeys...)

	return sb.String()
}
