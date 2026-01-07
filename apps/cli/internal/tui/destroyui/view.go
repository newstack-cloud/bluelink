package destroyui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
)

// overviewFooterHeight returns the height of the fixed footer in overview view.
func overviewFooterHeight() int {
	// Footer consists of: separator line + empty line + key hints line + empty line
	return 4
}

// renderError renders the error view for interactive mode.
func (m DestroyModel) renderError(err error) string {
	ctx := shared.DestroyErrorContext()

	// Check for validation errors
	if clientErr, isValidation := engineerrors.IsValidationError(err); isValidation {
		return shared.RenderValidationError(clientErr, ctx, m.styles)
	}

	// Check for stream errors
	if streamErr, ok := err.(*engineerrors.StreamError); ok {
		return shared.RenderStreamError(streamErr, ctx, m.styles)
	}

	// Generic error display
	return shared.RenderGenericError(err, "Destroy failed", m.styles)
}

func (m DestroyModel) renderDeployChangesetError() string {
	return shared.RenderChangesetTypeMismatchError(shared.ChangesetTypeMismatchParams{
		IsDestroyChangeset: false,
		InstanceName:       m.instanceName,
		ChangesetID:        m.changesetID,
	}, m.styles)
}

// renderOverviewView renders a full-screen destroy summary view.
func (m DestroyModel) renderOverviewView() string {
	sb := strings.Builder{}
	sb.WriteString(m.overviewViewport.View())
	sb.WriteString("\n")
	shared.RenderViewportOverlayFooter(&sb, "o", m.styles)
	return sb.String()
}

// renderOverviewContent builds the content for the overview viewport.
func (m DestroyModel) renderOverviewContent() string {
	sb := strings.Builder{}
	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())

	// Header
	sb.WriteString("\n")
	headerText := "Destroy Summary"
	switch m.finalStatus {
	case core.InstanceStatusDestroyFailed:
		headerText = "Destroy Failed"
	case core.InstanceStatusDestroyRollbackComplete:
		headerText = "Destroy Rolled Back"
	case core.InstanceStatusDestroyInterrupted:
		headerText = "Destroy Interrupted"
	}
	sb.WriteString(m.styles.Header.Render("  " + headerText))
	sb.WriteString("\n\n")

	// Instance info
	if m.instanceName != "" {
		sb.WriteString(m.styles.Muted.Render("  Instance: "))
		sb.WriteString(m.styles.Selected.Render(m.instanceName))
		sb.WriteString("\n")
	}
	if m.instanceID != "" {
		sb.WriteString(m.styles.Muted.Render("  ID: "))
		sb.WriteString(m.styles.Muted.Render(sdkstrings.TruncateString(m.instanceID, 40)))
		sb.WriteString("\n")
	}
	if m.changesetID != "" {
		sb.WriteString(m.styles.Muted.Render("  Changeset: "))
		sb.WriteString(m.styles.Muted.Render(m.changesetID))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Calculate content width for text wrapping
	contentWidth := m.overviewViewport.Width - 4

	// Destroyed elements
	m.renderDestroyedElements(&sb, successStyle)

	// Failed elements
	m.renderFailedElements(&sb, contentWidth)

	// Interrupted elements
	m.renderInterruptedElements(&sb)

	// Final status message
	if len(m.failureReasons) > 0 {
		sb.WriteString(m.styles.Muted.Render("  "))
		sb.WriteString(m.styles.Muted.Render(strings.Repeat("─", 40)))
		sb.WriteString("\n")
		sb.WriteString(m.styles.Error.Render("  Root cause:"))
		sb.WriteString("\n")
		for _, reason := range m.failureReasons {
			sb.WriteString(m.styles.Muted.Render("    "))
			sb.WriteString(m.styles.Muted.Render(reason))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// renderDestroyedElements renders the destroyed operations section.
func (m DestroyModel) renderDestroyedElements(sb *strings.Builder, successStyle lipgloss.Style) {
	if len(m.destroyedElements) == 0 {
		return
	}

	elementLabel := sdkstrings.Pluralize(len(m.destroyedElements), "Element", "Elements")
	sb.WriteString(successStyle.Render(fmt.Sprintf("  %d Destroyed %s:", len(m.destroyedElements), elementLabel)))
	sb.WriteString("\n\n")

	for _, elem := range m.destroyedElements {
		sb.WriteString(successStyle.Render("  ✓ "))
		sb.WriteString(m.styles.Selected.Render(elem.ElementPath))
		if elem.ElementType != "" && elem.ElementType != "child" && elem.ElementType != "link" {
			sb.WriteString(m.styles.Muted.Render(" (" + elem.ElementType + ")"))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

// renderFailedElements renders element failures with text wrapping and full element paths.
func (m DestroyModel) renderFailedElements(sb *strings.Builder, contentWidth int) {
	shared.RenderElementFailures(sb, m.elementFailures, contentWidth, true, m.styles)
}

// renderInterruptedElements renders interrupted elements with full paths.
func (m DestroyModel) renderInterruptedElements(sb *strings.Builder) {
	if len(m.interruptedElements) == 0 {
		return
	}

	elementLabel := sdkstrings.Pluralize(len(m.interruptedElements), "Element", "Elements")
	sb.WriteString(m.styles.Warning.Render(fmt.Sprintf("  %d %s Interrupted:", len(m.interruptedElements), elementLabel)))
	sb.WriteString("\n\n")

	for _, elem := range m.interruptedElements {
		sb.WriteString(m.styles.Warning.Render("  ⏹ "))
		sb.WriteString(m.styles.Selected.Render(elem.ElementPath))
		if elem.ElementType != "" && elem.ElementType != "child" && elem.ElementType != "link" {
			sb.WriteString(m.styles.Muted.Render(" (" + elem.ElementType + ")"))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("    These elements were interrupted and their state is unknown."))
	sb.WriteString("\n")
}

// renderPreDestroyStateView renders a full-screen pre-destroy instance state view.
func (m DestroyModel) renderPreDestroyStateView() string {
	sb := strings.Builder{}
	sb.WriteString(m.preDestroyStateViewport.View())
	sb.WriteString("\n")
	shared.RenderViewportOverlayFooter(&sb, "s", m.styles)
	return sb.String()
}

// renderPreDestroyStateContent builds the content for the pre-destroy state viewport.
func (m DestroyModel) renderPreDestroyStateContent() string {
	if m.preDestroyInstanceState == nil {
		return m.styles.Muted.Render("  No pre-destroy state available")
	}

	sb := strings.Builder{}

	// Header
	sb.WriteString("\n")
	sb.WriteString(m.styles.Header.Render("  Pre-Destroy Instance State"))
	sb.WriteString("\n\n")

	// Instance info
	if m.instanceName != "" {
		sb.WriteString(m.styles.Muted.Render("  Instance: "))
		sb.WriteString(m.styles.Selected.Render(m.instanceName))
		sb.WriteString("\n")
	}
	if m.preDestroyInstanceState.InstanceID != "" {
		sb.WriteString(m.styles.Muted.Render("  ID: "))
		sb.WriteString(m.styles.Muted.Render(m.preDestroyInstanceState.InstanceID))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Calculate content width for text wrapping
	contentWidth := m.preDestroyStateViewport.Width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Render the instance state hierarchy
	m.renderInstanceStateHierarchy(&sb, m.preDestroyInstanceState, "", 0, contentWidth)

	sb.WriteString("\n")
	return sb.String()
}

// renderInstanceStateHierarchy recursively renders instance state including children.
func (m DestroyModel) renderInstanceStateHierarchy(sb *strings.Builder, instanceState *state.InstanceState, prefix string, depth int, contentWidth int) {
	indent := strings.Repeat("  ", depth+1)
	// Calculate available width for values at this depth
	valueWidth := contentWidth - len(indent) - 4
	if valueWidth < 20 {
		valueWidth = 20
	}

	// Resources section - check ResourceIDs since that's the canonical list of resources
	if len(instanceState.ResourceIDs) > 0 {
		sb.WriteString(m.styles.Category.Render(indent + "Resources:"))
		sb.WriteString("\n")
		m.renderResourceStates(sb, instanceState, indent+"  ", valueWidth)
		sb.WriteString("\n")
	}

	// Links section
	if len(instanceState.Links) > 0 {
		sb.WriteString(m.styles.Category.Render(indent + "Links:"))
		sb.WriteString("\n")
		m.renderLinkStates(sb, instanceState.Links, indent+"  ")
		sb.WriteString("\n")
	}

	// Exports section
	if len(instanceState.Exports) > 0 {
		sb.WriteString(m.styles.Category.Render(indent + "Exports:"))
		sb.WriteString("\n")
		m.renderExports(sb, instanceState.Exports, indent+"  ", valueWidth)
		sb.WriteString("\n")
	}

	// Child blueprints
	if len(instanceState.ChildBlueprints) > 0 {
		sb.WriteString(m.styles.Category.Render(indent + "Child Blueprints:"))
		sb.WriteString("\n")

		childNames := make([]string, 0, len(instanceState.ChildBlueprints))
		for name := range instanceState.ChildBlueprints {
			childNames = append(childNames, name)
		}
		sort.Strings(childNames)

		for _, childName := range childNames {
			childState := instanceState.ChildBlueprints[childName]
			sb.WriteString(indent + "  ")
			sb.WriteString(m.styles.Selected.Render(childName))
			if childState.InstanceID != "" {
				sb.WriteString(m.styles.Muted.Render(" ("))
				sb.WriteString(m.styles.Muted.Render(childState.InstanceID))
				sb.WriteString(m.styles.Muted.Render(")"))
			}
			sb.WriteString("\n")

			// Recursively render child state
			m.renderInstanceStateHierarchy(sb, childState, prefix+childName+"/", depth+1, contentWidth)
		}
	}
}

// renderResourceStates renders the resources from instance state.
func (m DestroyModel) renderResourceStates(sb *strings.Builder, instanceState *state.InstanceState, indent string, valueWidth int) {
	resourceNames := make([]string, 0, len(instanceState.ResourceIDs))
	for name := range instanceState.ResourceIDs {
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)

	for _, name := range resourceNames {
		resourceID := instanceState.ResourceIDs[name]
		resourceState := instanceState.Resources[resourceID]

		sb.WriteString(indent)
		sb.WriteString(m.styles.Muted.Render("• "))
		sb.WriteString(name)
		if resourceState != nil && resourceState.Type != "" {
			sb.WriteString(m.styles.Muted.Render(" ("))
			sb.WriteString(m.styles.Muted.Render(resourceState.Type))
			sb.WriteString(m.styles.Muted.Render(")"))
		}
		sb.WriteString("\n")

		// Show resource ID
		sb.WriteString(indent + "  ")
		sb.WriteString(m.styles.Muted.Render("ID: "))
		sb.WriteString(m.styles.Muted.Render(resourceID))
		sb.WriteString("\n")

		if resourceState != nil && resourceState.SpecData != nil {
			computedFieldsSet := make(map[string]bool)
			for _, field := range resourceState.ComputedFields {
				computedFieldsSet[field] = true
			}

			// Show spec (non-computed fields)
			specFields := filterSpecFields(resourceState.SpecData, computedFieldsSet, false)
			if len(specFields) > 0 {
				sb.WriteString(indent + "  ")
				sb.WriteString(m.styles.Muted.Render("Spec:"))
				sb.WriteString("\n")
				m.renderFieldsWithWrapping(sb, specFields, indent+"    ", valueWidth)
			}

			// Show outputs (computed fields)
			outputFields := filterSpecFields(resourceState.SpecData, computedFieldsSet, true)
			if len(outputFields) > 0 {
				sb.WriteString(indent + "  ")
				sb.WriteString(m.styles.Muted.Render("Outputs:"))
				sb.WriteString("\n")
				m.renderFieldsWithWrapping(sb, outputFields, indent+"    ", valueWidth)
			}
		}
	}
}

// renderFieldsWithWrapping renders field name/value pairs with text wrapping for long values.
func (m DestroyModel) renderFieldsWithWrapping(sb *strings.Builder, fields []fieldInfo, indent string, valueWidth int) {
	for _, field := range fields {
		sb.WriteString(indent)
		sb.WriteString(m.styles.Muted.Render(field.Name + ": "))

		// Check if value contains newlines (pretty-printed JSON)
		if strings.Contains(field.Value, "\n") {
			sb.WriteString("\n")
			lines := strings.Split(field.Value, "\n")
			for _, line := range lines {
				sb.WriteString(indent + "  ")
				sb.WriteString(m.styles.Muted.Render(line))
				sb.WriteString("\n")
			}
		} else {
			// Wrap long single-line values
			wrappedLines := outpututil.WrapTextLines(field.Value, valueWidth)
			if len(wrappedLines) == 1 {
				sb.WriteString(m.styles.Muted.Render(wrappedLines[0]))
				sb.WriteString("\n")
			} else {
				sb.WriteString("\n")
				for _, line := range wrappedLines {
					sb.WriteString(indent + "  ")
					sb.WriteString(m.styles.Muted.Render(line))
					sb.WriteString("\n")
				}
			}
		}
	}
}

// fieldInfo holds name and value for a spec field.
type fieldInfo struct {
	Name  string
	Value string
}

// filterSpecFields extracts fields from SpecData, filtering by computed status.
// Uses pretty-printed JSON for complex nested objects and arrays.
func filterSpecFields(specData *core.MappingNode, computedFields map[string]bool, onlyComputed bool) []fieldInfo {
	if specData == nil || specData.Fields == nil {
		return nil
	}

	prettyOpts := headless.FormatMappingNodeOptions{PrettyPrint: true}

	var fields []fieldInfo
	fieldNames := make([]string, 0, len(specData.Fields))
	for name := range specData.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, name := range fieldNames {
		isComputed := computedFields[name]
		if onlyComputed != isComputed {
			continue
		}

		node := specData.Fields[name]
		valueStr := headless.FormatMappingNodeWithOptions(node, prettyOpts)
		if valueStr != "" && valueStr != "null" {
			fields = append(fields, fieldInfo{Name: name, Value: valueStr})
		}
	}

	return fields
}

// renderLinkStates renders the links from instance state.
func (m DestroyModel) renderLinkStates(sb *strings.Builder, links map[string]*state.LinkState, indent string) {
	linkNames := make([]string, 0, len(links))
	for name := range links {
		linkNames = append(linkNames, name)
	}
	sort.Strings(linkNames)

	for _, name := range linkNames {
		linkState := links[name]
		sb.WriteString(indent)
		sb.WriteString(m.styles.Muted.Render("• "))
		sb.WriteString(name)
		sb.WriteString("\n")

		if linkState.LinkID != "" {
			sb.WriteString(indent + "  ")
			sb.WriteString(m.styles.Muted.Render("ID: "))
			sb.WriteString(m.styles.Muted.Render(sdkstrings.TruncateString(linkState.LinkID, 50)))
			sb.WriteString("\n")
		}
	}
}

// renderExports renders the exports from instance state with text wrapping.
func (m DestroyModel) renderExports(sb *strings.Builder, exports map[string]*state.ExportState, indent string, valueWidth int) {
	exportNames := make([]string, 0, len(exports))
	for name := range exports {
		exportNames = append(exportNames, name)
	}
	sort.Strings(exportNames)

	prettyOpts := headless.FormatMappingNodeOptions{PrettyPrint: true}

	for _, name := range exportNames {
		export := exports[name]
		sb.WriteString(indent)
		sb.WriteString(m.styles.Muted.Render("• "))
		sb.WriteString(name)

		if export != nil && export.Value != nil {
			valueStr := headless.FormatMappingNodeWithOptions(export.Value, prettyOpts)
			if valueStr != "" && valueStr != "null" {
				sb.WriteString(m.styles.Muted.Render(": "))
				// Handle multi-line values (pretty-printed JSON)
				if strings.Contains(valueStr, "\n") {
					sb.WriteString("\n")
					lines := strings.Split(valueStr, "\n")
					for _, line := range lines {
						sb.WriteString(indent + "  ")
						sb.WriteString(m.styles.Muted.Render(line))
						sb.WriteString("\n")
					}
				} else {
					// Wrap long single-line values
					wrappedLines := outpututil.WrapTextLines(valueStr, valueWidth)
					if len(wrappedLines) == 1 {
						sb.WriteString(m.styles.Muted.Render(wrappedLines[0]))
						sb.WriteString("\n")
					} else {
						sb.WriteString("\n")
						for _, line := range wrappedLines {
							sb.WriteString(indent + "  ")
							sb.WriteString(m.styles.Muted.Render(line))
							sb.WriteString("\n")
						}
					}
				}
			} else {
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString("\n")
		}
	}
}
