package stageui

import (
	"bytes"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/stretchr/testify/suite"
)

type ViewHeadlessTestSuite struct {
	suite.Suite
}

func TestViewHeadlessTestSuite(t *testing.T) {
	suite.Run(t, new(ViewHeadlessTestSuite))
}

func (s *ViewHeadlessTestSuite) createTestModel() (*StageModel, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	prefixedWriter := headless.NewPrefixedWriter(buf, "[stage] ")
	printer := headless.NewPrinter(prefixedWriter, 80)

	model := &StageModel{
		changesetID: "cs-test-123",
		printer:     printer,
		items:       []StageItem{},
	}
	return model, buf
}

// --- printHeadlessHeader tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessHeader_prints_changeset_id() {
	model, buf := s.createTestModel()
	model.printHeadlessHeader()
	output := buf.String()
	s.Contains(output, "Starting change staging")
	s.Contains(output, "cs-test-123")
}

// --- printHeadlessResourceEvent tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessResourceEvent_shows_new_resource() {
	model, buf := s.createTestModel()
	data := &types.ResourceChangesEventData{
		ResourceChangesMessage: container.ResourceChangesMessage{
			ResourceName: "myResource",
			New:          true,
			Changes:      provider.Changes{},
		},
	}
	model.printHeadlessResourceEvent(data)
	output := buf.String()
	s.Contains(output, "myResource")
	s.Contains(output, "(new)")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessResourceEvent_shows_existing_resource() {
	model, buf := s.createTestModel()
	data := &types.ResourceChangesEventData{
		ResourceChangesMessage: container.ResourceChangesMessage{
			ResourceName: "existingResource",
			New:          false,
			Changes: provider.Changes{
				ModifiedFields: []provider.FieldChange{{FieldPath: "spec.field"}},
			},
		},
	}
	model.printHeadlessResourceEvent(data)
	output := buf.String()
	s.Contains(output, "existingResource")
	s.NotContains(output, "(new)")
}

// --- printHeadlessChildEvent tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessChildEvent_shows_new_child() {
	model, buf := s.createTestModel()
	data := &types.ChildChangesEventData{
		ChildChangesMessage: container.ChildChangesMessage{
			ChildBlueprintName: "myChild",
			New:                true,
			Changes: changes.BlueprintChanges{
				ResourceChanges: map[string]provider.Changes{"res1": {}, "res2": {}},
			},
		},
	}
	model.printHeadlessChildEvent(data)
	output := buf.String()
	s.Contains(output, "myChild")
	s.Contains(output, "(new")
	s.Contains(output, "2 resources")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessChildEvent_shows_existing_child() {
	model, buf := s.createTestModel()
	data := &types.ChildChangesEventData{
		ChildChangesMessage: container.ChildChangesMessage{
			ChildBlueprintName: "existingChild",
			New:                false,
			Changes: changes.BlueprintChanges{
				ResourceChanges: map[string]provider.Changes{"res1": {}},
			},
		},
	}
	model.printHeadlessChildEvent(data)
	output := buf.String()
	s.Contains(output, "existingChild")
	s.Contains(output, "1 resource")
}

// --- printHeadlessLinkEvent tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessLinkEvent_shows_new_link() {
	model, buf := s.createTestModel()
	data := &types.LinkChangesEventData{
		LinkChangesMessage: container.LinkChangesMessage{
			ResourceAName: "resA",
			ResourceBName: "resB",
			New:           true,
			Changes:       provider.LinkChanges{},
		},
	}
	model.printHeadlessLinkEvent(data)
	output := buf.String()
	s.Contains(output, "resA::resB")
	s.Contains(output, "(new)")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessLinkEvent_shows_existing_link() {
	model, buf := s.createTestModel()
	data := &types.LinkChangesEventData{
		LinkChangesMessage: container.LinkChangesMessage{
			ResourceAName: "resA",
			ResourceBName: "resB",
			New:           false,
			Changes:       provider.LinkChanges{},
		},
	}
	model.printHeadlessLinkEvent(data)
	output := buf.String()
	s.Contains(output, "resA::resB")
	s.NotContains(output, "(new)")
}

// --- printHeadlessSummary tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessSummary_shows_counts() {
	model, buf := s.createTestModel()
	model.items = []StageItem{
		{Type: ItemTypeResource, Name: "res1", Action: ActionCreate},
		{Type: ItemTypeResource, Name: "res2", Action: ActionUpdate},
		{Type: ItemTypeChild, Name: "child1", Action: ActionCreate},
	}
	model.printHeadlessSummary()
	output := buf.String()
	s.Contains(output, "Complete:")
	s.Contains(output, "2 resources")
	s.Contains(output, "1 child")
	s.Contains(output, "cs-test-123")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessSummary_shows_no_changes() {
	model, buf := s.createTestModel()
	model.items = []StageItem{}
	model.printHeadlessSummary()
	output := buf.String()
	s.Contains(output, "No changes to apply")
}

// --- printHeadlessApplyHint tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessApplyHint_deploy_with_instance_name() {
	model, buf := s.createTestModel()
	model.destroy = false
	model.instanceName = "my-instance"
	model.printHeadlessApplyHint()
	output := buf.String()
	s.Contains(output, "bluelink deploy")
	s.Contains(output, "--changeset-id cs-test-123")
	s.Contains(output, "--instance-name my-instance")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessApplyHint_deploy_with_instance_id() {
	model, buf := s.createTestModel()
	model.destroy = false
	model.instanceID = "inst-456"
	model.printHeadlessApplyHint()
	output := buf.String()
	s.Contains(output, "bluelink deploy")
	s.Contains(output, "--instance-id inst-456")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessApplyHint_destroy() {
	model, buf := s.createTestModel()
	model.destroy = true
	model.instanceName = "my-instance"
	model.printHeadlessApplyHint()
	output := buf.String()
	s.Contains(output, "bluelink destroy")
	s.Contains(output, "--changeset-id cs-test-123")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessApplyHint_placeholder_name() {
	model, buf := s.createTestModel()
	model.destroy = false
	model.printHeadlessApplyHint()
	output := buf.String()
	s.Contains(output, "--instance-name <name>")
}

// --- printHeadlessItemDetails tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessItemDetails_resource() {
	model, buf := s.createTestModel()
	item := StageItem{
		Type:   ItemTypeResource,
		Name:   "myRes",
		Action: ActionCreate,
	}
	model.printHeadlessItemDetails(item)
	output := buf.String()
	s.Contains(output, "myRes")
	s.Contains(output, "resource")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessItemDetails_child() {
	model, buf := s.createTestModel()
	item := StageItem{
		Type:   ItemTypeChild,
		Name:   "myChild",
		Action: ActionCreate,
	}
	model.printHeadlessItemDetails(item)
	output := buf.String()
	s.Contains(output, "myChild")
	s.Contains(output, "child")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessItemDetails_link() {
	model, buf := s.createTestModel()
	item := StageItem{
		Type:   ItemTypeLink,
		Name:   "resA::resB",
		Action: ActionCreate,
	}
	model.printHeadlessItemDetails(item)
	output := buf.String()
	s.Contains(output, "resA::resB")
	s.Contains(output, "link")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessItemDetails_with_parent() {
	model, buf := s.createTestModel()
	item := StageItem{
		Type:        ItemTypeResource,
		Name:        "nestedRes",
		Action:      ActionUpdate,
		ParentChild: "parentBlueprint",
	}
	model.printHeadlessItemDetails(item)
	output := buf.String()
	s.Contains(output, "parentBlueprint.nestedRes")
}

// --- printHeadlessResourceCurrentState tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessResourceCurrentState_shows_resource_id() {
	model, buf := s.createTestModel()
	resourceState := &state.ResourceState{
		ResourceID: "res-id-123",
		Type:       "aws/s3/bucket",
	}
	model.printHeadlessResourceCurrentState(resourceState)
	output := buf.String()
	s.Contains(output, "res-id-123")
	s.Contains(output, "aws/s3/bucket")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessResourceCurrentState_shows_outputs() {
	model, buf := s.createTestModel()
	bucketName := "my-bucket"
	resourceState := &state.ResourceState{
		ResourceID: "res-id-123",
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"bucketName": {Scalar: &core.ScalarValue{StringValue: &bucketName}},
			},
		},
		ComputedFields: []string{"bucketName"},
	}
	model.printHeadlessResourceCurrentState(resourceState)
	output := buf.String()
	s.Contains(output, "Current Outputs")
	s.Contains(output, "bucketName")
}

// --- printHeadlessChildChanges tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessChildChanges_shows_counts() {
	model, buf := s.createTestModel()
	childChanges := &changes.BlueprintChanges{
		NewResources:     map[string]provider.Changes{"res1": {}, "res2": {}},
		ResourceChanges:  map[string]provider.Changes{"res3": {}},
		RemovedResources: []string{"res4"},
	}
	model.printHeadlessChildChanges(childChanges)
	output := buf.String()
	s.Contains(output, "2")
	s.Contains(output, "1")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessChildChanges_no_changes() {
	model, buf := s.createTestModel()
	childChanges := &changes.BlueprintChanges{}
	model.printHeadlessChildChanges(childChanges)
	output := buf.String()
	s.Contains(output, "no")
}

// --- printHeadlessLinkChanges tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessLinkChanges_shows_field_changes() {
	model, buf := s.createTestModel()
	linkChanges := &provider.LinkChanges{
		NewFields: []*provider.FieldChange{
			{FieldPath: "newField"},
		},
		ModifiedFields: []*provider.FieldChange{
			{FieldPath: "modField"},
		},
		RemovedFields: []string{"removedField"},
	}
	model.printHeadlessLinkChanges(linkChanges)
	output := buf.String()
	s.Contains(output, "newField")
	s.Contains(output, "modField")
	s.Contains(output, "removedField")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessLinkChanges_no_changes() {
	model, buf := s.createTestModel()
	linkChanges := &provider.LinkChanges{}
	model.printHeadlessLinkChanges(linkChanges)
	output := buf.String()
	s.Contains(output, "no")
}

// --- printHeadlessResourceChanges tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessResourceChanges_shows_field_changes() {
	model, buf := s.createTestModel()
	resourceChanges := &provider.Changes{
		NewFields: []provider.FieldChange{
			{FieldPath: "spec.newField"},
		},
		ModifiedFields: []provider.FieldChange{
			{FieldPath: "spec.modField"},
		},
		RemovedFields: []string{"spec.removed"},
	}
	model.printHeadlessResourceChanges(resourceChanges)
	output := buf.String()
	s.Contains(output, "Field Changes")
	s.Contains(output, "newField")
	s.Contains(output, "modField")
	s.Contains(output, "removed")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessResourceChanges_shows_outbound_links() {
	model, buf := s.createTestModel()
	resourceChanges := &provider.Changes{
		NewOutboundLinks: map[string]provider.LinkChanges{
			"newLink": {},
		},
	}
	model.printHeadlessResourceChanges(resourceChanges)
	output := buf.String()
	s.Contains(output, "Outbound Link Changes")
	s.Contains(output, "newLink")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessResourceChanges_no_changes() {
	model, buf := s.createTestModel()
	resourceChanges := &provider.Changes{}
	model.printHeadlessResourceChanges(resourceChanges)
	output := buf.String()
	s.Contains(output, "no")
}

// --- printHeadlessOutboundLinkChanges tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessOutboundLinkChanges_shows_all_types() {
	model, buf := s.createTestModel()
	resourceChanges := &provider.Changes{
		NewOutboundLinks: map[string]provider.LinkChanges{
			"linkA": {},
		},
		OutboundLinkChanges: map[string]provider.LinkChanges{
			"linkB": {},
		},
		RemovedOutboundLinks: []string{"linkC"},
	}
	model.printHeadlessOutboundLinkChanges(resourceChanges)
	output := buf.String()
	s.Contains(output, "linkA")
	s.Contains(output, "(new link)")
	s.Contains(output, "linkB")
	s.Contains(output, "(link updated)")
	s.Contains(output, "linkC")
	s.Contains(output, "(link removed)")
}

// --- printHeadlessLinkFieldChanges tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessLinkFieldChanges_handles_nil() {
	model, buf := s.createTestModel()
	model.printHeadlessLinkFieldChanges(nil, "  ")
	output := buf.String()
	s.Empty(output)
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessLinkFieldChanges_shows_changes() {
	model, buf := s.createTestModel()
	linkChanges := &provider.LinkChanges{
		NewFields: []*provider.FieldChange{
			{FieldPath: "field1"},
		},
	}
	model.printHeadlessLinkFieldChanges(linkChanges, "    ")
	output := buf.String()
	s.Contains(output, "field1")
}

// --- printHeadlessError tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessError_generic_error() {
	model, buf := s.createTestModel()
	err := &testError{message: "something went wrong"}
	model.printHeadlessError(err)
	output := buf.String()
	s.Contains(output, "Error during change staging")
	s.Contains(output, "something went wrong")
}

// --- printHeadlessValidationError tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessValidationError_shows_validation_errors() {
	model, buf := s.createTestModel()
	clientErr := &engineerrors.ClientError{
		Message: "Validation failed",
		ValidationErrors: []*engineerrors.ValidationError{
			{Location: "resources.myRes", Message: "invalid field"},
		},
	}
	model.printHeadlessValidationError(clientErr)
	output := buf.String()
	s.Contains(output, "Failed to create changeset")
	s.Contains(output, "resources.myRes")
	s.Contains(output, "invalid field")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessValidationError_shows_diagnostics() {
	model, buf := s.createTestModel()
	clientErr := &engineerrors.ClientError{
		Message: "Validation failed",
		ValidationDiagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "Missing required field",
			},
		},
	}
	model.printHeadlessValidationError(clientErr)
	output := buf.String()
	s.Contains(output, "Blueprint Diagnostics")
	s.Contains(output, "Missing required field")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessValidationError_shows_message_fallback() {
	model, buf := s.createTestModel()
	clientErr := &engineerrors.ClientError{
		Message: "Some error message",
	}
	model.printHeadlessValidationError(clientErr)
	output := buf.String()
	s.Contains(output, "Some error message")
}

// --- printHeadlessStreamError tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessStreamError_shows_message() {
	model, buf := s.createTestModel()
	streamErr := &engineerrors.StreamError{
		Event: &types.StreamErrorMessageEvent{
			Message: "Stream error occurred",
		},
	}
	model.printHeadlessStreamError(streamErr)
	output := buf.String()
	s.Contains(output, "Error during change staging")
	s.Contains(output, "Stream error occurred")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessStreamError_shows_diagnostics() {
	model, buf := s.createTestModel()
	streamErr := &engineerrors.StreamError{
		Event: &types.StreamErrorMessageEvent{
			Message: "Error",
			Diagnostics: []*core.Diagnostic{
				{Level: core.DiagnosticLevelError, Message: "Diagnostic message"},
			},
		},
	}
	model.printHeadlessStreamError(streamErr)
	output := buf.String()
	s.Contains(output, "Diagnostics")
	s.Contains(output, "Diagnostic message")
}

// --- printHeadlessDiagnostic tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessDiagnostic_with_range() {
	model, buf := s.createTestModel()
	diag := &core.Diagnostic{
		Level:   core.DiagnosticLevelError,
		Message: "Error at location",
		Range: &core.DiagnosticRange{
			Start: &source.Meta{Position: source.Position{Line: 10, Column: 5}},
		},
	}
	model.printHeadlessDiagnostic(diag)
	output := buf.String()
	s.Contains(output, "line 10")
	s.Contains(output, "col 5")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessDiagnostic_without_range() {
	model, buf := s.createTestModel()
	diag := &core.Diagnostic{
		Level:   core.DiagnosticLevelWarning,
		Message: "Warning message",
	}
	model.printHeadlessDiagnostic(diag)
	output := buf.String()
	s.Contains(output, "Warning message")
}

// --- printHeadlessExportChanges tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessExportChanges_handles_nil() {
	model, buf := s.createTestModel()
	model.printHeadlessExportChanges(nil, "")
	output := buf.String()
	s.Empty(output)
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessExportChanges_shows_new_exports() {
	model, buf := s.createTestModel()
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"exportA": {},
		},
	}
	model.printHeadlessExportChanges(bc, "")
	output := buf.String()
	s.Contains(output, "New Exports")
	s.Contains(output, "exportA")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessExportChanges_shows_modified_exports() {
	model, buf := s.createTestModel()
	oldVal := "old"
	bc := &changes.BlueprintChanges{
		ExportChanges: map[string]provider.FieldChange{
			"modifiedExport": {PrevValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &oldVal}}},
		},
	}
	model.printHeadlessExportChanges(bc, "")
	output := buf.String()
	s.Contains(output, "Modified Exports")
	s.Contains(output, "modifiedExport")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessExportChanges_shows_removed_exports() {
	model, buf := s.createTestModel()
	bc := &changes.BlueprintChanges{
		RemovedExports: []string{"removedExport"},
	}
	model.printHeadlessExportChanges(bc, "")
	output := buf.String()
	s.Contains(output, "Removed Exports")
	s.Contains(output, "removedExport")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessExportChanges_with_prefix() {
	model, buf := s.createTestModel()
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"childExport": {},
		},
	}
	model.printHeadlessExportChanges(bc, "parentChild")
	output := buf.String()
	s.Contains(output, "parentChild")
}

// --- printHeadlessExportField tests ---

func (s *ViewHeadlessTestSuite) Test_printHeadlessExportField_new_field() {
	model, buf := s.createTestModel()
	newVal := "value123"
	change := &provider.FieldChange{NewValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &newVal}}}
	model.printHeadlessExportField("myExport", change, nil, true, false)
	output := buf.String()
	s.Contains(output, "myExport")
	s.Contains(output, "value123")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessExportField_modified_field() {
	model, buf := s.createTestModel()
	oldVal := "oldVal"
	newVal := "newVal"
	change := &provider.FieldChange{
		PrevValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &oldVal}},
		NewValue:  &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &newVal}},
	}
	model.printHeadlessExportField("myExport", change, nil, false, true)
	output := buf.String()
	s.Contains(output, "myExport")
	s.Contains(output, "oldVal")
	s.Contains(output, "newVal")
}

func (s *ViewHeadlessTestSuite) Test_printHeadlessExportField_computed_at_deploy() {
	model, buf := s.createTestModel()
	val := "value"
	change := &provider.FieldChange{NewValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &val}}}
	// resolveOnDeploy expects the full path "exports.<name>" format
	model.printHeadlessExportField("computedExport", change, []string{"exports.computedExport"}, true, false)
	output := buf.String()
	s.Contains(output, "known on deploy")
}

// --- helper test error type ---

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
