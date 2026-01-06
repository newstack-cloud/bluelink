package inspectui

import (
	"sort"
	"strings"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/deployui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// InspectDetailsRenderer implements splitpane.DetailsRenderer for inspect UI.
type InspectDetailsRenderer struct {
	MaxExpandDepth       int
	NavigationStackDepth int
	InstanceState        *state.InstanceState
	Finished             bool
}

var _ splitpane.DetailsRenderer = (*InspectDetailsRenderer)(nil)

// RenderDetails renders the right pane content for a selected item.
func (r *InspectDetailsRenderer) RenderDetails(item splitpane.Item, width int, s *styles.Styles) string {
	deployItem, ok := item.(*deployui.DeployItem)
	if !ok {
		return s.Muted.Render("Unknown item type")
	}

	switch deployItem.Type {
	case deployui.ItemTypeResource:
		return r.renderResourceDetails(deployItem, width, s)
	case deployui.ItemTypeChild:
		return r.renderChildDetails(deployItem, width, s)
	case deployui.ItemTypeLink:
		return r.renderLinkDetails(deployItem, width, s)
	default:
		return s.Muted.Render("Unknown item type")
	}
}

func (r *InspectDetailsRenderer) renderResourceDetails(item *deployui.DeployItem, width int, s *styles.Styles) string {
	res := item.Resource
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

	resourceState := res.ResourceState
	if resourceState == nil {
		// Check item's instance state first (handles nested blueprints)
		if item.InstanceState != nil {
			resourceState = findResourceStateByName(item.InstanceState, res.Name)
		}
		// Fall back to root instance state
		if resourceState == nil && r.InstanceState != nil {
			resourceState = findResourceStateByName(r.InstanceState, res.Name)
		}
	}

	// Resource ID
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

	// Resource type
	resourceType := res.ResourceType
	if resourceType == "" && resourceState != nil {
		resourceType = resourceState.Type
	}
	if resourceType != "" {
		sb.WriteString(s.Muted.Render("Type: "))
		sb.WriteString(resourceType)
		sb.WriteString("\n")
	}

	// Status - prefer resourceState, fall back to item status for streaming resources
	if resourceState != nil {
		sb.WriteString(s.Muted.Render("Status: "))
		sb.WriteString(shared.RenderResourceStatus(resourceState.Status, s))
		sb.WriteString("\n")
	} else if res.Status != 0 {
		sb.WriteString(s.Muted.Render("Status: "))
		sb.WriteString(shared.RenderResourceStatus(res.Status, s))
		sb.WriteString("\n")
	}

	// Outputs section
	if resourceState != nil {
		outputsContent := r.renderOutputsSection(resourceState, width, s)
		if outputsContent != "" {
			sb.WriteString("\n")
			sb.WriteString(outputsContent)
		}

		// Spec hint
		specHint := r.renderSpecHint(resourceState, s)
		if specHint != "" {
			sb.WriteString("\n")
			sb.WriteString(specHint)
			sb.WriteString("\n")
		}
	}

	// Outbound links section
	outboundLinks := r.renderOutboundLinksSection(res.Name, s)
	if outboundLinks != "" {
		sb.WriteString("\n")
		sb.WriteString(outboundLinks)
	}

	return sb.String()
}

func (r *InspectDetailsRenderer) renderOutputsSection(resourceState *state.ResourceState, width int, s *styles.Styles) string {
	if resourceState == nil || resourceState.SpecData == nil {
		return ""
	}

	fields := outpututil.CollectOutputFields(resourceState.SpecData, resourceState.ComputedFields)
	if len(fields) == 0 {
		return ""
	}

	return outpututil.RenderOutputFieldsWithLabel(fields, "Outputs:", width, s)
}

func (r *InspectDetailsRenderer) renderSpecHint(resourceState *state.ResourceState, s *styles.Styles) string {
	// Show spec hint when the resource has spec data available
	// (either deployment is finished or the individual resource has completed)
	if resourceState == nil || resourceState.SpecData == nil {
		return ""
	}

	return outpututil.RenderSpecHint(resourceState.SpecData, resourceState.ComputedFields, s)
}

func (r *InspectDetailsRenderer) renderOutboundLinksSection(resourceName string, s *styles.Styles) string {
	if r.InstanceState == nil || len(r.InstanceState.Links) == 0 {
		return ""
	}

	prefix := resourceName + "::"
	var outboundLinks []string
	for linkName, linkState := range r.InstanceState.Links {
		if strings.HasPrefix(linkName, prefix) {
			targetResource := strings.TrimPrefix(linkName, prefix)
			statusStr := formatLinkStatus(linkState.Status)
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

func (r *InspectDetailsRenderer) renderChildDetails(item *deployui.DeployItem, width int, s *styles.Styles) string {
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

	// Instance ID
	childInstanceID := child.ChildInstanceID
	if childInstanceID == "" && item.InstanceState != nil {
		childInstanceID = item.InstanceState.InstanceID
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
	sb.WriteString(shared.RenderInstanceStatus(child.Status, s))
	sb.WriteString("\n")

	// Show inspect hint for children at max expand depth
	effectiveDepth := item.Depth + r.NavigationStackDepth
	if effectiveDepth >= r.MaxExpandDepth && item.InstanceState != nil {
		sb.WriteString("\n")
		sb.WriteString(s.Hint.Render("Press enter to inspect this child blueprint"))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (r *InspectDetailsRenderer) renderLinkDetails(item *deployui.DeployItem, width int, s *styles.Styles) string {
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

	// Link ID
	if link.LinkID != "" {
		sb.WriteString(s.Muted.Render("Link ID: "))
		sb.WriteString(link.LinkID)
		sb.WriteString("\n")
	}

	// Status
	sb.WriteString(s.Muted.Render("Status: "))
	sb.WriteString(shared.RenderLinkStatus(link.Status, s))
	sb.WriteString("\n")

	return sb.String()
}

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

// InspectSectionGrouper implements splitpane.SectionGrouper for inspect UI.
type InspectSectionGrouper struct {
	shared.SectionGrouper
}

var _ splitpane.SectionGrouper = (*InspectSectionGrouper)(nil)

// InspectFooterRenderer implements splitpane.FooterRenderer for inspect UI.
type InspectFooterRenderer struct {
	InstanceID       string
	InstanceName     string
	CurrentStatus    core.InstanceStatus
	Streaming        bool
	Finished         bool
	SpinnerView      string
	HasInstanceState bool
	EmbeddedInList   bool // When true, shows "esc back to list" instead of "q quit"
}

var _ splitpane.FooterRenderer = (*InspectFooterRenderer)(nil)

// RenderFooter renders the inspect-specific footer.
func (r *InspectFooterRenderer) RenderFooter(model *splitpane.Model, s *styles.Styles) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	if model.IsInDrillDown() {
		shared.RenderBreadcrumb(&sb, model.NavigationPath(), s)
		shared.RenderFooterNavigation(&sb, s, shared.KeyHint{Key: "esc", Desc: "back"})
		return sb.String()
	}

	// Instance info
	sb.WriteString("  ")
	if r.Streaming && r.SpinnerView != "" {
		sb.WriteString(r.SpinnerView)
		sb.WriteString(" ")
	}

	if r.InstanceName != "" {
		sb.WriteString(s.Selected.Render(r.InstanceName))
	} else if r.InstanceID != "" {
		sb.WriteString(s.Selected.Render(r.InstanceID))
	}

	sb.WriteString(s.Muted.Render(" • "))
	sb.WriteString(shared.RenderInstanceStatus(r.CurrentStatus, s))
	sb.WriteString("\n")

	// View shortcuts
	if r.Finished && r.HasInstanceState {
		sb.WriteString(s.Muted.Render("  press "))
		sb.WriteString(s.Key.Render("o"))
		sb.WriteString(s.Muted.Render(" for overview, "))
		sb.WriteString(s.Key.Render("e"))
		sb.WriteString(s.Muted.Render(" for exports"))
		sb.WriteString("\n")
	} else if r.Streaming {
		sb.WriteString(s.Muted.Render("  streaming deployment events..."))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Navigation help
	sb.WriteString(s.Muted.Render("  "))
	sb.WriteString(s.Key.Render("↑/↓"))
	sb.WriteString(s.Muted.Render(" navigate  "))
	sb.WriteString(s.Key.Render("tab"))
	sb.WriteString(s.Muted.Render(" switch pane  "))
	if r.EmbeddedInList {
		sb.WriteString(s.Key.Render("esc"))
		sb.WriteString(s.Muted.Render(" back to list  "))
	}
	sb.WriteString(s.Key.Render("q"))
	sb.WriteString(s.Muted.Render(" quit"))
	sb.WriteString("\n")

	return sb.String()
}
