package driftui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// DriftItemType represents the type of drift item.
type DriftItemType string

const (
	DriftItemTypeResource DriftItemType = "resource"
	DriftItemTypeLink     DriftItemType = "link"
	DriftItemTypeChild    DriftItemType = "child"
)

// DriftItem represents a resource or link with drift for the split pane.
type DriftItem struct {
	Type         DriftItemType
	Name         string
	ResourceType string // For resources: the resource type (e.g., "aws/s3/bucket")
	ChildPath    string // Path to child blueprint (e.g., "childA" or "childA.childB")
	DriftType    container.ReconciliationType
	// Changes for resources
	ResourceResult *container.ResourceReconcileResult
	// ResourceState from instance state (for computed fields/outputs)
	ResourceState *state.ResourceState
	// Changes for links
	LinkResult  *container.LinkReconcileResult
	Recommended container.ReconciliationAction
	// For hierarchical display
	Depth       int
	ParentChild string
	// Children for child blueprint summary items
	Children []*DriftItem
}

// Ensure DriftItem implements splitpane.Item
var _ splitpane.Item = (*DriftItem)(nil)

// GetID returns a unique identifier for the item.
func (d *DriftItem) GetID() string {
	if d.ChildPath != "" {
		return fmt.Sprintf("%s:%s", d.ChildPath, d.Name)
	}
	return d.Name
}

// GetName returns the display name for the item.
func (d *DriftItem) GetName() string {
	return d.Name
}

// GetIcon returns an icon for the item.
func (d *DriftItem) GetIcon(selected bool) string {
	return d.getIconChar()
}

// getIconChar returns the icon character.
func (d *DriftItem) getIconChar() string {
	switch d.DriftType {
	case container.ReconciliationTypeDrift:
		return "⚠"
	case container.ReconciliationTypeInterrupted:
		return "!"
	default:
		return "○"
	}
}

// GetIconStyled returns a styled icon for the item.
func (d *DriftItem) GetIconStyled(s *styles.Styles, styled bool) string {
	icon := d.getIconChar()
	if !styled {
		return icon
	}

	switch d.DriftType {
	case container.ReconciliationTypeDrift:
		return s.Warning.Render(icon)
	case container.ReconciliationTypeInterrupted:
		return s.Error.Render(icon)
	default:
		return s.Muted.Render(icon)
	}
}

// GetAction returns the action badge text.
func (d *DriftItem) GetAction() string {
	return HumanReadableDriftType(d.DriftType)
}

// GetDepth returns the nesting depth for indentation.
func (d *DriftItem) GetDepth() int {
	return d.Depth
}

// GetParentID returns the parent item ID.
func (d *DriftItem) GetParentID() string {
	return d.ParentChild
}

// GetItemType returns the type for section grouping.
func (d *DriftItem) GetItemType() string {
	return string(d.Type)
}

// IsExpandable returns true if the item can be expanded in-place.
func (d *DriftItem) IsExpandable() bool {
	return d.Type == DriftItemTypeChild && len(d.Children) > 0
}

// CanDrillDown returns true if the item can be drilled into.
func (d *DriftItem) CanDrillDown() bool {
	return d.Type == DriftItemTypeChild && len(d.Children) > 0
}

// GetChildren returns child items when expanded.
func (d *DriftItem) GetChildren() []splitpane.Item {
	if d.Type != DriftItemTypeChild || len(d.Children) == 0 {
		return nil
	}

	items := make([]splitpane.Item, len(d.Children))
	for i, child := range d.Children {
		items[i] = child
	}
	return items
}

// BuildDriftItems creates DriftItems from a ReconciliationCheckResult.
// instanceState is optional - when provided, it enables resource state lookup
// for displaying computed fields/outputs.
func BuildDriftItems(
	result *container.ReconciliationCheckResult,
	instanceState *state.InstanceState,
) []splitpane.Item {
	var items []splitpane.Item

	// Build tree for child blueprints
	childTree := buildDriftChildTree(result, instanceState)

	// Add parent-level resources (ChildPath == "")
	for i := range result.Resources {
		r := &result.Resources[i]
		if r.ChildPath == "" {
			resourceState := findResourceState(instanceState, r.ResourceName)
			items = append(items, &DriftItem{
				Type:           DriftItemTypeResource,
				Name:           r.ResourceName,
				ResourceType:   r.ResourceType,
				DriftType:      r.Type,
				ResourceResult: r,
				ResourceState:  resourceState,
				Recommended:    r.RecommendedAction,
				Depth:          0,
			})
		}
	}

	// Add parent-level links (ChildPath == "")
	for i := range result.Links {
		l := &result.Links[i]
		if l.ChildPath == "" {
			items = append(items, &DriftItem{
				Type:        DriftItemTypeLink,
				Name:        l.LinkName,
				DriftType:   l.Type,
				LinkResult:  l,
				Recommended: l.RecommendedAction,
				Depth:       0,
			})
		}
	}

	// Add child blueprint summary items
	for _, child := range childTree {
		items = append(items, child)
	}

	return items
}

// findResourceState finds a resource state by name using the instance state's
// ResourceIDs map to look up the resource ID, then retrieves the state from Resources.
func findResourceState(instanceState *state.InstanceState, name string) *state.ResourceState {
	if instanceState == nil || instanceState.ResourceIDs == nil || instanceState.Resources == nil {
		return nil
	}
	resourceID, ok := instanceState.ResourceIDs[name]
	if !ok {
		return nil
	}
	return instanceState.Resources[resourceID]
}

// findChildInstanceState finds a child instance state by name from the parent.
func findChildInstanceState(instanceState *state.InstanceState, childName string) *state.InstanceState {
	if instanceState == nil || instanceState.ChildBlueprints == nil {
		return nil
	}
	return instanceState.ChildBlueprints[childName]
}

// driftChildNode is a temporary structure for building the child tree.
type driftChildNode struct {
	name          string
	fullPath      string
	resources     []*container.ResourceReconcileResult
	links         []*container.LinkReconcileResult
	children      map[string]*driftChildNode
	instanceState *state.InstanceState
}

// buildDriftChildTree builds a tree of child blueprints from resources and links.
func buildDriftChildTree(
	result *container.ReconciliationCheckResult,
	instanceState *state.InstanceState,
) []*DriftItem {
	root := &driftChildNode{
		children:      make(map[string]*driftChildNode),
		instanceState: instanceState,
	}

	// Insert all resources with non-empty ChildPath into the tree
	for i := range result.Resources {
		r := &result.Resources[i]
		if r.ChildPath != "" {
			insertDriftResourceIntoTree(root, r.ChildPath, r, instanceState)
		}
	}

	// Insert all links with non-empty ChildPath into the tree
	for i := range result.Links {
		l := &result.Links[i]
		if l.ChildPath != "" {
			insertDriftLinkIntoTree(root, l.ChildPath, l, instanceState)
		}
	}

	// Convert tree to DriftItems
	return flattenDriftTree(root, 0)
}

func insertDriftResourceIntoTree(
	root *driftChildNode,
	childPath string,
	r *container.ResourceReconcileResult,
	instanceState *state.InstanceState,
) {
	node := getOrCreateDriftNode(root, childPath, instanceState)
	node.resources = append(node.resources, r)
}

func insertDriftLinkIntoTree(
	root *driftChildNode,
	childPath string,
	l *container.LinkReconcileResult,
	instanceState *state.InstanceState,
) {
	node := getOrCreateDriftNode(root, childPath, instanceState)
	node.links = append(node.links, l)
}

// getOrCreateDriftNode traverses/creates the tree path and returns the leaf node.
// It also propagates instance state through the tree hierarchy.
func getOrCreateDriftNode(
	root *driftChildNode,
	childPath string,
	instanceState *state.InstanceState,
) *driftChildNode {
	segments := splitDriftChildPath(childPath)
	current := root
	currentInstanceState := instanceState
	for i, segment := range segments {
		if current.children == nil {
			current.children = make(map[string]*driftChildNode)
		}
		child, exists := current.children[segment]
		if !exists {
			// Get the child's instance state from the parent
			childInstanceState := findChildInstanceState(currentInstanceState, segment)
			child = &driftChildNode{
				name:          segment,
				fullPath:      joinDriftChildPath(segments[:i+1]),
				children:      make(map[string]*driftChildNode),
				instanceState: childInstanceState,
			}
			current.children[segment] = child
		}
		current = child
		currentInstanceState = current.instanceState
	}
	return current
}

func splitDriftChildPath(childPath string) []string {
	if childPath == "" {
		return nil
	}
	return strings.Split(childPath, ".")
}

func joinDriftChildPath(segments []string) string {
	return strings.Join(segments, ".")
}

func flattenDriftTree(node *driftChildNode, depth int) []*DriftItem {
	var items []*DriftItem

	childNames := sortedChildNames(node.children)
	for _, name := range childNames {
		child := node.children[name]
		childItem := buildChildDriftItem(name, child.fullPath, depth)
		childItem.Children = buildChildItems(child, name, depth)
		items = append(items, childItem)
	}

	return items
}

func sortedChildNames(children map[string]*driftChildNode) []string {
	names := make([]string, 0, len(children))
	for name := range children {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func buildChildDriftItem(name, fullPath string, depth int) *DriftItem {
	return &DriftItem{
		Type:      DriftItemTypeChild,
		Name:      name,
		ChildPath: fullPath,
		DriftType: container.ReconciliationTypeDrift,
		Depth:     depth,
	}
}

func buildChildItems(child *driftChildNode, parentName string, depth int) []*DriftItem {
	var items []*DriftItem
	addResourceDriftItems(child.resources, child.instanceState, &items, depth+1, parentName)
	addLinkDriftItems(child.links, &items, depth+1, parentName)
	addNestedChildren(child, parentName, depth, &items)
	return items
}

func addResourceDriftItems(
	resources []*container.ResourceReconcileResult,
	instanceState *state.InstanceState,
	items *[]*DriftItem,
	depth int,
	parentChild string,
) {
	for _, r := range resources {
		resourceState := findResourceState(instanceState, r.ResourceName)
		*items = append(*items, &DriftItem{
			Type:           DriftItemTypeResource,
			Name:           r.ResourceName,
			ResourceType:   r.ResourceType,
			ChildPath:      r.ChildPath,
			DriftType:      r.Type,
			ResourceResult: r,
			ResourceState:  resourceState,
			Recommended:    r.RecommendedAction,
			Depth:          depth,
			ParentChild:    parentChild,
		})
	}
}

func addLinkDriftItems(
	links []*container.LinkReconcileResult,
	items *[]*DriftItem,
	depth int,
	parentChild string,
) {
	for _, l := range links {
		*items = append(*items, &DriftItem{
			Type:        DriftItemTypeLink,
			Name:        l.LinkName,
			ChildPath:   l.ChildPath,
			DriftType:   l.Type,
			LinkResult:  l,
			Recommended: l.RecommendedAction,
			Depth:       depth,
			ParentChild: parentChild,
		})
	}
}

func addNestedChildren(child *driftChildNode, parentName string, depth int, items *[]*DriftItem) {
	nestedChildren := flattenDriftTree(child, depth+1)
	for _, nested := range nestedChildren {
		nested.ParentChild = parentName
		*items = append(*items, nested)
	}
}

// DriftDetailsRenderer implements splitpane.DetailsRenderer for drift review UI.
type DriftDetailsRenderer struct {
	MaxExpandDepth       int
	NavigationStackDepth int
}

// Ensure DriftDetailsRenderer implements splitpane.DetailsRenderer
var _ splitpane.DetailsRenderer = (*DriftDetailsRenderer)(nil)

// RenderDetails renders the right pane content for a selected drift item.
func (r *DriftDetailsRenderer) RenderDetails(item splitpane.Item, width int, s *styles.Styles) string {
	driftItem, ok := item.(*DriftItem)
	if !ok {
		return s.Muted.Render("Unknown item type")
	}

	switch driftItem.Type {
	case DriftItemTypeResource:
		return r.renderResourceDetails(driftItem, width, s)
	case DriftItemTypeLink:
		return r.renderLinkDetails(driftItem, width, s)
	case DriftItemTypeChild:
		return r.renderChildDetails(driftItem, width, s)
	default:
		return s.Muted.Render("Unknown item type")
	}
}

func (r *DriftDetailsRenderer) renderResourceDetails(item *DriftItem, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(item.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Resource type
	if item.ResourceType != "" {
		sb.WriteString(s.Muted.Render("Type: "))
		sb.WriteString(item.ResourceType)
		sb.WriteString("\n")
	}

	// Drift type
	sb.WriteString(s.Muted.Render("Drift Type: "))
	sb.WriteString(HumanReadableDriftTypeLabel(item.DriftType))
	sb.WriteString("\n")

	// Child path if present
	if item.ChildPath != "" {
		sb.WriteString(s.Muted.Render("Child Path: "))
		sb.WriteString(item.ChildPath)
		sb.WriteString("\n")
	}

	// Resource ID
	if item.ResourceResult != nil && item.ResourceResult.ResourceID != "" {
		sb.WriteString(s.Muted.Render("Resource ID: "))
		sb.WriteString(item.ResourceResult.ResourceID)
		sb.WriteString("\n")
	}

	// For interrupted resources, show whether resource exists externally
	if item.DriftType == container.ReconciliationTypeInterrupted && item.ResourceResult != nil {
		sb.WriteString(s.Muted.Render("Resource exists: "))
		if item.ResourceResult.ResourceExists {
			sb.WriteString("Yes")
		} else {
			sb.WriteString("No")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Changes
	if item.ResourceResult != nil && item.ResourceResult.Changes != nil {
		sb.WriteString(r.renderResourceChanges(item.ResourceResult, s))
	} else if item.DriftType == container.ReconciliationTypeInterrupted && item.ResourceResult != nil {
		// For interrupted resources without computed changes, show the state comparison
		sb.WriteString(r.renderInterruptedResourceState(item.ResourceResult, s))
	}

	// Current outputs - use ResourceState.ComputedFields as the single source of truth
	if item.ResourceState != nil && item.ResourceState.SpecData != nil && len(item.ResourceState.ComputedFields) > 0 {
		outputsSection := outpututil.RenderOutputsFromState(item.ResourceState, width, s)
		if outputsSection != "" {
			sb.WriteString("\n")
			sb.WriteString(outputsSection)
		}
	}

	// Recommended action
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render("Recommended: "))
	sb.WriteString(HumanReadableAction(item.Recommended))
	sb.WriteString("\n")

	// Show manual cleanup instructions when external state could not be retrieved
	if item.Recommended == container.ReconciliationActionManualCleanupRequired {
		sb.WriteString("\n")
		sb.WriteString(s.Warning.Render("⚠ External state could not be retrieved"))
		sb.WriteString("\n\n")
		sb.WriteString(s.Muted.Render("This resource was interrupted during creation and its external"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("state cannot be automatically retrieved (tag-based lookup may"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("not be supported for this resource type)."))
		sb.WriteString("\n\n")
		sb.WriteString(s.Category.Render("Recommended steps:"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("  1. Check if the resource exists in your provider console"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("  2. If it exists, delete it manually"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("  3. Destroy this instance and re-run the deployment"))
		sb.WriteString("\n\n")
		sb.WriteString(s.Muted.Render("If you destroy this instance without manual cleanup, this"))
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("resource may remain orphaned in your provider."))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *DriftDetailsRenderer) renderResourceChanges(
	result *container.ResourceReconcileResult,
	s *styles.Styles,
) string {
	sb := strings.Builder{}
	changes := result.Changes

	hasChanges := len(changes.NewFields) > 0 ||
		len(changes.ModifiedFields) > 0 ||
		len(changes.RemovedFields) > 0

	if !hasChanges {
		sb.WriteString(s.Muted.Render("No field changes detected"))
		return sb.String()
	}

	successStyle := lipgloss.NewStyle().Foreground(s.Palette.Success())

	sb.WriteString(s.Category.Render("Changes (external):"))
	sb.WriteString("\n")

	// Modified fields - external changes from persisted state
	for _, field := range changes.ModifiedFields {
		prevValue := headless.FormatMappingNode(field.PrevValue)
		newValue := headless.FormatMappingNode(field.NewValue)
		line := fmt.Sprintf("  ± %s: %s → %s", field.FieldPath, prevValue, newValue)
		sb.WriteString(s.Warning.Render(line))
		sb.WriteString("\n")
	}

	// New fields - added externally
	for _, field := range changes.NewFields {
		line := fmt.Sprintf("  + %s: %s", field.FieldPath, headless.FormatMappingNode(field.NewValue))
		sb.WriteString(successStyle.Render(line))
		sb.WriteString("\n")
	}

	// Removed fields - removed externally
	for _, fieldPath := range changes.RemovedFields {
		line := fmt.Sprintf("  - %s", fieldPath)
		sb.WriteString(s.Error.Render(line))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *DriftDetailsRenderer) renderInterruptedResourceState(
	result *container.ResourceReconcileResult,
	s *styles.Styles,
) string {
	sb := strings.Builder{}

	// If the resource doesn't exist externally, indicate that
	if !result.ResourceExists {
		sb.WriteString(s.Category.Render("State:"))
		sb.WriteString("\n")
		sb.WriteString(s.Error.Render("  Resource not found externally"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Show external state if available
	if result.ExternalState != nil {
		sb.WriteString(s.Category.Render("External state:"))
		sb.WriteString("\n")
		sb.WriteString(renderMappingNodeSummary(result.ExternalState, s, "  "))
	} else {
		sb.WriteString(s.Muted.Render("No external state available"))
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderMappingNodeSummary(node *core.MappingNode, s *styles.Styles, indent string) string {
	sb := strings.Builder{}

	if node == nil {
		return sb.String()
	}

	formatted := headless.FormatMappingNode(node)
	// Limit the output to avoid overwhelming the UI
	lines := strings.Split(formatted, "\n")
	maxLines := 20
	for i, line := range lines {
		if i >= maxLines {
			sb.WriteString(s.Muted.Render(fmt.Sprintf("%s... (%d more lines)", indent, len(lines)-maxLines)))
			sb.WriteString("\n")
			break
		}
		sb.WriteString(indent)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *DriftDetailsRenderer) renderLinkDetails(item *DriftItem, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(item.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Drift type
	sb.WriteString(s.Muted.Render("Drift Type: "))
	sb.WriteString(HumanReadableDriftTypeLabel(item.DriftType))
	sb.WriteString("\n")

	// Child path if present
	if item.ChildPath != "" {
		sb.WriteString(s.Muted.Render("Child Path: "))
		sb.WriteString(item.ChildPath)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Link data updates
	if item.LinkResult != nil && len(item.LinkResult.LinkDataUpdates) > 0 {
		sb.WriteString(s.Category.Render("Link Data Affected:"))
		sb.WriteString("\n")
		for path := range item.LinkResult.LinkDataUpdates {
			sb.WriteString(s.Warning.Render(fmt.Sprintf("  ± %s", path)))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Recommended action
	sb.WriteString(s.Muted.Render("Recommended: "))
	sb.WriteString(HumanReadableAction(item.Recommended))
	sb.WriteString("\n")

	return sb.String()
}

func (r *DriftDetailsRenderer) renderChildDetails(item *DriftItem, width int, s *styles.Styles) string {
	sb := strings.Builder{}

	// Header
	sb.WriteString(s.Header.Render(item.Name))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")

	// Child path
	if item.ChildPath != "" {
		sb.WriteString(s.Muted.Render("Path: "))
		sb.WriteString(item.ChildPath)
		sb.WriteString("\n\n")
	}

	// Summary of items with drift
	resourceCount := 0
	linkCount := 0
	childCount := 0
	for _, child := range item.Children {
		switch child.Type {
		case DriftItemTypeResource:
			resourceCount++
		case DriftItemTypeLink:
			linkCount++
		case DriftItemTypeChild:
			childCount++
		}
	}

	sb.WriteString(s.Category.Render("Drift Summary:"))
	sb.WriteString("\n")
	if resourceCount > 0 {
		resourceLabel := sdkstrings.Pluralize(resourceCount, "resource", "resources")
		sb.WriteString(s.Warning.Render(fmt.Sprintf("  %d %s with drift", resourceCount, resourceLabel)))
		sb.WriteString("\n")
	}
	if linkCount > 0 {
		linkLabel := sdkstrings.Pluralize(linkCount, "link", "links")
		sb.WriteString(s.Warning.Render(fmt.Sprintf("  %d %s with drift", linkCount, linkLabel)))
		sb.WriteString("\n")
	}
	if childCount > 0 {
		childLabel := sdkstrings.Pluralize(childCount, "blueprint", "blueprints")
		sb.WriteString(s.Warning.Render(fmt.Sprintf("  %d nested child %s", childCount, childLabel)))
		sb.WriteString("\n")
	}

	// Show hint for expanding
	effectiveDepth := item.Depth + r.NavigationStackDepth
	if effectiveDepth >= r.MaxExpandDepth && len(item.Children) > 0 {
		sb.WriteString("\n")
		sb.WriteString(s.Hint.Render("Press enter to inspect this child blueprint"))
		sb.WriteString("\n")
	}

	return sb.String()
}

// DriftSectionGrouper implements splitpane.SectionGrouper for drift review UI.
type DriftSectionGrouper struct {
	MaxExpandDepth int
}

// Ensure DriftSectionGrouper implements splitpane.SectionGrouper
var _ splitpane.SectionGrouper = (*DriftSectionGrouper)(nil)

// GroupItems organizes drift items into sections: Resources, Links, Child Blueprints.
func (g *DriftSectionGrouper) GroupItems(
	items []splitpane.Item,
	isExpanded func(id string) bool,
) []splitpane.Section {
	var resources []splitpane.Item
	var links []splitpane.Item
	var children []splitpane.Item

	for _, item := range items {
		driftItem, ok := item.(*DriftItem)
		if !ok {
			continue
		}

		// Nested items (with ParentChild set) go to children section
		if driftItem.ParentChild != "" {
			children = append(children, item)
			continue
		}

		switch driftItem.Type {
		case DriftItemTypeResource:
			resources = append(resources, item)
		case DriftItemTypeLink:
			links = append(links, item)
		case DriftItemTypeChild:
			children = append(children, item)
			// If expanded, recursively add children inline
			children = g.appendExpandedChildren(children, item, isExpanded)
		}
	}

	// Sort each section for consistent ordering
	SortDriftItems(resources)
	SortDriftItems(links)

	var sections []splitpane.Section

	if len(resources) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Resources",
			Items: resources,
		})
	}

	if len(links) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Links",
			Items: links,
		})
	}

	if len(children) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Child Blueprints",
			Items: children,
		})
	}

	return sections
}

// appendExpandedChildren recursively appends children of an expanded item.
func (g *DriftSectionGrouper) appendExpandedChildren(
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
	SortDriftItems(childItems)

	for _, child := range childItems {
		children = append(children, child)
		if child.IsExpandable() {
			children = g.appendExpandedChildren(children, child, isExpanded)
		}
	}

	return children
}

// SortDriftItems sorts items alphabetically by name.
func SortDriftItems(items []splitpane.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
}

// DriftFooterRenderer implements splitpane.FooterRenderer for drift review UI.
type DriftFooterRenderer struct {
	Context DriftContext
}

// Ensure DriftFooterRenderer implements splitpane.FooterRenderer
var _ splitpane.FooterRenderer = (*DriftFooterRenderer)(nil)

// RenderFooter renders the drift review footer with options and contextual hint.
func (r *DriftFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
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

		// Navigation help for drill-down view
		sb.WriteString(s.Muted.Render("  "))
		sb.WriteString(s.Key.Render("esc"))
		sb.WriteString(s.Muted.Render(" back  "))
		sb.WriteString(s.Key.Render("↑/↓"))
		sb.WriteString(s.Muted.Render(" navigate  "))
		sb.WriteString(s.Key.Render("enter"))
		sb.WriteString(s.Muted.Render(" expand  "))
		sb.WriteString(s.Key.Render("a"))
		sb.WriteString(s.Muted.Render(" accept  "))
		sb.WriteString(s.Key.Render("q"))
		sb.WriteString(s.Muted.Render(" quit"))
		sb.WriteString("\n")

		return sb.String()
	}

	// Options
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("a"))
	sb.WriteString(s.Muted.Render(" accept external changes  "))
	sb.WriteString(s.Key.Render("q"))
	sb.WriteString(s.Muted.Render(" quit"))
	sb.WriteString("\n\n")

	// Contextual hint
	hint := HintForContext(r.Context)
	if hint != "" {
		sb.WriteString(s.Muted.Render("  Hint: "))
		sb.WriteString(s.Hint.Render(hint))
		sb.WriteString("\n\n")
	}

	// Navigation help
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" navigate  "))
	sb.WriteString(s.Key.Render("enter"))
	sb.WriteString(s.Muted.Render(" expand  "))
	sb.WriteString(s.Key.Render("tab"))
	sb.WriteString(s.Muted.Render(" switch pane  "))
	sb.WriteString(s.Key.Render("a"))
	sb.WriteString(s.Muted.Render(" accept  "))
	sb.WriteString(s.Key.Render("q"))
	sb.WriteString(s.Muted.Render(" quit"))
	sb.WriteString("\n")

	return sb.String()
}

// HumanReadableDriftType converts a ReconciliationType to a short uppercase label.
func HumanReadableDriftType(t container.ReconciliationType) string {
	switch t {
	case container.ReconciliationTypeDrift:
		return "DRIFT"
	case container.ReconciliationTypeInterrupted:
		return "INTERRUPTED"
	case container.ReconciliationTypeStateRefresh:
		return "STATE REFRESH"
	default:
		return string(t)
	}
}

// HumanReadableAction converts a ReconciliationAction to a human-readable label.
func HumanReadableAction(action container.ReconciliationAction) string {
	switch action {
	case container.ReconciliationActionAcceptExternal:
		return "Accept external state"
	case container.ReconciliationActionUpdateStatus:
		return "Update status only"
	case container.ReconciliationActionManualCleanupRequired:
		return "Manual cleanup required"
	default:
		return string(action)
	}
}

// HumanReadableDriftTypeLabel converts a ReconciliationType to a human-readable label.
func HumanReadableDriftTypeLabel(t container.ReconciliationType) string {
	switch t {
	case container.ReconciliationTypeDrift:
		return "Drift"
	case container.ReconciliationTypeInterrupted:
		return "Interrupted"
	case container.ReconciliationTypeStateRefresh:
		return "State refresh"
	default:
		return string(t)
	}
}
