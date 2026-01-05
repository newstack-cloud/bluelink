package deployui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
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

	// Metadata - Resource ID first (top section)
	// Try item's ResourceID first, then fall back to state
	resourceID := res.ResourceID
	if resourceID == "" && resourceState != nil {
		resourceID = resourceState.ResourceID
	}
	if resourceID != "" {
		sb.WriteString(s.Muted.Render("Resource ID: "))
		sb.WriteString(resourceID)
		sb.WriteString("\n")
	}
	if res.DisplayName != "" {
		sb.WriteString(s.Muted.Render("Name: "))
		sb.WriteString(res.Name)
		sb.WriteString("\n")
	}
	// Resource type - try item first, then fall back to state
	resourceType := res.ResourceType
	if resourceType == "" && resourceState != nil {
		resourceType = resourceState.Type
	}
	if resourceType != "" {
		sb.WriteString(s.Muted.Render("Type: "))
		sb.WriteString(resourceType)
		sb.WriteString("\n")
	}

	// Status - only show for items that will be/were deployed
	if res.Action != ActionNoChange {
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
	}

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

		// Spec hint - show field count and shortcut (only when deployment finished)
		if r.Finished {
			specHint := r.renderSpecHint(resourceState, s)
			if specHint != "" {
				sb.WriteString("\n")
				sb.WriteString(specHint)
				sb.WriteString("\n")
			}
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

// findResourceStateByName finds a resource state by name using the instance state's
// ResourceIDs map to look up the resource ID, then retrieves the state from Resources.
func findResourceStateByName(instanceState *state.InstanceState, name string) *state.ResourceState {
	if instanceState == nil || instanceState.ResourceIDs == nil || instanceState.Resources == nil {
		return nil
	}
	resourceID, ok := instanceState.ResourceIDs[name]
	if !ok {
		return nil
	}
	return instanceState.Resources[resourceID]
}

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

// findChildInstanceIDByPath finds a child blueprint's instance ID by traversing the instance state hierarchy.
// The path format is "childA/childB" where each segment is a child blueprint name.
func findChildInstanceIDByPath(instanceState *state.InstanceState, path string) string {
	if instanceState == nil || path == "" {
		return ""
	}

	// Parse the path to navigate to the child blueprint
	segments := strings.Split(path, "/")

	// Navigate to the target child blueprint
	currentState := instanceState
	for _, childName := range segments {
		if currentState.ChildBlueprints == nil {
			return ""
		}
		childState, ok := currentState.ChildBlueprints[childName]
		if !ok || childState == nil {
			return ""
		}
		currentState = childState
	}

	// Return the instance ID of the target child
	return currentState.InstanceID
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

// findLinkIDByPath finds a link's ID by traversing the instance state hierarchy.
// The path format is "childA/childB/linkName" where the last segment is the link name
// and the preceding segments are child blueprint names.
func findLinkIDByPath(instanceState *state.InstanceState, path string, linkName string) string {
	if instanceState == nil {
		return ""
	}

	// Parse the path to extract child blueprint names
	// Path format: "childA/childB/linkName" or just "linkName" for top-level
	segments := strings.Split(path, "/")

	// Navigate to the correct child blueprint
	currentState := instanceState
	for i := 0; i < len(segments)-1; i++ {
		childName := segments[i]
		if currentState.ChildBlueprints == nil {
			return ""
		}
		childState, ok := currentState.ChildBlueprints[childName]
		if !ok || childState == nil {
			return ""
		}
		currentState = childState
	}

	// Now look up the link in the target instance state
	if currentState.Links == nil {
		return ""
	}
	linkState, ok := currentState.Links[linkName]
	if !ok || linkState == nil {
		return ""
	}
	return linkState.LinkID
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
	if r.PostDeployInstanceState == nil || len(r.PostDeployInstanceState.Links) == 0 {
		return ""
	}

	// Find links that originate from this resource (linkName starts with "resourceName::")
	prefix := resourceName + "::"
	var outboundLinks []string
	for linkName, linkState := range r.PostDeployInstanceState.Links {
		if strings.HasPrefix(linkName, prefix) {
			// Extract target resource name
			targetResource := strings.TrimPrefix(linkName, prefix)
			statusStr := formatLinkStatus(linkState.Status)
			outboundLinks = append(outboundLinks, fmt.Sprintf("→ %s (%s)", targetResource, statusStr))
		}
	}

	if len(outboundLinks) == 0 {
		return ""
	}

	// Sort for consistent display
	sort.Strings(outboundLinks)

	sb := strings.Builder{}
	sb.WriteString(s.Category.Render("Outbound Links:"))
	sb.WriteString("\n")
	for _, link := range outboundLinks {
		sb.WriteString(s.Muted.Render("  " + link))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatLinkStatus returns a human-readable status string for a link.
func formatLinkStatus(status core.LinkStatus) string {
	switch status {
	case core.LinkStatusCreated:
		return "created"
	case core.LinkStatusUpdated:
		return "updated"
	case core.LinkStatusDestroyed:
		return "destroyed"
	case core.LinkStatusCreating:
		return "creating"
	case core.LinkStatusUpdating:
		return "updating"
	case core.LinkStatusDestroying:
		return "destroying"
	case core.LinkStatusCreateFailed, core.LinkStatusUpdateFailed, core.LinkStatusDestroyFailed:
		return "failed"
	default:
		return status.String()
	}
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
	case core.ResourceStatusCreateInterrupted,
		core.ResourceStatusUpdateInterrupted,
		core.ResourceStatusDestroyInterrupted:
		return s.Warning.Render("Interrupted")
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
	case core.PreciseResourceStatusCreateInterrupted:
		return "Resource creation was interrupted (actual state unknown)"
	case core.PreciseResourceStatusUpdateInterrupted:
		return "Resource update was interrupted (actual state unknown)"
	case core.PreciseResourceStatusDestroyInterrupted:
		return "Resource destruction was interrupted (actual state unknown)"
	default:
		return "Pending"
	}
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
	case ActionNoChange:
		return s.Muted.Render("NO CHANGE")
	default:
		return s.Muted.Render(string(action))
	}
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
		sb.WriteString(renderInstanceStatus(child.Status, s))
		sb.WriteString("\n")
	}

	// Action
	if child.Action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(renderAction(child.Action, s))
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
	case core.InstanceStatusDeployInterrupted,
		core.InstanceStatusUpdateInterrupted,
		core.InstanceStatusDestroyInterrupted:
		return s.Warning.Render("Interrupted")
	default:
		return s.Muted.Render("Unknown")
	}
}

func (r *DeployDetailsRenderer) renderLinkDetails(item *DeployItem, width int, s *styles.Styles) string {
	link := item.Link
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

	// Link ID - try item first, then fall back to post-deploy state
	linkID := link.LinkID
	if linkID == "" {
		linkID = r.getLinkID(item.Path, link.LinkName)
	}
	if linkID != "" {
		sb.WriteString(s.Muted.Render("Link ID: "))
		sb.WriteString(linkID)
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
	case core.LinkStatusCreateInterrupted,
		core.LinkStatusUpdateInterrupted,
		core.LinkStatusDestroyInterrupted:
		return s.Warning.Render("Interrupted")
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
	case core.PreciseLinkStatusResourceAUpdateInterrupted:
		return "Resource A update was interrupted (actual state unknown)"
	case core.PreciseLinkStatusResourceBUpdateInterrupted:
		return "Resource B update was interrupted (actual state unknown)"
	case core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted:
		return "Intermediary resource update was interrupted (actual state unknown)"
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

		// Show summary of successful, failed, and interrupted elements
		hasSummary := len(r.SuccessfulElements) > 0 || len(r.ElementFailures) > 0 || len(r.InterruptedElements) > 0
		if hasSummary {
			sb.WriteString("  ")
			needsComma := false
			if len(r.SuccessfulElements) > 0 {
				successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())
				sb.WriteString(successStyle.Render(fmt.Sprintf("%d successful", len(r.SuccessfulElements))))
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
	} else {
		// Deployment in progress with animated spinner
		sb.WriteString("  ")
		if r.SpinnerView != "" {
			sb.WriteString(r.SpinnerView)
			sb.WriteString(" ")
		}
		sb.WriteString(s.Info.Render("Deploying "))
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

		// Show exports hint when in progress
		if r.HasInstanceState {
			sb.WriteString(s.Muted.Render("  press "))
			sb.WriteString(s.Key.Render("e"))
			sb.WriteString(s.Muted.Render(" for exports"))
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

	// Line 1: Staging summary with changeset ID and overview hint
	sb.WriteString(s.Muted.Render("  Staging complete. Changeset: "))
	sb.WriteString(s.Selected.Render(r.ChangesetID))
	sb.WriteString(s.Muted.Render(" - press "))
	sb.WriteString(s.Key.Render("o"))
	sb.WriteString(s.Muted.Render(" for overview"))
	sb.WriteString("\n")

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
	if r.HasExportChanges {
		sb.WriteString(s.Key.Render("e"))
		sb.WriteString(s.Muted.Render(" exports  "))
	}
	sb.WriteString(s.Key.Render("q"))
	sb.WriteString(s.Muted.Render(" quit"))
	sb.WriteString("\n")

	return sb.String()
}
