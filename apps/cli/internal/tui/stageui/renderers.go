package stageui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// StageDetailsRenderer implements splitpane.DetailsRenderer for stage UI.
type StageDetailsRenderer struct {
	// MaxExpandDepth is used to determine when to show the drill-down hint
	MaxExpandDepth int
	// NavigationStackDepth is the current depth of navigation stack
	NavigationStackDepth int
}

// Ensure StageDetailsRenderer implements splitpane.DetailsRenderer
var _ splitpane.DetailsRenderer = (*StageDetailsRenderer)(nil)

// RenderDetails renders the right pane content for a selected item.
func (r *StageDetailsRenderer) RenderDetails(item splitpane.Item, width int, s *styles.Styles) string {
	stageItem, ok := item.(*StageItem)
	if !ok {
		return s.Muted.Render("Unknown item type")
	}

	switch stageItem.Type {
	case ItemTypeResource:
		return r.renderResourceDetails(stageItem, width, s)
	case ItemTypeChild:
		return r.renderChildDetails(stageItem, width, s)
	case ItemTypeLink:
		return r.renderLinkDetails(stageItem, width, s)
	default:
		return s.Muted.Render("Unknown item type")
	}
}

func (r *StageDetailsRenderer) renderResourceDetails(item *StageItem, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	r.renderResourceHeader(&sb, item, width, s)
	r.renderResourceMetadata(&sb, item, s)
	sb.WriteString("\n")
	r.renderResourceBody(&sb, item, width, s)
	r.renderResourceOutputs(&sb, item, width, s)

	return sb.String()
}

func (r *StageDetailsRenderer) renderResourceHeader(sb *strings.Builder, item *StageItem, width int, s *styles.Styles) {
	headerText := item.Name
	if item.DisplayName != "" {
		headerText = item.DisplayName
	}
	sb.WriteString(s.Header.Render(headerText))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")
}

func (r *StageDetailsRenderer) renderResourceMetadata(sb *strings.Builder, item *StageItem, s *styles.Styles) {
	if item.DisplayName != "" {
		sb.WriteString(s.Muted.Render("Name: "))
		sb.WriteString(item.Name)
		sb.WriteString("\n")
	}

	if item.ResourceType != "" {
		sb.WriteString(s.Muted.Render("Type: "))
		sb.WriteString(item.ResourceType)
		sb.WriteString("\n")
	}

	sb.WriteString(s.Muted.Render("Action: "))
	sb.WriteString(renderActionBadge(item.Action, s))
	sb.WriteString("\n")

	resourceID := r.getResourceID(item)
	if resourceID != "" {
		sb.WriteString(s.Muted.Render("Resource ID: "))
		sb.WriteString(resourceID)
		sb.WriteString("\n")
	}
}

func (r *StageDetailsRenderer) getResourceID(item *StageItem) string {
	if resourceChanges, ok := item.Changes.(*provider.Changes); ok && resourceChanges != nil {
		if resourceChanges.AppliedResourceInfo.ResourceID != "" {
			return resourceChanges.AppliedResourceInfo.ResourceID
		}
	}
	if item.ResourceState != nil {
		return item.ResourceState.ResourceID
	}
	return ""
}

func (r *StageDetailsRenderer) renderResourceBody(sb *strings.Builder, item *StageItem, width int, s *styles.Styles) {
	if item.Removed {
		sb.WriteString(s.Error.Render("This resource will be destroyed"))
		sb.WriteString("\n")
		return
	}

	resourceChanges, ok := item.Changes.(*provider.Changes)
	if !ok || resourceChanges == nil {
		sb.WriteString(s.Muted.Render("No changes"))
		sb.WriteString("\n")
		return
	}

	sb.WriteString(r.renderResourceChanges(resourceChanges, width, s))
}

func (r *StageDetailsRenderer) renderResourceOutputs(sb *strings.Builder, item *StageItem, width int, s *styles.Styles) {
	if item.ResourceState == nil || item.ResourceState.SpecData == nil {
		return
	}
	if len(item.ResourceState.ComputedFields) == 0 {
		return
	}

	outputsSection := outpututil.RenderOutputsFromState(item.ResourceState, width, s)
	if outputsSection != "" {
		sb.WriteString("\n")
		sb.WriteString(outputsSection)
	}
}

func (r *StageDetailsRenderer) renderResourceChanges(resourceChanges *provider.Changes, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	hasFieldChanges := provider.ChangesHasFieldChanges(resourceChanges)
	hasOutboundLinkChanges := len(resourceChanges.NewOutboundLinks) > 0 ||
		len(resourceChanges.OutboundLinkChanges) > 0 ||
		len(resourceChanges.RemovedOutboundLinks) > 0

	if !hasFieldChanges && !hasOutboundLinkChanges {
		sb.WriteString(s.Muted.Render("No changes"))
		sb.WriteString("\n")
		return sb.String()
	}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	// Render field changes section
	sb.WriteString(s.Category.Render("Field Changes:"))
	sb.WriteString("\n")

	// Calculate available width for field values (accounting for indent and prefix)
	// Format: "  + fieldPath: value" or "  ± fieldPath: prev -> new"
	fieldIndent := 4 // "  + " or "  ± " or "  - "

	if hasFieldChanges {
		// New fields (additions)
		for _, field := range resourceChanges.NewFields {
			r.renderFieldChange(&sb, "+", field.FieldPath, "", headless.FormatMappingNode(field.NewValue), width, fieldIndent, successStyle)
		}

		// Modified fields
		for _, field := range resourceChanges.ModifiedFields {
			prevValue := headless.FormatMappingNode(field.PrevValue)
			newValue := headless.FormatMappingNode(field.NewValue)
			r.renderFieldChange(&sb, "±", field.FieldPath, prevValue, newValue, width, fieldIndent, s.Warning)
		}

		// Removed fields
		for _, fieldPath := range resourceChanges.RemovedFields {
			line := fmt.Sprintf("  - %s", fieldPath)
			sb.WriteString(s.Error.Render(line))
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(s.Muted.Render("  None"))
		sb.WriteString("\n")
	}

	// Render outbound link changes if present
	if hasOutboundLinkChanges {
		sb.WriteString("\n")
		sb.WriteString(r.renderOutboundLinkChanges(resourceChanges, s))
	}

	return sb.String()
}

// fieldChangeLayout holds pre-calculated layout values for field change rendering.
type fieldChangeLayout struct {
	baseIndent   string
	valueIndent  string
	contentWidth int
	valueWidth   int
}

func newFieldChangeLayout(prefix string, width, indent int) fieldChangeLayout {
	baseIndent := "  " + prefix + " "
	baseIndentLen := len(baseIndent)

	contentWidth := width - indent - baseIndentLen
	if contentWidth < 20 {
		contentWidth = 20
	}

	valueIndent := strings.Repeat(" ", baseIndentLen+2)
	valueWidth := width - indent - len(valueIndent)
	if valueWidth < 20 {
		valueWidth = 20
	}

	return fieldChangeLayout{
		baseIndent:   baseIndent,
		valueIndent:  valueIndent,
		contentWidth: contentWidth,
		valueWidth:   valueWidth,
	}
}

func (r *StageDetailsRenderer) renderFieldChange(
	sb *strings.Builder,
	prefix string,
	fieldPath string,
	prevValue string,
	newValue string,
	width int,
	indent int,
	style lipgloss.Style,
) {
	layout := newFieldChangeLayout(prefix, width, indent)
	fieldPathWithColon := fieldPath + ":"
	minValueSpace := 10

	fieldPathFits := len(fieldPathWithColon) <= layout.contentWidth-minValueSpace

	if prevValue == "" {
		r.renderNewFieldChange(sb, fieldPathWithColon, newValue, layout, fieldPathFits, style)
	} else {
		r.renderModifiedFieldChange(sb, fieldPathWithColon, prevValue, newValue, layout, fieldPathFits, style)
	}
}

func (r *StageDetailsRenderer) renderNewFieldChange(
	sb *strings.Builder,
	fieldPathWithColon string,
	newValue string,
	layout fieldChangeLayout,
	fieldPathFits bool,
	style lipgloss.Style,
) {
	if !fieldPathFits {
		sb.WriteString(style.Render(layout.baseIndent + fieldPathWithColon))
		sb.WriteString("\n")
		r.renderWrappedValue(sb, newValue, layout.valueIndent, layout.valueWidth, style)
		return
	}

	fullLine := fieldPathWithColon + " " + newValue
	if len(fullLine) <= layout.contentWidth {
		sb.WriteString(style.Render(layout.baseIndent + fullLine))
		sb.WriteString("\n")
		return
	}

	sb.WriteString(style.Render(layout.baseIndent + fieldPathWithColon))
	sb.WriteString("\n")
	r.renderWrappedValue(sb, newValue, layout.valueIndent, layout.valueWidth, style)
}

func (r *StageDetailsRenderer) renderModifiedFieldChange(
	sb *strings.Builder,
	fieldPathWithColon string,
	prevValue string,
	newValue string,
	layout fieldChangeLayout,
	fieldPathFits bool,
	style lipgloss.Style,
) {
	if !fieldPathFits {
		sb.WriteString(style.Render(layout.baseIndent + fieldPathWithColon))
		sb.WriteString("\n")
		r.renderPrevToNewValue(sb, prevValue, newValue, layout, style)
		return
	}

	fullLine := fieldPathWithColon + " " + prevValue + " -> " + newValue
	if len(fullLine) <= layout.contentWidth {
		sb.WriteString(style.Render(layout.baseIndent + fullLine))
		sb.WriteString("\n")
		return
	}

	sb.WriteString(style.Render(layout.baseIndent + fieldPathWithColon))
	sb.WriteString("\n")
	r.renderPrevToNewValue(sb, prevValue, newValue, layout, style)
}

func (r *StageDetailsRenderer) renderPrevToNewValue(
	sb *strings.Builder,
	prevValue string,
	newValue string,
	layout fieldChangeLayout,
	style lipgloss.Style,
) {
	r.renderWrappedValue(sb, prevValue, layout.valueIndent, layout.valueWidth, style)
	sb.WriteString(style.Render(layout.valueIndent + "->"))
	sb.WriteString("\n")
	r.renderWrappedValue(sb, newValue, layout.valueIndent, layout.valueWidth, style)
}

// renderWrappedValue renders a value with text wrapping, breaking at reasonable points.
func (r *StageDetailsRenderer) renderWrappedValue(
	sb *strings.Builder,
	value string,
	indent string,
	width int,
	style lipgloss.Style,
) {
	if len(value) <= width {
		sb.WriteString(style.Render(indent + value))
		sb.WriteString("\n")
		return
	}

	// Try to wrap at spaces first
	wrapped := outpututil.WrapText(value, width)
	lines := strings.Split(wrapped, "\n")

	// If WrapText didn't help (no spaces to break on), force break at width
	if len(lines) == 1 && len(value) > width {
		lines = r.forceWrap(value, width)
	}

	for _, line := range lines {
		sb.WriteString(style.Render(indent + line))
		sb.WriteString("\n")
	}
}

// forceWrap breaks text at the specified width regardless of word boundaries.
func (r *StageDetailsRenderer) forceWrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	for len(text) > width {
		lines = append(lines, text[:width])
		text = text[width:]
	}
	if len(text) > 0 {
		lines = append(lines, text)
	}
	return lines
}

func (r *StageDetailsRenderer) renderOutboundLinkChanges(resourceChanges *provider.Changes, s *styles.Styles) string {
	sb := strings.Builder{}

	sb.WriteString(s.Category.Render("Outbound Link Changes:"))
	sb.WriteString("\n")

	r.renderNewOutboundLinks(&sb, resourceChanges.NewOutboundLinks, s)
	r.renderModifiedOutboundLinks(&sb, resourceChanges.OutboundLinkChanges, s)
	r.renderRemovedOutboundLinks(&sb, resourceChanges.RemovedOutboundLinks, s)

	return sb.String()
}

func (r *StageDetailsRenderer) renderNewOutboundLinks(
	sb *strings.Builder,
	links map[string]provider.LinkChanges,
	s *styles.Styles,
) {
	if len(links) == 0 {
		return
	}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())
	linkNames := sortedMapKeys(links)

	for _, linkName := range linkNames {
		linkChanges := links[linkName]
		line := fmt.Sprintf("  + %s (new link)", linkName)
		sb.WriteString(successStyle.Render(line))
		sb.WriteString("\n")
		sb.WriteString(r.renderLinkFieldChanges(&linkChanges, s, "      "))
	}
}

func (r *StageDetailsRenderer) renderModifiedOutboundLinks(
	sb *strings.Builder,
	links map[string]provider.LinkChanges,
	s *styles.Styles,
) {
	if len(links) == 0 {
		return
	}

	linkNames := sortedMapKeys(links)

	for _, linkName := range linkNames {
		linkChanges := links[linkName]
		line := fmt.Sprintf("  ± %s (link updated)", linkName)
		sb.WriteString(s.Warning.Render(line))
		sb.WriteString("\n")
		sb.WriteString(r.renderLinkFieldChanges(&linkChanges, s, "      "))
	}
}

func (r *StageDetailsRenderer) renderRemovedOutboundLinks(
	sb *strings.Builder,
	links []string,
	s *styles.Styles,
) {
	if len(links) == 0 {
		return
	}

	removedLinks := make([]string, len(links))
	copy(removedLinks, links)
	sort.Strings(removedLinks)

	for _, linkName := range removedLinks {
		line := fmt.Sprintf("  - %s (link removed)", linkName)
		sb.WriteString(s.Error.Render(line))
		sb.WriteString("\n")
	}
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (r *StageDetailsRenderer) renderLinkFieldChanges(linkChanges *provider.LinkChanges, s *styles.Styles, indent string) string {
	if linkChanges == nil {
		return ""
	}

	hasChanges := len(linkChanges.NewFields) > 0 ||
		len(linkChanges.ModifiedFields) > 0 ||
		len(linkChanges.RemovedFields) > 0

	if !hasChanges {
		return ""
	}

	sb := strings.Builder{}
	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	// New fields
	for _, field := range linkChanges.NewFields {
		line := fmt.Sprintf("%s+ %s: %s", indent, field.FieldPath, headless.FormatMappingNode(field.NewValue))
		sb.WriteString(successStyle.Render(line))
		sb.WriteString("\n")
	}

	// Modified fields
	for _, field := range linkChanges.ModifiedFields {
		prevValue := headless.FormatMappingNode(field.PrevValue)
		newValue := headless.FormatMappingNode(field.NewValue)
		line := fmt.Sprintf("%s± %s: %s -> %s", indent, field.FieldPath, prevValue, newValue)
		sb.WriteString(s.Warning.Render(line))
		sb.WriteString("\n")
	}

	// Removed fields
	for _, fieldPath := range linkChanges.RemovedFields {
		line := fmt.Sprintf("%s- %s", indent, fieldPath)
		sb.WriteString(s.Error.Render(line))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *StageDetailsRenderer) renderChildDetails(item *StageItem, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(item.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Status and action
	sb.WriteString(s.Muted.Render("Status: "))
	sb.WriteString("Changes computed")
	sb.WriteString("\n")

	sb.WriteString(s.Muted.Render("Action: "))
	sb.WriteString(renderActionBadge(item.Action, s))
	sb.WriteString("\n")

	// Instance ID (if available from instance state)
	if item.InstanceState != nil && item.InstanceState.InstanceID != "" {
		sb.WriteString(s.Muted.Render("Instance ID: "))
		sb.WriteString(item.InstanceState.InstanceID)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// For removed children, show destruction message
	if item.Removed {
		sb.WriteString(s.Error.Render("This child blueprint will be destroyed"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Show inspect hint for children at max expand depth
	effectiveDepth := item.Depth + r.NavigationStackDepth
	if effectiveDepth >= r.MaxExpandDepth && item.Changes != nil {
		sb.WriteString(s.Hint.Render("Press enter to inspect this child blueprint"))
		sb.WriteString("\n\n")
	}

	// Child changes summary
	if childChanges, ok := item.Changes.(*changes.BlueprintChanges); ok && childChanges != nil {
		sb.WriteString(r.renderChildChangesSummary(childChanges, s))
	}

	return sb.String()
}

func (r *StageDetailsRenderer) renderChildChangesSummary(childChanges *changes.BlueprintChanges, s *styles.Styles) string {
	sb := strings.Builder{}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	newCount := len(childChanges.NewResources)
	updateCount := len(childChanges.ResourceChanges)
	removeCount := len(childChanges.RemovedResources)

	// Child blueprints
	newChildren := len(childChanges.NewChildren)
	childUpdates := len(childChanges.ChildChanges)
	removedChildren := len(childChanges.RemovedChildren)

	// Only render summary if there's something to show
	hasResourceChanges := newCount > 0 || updateCount > 0 || removeCount > 0
	hasChildChanges := newChildren > 0 || childUpdates > 0 || removedChildren > 0

	if !hasResourceChanges && !hasChildChanges {
		return ""
	}

	sb.WriteString(s.Category.Render("Summary:"))
	sb.WriteString("\n")

	if newCount > 0 {
		sb.WriteString(successStyle.Render(fmt.Sprintf("  %d %s to be created", newCount, sdkstrings.Pluralize(newCount, "resource", "resources"))))
		sb.WriteString("\n")
	}
	if updateCount > 0 {
		sb.WriteString(s.Warning.Render(fmt.Sprintf("  %d %s to be updated", updateCount, sdkstrings.Pluralize(updateCount, "resource", "resources"))))
		sb.WriteString("\n")
	}
	if removeCount > 0 {
		sb.WriteString(s.Error.Render(fmt.Sprintf("  %d %s to be removed", removeCount, sdkstrings.Pluralize(removeCount, "resource", "resources"))))
		sb.WriteString("\n")
	}

	if hasChildChanges {
		if hasResourceChanges {
			sb.WriteString("\n")
		}
		if newChildren > 0 {
			sb.WriteString(successStyle.Render(fmt.Sprintf("  %d child %s to be created", newChildren, sdkstrings.Pluralize(newChildren, "blueprint", "blueprints"))))
			sb.WriteString("\n")
		}
		if childUpdates > 0 {
			sb.WriteString(s.Warning.Render(fmt.Sprintf("  %d child %s to be updated", childUpdates, sdkstrings.Pluralize(childUpdates, "blueprint", "blueprints"))))
			sb.WriteString("\n")
		}
		if removedChildren > 0 {
			sb.WriteString(s.Error.Render(fmt.Sprintf("  %d child %s to be removed", removedChildren, sdkstrings.Pluralize(removedChildren, "blueprint", "blueprints"))))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (r *StageDetailsRenderer) renderLinkDetails(item *StageItem, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(item.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Status and action
	sb.WriteString(s.Muted.Render("Status: "))
	sb.WriteString("Changes computed")
	sb.WriteString("\n")

	sb.WriteString(s.Muted.Render("Action: "))
	sb.WriteString(renderActionBadge(item.Action, s))
	sb.WriteString("\n")

	// Link ID (if available from link state)
	if item.LinkState != nil && item.LinkState.LinkID != "" {
		sb.WriteString(s.Muted.Render("Link ID: "))
		sb.WriteString(item.LinkState.LinkID)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Link changes or deletion message
	if item.Removed {
		sb.WriteString(s.Error.Render("This link will be destroyed"))
		sb.WriteString("\n")
	} else if linkChanges, ok := item.Changes.(*provider.LinkChanges); ok && linkChanges != nil {
		sb.WriteString(r.renderLinkChanges(linkChanges, s))
	}

	return sb.String()
}

func (r *StageDetailsRenderer) renderLinkChanges(linkChanges *provider.LinkChanges, s *styles.Styles) string {
	sb := strings.Builder{}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	hasChanges := len(linkChanges.NewFields) > 0 || len(linkChanges.ModifiedFields) > 0 || len(linkChanges.RemovedFields) > 0

	if !hasChanges {
		sb.WriteString(s.Muted.Render("No field changes"))
		return sb.String()
	}

	sb.WriteString(s.Category.Render("Changes:"))
	sb.WriteString("\n")

	// New fields (additions)
	for _, field := range linkChanges.NewFields {
		line := fmt.Sprintf("  + %s: %s", field.FieldPath, headless.FormatMappingNode(field.NewValue))
		sb.WriteString(successStyle.Render(line))
		sb.WriteString("\n")
	}

	// Modified fields
	for _, field := range linkChanges.ModifiedFields {
		prevValue := headless.FormatMappingNode(field.PrevValue)
		newValue := headless.FormatMappingNode(field.NewValue)
		line := fmt.Sprintf("  ± %s: %s -> %s", field.FieldPath, prevValue, newValue)
		sb.WriteString(s.Warning.Render(line))
		sb.WriteString("\n")
	}

	// Removed fields
	for _, fieldPath := range linkChanges.RemovedFields {
		line := fmt.Sprintf("  - %s", fieldPath)
		sb.WriteString(s.Error.Render(line))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderActionBadge renders an action badge with appropriate styling.
func renderActionBadge(action ActionType, s *styles.Styles) string {
	return shared.RenderActionBadge(action, s)
}

// StageSectionGrouper implements splitpane.SectionGrouper for stage UI.
type StageSectionGrouper struct {
	shared.SectionGrouper
}

// Ensure StageSectionGrouper implements splitpane.SectionGrouper
var _ splitpane.SectionGrouper = (*StageSectionGrouper)(nil)

// StageFooterRenderer implements splitpane.FooterRenderer for stage UI.
// It supports a delegate pattern to allow custom footer rendering (e.g., for deploy flow).
type StageFooterRenderer struct {
	ChangesetID  string
	InstanceID   string
	InstanceName string
	// Destroy indicates whether this is a destroy operation
	Destroy bool
	// Change summary counts for footer display
	CreateCount   int
	UpdateCount   int
	RecreateCount int
	DeleteCount   int
	// HasExportChanges indicates whether there are export changes to show
	HasExportChanges bool
	// Delegate is an optional custom footer renderer that takes precedence when set.
	// This allows the deploy flow to inject its own footer (e.g., confirmation form).
	Delegate splitpane.FooterRenderer
}

// Ensure StageFooterRenderer implements splitpane.FooterRenderer
var _ splitpane.FooterRenderer = (*StageFooterRenderer)(nil)

// RenderFooter renders the stage-specific footer with changeset and deploy instructions.
// If a Delegate is set, it defers to the delegate for rendering.
func (r *StageFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	// If a delegate is set, use it for rendering
	if r.Delegate != nil {
		return r.Delegate.RenderFooter(model, s)
	}
	sb := strings.Builder{}
	sb.WriteString("\n")

	// Show different footer when viewing a child blueprint
	if model.IsInDrillDown() {
		shared.RenderBreadcrumb(&sb, model.NavigationPath(), s)
		shared.RenderFooterNavigation(&sb, s,
			shared.KeyHint{Key: "esc", Desc: "back"},
			shared.KeyHint{Key: "enter", Desc: "expand/inspect"},
		)
		return sb.String()
	}

	sb.WriteString(s.Muted.Render("  Staging complete. Changeset ID: "))
	sb.WriteString(s.Selected.Render(r.ChangesetID))
	sb.WriteString(s.Muted.Render(" - press "))
	sb.WriteString(s.Key.Render("o"))
	sb.WriteString(s.Muted.Render(" for overview"))
	sb.WriteString("\n")

	// Show summary counts
	sb.WriteString(r.renderChangeSummary(s))
	sb.WriteString("\n")

	// Check if there are any changes
	hasChanges := r.CreateCount > 0 || r.UpdateCount > 0 || r.DeleteCount > 0 || r.RecreateCount > 0

	if hasChanges {
		// Deploy/destroy instructions
		if r.Destroy {
			sb.WriteString(s.Muted.Render("  To destroy, run "))
			sb.WriteString(s.Command.Render("bluelink destroy"))
		} else {
			sb.WriteString(s.Muted.Render("  To deploy, run "))
			sb.WriteString(s.Command.Render("bluelink deploy"))
		}
		sb.WriteString(s.Muted.Render(" with changeset "))
		sb.WriteString(s.Selected.Render(r.ChangesetID))
		sb.WriteString("\n\n")
	} else {
		// No changes message
		sb.WriteString(s.Muted.Render("  No changes to apply. Press "))
		sb.WriteString(s.Key.Render("q"))
		sb.WriteString(s.Muted.Render(" to quit."))
		sb.WriteString("\n\n")
	}

	// Navigation help
	extraKeys := []shared.KeyHint{{Key: "enter", Desc: "expand/collapse"}}
	if r.HasExportChanges {
		extraKeys = append(extraKeys, shared.KeyHint{Key: "e", Desc: "exports"})
	}
	shared.RenderFooterNavigation(&sb, s, extraKeys...)

	return sb.String()
}

// renderChangeSummary renders the change summary counts (e.g., "3 creates, 2 updates, 1 delete").
func (r *StageFooterRenderer) renderChangeSummary(s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("  ")

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())
	needsComma := false

	if r.CreateCount > 0 {
		sb.WriteString(successStyle.Render(fmt.Sprintf("%d %s",
			r.CreateCount, sdkstrings.Pluralize(r.CreateCount, "create", "creates"))))
		needsComma = true
	}
	if r.UpdateCount > 0 {
		if needsComma {
			sb.WriteString(s.Muted.Render(", "))
		}
		sb.WriteString(s.Warning.Render(fmt.Sprintf("%d %s",
			r.UpdateCount, sdkstrings.Pluralize(r.UpdateCount, "update", "updates"))))
		needsComma = true
	}
	if r.RecreateCount > 0 {
		if needsComma {
			sb.WriteString(s.Muted.Render(", "))
		}
		sb.WriteString(s.Info.Render(fmt.Sprintf("%d %s",
			r.RecreateCount, sdkstrings.Pluralize(r.RecreateCount, "recreate", "recreates"))))
		needsComma = true
	}
	if r.DeleteCount > 0 {
		if needsComma {
			sb.WriteString(s.Muted.Render(", "))
		}
		sb.WriteString(s.Error.Render(fmt.Sprintf("%d %s",
			r.DeleteCount, sdkstrings.Pluralize(r.DeleteCount, "delete", "deletes"))))
	}

	return sb.String()
}
