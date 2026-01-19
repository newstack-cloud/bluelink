package deployui

import (
	"strings"

	"github.com/newstack-cloud/bluelink/apps/cli/diagutils"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
)

// Headless output methods for DeployModel.
// These methods handle rendering deployment progress and results in non-interactive mode.

func (m *DeployModel) printHeadlessHeader() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println("Starting deployment...")
	w.Printf("Instance ID: %s\n", m.instanceID)
	if m.instanceName != "" {
		w.Printf("Instance Name: %s\n", m.instanceName)
	}
	w.Printf("Changeset: %s\n", m.changesetID)
	w.DoubleSeparator(72)
	w.PrintlnEmpty()
}

func (m *DeployModel) printHeadlessResourceEvent(data *container.ResourceDeployUpdateMessage) {
	statusIcon := shared.ResourceStatusHeadlessIcon(data.Status)
	statusText := shared.ResourceStatusHeadlessText(data.Status)
	resourcePath := m.buildResourcePath(data.InstanceID, data.ResourceName)
	displayPath := strings.ReplaceAll(resourcePath, "/", ".")
	m.printer.ProgressItem(statusIcon, "resource", displayPath, statusText, "")
}

func (m *DeployModel) printHeadlessChildEvent(data *container.ChildDeployUpdateMessage) {
	statusIcon := shared.InstanceStatusHeadlessIcon(data.Status)
	statusText := shared.InstanceStatusHeadlessText(data.Status)
	childPath := m.buildInstancePath(data.ParentInstanceID, data.ChildName)
	displayPath := strings.ReplaceAll(childPath, "/", ".")
	m.printer.ProgressItem(statusIcon, "child", displayPath, statusText, "")
}

func (m *DeployModel) printHeadlessLinkEvent(data *container.LinkDeployUpdateMessage) {
	statusIcon := shared.LinkStatusHeadlessIcon(data.Status)
	statusText := shared.LinkStatusHeadlessText(data.Status)
	linkPath := m.buildResourcePath(data.InstanceID, data.LinkName)
	displayPath := strings.ReplaceAll(linkPath, "/", ".")
	m.printer.ProgressItem(statusIcon, "link", displayPath, statusText, "")
}

func (m *DeployModel) printHeadlessSummary() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.DoubleSeparator(72)
	w.Println(m.getHeadlessSummaryHeader())
	w.DoubleSeparator(72)
	w.PrintlnEmpty()

	m.printHeadlessDeployedItems()

	if !m.isDeployRollbackComplete() {
		m.printHeadlessExports()
	}

	m.printHeadlessSkippedRollbackItems()

	resourceCount := len(m.resourcesByName)
	childCount := len(m.childrenByName)
	linkCount := len(m.linksByName)

	w.DoubleSeparator(72)
	w.Printf("Complete: %d %s, %d %s, %d %s\n",
		resourceCount, sdkstrings.Pluralize(resourceCount, "resource", "resources"),
		childCount, sdkstrings.Pluralize(childCount, "child", "children"),
		linkCount, sdkstrings.Pluralize(linkCount, "link", "links"))
	w.PrintlnEmpty()

	if !m.isDeployRollbackComplete() {
		w.Printf("Instance ID: %s\n", m.instanceID)
		if m.instanceName != "" {
			w.Printf("Instance Name: %s\n", m.instanceName)
		}
	}

	if m.postDeployInstanceState != nil && m.postDeployInstanceState.Durations != nil {
		durations := m.postDeployInstanceState.Durations
		if durations.PrepareDuration != nil && *durations.PrepareDuration > 0 {
			w.Printf("Prepare Duration: %s\n", outpututil.FormatDuration(*durations.PrepareDuration))
		}
		if durations.TotalDuration != nil && *durations.TotalDuration > 0 {
			w.Printf("Total Duration: %s\n", outpututil.FormatDuration(*durations.TotalDuration))
		}
	}

	w.PrintlnEmpty()
}

var summaryHeaders = map[core.InstanceStatus]string{
	core.InstanceStatusDeployRollbackComplete:  "Deployment rolled back",
	core.InstanceStatusUpdateRollbackComplete:  "Update rolled back",
	core.InstanceStatusDestroyRollbackComplete: "Destroy rolled back",
	core.InstanceStatusDeployRollbackFailed:    "Deployment rollback failed",
	core.InstanceStatusUpdateRollbackFailed:    "Update rollback failed",
	core.InstanceStatusDestroyRollbackFailed:   "Destroy rollback failed",
	core.InstanceStatusDeployFailed:            "Deployment failed",
	core.InstanceStatusUpdateFailed:            "Update failed",
	core.InstanceStatusDestroyFailed:           "Destroy failed",
	core.InstanceStatusDeployed:                "Deployment completed",
	core.InstanceStatusUpdated:                 "Update completed",
	core.InstanceStatusDestroyed:               "Destroy completed",
}

func (m *DeployModel) getHeadlessSummaryHeader() string {
	if header, ok := summaryHeaders[m.finalStatus]; ok {
		return header
	}
	return "Deployment completed"
}

func (m *DeployModel) isDeployRollbackComplete() bool {
	return m.finalStatus == core.InstanceStatusDeployRollbackComplete
}

func (m *DeployModel) printHeadlessDeployedItems() {
	for path, res := range m.resourcesByName {
		displayPath := strings.ReplaceAll(path, "/", ".")
		m.printHeadlessResourceDetailsWithPath(res, displayPath, path)
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

func (m *DeployModel) printHeadlessResourceDetailsWithPath(res *ResourceDeployItem, displayPath string, stateLookupPath string) {
	if res == nil {
		return
	}

	w := m.printer.Writer()

	statusText := shared.ResourceStatusHeadlessText(res.Status)
	m.printer.ItemHeader("resource", displayPath, statusText)
	w.SingleSeparator(72)

	resourceState := m.resolveResourceState(res, stateLookupPath)

	m.printResourceBasicInfo(w, res, resourceState, statusText)
	m.printResourceTiming(w, res)
	printResourceOutputs(w, resourceState)
	printResourceSpec(w, resourceState)

	w.PrintlnEmpty()
	w.PrintlnEmpty()
}

func (m *DeployModel) resolveResourceState(res *ResourceDeployItem, stateLookupPath string) *state.ResourceState {
	var resourceState *state.ResourceState
	if m.postDeployInstanceState != nil {
		resourceState = findResourceStateByPath(m.postDeployInstanceState, stateLookupPath, res.Name)
	}
	if resourceState == nil && res.ResourceState != nil {
		resourceState = res.ResourceState
	}
	return resourceState
}

func (m *DeployModel) printResourceBasicInfo(w *headless.PrefixedWriter, res *ResourceDeployItem, resourceState *state.ResourceState, statusText string) {
	resourceID := res.ResourceID
	if resourceID == "" && resourceState != nil {
		resourceID = resourceState.ResourceID
	}
	if resourceID != "" {
		w.Printf("Resource ID: %s\n", resourceID)
	}

	resourceType := res.ResourceType
	if resourceType == "" && resourceState != nil {
		resourceType = resourceState.Type
	}
	if resourceType != "" {
		w.Printf("Type: %s\n", resourceType)
	}

	w.Printf("Status: %s\n", statusText)
}

func (m *DeployModel) printResourceTiming(w *headless.PrefixedWriter, res *ResourceDeployItem) {
	if res.Durations == nil {
		return
	}

	hasDurations := false
	if res.Durations.ConfigCompleteDuration != nil && *res.Durations.ConfigCompleteDuration > 0 {
		if !hasDurations {
			w.PrintlnEmpty()
			w.Println("Timing:")
			hasDurations = true
		}
		w.Printf("  Config Complete: %s\n", outpututil.FormatDuration(*res.Durations.ConfigCompleteDuration))
	}
	if res.Durations.TotalDuration != nil && *res.Durations.TotalDuration > 0 {
		if !hasDurations {
			w.PrintlnEmpty()
			w.Println("Timing:")
		}
		w.Printf("  Total: %s\n", outpututil.FormatDuration(*res.Durations.TotalDuration))
	}
}

func printResourceOutputs(w *headless.PrefixedWriter, resourceState *state.ResourceState) {
	if resourceState == nil || resourceState.SpecData == nil || len(resourceState.ComputedFields) == 0 {
		return
	}

	outputs := outpututil.CollectOutputFields(resourceState.SpecData, resourceState.ComputedFields)
	if len(outputs) == 0 {
		return
	}

	w.PrintlnEmpty()
	w.Println("Outputs:")
	for _, field := range outputs {
		w.Printf("  %s: %s\n", field.Name, field.Value)
	}
}

func printResourceSpec(w *headless.PrefixedWriter, resourceState *state.ResourceState) {
	if resourceState == nil || resourceState.SpecData == nil {
		return
	}

	specFields := outpututil.CollectNonComputedFieldsPretty(resourceState.SpecData, resourceState.ComputedFields)
	if len(specFields) == 0 {
		return
	}

	w.PrintlnEmpty()
	w.Println("Spec:")
	for _, field := range specFields {
		printHeadlessField(w, field.Name, field.Value)
	}
}

func printHeadlessField(w *headless.PrefixedWriter, name, value string) {
	if strings.Contains(value, "\n") {
		w.Printf("  %s:\n", name)
		for line := range strings.SplitSeq(value, "\n") {
			w.Printf("    %s\n", line)
		}
	} else {
		w.Printf("  %s: %s\n", name, value)
	}
}

func (m *DeployModel) printHeadlessChildDetailsWithPath(child *ChildDeployItem, displayPath string) {
	if child == nil {
		return
	}

	w := m.printer.Writer()

	statusText := shared.InstanceStatusHeadlessText(child.Status)
	m.printer.ItemHeader("child", displayPath, statusText)
	w.SingleSeparator(72)

	if child.ChildInstanceID != "" {
		w.Printf("Instance ID: %s\n", child.ChildInstanceID)
	}

	w.Printf("Status: %s\n", statusText)

	w.PrintlnEmpty()
	w.PrintlnEmpty()
}

func (m *DeployModel) printHeadlessLinkDetailsWithPath(link *LinkDeployItem, displayPath string) {
	if link == nil {
		return
	}

	w := m.printer.Writer()

	statusText := shared.LinkStatusHeadlessText(link.Status)
	m.printer.ItemHeader("link", displayPath, statusText)
	w.SingleSeparator(72)

	if link.LinkID != "" {
		w.Printf("Link ID: %s\n", link.LinkID)
	}

	w.Printf("Status: %s\n", statusText)

	if link.ResourceAName != "" && link.ResourceBName != "" {
		w.Printf("Connection: %s -> %s\n", link.ResourceAName, link.ResourceBName)
	}

	w.PrintlnEmpty()
	w.PrintlnEmpty()
}

func (m *DeployModel) printHeadlessExports() {
	if m.postDeployInstanceState == nil {
		return
	}
	m.printHeadlessInstanceExports("", m.postDeployInstanceState)
}

func (m *DeployModel) printHeadlessInstanceExports(path string, instanceState *state.InstanceState) {
	if instanceState == nil {
		return
	}

	w := m.printer.Writer()

	if len(instanceState.Exports) > 0 {
		fields := outpututil.CollectExportFieldsPretty(instanceState.Exports)
		if len(fields) > 0 {
			instanceName := "Root Instance"
			if path != "" {
				instanceName = path
			}
			m.printer.ItemHeader("exports", instanceName, "")
			w.SingleSeparator(72)

			for _, field := range fields {
				m.printHeadlessExportField(field)
			}
			w.PrintlnEmpty()
		}
	}

	for childName, childState := range instanceState.ChildBlueprints {
		childPath := childName
		if path != "" {
			childPath = path + "." + childName
		}
		m.printHeadlessInstanceExports(childPath, childState)
	}
}

func (m *DeployModel) printHeadlessExportField(field outpututil.ExportField) {
	w := m.printer.Writer()

	w.Printf("  %s:\n", field.Name)

	if field.Type != "" {
		w.Printf("    Type: %s\n", field.Type)
	}

	if field.Field != "" {
		w.Printf("    Field: %s\n", field.Field)
	}

	if field.Description != "" {
		w.Printf("    Description: %s\n", field.Description)
	}

	w.Println("    Value:")
	if field.Value == "" || field.Value == "null" {
		w.Println("      null")
	} else {
		lines := strings.Split(field.Value, "\n")
		for _, line := range lines {
			w.Printf("      %s\n", line)
		}
	}
	w.PrintlnEmpty()
}

func (m *DeployModel) printHeadlessSkippedRollbackItems() {
	if len(m.skippedRollbackItems) == 0 {
		return
	}

	w := m.printer.Writer()
	m.printer.ItemHeader("warning", "Skipped Rollback Items", "")
	w.SingleSeparator(72)
	w.Println("The following items were not rolled back because they were not in a safe state:")
	w.PrintlnEmpty()

	for _, item := range m.skippedRollbackItems {
		itemPath := item.Name
		if item.ChildPath != "" {
			itemPath = item.ChildPath + "." + item.Name
		}
		w.Printf("  - %s (%s)\n", itemPath, item.Type)
		w.Printf("    Status: %s\n", item.Status)
		w.Printf("    Reason: %s\n", item.Reason)
	}
	w.PrintlnEmpty()
}

func (m *DeployModel) printHeadlessError(err error) {
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

	w.Println("ERR Deployment failed")
	w.PrintlnEmpty()
	w.Printf("  Error: %s\n", err.Error())
}

func (m *DeployModel) printHeadlessDestroyChangesetError() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println("ERR Cannot deploy using a destroy changeset")
	w.PrintlnEmpty()
	w.Println("The changeset you specified was created for a destroy operation and cannot")
	w.Println("be used with the deploy command.")
	w.PrintlnEmpty()
	w.Println("To resolve this issue, you can either:")
	w.PrintlnEmpty()
	w.Println("  1. Use the 'destroy' command to apply this changeset:")
	w.Printf("     bluelink destroy --instance-name %s --change-set-id %s\n", m.instanceName, m.changesetID)
	w.PrintlnEmpty()
	w.Println("  2. Create a new changeset for deployment (without --destroy):")
	w.Printf("     bluelink stage --instance-name %s\n", m.instanceName)
	w.PrintlnEmpty()
}

func (m *DeployModel) printHeadlessValidationError(clientErr *engineerrors.ClientError) {
	w := m.printer.Writer()
	w.Println("ERR Failed to start deployment")
	w.PrintlnEmpty()
	w.Println("The following issues must be resolved before deployment can proceed:")
	w.PrintlnEmpty()

	if len(clientErr.ValidationErrors) > 0 {
		w.Println("Validation Errors:")
		w.SingleSeparator(72)
		for _, valErr := range clientErr.ValidationErrors {
			location := valErr.Location
			if location == "" {
				location = "unknown"
			}
			w.Printf("  - %s: %s\n", location, valErr.Message)
		}
		w.PrintlnEmpty()
	}

	if len(clientErr.ValidationDiagnostics) > 0 {
		w.Println("Blueprint Diagnostics:")
		w.SingleSeparator(72)
		for _, diag := range clientErr.ValidationDiagnostics {
			m.printHeadlessDiagnostic(diag)
		}
		w.PrintlnEmpty()
	}

	if len(clientErr.ValidationErrors) == 0 && len(clientErr.ValidationDiagnostics) == 0 {
		w.Printf("  %s\n", clientErr.Message)
	}
}

func (m *DeployModel) printHeadlessStreamError(streamErr *engineerrors.StreamError) {
	w := m.printer.Writer()
	w.Println("ERR Error during deployment")
	w.PrintlnEmpty()
	w.Println("The following issues occurred during deployment:")
	w.PrintlnEmpty()
	w.Printf("  %s\n", streamErr.Event.Message)
	w.PrintlnEmpty()

	if len(streamErr.Event.Diagnostics) > 0 {
		w.Println("Diagnostics:")
		w.SingleSeparator(72)
		for _, diag := range streamErr.Event.Diagnostics {
			m.printHeadlessDiagnostic(diag)
		}
		w.PrintlnEmpty()
	}
}

func (m *DeployModel) printHeadlessDiagnostic(diag *core.Diagnostic) {
	w := m.printer.Writer()

	levelName := getDiagnosticLevelName(diag.Level)
	line, col := getDiagnosticPosition(diag.Range)

	if line > 0 {
		w.Printf("  [%s] line %d, col %d: %s\n", levelName, line, col, diag.Message)
	} else {
		w.Printf("  [%s] %s\n", levelName, diag.Message)
	}

	if diag.Context != nil && len(diag.Context.SuggestedActions) > 0 {
		printHeadlessSuggestedActions(w, diag.Context)
	}
}

func getDiagnosticLevelName(level core.DiagnosticLevel) string {
	switch level {
	case core.DiagnosticLevelError:
		return "ERROR"
	case core.DiagnosticLevelWarning:
		return "WARNING"
	default:
		return "INFO"
	}
}

func getDiagnosticPosition(r *core.DiagnosticRange) (int, int) {
	if r == nil {
		return 0, 0
	}
	return r.Start.Line, r.Start.Column
}

func printHeadlessSuggestedActions(w *headless.PrefixedWriter, ctx *errors.ErrorContext) {
	w.Println("  Suggested Actions:")
	for i, action := range ctx.SuggestedActions {
		printHeadlessSuggestedAction(w, i+1, action, ctx.Metadata)
	}
}

func printHeadlessSuggestedAction(
	w *headless.PrefixedWriter,
	index int,
	action errors.SuggestedAction,
	metadata map[string]any,
) {
	w.Printf("    %d. %s\n", index, action.Title)
	if action.Description != "" {
		w.Printf("       %s\n", action.Description)
	}

	concrete := diagutils.GetConcreteAction(action, metadata)
	if concrete == nil {
		return
	}

	for _, cmd := range concrete.Commands {
		w.Printf("       Run: %s\n", cmd)
	}
	for _, link := range concrete.Links {
		w.Printf("       See: %s\n", link.URL)
	}
}

func (m *DeployModel) printHeadlessDriftDetected() {
	printer := driftui.NewHeadlessDriftPrinter(m.printer, m.driftContext)
	printer.PrintDriftDetected(m.driftResult)
}

func (m *DeployModel) printHeadlessPreRollbackState(data *container.PreRollbackStateMessage) {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.DoubleSeparator(72)
	w.Println("Pre-Rollback State Captured")
	w.DoubleSeparator(72)
	w.PrintlnEmpty()

	w.Printf("Instance ID: %s\n", data.InstanceID)
	w.Printf("Instance Name: %s\n", data.InstanceName)
	w.Printf("Status: %s\n", data.Status.String())
	w.PrintlnEmpty()

	if len(data.FailureReasons) > 0 {
		w.Println("Failure Reasons:")
		for _, reason := range data.FailureReasons {
			w.Printf("  - %s\n", reason)
		}
		w.PrintlnEmpty()
	}

	if len(data.Resources) > 0 {
		w.Printf("Resources (%d):\n", len(data.Resources))
		for _, r := range data.Resources {
			m.printHeadlessResourceSnapshot(w, &r, "  ")
		}
		w.PrintlnEmpty()
	}

	if len(data.Links) > 0 {
		w.Printf("Links (%d):\n", len(data.Links))
		for _, l := range data.Links {
			w.Printf("  - %s: %s\n", l.LinkName, l.Status.String())
		}
		w.PrintlnEmpty()
	}

	if len(data.Children) > 0 {
		w.Printf("Children (%d):\n", len(data.Children))
		for _, c := range data.Children {
			m.printHeadlessChildSnapshot(w, &c, "  ")
		}
		w.PrintlnEmpty()
	}

	w.Println("Auto-rollback is starting...")
	w.PrintlnEmpty()
}

func (m *DeployModel) printHeadlessResourceSnapshot(w *headless.PrefixedWriter, r *container.ResourceSnapshot, indent string) {
	w.Printf("%s- %s (%s): %s\n", indent, r.ResourceName, r.ResourceType, r.Status.String())

	if r.SpecData != nil {
		fields := outpututil.CollectOutputFields(r.SpecData, r.ComputedFields)
		if len(fields) > 0 {
			w.Printf("%s  Outputs:\n", indent)
			for _, field := range fields {
				w.Printf("%s    %s: %s\n", indent, field.Name, field.Value)
			}
		}
	}
}

func (m *DeployModel) printHeadlessChildSnapshot(w *headless.PrefixedWriter, c *container.ChildSnapshot, indent string) {
	w.Printf("%s- %s: %s\n", indent, c.ChildName, c.Status.String())

	if len(c.Resources) > 0 {
		w.Printf("%s  Resources (%d):\n", indent, len(c.Resources))
		for _, r := range c.Resources {
			m.printHeadlessResourceSnapshot(w, &r, indent+"    ")
		}
	}

	if len(c.Links) > 0 {
		w.Printf("%s  Links (%d):\n", indent, len(c.Links))
		for _, l := range c.Links {
			w.Printf("%s    - %s: %s\n", indent, l.LinkName, l.Status.String())
		}
	}

	if len(c.Children) > 0 {
		w.Printf("%s  Children (%d):\n", indent, len(c.Children))
		for _, nested := range c.Children {
			m.printHeadlessChildSnapshot(w, &nested, indent+"    ")
		}
	}
}
