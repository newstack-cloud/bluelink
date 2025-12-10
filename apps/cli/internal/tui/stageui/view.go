package stageui

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
)

// MaxExpandDepth is the maximum nesting depth for expanding child blueprints.
// Children at this depth will be shown but cannot be expanded further in the left pane.
// Their details are still viewable in the right pane when selected.
const MaxExpandDepth = 2

// Headless rendering methods

func (m *StageModel) printHeadlessHeader() {
	w := m.printer.Writer()
	w.Println("Starting change staging...")
	w.Printf("Changeset: %s\n", m.changesetID)
	w.DoubleSeparator(72)
	w.PrintlnEmpty()
}

func (m *StageModel) printHeadlessResourceEvent(data *types.ResourceChangesEventData) {
	action := m.determineResourceAction(data)
	suffix := ""
	if data.New {
		suffix = "(new)"
	}
	m.printer.ProgressItem("✓", "resource", data.ResourceName, string(action), suffix)
}

func (m *StageModel) printHeadlessChildEvent(data *types.ChildChangesEventData) {
	action := m.determineChildAction(data)
	resourceCount := len(data.Changes.NewResources) + len(data.Changes.ResourceChanges)
	suffix := ""
	if data.New {
		suffix = fmt.Sprintf("(new, %d %s)", resourceCount, sdkstrings.Pluralize(resourceCount, "resource", "resources"))
	} else {
		suffix = fmt.Sprintf("(%d %s)", resourceCount, sdkstrings.Pluralize(resourceCount, "resource", "resources"))
	}
	m.printer.ProgressItem("✓", "child", data.ChildBlueprintName, string(action), suffix)
}

func (m *StageModel) printHeadlessLinkEvent(data *types.LinkChangesEventData) {
	action := m.determineLinkAction(data)
	linkName := fmt.Sprintf("%s::%s", data.ResourceAName, data.ResourceBName)
	suffix := ""
	if data.New {
		suffix = "(new)"
	}
	m.printer.ProgressItem("✓", "link", linkName, string(action), suffix)
}

func (m *StageModel) printHeadlessSummary() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.DoubleSeparator(72)
	w.Println("Change staging complete")
	w.DoubleSeparator(72)
	w.PrintlnEmpty()

	// Print detailed changes for each item
	for _, item := range m.items {
		m.printHeadlessItemDetails(item)
	}

	// Summary
	resources, children, links := m.countByType()
	create, update, delete, recreate := m.countChangeSummary()

	w.DoubleSeparator(72)
	w.Printf("Complete: %d %s, %d %s, %d %s\n",
		resources, sdkstrings.Pluralize(resources, "resource", "resources"),
		children, sdkstrings.Pluralize(children, "child", "children"),
		links, sdkstrings.Pluralize(links, "link", "links"))
	w.Printf("Actions: %d create, %d update, %d delete, %d recreate\n", create, update, delete, recreate)
	w.PrintlnEmpty()
	w.Printf("Changeset ID: %s\n", m.changesetID)
	w.PrintlnEmpty()

	// Build deploy command
	deployCmd := fmt.Sprintf("bluelink deploy --changeset-id %s", m.changesetID)
	if m.instanceName != "" {
		deployCmd += fmt.Sprintf(" --instance-name %s", m.instanceName)
	} else if m.instanceID != "" {
		deployCmd += fmt.Sprintf(" --instance-id %s", m.instanceID)
	} else {
		deployCmd += " --instance-name <name>"
	}
	m.printer.NextStep("To apply these changes, run:", deployCmd)
}

func (m *StageModel) printHeadlessItemDetails(item StageItem) {
	w := m.printer.Writer()

	// Item header with action
	m.printer.ItemHeader(string(item.Type), item.Name, string(item.Action))
	w.SingleSeparator(72)

	switch item.Type {
	case ItemTypeResource:
		if resourceChanges, ok := item.Changes.(*provider.Changes); ok {
			m.printHeadlessResourceChanges(resourceChanges)
		}
	case ItemTypeChild:
		if childChanges, ok := item.Changes.(*changes.BlueprintChanges); ok {
			m.printHeadlessChildChanges(childChanges)
		}
	case ItemTypeLink:
		if linkChanges, ok := item.Changes.(*provider.LinkChanges); ok {
			m.printHeadlessLinkChanges(linkChanges)
		}
	}

	w.PrintlnEmpty()
}

func (m *StageModel) printHeadlessResourceChanges(resourceChanges *provider.Changes) {
	hasChanges := len(resourceChanges.NewFields) > 0 || len(resourceChanges.ModifiedFields) > 0 || len(resourceChanges.RemovedFields) > 0

	if !hasChanges {
		m.printer.NoChanges()
		return
	}

	// New fields
	for _, field := range resourceChanges.NewFields {
		m.printer.FieldAdd(field.FieldPath, headless.FormatMappingNode(field.NewValue))
	}

	// Modified fields
	for _, field := range resourceChanges.ModifiedFields {
		m.printer.FieldModify(
			field.FieldPath,
			headless.FormatMappingNode(field.PrevValue),
			headless.FormatMappingNode(field.NewValue),
		)
	}

	// Removed fields
	for _, fieldPath := range resourceChanges.RemovedFields {
		m.printer.FieldRemove(fieldPath)
	}
}

func (m *StageModel) printHeadlessChildChanges(childChanges *changes.BlueprintChanges) {
	newCount := len(childChanges.NewResources)
	updateCount := len(childChanges.ResourceChanges)
	removeCount := len(childChanges.RemovedResources)

	m.printer.CountSummary(newCount, "resource", "resources", "to be created")
	m.printer.CountSummary(updateCount, "resource", "resources", "to be updated")
	m.printer.CountSummary(removeCount, "resource", "resources", "to be removed")

	if newCount == 0 && updateCount == 0 && removeCount == 0 {
		m.printer.NoChanges()
	}
}

func (m *StageModel) printHeadlessLinkChanges(linkChanges *provider.LinkChanges) {
	hasChanges := len(linkChanges.NewFields) > 0 || len(linkChanges.ModifiedFields) > 0 || len(linkChanges.RemovedFields) > 0

	if !hasChanges {
		m.printer.NoChanges()
		return
	}

	// New fields
	for _, field := range linkChanges.NewFields {
		m.printer.FieldAdd(field.FieldPath, headless.FormatMappingNode(field.NewValue))
	}

	// Modified fields
	for _, field := range linkChanges.ModifiedFields {
		m.printer.FieldModify(
			field.FieldPath,
			headless.FormatMappingNode(field.PrevValue),
			headless.FormatMappingNode(field.NewValue),
		)
	}

	// Removed fields
	for _, fieldPath := range linkChanges.RemovedFields {
		m.printer.FieldRemove(fieldPath)
	}
}

func (m *StageModel) printHeadlessError(err error) {
	w := m.printer.Writer()
	w.PrintlnEmpty()

	// Check for validation errors (ClientError with ValidationErrors or ValidationDiagnostics)
	if clientErr, isValidation := engineerrors.IsValidationError(err); isValidation {
		m.printHeadlessValidationError(clientErr)
		return
	}

	// Check for stream errors with diagnostics
	if streamErr, ok := err.(*engineerrors.StreamError); ok {
		m.printHeadlessStreamError(streamErr)
		return
	}

	// Generic error display
	w.Println("✗ Error during change staging")
	w.PrintlnEmpty()
	w.Printf("  Error: %s\n", err.Error())
}

func (m *StageModel) printHeadlessValidationError(clientErr *engineerrors.ClientError) {
	w := m.printer.Writer()
	w.Println("✗ Failed to create changeset")
	w.PrintlnEmpty()
	w.Println("The following issues must be resolved in the blueprint before changes can be staged:")
	w.PrintlnEmpty()

	// Render validation errors (input validation)
	if len(clientErr.ValidationErrors) > 0 {
		w.Println("Validation Errors:")
		w.SingleSeparator(72)
		for _, valErr := range clientErr.ValidationErrors {
			location := valErr.Location
			if location == "" {
				location = "unknown"
			}
			w.Printf("  • %s: %s\n", location, valErr.Message)
		}
		w.PrintlnEmpty()
	}

	// Render validation diagnostics (blueprint issues)
	if len(clientErr.ValidationDiagnostics) > 0 {
		w.Println("Blueprint Diagnostics:")
		w.SingleSeparator(72)
		for _, diag := range clientErr.ValidationDiagnostics {
			m.printHeadlessDiagnostic(diag)
		}
		w.PrintlnEmpty()
	}

	// If no specific errors, show the general message
	if len(clientErr.ValidationErrors) == 0 && len(clientErr.ValidationDiagnostics) == 0 {
		w.Printf("  %s\n", clientErr.Message)
	}
}

func (m *StageModel) printHeadlessStreamError(streamErr *engineerrors.StreamError) {
	w := m.printer.Writer()
	w.Println("✗ Error during change staging")
	w.PrintlnEmpty()
	w.Println("The following issues occurred during change staging:")
	w.PrintlnEmpty()
	w.Printf("  %s\n", streamErr.Event.Message)
	w.PrintlnEmpty()

	// Render diagnostics if present
	if len(streamErr.Event.Diagnostics) > 0 {
		w.Println("Diagnostics:")
		w.SingleSeparator(72)
		for _, diag := range streamErr.Event.Diagnostics {
			m.printHeadlessDiagnostic(diag)
		}
		w.PrintlnEmpty()
	}
}

func (m *StageModel) printHeadlessDiagnostic(diag *core.Diagnostic) {
	level := headless.DiagnosticLevelFromCore(diag.Level)
	levelName := headless.DiagnosticLevelName(level)

	line, col := 0, 0
	if diag.Range != nil {
		line = diag.Range.Start.Line
		col = diag.Range.Start.Column
	}

	m.printer.Diagnostic(levelName, diag.Message, line, col)
}

