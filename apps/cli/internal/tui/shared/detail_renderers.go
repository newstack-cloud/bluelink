package shared

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
)

// RenderSectionHeader renders a styled section header with a separator line.
func RenderSectionHeader(sb *strings.Builder, headerText string, width int, s *styles.Styles) {
	sb.WriteString(s.Header.Render(headerText))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")
}

// RenderLabelValue renders a label-value pair with muted label styling.
func RenderLabelValue(sb *strings.Builder, label, value string, s *styles.Styles) {
	sb.WriteString(s.Muted.Render(label + ": "))
	sb.WriteString(value)
	sb.WriteString("\n")
}

// FindResourceStateByName finds a resource state by name using the instance state's
// ResourceIDs map to look up the resource ID, then retrieves the state from Resources.
func FindResourceStateByName(instanceState *state.InstanceState, name string) *state.ResourceState {
	if instanceState == nil || instanceState.ResourceIDs == nil || instanceState.Resources == nil {
		return nil
	}
	resourceID, ok := instanceState.ResourceIDs[name]
	if !ok {
		return nil
	}
	return instanceState.Resources[resourceID]
}

// FindChildInstanceIDByPath finds a child blueprint's instance ID by traversing the instance state hierarchy.
// The path format is "childA/childB" where each segment is a child blueprint name.
func FindChildInstanceIDByPath(instanceState *state.InstanceState, path string) string {
	if instanceState == nil || path == "" {
		return ""
	}

	segments := strings.Split(path, "/")
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

	return currentState.InstanceID
}

// FindLinkIDByPath finds a link's ID by traversing the instance state hierarchy.
// The path format is "childA/childB/linkName" where the preceding segments are child blueprint names.
func FindLinkIDByPath(instanceState *state.InstanceState, path, linkName string) string {
	if instanceState == nil {
		return ""
	}

	segments := strings.Split(path, "/")
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

	if currentState.Links == nil {
		return ""
	}
	linkState, ok := currentState.Links[linkName]
	if !ok || linkState == nil {
		return ""
	}
	return linkState.LinkID
}

// FindResourceIDByPath finds a resource ID by traversing the instance state hierarchy.
// The path format is "childA/childB/resourceName" where the preceding segments are child blueprint names.
func FindResourceIDByPath(instanceState *state.InstanceState, path, resourceName string) string {
	if instanceState == nil {
		return ""
	}

	segments := strings.Split(path, "/")
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

	if currentState.ResourceIDs == nil {
		return ""
	}
	return currentState.ResourceIDs[resourceName]
}

// KeyHint represents a keyboard shortcut hint for footer navigation.
type KeyHint struct {
	Key  string
	Desc string
}

// RenderBreadcrumb renders navigation breadcrumb for drill-down views.
func RenderBreadcrumb(sb *strings.Builder, navigationPath []string, s *styles.Styles) {
	sb.WriteString(s.Muted.Render("  Viewing: "))
	for i, name := range navigationPath {
		if i > 0 {
			sb.WriteString(s.Muted.Render(" > "))
		}
		sb.WriteString(s.Selected.Render(name))
	}
	sb.WriteString("\n\n")
}

// RenderFooterNavigation renders standard keyboard navigation hints.
func RenderFooterNavigation(sb *strings.Builder, s *styles.Styles, extraKeys ...KeyHint) {
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" navigate  "))
	sb.WriteString(s.Key.Render("tab"))
	sb.WriteString(s.Muted.Render(" switch pane  "))
	for _, key := range extraKeys {
		sb.WriteString(s.Key.Render(key.Key))
		sb.WriteString(s.Muted.Render(" " + key.Desc + "  "))
	}
	sb.WriteString(s.Key.Render("q"))
	sb.WriteString(s.Muted.Render(" quit"))
	sb.WriteString("\n")
}

// RenderExportsFooterNavigation renders navigation hints for exports overlay views.
func RenderExportsFooterNavigation(sb *strings.Builder, s *styles.Styles) {
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" navigate  "))
	sb.WriteString(s.Key.Render("tab"))
	sb.WriteString(s.Muted.Render(" switch pane  "))
	sb.WriteString(s.Key.Render("e"))
	sb.WriteString(s.Muted.Render("/"))
	sb.WriteString(s.Key.Render("esc"))
	sb.WriteString(s.Muted.Render(" close"))
	sb.WriteString("\n")
}

// RenderFailureReasons renders failure reasons with word wrapping.
func RenderFailureReasons(sb *strings.Builder, reasons []string, width int, s *styles.Styles) {
	if len(reasons) == 0 {
		return
	}
	sb.WriteString("\n")
	sb.WriteString(s.Error.Render("Failure Reasons:"))
	sb.WriteString("\n\n")
	reasonWidth := ui.SafeWidth(width - 2)
	wrapStyle := lipgloss.NewStyle().Width(reasonWidth)
	for i, reason := range reasons {
		sb.WriteString(s.Error.Render(wrapStyle.Render(reason)))
		if i < len(reasons)-1 {
			sb.WriteString("\n\n")
		}
	}
	sb.WriteString("\n")
}

// ResourceMetadata holds basic resource information for rendering.
type ResourceMetadata struct {
	ResourceID   string
	DisplayName  string
	Name         string
	ResourceType string
}

// RenderResourceMetadata renders the common resource metadata fields (ID, name, type).
func RenderResourceMetadata(sb *strings.Builder, meta ResourceMetadata, resourceState *state.ResourceState, s *styles.Styles) {
	// Resource ID - try provided ID first, then fall back to state
	resourceID := meta.ResourceID
	if resourceID == "" && resourceState != nil {
		resourceID = resourceState.ResourceID
	}
	if resourceID != "" {
		sb.WriteString(s.Muted.Render("Resource ID: "))
		sb.WriteString(resourceID)
		sb.WriteString("\n")
	}

	// Display name (only if different from logical name)
	if meta.DisplayName != "" {
		sb.WriteString(s.Muted.Render("Name: "))
		sb.WriteString(meta.Name)
		sb.WriteString("\n")
	}

	// Resource type - try provided type first, then fall back to state
	resourceType := meta.ResourceType
	if resourceType == "" && resourceState != nil {
		resourceType = resourceState.Type
	}
	if resourceType != "" {
		sb.WriteString(s.Muted.Render("Type: "))
		sb.WriteString(resourceType)
		sb.WriteString("\n")
	}
}

// LinkDetailsBase holds the common fields for rendering link details across UIs.
type LinkDetailsBase struct {
	LinkName      string
	ResourceAName string
	ResourceBName string
	LinkID        string
	Action        string
}

// RenderLinkDetailsBase renders the common link details header and metadata.
// Returns after writing the metadata section, allowing callers to add UI-specific content.
func RenderLinkDetailsBase(sb *strings.Builder, link LinkDetailsBase, width int, s *styles.Styles) {
	// Header
	RenderSectionHeader(sb, link.LinkName, width, s)

	// Resources
	RenderLabelValue(sb, "Resource A", link.ResourceAName, s)
	RenderLabelValue(sb, "Resource B", link.ResourceBName, s)

	// Link ID
	if link.LinkID != "" {
		RenderLabelValue(sb, "Link ID", link.LinkID, s)
	}
}

// RenderLinkAction renders the action badge for a link if action is non-empty.
func RenderLinkAction(sb *strings.Builder, action string, s *styles.Styles) {
	if action != "" {
		sb.WriteString(s.Muted.Render("Action: "))
		sb.WriteString(RenderActionBadge(ActionType(action), s))
		sb.WriteString("\n")
	}
}

// RenderStagingDrillDownFooter renders the common drill-down footer for staging views.
// Returns true if footer was rendered (i.e., model is in drill-down), false otherwise.
func RenderStagingDrillDownFooter(sb *strings.Builder, model SplitPaneModel, s *styles.Styles) bool {
	if !model.IsInDrillDown() {
		return false
	}
	RenderBreadcrumb(sb, model.NavigationPath(), s)
	RenderFooterNavigation(sb, s,
		KeyHint{Key: "esc", Desc: "back"},
		KeyHint{Key: "enter", Desc: "expand/inspect"},
	)
	return true
}

// SplitPaneModel is the interface for split pane model methods needed by footer rendering.
type SplitPaneModel interface {
	IsInDrillDown() bool
	NavigationPath() []string
}

// RenderStagingCompleteHeader renders the "Staging complete. Changeset: X" header line.
func RenderStagingCompleteHeader(sb *strings.Builder, changesetID string, s *styles.Styles) {
	sb.WriteString(s.Muted.Render("  Staging complete. Changeset: "))
	sb.WriteString(s.Selected.Render(changesetID))
	sb.WriteString(s.Muted.Render(" - press "))
	sb.WriteString(s.Key.Render("o"))
	sb.WriteString(s.Muted.Render(" for overview"))
	sb.WriteString("\n")
}

// RenderConfirmationPrompt renders a y/n confirmation prompt.
func RenderConfirmationPrompt(sb *strings.Builder, promptText string, s *styles.Styles) {
	sb.WriteString(s.Muted.Render("  " + promptText + " "))
	sb.WriteString(s.Key.Render("y"))
	sb.WriteString(s.Muted.Render("/"))
	sb.WriteString(s.Key.Render("n"))
	sb.WriteString("\n\n")
}

// ElementSummary holds counts of successful, failed, and interrupted elements for footer rendering.
type ElementSummary struct {
	SuccessCount     int
	SuccessLabel     string // e.g., "successful" or "destroyed"
	FailureCount     int
	InterruptedCount int
}

// RenderElementSummary renders a summary line of element counts (successful/destroyed, failures, interrupted).
func RenderElementSummary(sb *strings.Builder, summary ElementSummary, s *styles.Styles) {
	hasSummary := summary.SuccessCount > 0 || summary.FailureCount > 0 || summary.InterruptedCount > 0
	if !hasSummary {
		return
	}

	sb.WriteString("  ")
	needsComma := false
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	if summary.SuccessCount > 0 {
		label := summary.SuccessLabel
		if label == "" {
			label = "successful"
		}
		sb.WriteString(successStyle.Render(fmt.Sprintf("%d %s", summary.SuccessCount, label)))
		needsComma = true
	}
	if summary.FailureCount > 0 {
		if needsComma {
			sb.WriteString(s.Muted.Render(", "))
		}
		sb.WriteString(s.Error.Render(fmt.Sprintf("%d %s", summary.FailureCount, sdkstrings.Pluralize(summary.FailureCount, "failure", "failures"))))
		needsComma = true
	}
	if summary.InterruptedCount > 0 {
		if needsComma {
			sb.WriteString(s.Muted.Render(", "))
		}
		sb.WriteString(s.Warning.Render(fmt.Sprintf("%d interrupted", summary.InterruptedCount)))
	}
	sb.WriteString("\n")
}

// StreamingFooterParams holds parameters for rendering a streaming operation footer.
type StreamingFooterParams struct {
	SpinnerView      string
	ActionVerb       string // e.g., "Deploying" or "Destroying"
	InstanceName     string
	ChangesetID      string
	HasInstanceState bool
	StateHintKey     string // e.g., "e" for exports or "s" for pre-destroy state
	StateHintLabel   string // e.g., "exports" or "pre-destroy state"
}

// RenderStreamingFooter renders a footer for an in-progress streaming operation.
func RenderStreamingFooter(sb *strings.Builder, params StreamingFooterParams, s *styles.Styles) {
	sb.WriteString("  ")
	if params.SpinnerView != "" {
		sb.WriteString(params.SpinnerView)
		sb.WriteString(" ")
	}
	sb.WriteString(s.Info.Render(params.ActionVerb + " "))
	if params.InstanceName != "" {
		italicStyle := lipgloss.NewStyle().Italic(true)
		sb.WriteString(italicStyle.Render(params.InstanceName))
	}
	sb.WriteString("\n")

	if params.ChangesetID != "" {
		sb.WriteString(s.Muted.Render("  Changeset: "))
		sb.WriteString(s.Selected.Render(params.ChangesetID))
		sb.WriteString("\n")
	}

	if params.HasInstanceState && params.StateHintKey != "" {
		sb.WriteString(s.Muted.Render("  press "))
		sb.WriteString(s.Key.Render(params.StateHintKey))
		sb.WriteString(s.Muted.Render(" for " + params.StateHintLabel))
		sb.WriteString("\n")
	}
}

// RenderViewportOverlayFooter renders a footer for viewport overlay views (overview, spec, etc.).
// The toggleKey is the key that toggles the overlay (e.g., "o" for overview, "s" for spec).
func RenderViewportOverlayFooter(sb *strings.Builder, toggleKey string, s *styles.Styles) {
	sb.WriteString(s.Muted.Render("  " + strings.Repeat("─", 60)))
	sb.WriteString("\n")
	RenderViewportScrollHints(sb, toggleKey, s)
}

// RenderViewportScrollHints renders just the scroll/return/quit key hints for viewport overlays.
// Use this when you don't want the separator line (e.g., for inspect model).
func RenderViewportScrollHints(sb *strings.Builder, toggleKey string, s *styles.Styles) {
	keyStyle := lipgloss.NewStyle().Foreground(s.Palette.Primary()).Bold(true)
	sb.WriteString(s.Muted.Render("  Press "))
	sb.WriteString(keyStyle.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" to scroll  "))
	sb.WriteString(keyStyle.Render("esc"))
	sb.WriteString(s.Muted.Render("/"))
	sb.WriteString(keyStyle.Render(toggleKey))
	sb.WriteString(s.Muted.Render(" to return  "))
	sb.WriteString(keyStyle.Render("q"))
	sb.WriteString(s.Muted.Render(" to quit"))
	sb.WriteString("\n")
}

// RenderElementFailures renders a list of element failures with text wrapping.
// Set showElementType to true to display the element type for resources (used by destroy).
func RenderElementFailures(sb *strings.Builder, failures []ElementFailure, contentWidth int, showElementType bool, s *styles.Styles) {
	if len(failures) == 0 {
		return
	}

	failureLabel := sdkstrings.Pluralize(len(failures), "Failure", "Failures")
	sb.WriteString(s.Error.Render(fmt.Sprintf("  %d %s:", len(failures), failureLabel)))
	sb.WriteString("\n\n")

	reasonWidth := contentWidth - 8

	for _, failure := range failures {
		sb.WriteString(s.Error.Render("  ✗ "))
		sb.WriteString(s.Selected.Render(failure.ElementPath))
		if showElementType && failure.ElementType != "" && failure.ElementType != "child" && failure.ElementType != "link" {
			sb.WriteString(s.Muted.Render(" (" + failure.ElementType + ")"))
		}
		sb.WriteString("\n")
		RenderWrappedFailureReasons(sb, failure.FailureReasons, reasonWidth, s)
		sb.WriteString("\n")
	}
}

// RenderWrappedFailureReasons renders failure reasons with bullet points and text wrapping.
func RenderWrappedFailureReasons(sb *strings.Builder, reasons []string, width int, s *styles.Styles) {
	for _, reason := range reasons {
		wrappedLines := outpututil.WrapTextLines(reason, width)
		for i, line := range wrappedLines {
			sb.WriteString("      ")
			if i == 0 {
				sb.WriteString(s.Error.Render("• "))
			} else {
				sb.WriteString("  ")
			}
			sb.WriteString(s.Error.Render(line))
			sb.WriteString("\n")
		}
	}
}

// RenderOutboundLinksSection renders the outbound links from a resource.
// It searches the provided links map for links that originate from the given resourceName.
func RenderOutboundLinksSection(resourceName string, links map[string]*state.LinkState, s *styles.Styles) string {
	if len(links) == 0 {
		return ""
	}

	prefix := resourceName + "::"
	var outboundLinks []string
	for linkName, linkState := range links {
		if strings.HasPrefix(linkName, prefix) {
			targetResource := strings.TrimPrefix(linkName, prefix)
			statusStr := FormatLinkStatus(linkState.Status)
			outboundLinks = append(outboundLinks, "→ "+targetResource+" ("+statusStr+")")
		}
	}

	if len(outboundLinks) == 0 {
		return ""
	}

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
