package destroyui

import (
	"sort"
	"strings"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
)

// Headless output methods for DestroyModel.

func (m *DestroyModel) printHeadlessHeader() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println("Starting destroy...")
	w.Printf("Instance ID: %s\n", m.instanceID)
	if m.instanceName != "" {
		w.Printf("Instance Name: %s\n", m.instanceName)
	}
	w.Printf("Changeset: %s\n", m.changesetID)
	w.DoubleSeparator(72)
	w.PrintlnEmpty()
}

func (m *DestroyModel) printHeadlessResourceEvent(data *container.ResourceDeployUpdateMessage) {
	statusIcon := shared.ResourceStatusHeadlessIcon(data.Status)
	statusText := shared.ResourceStatusHeadlessText(data.Status)
	resourcePath := m.buildItemPath(data.InstanceID, data.ResourceName)
	displayPath := strings.ReplaceAll(resourcePath, "/", ".")
	m.printer.ProgressItem(statusIcon, "resource", displayPath, statusText, "")
}

func (m *DestroyModel) printHeadlessChildEvent(data *container.ChildDeployUpdateMessage) {
	statusIcon := shared.InstanceStatusHeadlessIcon(data.Status)
	statusText := shared.InstanceStatusHeadlessText(data.Status)
	childPath := m.buildInstancePath(data.ParentInstanceID, data.ChildName)
	displayPath := strings.ReplaceAll(childPath, "/", ".")
	m.printer.ProgressItem(statusIcon, "child", displayPath, statusText, "")
}

func (m *DestroyModel) printHeadlessLinkEvent(data *container.LinkDeployUpdateMessage) {
	statusIcon := shared.LinkStatusHeadlessIcon(data.Status)
	statusText := shared.LinkStatusHeadlessText(data.Status)
	linkPath := m.buildItemPath(data.InstanceID, data.LinkName)
	displayPath := strings.ReplaceAll(linkPath, "/", ".")
	m.printer.ProgressItem(statusIcon, "link", displayPath, statusText, "")
}

func (m *DestroyModel) printHeadlessSummary() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.DoubleSeparator(72)
	w.Println(m.getHeadlessSummaryHeader())
	w.DoubleSeparator(72)
	w.PrintlnEmpty()

	m.printHeadlessDestroyedItems()

	resourceCount := len(m.resourcesByName)
	childCount := len(m.childrenByName)
	linkCount := len(m.linksByName)

	w.DoubleSeparator(72)
	w.Printf("Complete: %d %s, %d %s, %d %s\n",
		resourceCount, sdkstrings.Pluralize(resourceCount, "resource", "resources"),
		childCount, sdkstrings.Pluralize(childCount, "child", "children"),
		linkCount, sdkstrings.Pluralize(linkCount, "link", "links"))
	w.PrintlnEmpty()

	if m.postDestroyInstanceState != nil && m.postDestroyInstanceState.Durations != nil {
		durations := m.postDestroyInstanceState.Durations
		if durations.TotalDuration != nil && *durations.TotalDuration > 0 {
			w.Printf("Total Duration: %s\n", outpututil.FormatDuration(*durations.TotalDuration))
		}
	}

	w.PrintlnEmpty()

	// Print pre-destroy state after the summary
	m.printHeadlessPreDestroyState()
}

var destroySummaryHeaders = map[core.InstanceStatus]string{
	core.InstanceStatusDestroyed:               "Destroy completed",
	core.InstanceStatusDestroyFailed:           "Destroy failed",
	core.InstanceStatusDestroyInterrupted:      "Destroy interrupted",
	core.InstanceStatusDestroyRollingBack:      "Destroy rolling back",
	core.InstanceStatusDestroyRollbackComplete: "Destroy rolled back",
	core.InstanceStatusDestroyRollbackFailed:   "Destroy rollback failed",
}

func (m *DestroyModel) getHeadlessSummaryHeader() string {
	if header, ok := destroySummaryHeaders[m.finalStatus]; ok {
		return header
	}
	return "Destroy completed"
}

func (m *DestroyModel) printHeadlessDestroyedItems() {
	for path, res := range m.resourcesByName {
		displayPath := strings.ReplaceAll(path, "/", ".")
		m.printHeadlessResourceDetailsWithPath(res, displayPath)
	}

	for path, child := range m.childrenByName {
		displayPath := strings.ReplaceAll(path, "/", ".")
		m.printHeadlessChildDetailsWithPath(child, displayPath)
	}

	for path, link := range m.linksByName {
		displayPath := strings.ReplaceAll(path, "/", ".")
		m.printHeadlessLinkDetailsWithPath(link, displayPath)
	}
}

func (m *DestroyModel) printHeadlessResourceDetailsWithPath(res *ResourceDestroyItem, displayPath string) {
	if res == nil {
		return
	}

	w := m.printer.Writer()

	statusIcon := shared.ResourceStatusHeadlessIcon(res.Status)
	statusText := shared.ResourceStatusHeadlessText(res.Status)

	w.Printf("%s resource %s: %s\n", statusIcon, displayPath, statusText)

	if len(res.FailureReasons) > 0 {
		for _, reason := range res.FailureReasons {
			w.Printf("    Error: %s\n", reason)
		}
	}
}

func (m *DestroyModel) printHeadlessChildDetailsWithPath(child *ChildDestroyItem, displayPath string) {
	if child == nil {
		return
	}

	w := m.printer.Writer()

	statusIcon := shared.InstanceStatusHeadlessIcon(child.Status)
	statusText := shared.InstanceStatusHeadlessText(child.Status)

	w.Printf("%s child %s: %s\n", statusIcon, displayPath, statusText)

	if len(child.FailureReasons) > 0 {
		for _, reason := range child.FailureReasons {
			w.Printf("    Error: %s\n", reason)
		}
	}
}

func (m *DestroyModel) printHeadlessLinkDetailsWithPath(link *LinkDestroyItem, displayPath string) {
	if link == nil {
		return
	}

	w := m.printer.Writer()

	statusIcon := shared.LinkStatusHeadlessIcon(link.Status)
	statusText := shared.LinkStatusHeadlessText(link.Status)

	w.Printf("%s link %s: %s\n", statusIcon, displayPath, statusText)

	if len(link.FailureReasons) > 0 {
		for _, reason := range link.FailureReasons {
			w.Printf("    Error: %s\n", reason)
		}
	}
}

func (m *DestroyModel) printHeadlessError(err error) {
	w := m.printer.Writer()
	w.PrintlnEmpty()

	if clientErr, isValidation := engineerrors.IsValidationError(err); isValidation {
		m.printHeadlessValidationError(clientErr)
		return
	}

	if streamErr, ok := err.(*engineerrors.StreamError); ok {
		m.printHeadlessStreamError(streamErr)
		return
	}

	w.Println("ERR Destroy failed")
	w.PrintlnEmpty()
	w.Printf("  Error: %s\n", err.Error())
}

func (m *DestroyModel) printHeadlessValidationError(clientErr *engineerrors.ClientError) {
	w := m.printer.Writer()
	w.Println("ERR Failed to start destroy")
	w.PrintlnEmpty()
	w.Println("The following issues must be resolved before destroy can proceed:")
	w.PrintlnEmpty()

	if len(clientErr.ValidationErrors) > 0 {
		for _, ve := range clientErr.ValidationErrors {
			w.Printf("  • %s\n", ve.Message)
			if ve.Location != "" {
				w.Printf("    Location: %s\n", ve.Location)
			}
		}
	}

	if len(clientErr.ValidationDiagnostics) > 0 {
		for _, diag := range clientErr.ValidationDiagnostics {
			w.Printf("  • %s\n", diag.Message)
		}
	}

	if len(clientErr.ValidationErrors) == 0 && len(clientErr.ValidationDiagnostics) == 0 {
		w.Printf("  %s\n", clientErr.Message)
	}
	w.PrintlnEmpty()
}

func (m *DestroyModel) printHeadlessStreamError(streamErr *engineerrors.StreamError) {
	w := m.printer.Writer()
	w.Println("ERR Error during destroy")
	w.PrintlnEmpty()
	w.Printf("  %s\n", streamErr.Event.Message)

	if len(streamErr.Event.Diagnostics) > 0 {
		w.PrintlnEmpty()
		w.Println("  Diagnostics:")
		for _, diag := range streamErr.Event.Diagnostics {
			w.Printf("    • %s\n", diag.Message)
		}
	}
	w.PrintlnEmpty()
}

func (m *DestroyModel) printHeadlessDriftDetected() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.DoubleSeparator(72)
	w.Println("Drift detected")
	w.DoubleSeparator(72)
	w.Println(m.driftMessage)
	w.PrintlnEmpty()

	if m.driftResult != nil {
		for _, r := range m.driftResult.Resources {
			w.Printf("  Resource: %s (%s)\n", r.ResourceName, r.Type)
		}
		for _, l := range m.driftResult.Links {
			w.Printf("  Link: %s (%s)\n", l.LinkName, l.Type)
		}
	}
	w.PrintlnEmpty()
}

func (m *DestroyModel) printHeadlessDeployChangesetError() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println("ERR Cannot destroy using a deploy changeset")
	w.PrintlnEmpty()
	w.Println("The changeset you specified was created for a deploy operation and cannot")
	w.Println("be used with the destroy command.")
	w.PrintlnEmpty()
	w.Println("To resolve this issue, you can either:")
	w.PrintlnEmpty()
	w.Println("  1. Use the 'deploy' command to apply this changeset:")
	w.Printf("     bluelink deploy --instance-name %s --change-set-id %s\n", m.instanceName, m.changesetID)
	w.PrintlnEmpty()
	w.Println("  2. Create a new changeset for destroy:")
	w.Printf("     bluelink stage --instance-name %s --destroy\n", m.instanceName)
	w.PrintlnEmpty()
}

func (m *DestroyModel) printHeadlessPreDestroyState() {
	if m.preDestroyInstanceState == nil {
		return
	}

	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.DoubleSeparator(72)
	w.Println("Pre-Destroy Instance State")
	w.DoubleSeparator(72)
	w.PrintlnEmpty()

	if m.instanceName != "" {
		w.Printf("Instance: %s\n", m.instanceName)
	}
	if m.preDestroyInstanceState.InstanceID != "" {
		w.Printf("ID: %s\n", m.preDestroyInstanceState.InstanceID)
	}
	w.PrintlnEmpty()

	m.printHeadlessInstanceStateHierarchy(w, m.preDestroyInstanceState, "")
}

func (m *DestroyModel) printHeadlessInstanceStateHierarchy(w *headless.PrefixedWriter, instanceState *state.InstanceState, indent string) {
	// Resources section
	if len(instanceState.ResourceIDs) > 0 {
		w.Printf("%sResources:\n", indent)
		printHeadlessResourceStates(w, instanceState, indent+"  ")
		w.PrintlnEmpty()
	}

	// Links section
	if len(instanceState.Links) > 0 {
		w.Printf("%sLinks:\n", indent)
		printHeadlessLinkStates(w, instanceState.Links, indent+"  ")
		w.PrintlnEmpty()
	}

	// Exports section
	if len(instanceState.Exports) > 0 {
		w.Printf("%sExports:\n", indent)
		printHeadlessExports(w, instanceState.Exports, indent+"  ")
		w.PrintlnEmpty()
	}

	// Child blueprints
	if len(instanceState.ChildBlueprints) > 0 {
		w.Printf("%sChild Blueprints:\n", indent)
		childNames := make([]string, 0, len(instanceState.ChildBlueprints))
		for name := range instanceState.ChildBlueprints {
			childNames = append(childNames, name)
		}
		sort.Strings(childNames)

		for _, childName := range childNames {
			childState := instanceState.ChildBlueprints[childName]
			w.PrintlnEmpty()
			if childState.InstanceID != "" {
				w.Printf("%s  %s (%s):\n", indent, childName, childState.InstanceID)
			} else {
				w.Printf("%s  %s:\n", indent, childName)
			}
			m.printHeadlessInstanceStateHierarchy(w, childState, indent+"    ")
		}
	}
}

func printHeadlessResourceStates(w *headless.PrefixedWriter, instanceState *state.InstanceState, indent string) {
	resourceNames := make([]string, 0, len(instanceState.ResourceIDs))
	for name := range instanceState.ResourceIDs {
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)

	for _, name := range resourceNames {
		resourceID := instanceState.ResourceIDs[name]
		resourceState := instanceState.Resources[resourceID]

		if resourceState != nil && resourceState.Type != "" {
			w.Printf("%s%s (%s):\n", indent, name, resourceState.Type)
		} else {
			w.Printf("%s%s:\n", indent, name)
		}
		w.Printf("%s  ID: %s\n", indent, resourceID)

		if resourceState != nil && resourceState.SpecData != nil {
			// Show spec (non-computed fields)
			specFields := outpututil.CollectNonComputedFieldsPretty(resourceState.SpecData, resourceState.ComputedFields)
			if len(specFields) > 0 {
				w.Printf("%s  Spec:\n", indent)
				for _, field := range specFields {
					printHeadlessFieldIndented(w, indent+"    ", field.Name, field.Value)
				}
			}

			// Show outputs (computed fields)
			outputFields := collectOutputFieldsPretty(resourceState.SpecData, resourceState.ComputedFields)
			if len(outputFields) > 0 {
				w.Printf("%s  Outputs:\n", indent)
				for _, field := range outputFields {
					printHeadlessFieldIndented(w, indent+"    ", field.Name, field.Value)
				}
			}
		}
	}
}

func printHeadlessLinkStates(w *headless.PrefixedWriter, links map[string]*state.LinkState, indent string) {
	linkNames := make([]string, 0, len(links))
	for name := range links {
		linkNames = append(linkNames, name)
	}
	sort.Strings(linkNames)

	for _, name := range linkNames {
		linkState := links[name]
		w.Printf("%s%s\n", indent, name)
		if linkState != nil && linkState.LinkID != "" {
			w.Printf("%s  ID: %s\n", indent, linkState.LinkID)
		}
	}
}

func printHeadlessExports(w *headless.PrefixedWriter, exports map[string]*state.ExportState, indent string) {
	exportNames := make([]string, 0, len(exports))
	for name := range exports {
		exportNames = append(exportNames, name)
	}
	sort.Strings(exportNames)

	prettyOpts := headless.FormatMappingNodeOptions{PrettyPrint: true}

	for _, name := range exportNames {
		export := exports[name]
		if export != nil && export.Value != nil {
			valueStr := headless.FormatMappingNodeWithOptions(export.Value, prettyOpts)
			if valueStr != "" && valueStr != "null" {
				printHeadlessFieldIndented(w, indent, name, valueStr)
			} else {
				w.Printf("%s%s\n", indent, name)
			}
		} else {
			w.Printf("%s%s\n", indent, name)
		}
	}
}

func printHeadlessFieldIndented(w *headless.PrefixedWriter, indent, name, value string) {
	if strings.Contains(value, "\n") {
		w.Printf("%s%s:\n", indent, name)
		for line := range strings.SplitSeq(value, "\n") {
			w.Printf("%s  %s\n", indent, line)
		}
	} else {
		w.Printf("%s%s: %s\n", indent, name, value)
	}
}

// collectOutputFieldsPretty extracts computed fields from spec data with pretty-printed JSON.
func collectOutputFieldsPretty(specData *core.MappingNode, computedFields []string) []outpututil.OutputField {
	if specData == nil || specData.Fields == nil || len(computedFields) == 0 {
		return nil
	}

	// Build a set of computed field names for quick lookup
	computedSet := make(map[string]bool, len(computedFields))
	for _, path := range computedFields {
		// Strip "spec." prefix to get the field name
		fieldName := strings.TrimPrefix(path, "spec.")
		// Only consider top-level fields
		if !strings.Contains(fieldName, ".") {
			computedSet[fieldName] = true
		}
	}

	// Collect computed field names and sort them
	var fieldNames []string
	for fieldName := range specData.Fields {
		if computedSet[fieldName] {
			fieldNames = append(fieldNames, fieldName)
		}
	}
	sort.Strings(fieldNames)

	prettyOpts := headless.FormatMappingNodeOptions{PrettyPrint: true}
	var fields []outpututil.OutputField
	for _, fieldName := range fieldNames {
		fieldValue := specData.Fields[fieldName]
		formattedValue := headless.FormatMappingNodeWithOptions(fieldValue, prettyOpts)
		if formattedValue != "" && formattedValue != "null" && formattedValue != "<nil>" {
			fields = append(fields, outpututil.OutputField{Name: fieldName, Value: formattedValue})
		}
	}
	return fields
}
