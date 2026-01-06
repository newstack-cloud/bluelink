package inspectui

import (
	"sort"
	"strings"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
)

func (m *InspectModel) printHeadlessInstanceState() {
	if m.printer == nil {
		return
	}

	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println("Instance State")
	w.DoubleSeparator(72)

	if m.instanceState == nil {
		w.Println("No instance state available")
		return
	}

	// Basic info
	w.Printf("Instance ID: %s\n", m.instanceState.InstanceID)
	w.Printf("Instance Name: %s\n", m.instanceState.InstanceName)
	w.Printf("Status: %s\n", shared.InstanceStatusHeadlessText(m.instanceState.Status))
	w.PrintlnEmpty()

	// Resources (with spec and outputs)
	if len(m.instanceState.Resources) > 0 {
		w.Printf("Resources (%d):\n", len(m.instanceState.Resources))
		w.SingleSeparator(72)
		m.printHeadlessResourcesWithDetails(m.instanceState.Resources, "")
		w.PrintlnEmpty()
	}

	// Child Blueprints (with nested resources)
	if len(m.instanceState.ChildBlueprints) > 0 {
		w.Printf("Child Blueprints (%d):\n", len(m.instanceState.ChildBlueprints))
		w.SingleSeparator(72)
		m.printHeadlessChildBlueprintsRecursive(m.instanceState.ChildBlueprints, "")
		w.PrintlnEmpty()
	}

	// Links
	if len(m.instanceState.Links) > 0 {
		w.Printf("Links (%d):\n", len(m.instanceState.Links))
		w.SingleSeparator(72)
		for linkName, linkState := range m.instanceState.Links {
			m.printHeadlessLinkState(linkName, linkState.LinkID, linkState.Status.String())
		}
		w.PrintlnEmpty()
	}

	// Exports
	if len(m.instanceState.Exports) > 0 {
		w.Println("Exports:")
		w.SingleSeparator(72)
		fields := outpututil.CollectExportFieldsPretty(m.instanceState.Exports)
		for _, field := range fields {
			w.Printf("  %s: %s\n", field.Name, field.Value)
		}
		w.PrintlnEmpty()
	}

	// Durations
	if m.instanceState.Durations != nil {
		durations := m.instanceState.Durations
		if durations.PrepareDuration != nil && *durations.PrepareDuration > 0 {
			w.Printf("Prepare Duration: %s\n", outpututil.FormatDuration(*durations.PrepareDuration))
		}
		if durations.TotalDuration != nil && *durations.TotalDuration > 0 {
			w.Printf("Total Duration: %s\n", outpututil.FormatDuration(*durations.TotalDuration))
		}
	}

	// Summary
	w.DoubleSeparator(72)
	resourceCount := len(m.instanceState.Resources)
	childCount := len(m.instanceState.ChildBlueprints)
	linkCount := len(m.instanceState.Links)
	w.Printf("Total: %d %s, %d %s, %d %s\n",
		resourceCount, sdkstrings.Pluralize(resourceCount, "resource", "resources"),
		childCount, sdkstrings.Pluralize(childCount, "child", "children"),
		linkCount, sdkstrings.Pluralize(linkCount, "link", "links"))
	w.PrintlnEmpty()
}

func (m *InspectModel) printHeadlessResourcesWithDetails(resources map[string]*state.ResourceState, indent string) {
	// Sort resource names for consistent output
	var resourceNames []string
	for name := range resources {
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)

	for _, name := range resourceNames {
		resourceState := resources[name]
		m.printHeadlessResourceWithDetails(resourceState, indent)
	}
}

func (m *InspectModel) printHeadlessResourceWithDetails(resourceState *state.ResourceState, indent string) {
	w := m.printer.Writer()
	w.Printf("%s  %s\n", indent, resourceState.Name)
	w.Printf("%s    ID: %s\n", indent, resourceState.ResourceID)
	w.Printf("%s    Type: %s\n", indent, resourceState.Type)
	w.Printf("%s    Status: %s\n", indent, resourceState.Status.String())

	// Outputs (computed fields)
	if resourceState.SpecData != nil && len(resourceState.ComputedFields) > 0 {
		outputFields := outpututil.CollectOutputFields(resourceState.SpecData, resourceState.ComputedFields)
		if len(outputFields) > 0 {
			w.Printf("%s    Outputs:\n", indent)
			for _, field := range outputFields {
				w.Printf("%s      %s: %s\n", indent, field.Name, field.Value)
			}
		}
	}

	// Spec (non-computed fields)
	if resourceState.SpecData != nil {
		specFields := outpututil.CollectNonComputedFieldsPretty(resourceState.SpecData, resourceState.ComputedFields)
		if len(specFields) > 0 {
			w.Printf("%s    Spec:\n", indent)
			for _, field := range specFields {
				m.printHeadlessSpecField(field, indent+"      ")
			}
		}
	}
}

func (m *InspectModel) printHeadlessSpecField(field outpututil.OutputField, indent string) {
	w := m.printer.Writer()
	if strings.ContainsRune(field.Value, '\n') {
		w.Printf("%s%s:\n", indent, field.Name)
		for _, line := range strings.Split(field.Value, "\n") {
			w.Printf("%s  %s\n", indent, line)
		}
	} else {
		w.Printf("%s%s: %s\n", indent, field.Name, field.Value)
	}
}

func (m *InspectModel) printHeadlessChildBlueprintsRecursive(children map[string]*state.InstanceState, indent string) {
	w := m.printer.Writer()

	// Sort child names for consistent output
	var childNames []string
	for name := range children {
		childNames = append(childNames, name)
	}
	sort.Strings(childNames)

	for _, name := range childNames {
		childState := children[name]
		w.Printf("%s  %s\n", indent, name)
		w.Printf("%s    Instance ID: %s\n", indent, childState.InstanceID)
		w.Printf("%s    Status: %s\n", indent, childState.Status.String())

		// Print child's resources
		if len(childState.Resources) > 0 {
			w.Printf("%s    Resources (%d):\n", indent, len(childState.Resources))
			m.printHeadlessResourcesWithDetails(childState.Resources, indent+"    ")
		}

		// Print child's links
		if len(childState.Links) > 0 {
			w.Printf("%s    Links (%d):\n", indent, len(childState.Links))
			for linkName, linkState := range childState.Links {
				w.Printf("%s      %s\n", indent, linkName)
				w.Printf("%s        Link ID: %s\n", indent, linkState.LinkID)
				w.Printf("%s        Status: %s\n", indent, linkState.Status.String())
			}
		}

		// Recursively print nested child blueprints
		if len(childState.ChildBlueprints) > 0 {
			w.Printf("%s    Nested Blueprints (%d):\n", indent, len(childState.ChildBlueprints))
			m.printHeadlessChildBlueprintsRecursive(childState.ChildBlueprints, indent+"    ")
		}
	}
}


func (m *InspectModel) printHeadlessLinkState(linkName, linkID, status string) {
	w := m.printer.Writer()
	w.Printf("  %s\n", linkName)
	w.Printf("    Link ID: %s\n", linkID)
	w.Printf("    Status: %s\n", status)
}

func (m *InspectModel) printHeadlessResourceEvent(data *container.ResourceDeployUpdateMessage) {
	if m.printer == nil {
		return
	}
	statusIcon := shared.ResourceStatusHeadlessIcon(data.Status)
	statusText := shared.ResourceStatusHeadlessText(data.Status)
	m.printer.ProgressItem(statusIcon, "resource", data.ResourceName, statusText, "")
}

func (m *InspectModel) printHeadlessChildEvent(data *container.ChildDeployUpdateMessage) {
	if m.printer == nil {
		return
	}
	statusIcon := shared.InstanceStatusHeadlessIcon(data.Status)
	statusText := shared.InstanceStatusHeadlessText(data.Status)
	m.printer.ProgressItem(statusIcon, "child", data.ChildName, statusText, "")
}

func (m *InspectModel) printHeadlessLinkEvent(data *container.LinkDeployUpdateMessage) {
	if m.printer == nil {
		return
	}
	statusIcon := shared.LinkStatusHeadlessIcon(data.Status)
	statusText := shared.LinkStatusHeadlessText(data.Status)
	m.printer.ProgressItem(statusIcon, "link", data.LinkName, statusText, "")
}

func (m *InspectModel) printHeadlessError(err error) {
	if m.printer == nil {
		return
	}
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println("ERR Inspect failed")
	w.PrintlnEmpty()
	w.Printf("  Error: %s\n", err.Error())
}
