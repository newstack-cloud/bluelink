package deployui

import (
	"bytes"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/outpututil"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/stretchr/testify/suite"
)

type HeadlessOutputTestSuite struct {
	suite.Suite
}

func TestHeadlessOutputTestSuite(t *testing.T) {
	suite.Run(t, new(HeadlessOutputTestSuite))
}

func (s *HeadlessOutputTestSuite) createTestModel(buf *bytes.Buffer) *DeployModel {
	prefixedWriter := headless.NewPrefixedWriter(buf, "[deploy] ")
	printer := headless.NewPrinter(prefixedWriter, 80)
	return &DeployModel{
		printer:               printer,
		instanceID:            "test-instance-123",
		instanceName:          "test-instance",
		changesetID:           "changeset-456",
		resourcesByName:       make(map[string]*ResourceDeployItem),
		childrenByName:        make(map[string]*ChildDeployItem),
		linksByName:           make(map[string]*LinkDeployItem),
		instanceIDToChildName: make(map[string]string),
		instanceIDToParentID:  make(map[string]string),
	}
}

// getHeadlessSummaryHeader tests

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_deploy_rollback_complete() {
	m := &DeployModel{finalStatus: core.InstanceStatusDeployRollbackComplete}
	s.Equal("Deployment rolled back", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_update_rollback_complete() {
	m := &DeployModel{finalStatus: core.InstanceStatusUpdateRollbackComplete}
	s.Equal("Update rolled back", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_destroy_rollback_complete() {
	m := &DeployModel{finalStatus: core.InstanceStatusDestroyRollbackComplete}
	s.Equal("Destroy rolled back", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_deploy_rollback_failed() {
	m := &DeployModel{finalStatus: core.InstanceStatusDeployRollbackFailed}
	s.Equal("Deployment rollback failed", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_update_rollback_failed() {
	m := &DeployModel{finalStatus: core.InstanceStatusUpdateRollbackFailed}
	s.Equal("Update rollback failed", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_destroy_rollback_failed() {
	m := &DeployModel{finalStatus: core.InstanceStatusDestroyRollbackFailed}
	s.Equal("Destroy rollback failed", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_deploy_failed() {
	m := &DeployModel{finalStatus: core.InstanceStatusDeployFailed}
	s.Equal("Deployment failed", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_update_failed() {
	m := &DeployModel{finalStatus: core.InstanceStatusUpdateFailed}
	s.Equal("Update failed", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_destroy_failed() {
	m := &DeployModel{finalStatus: core.InstanceStatusDestroyFailed}
	s.Equal("Destroy failed", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_deployed() {
	m := &DeployModel{finalStatus: core.InstanceStatusDeployed}
	s.Equal("Deployment completed", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_updated() {
	m := &DeployModel{finalStatus: core.InstanceStatusUpdated}
	s.Equal("Update completed", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_destroyed() {
	m := &DeployModel{finalStatus: core.InstanceStatusDestroyed}
	s.Equal("Destroy completed", m.getHeadlessSummaryHeader())
}

func (s *HeadlessOutputTestSuite) Test_getHeadlessSummaryHeader_default() {
	m := &DeployModel{finalStatus: core.InstanceStatusPreparing}
	s.Equal("Deployment completed", m.getHeadlessSummaryHeader())
}

// isDeployRollbackComplete tests

func (s *HeadlessOutputTestSuite) Test_isDeployRollbackComplete_true() {
	m := &DeployModel{finalStatus: core.InstanceStatusDeployRollbackComplete}
	s.True(m.isDeployRollbackComplete())
}

func (s *HeadlessOutputTestSuite) Test_isDeployRollbackComplete_false_for_deployed() {
	m := &DeployModel{finalStatus: core.InstanceStatusDeployed}
	s.False(m.isDeployRollbackComplete())
}

func (s *HeadlessOutputTestSuite) Test_isDeployRollbackComplete_false_for_other_rollback() {
	m := &DeployModel{finalStatus: core.InstanceStatusUpdateRollbackComplete}
	s.False(m.isDeployRollbackComplete())
}

// printHeadlessHeader tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessHeader_outputs_instance_info() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	m.printHeadlessHeader()

	output := buf.String()
	s.Contains(output, "Starting deployment...")
	s.Contains(output, "Instance ID: test-instance-123")
	s.Contains(output, "Instance Name: test-instance")
	s.Contains(output, "Changeset: changeset-456")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessHeader_without_instance_name() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)
	m.instanceName = ""

	m.printHeadlessHeader()

	output := buf.String()
	s.Contains(output, "Instance ID: test-instance-123")
	s.NotContains(output, "Instance Name:")
}

// printHeadlessResourceEvent tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessResourceEvent_outputs_resource_status() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)
	// buildResourcePath uses instanceID to determine if it's root instance

	data := &container.ResourceDeployUpdateMessage{
		InstanceID:   "test-instance-123", // Same as model's instanceID, so root level
		ResourceName: "myResource",
		Status:       core.ResourceStatusCreated,
	}

	m.printHeadlessResourceEvent(data)

	output := buf.String()
	s.Contains(output, "resource")
	s.Contains(output, "myResource")
	s.Contains(output, "created")
}

// printHeadlessChildEvent tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessChildEvent_outputs_child_status() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)
	// buildInstancePath uses instanceID to determine if it's root instance

	data := &container.ChildDeployUpdateMessage{
		ParentInstanceID: "test-instance-123", // Same as model's instanceID, so root level
		ChildName:        "childBlueprint",
		Status:           core.InstanceStatusDeployed,
	}

	m.printHeadlessChildEvent(data)

	output := buf.String()
	s.Contains(output, "child")
	s.Contains(output, "childBlueprint")
	s.Contains(output, "deployed")
}

// printHeadlessLinkEvent tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessLinkEvent_outputs_link_status() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)
	// buildResourcePath uses instanceID to determine if it's root instance

	data := &container.LinkDeployUpdateMessage{
		InstanceID: "test-instance-123", // Same as model's instanceID, so root level
		LinkName:   "resourceA::resourceB",
		Status:     core.LinkStatusCreated,
	}

	m.printHeadlessLinkEvent(data)

	output := buf.String()
	s.Contains(output, "link")
	s.Contains(output, "resourceA::resourceB")
	s.Contains(output, "created")
}

// printHeadlessResourceDetailsWithPath tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessResourceDetailsWithPath_nil_resource() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	m.printHeadlessResourceDetailsWithPath(nil, "myResource", "myResource")

	// Should not panic and output should be empty (after trimming prefix lines)
	s.Empty(buf.String())
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessResourceDetailsWithPath_outputs_details() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	res := &ResourceDeployItem{
		Name:         "testResource",
		ResourceID:   "res-123",
		ResourceType: "aws/s3/bucket",
		Status:       core.ResourceStatusCreated,
	}

	m.printHeadlessResourceDetailsWithPath(res, "testResource", "testResource")

	output := buf.String()
	s.Contains(output, "resource")
	s.Contains(output, "testResource")
	s.Contains(output, "Resource ID: res-123")
	s.Contains(output, "Type: aws/s3/bucket")
	s.Contains(output, "Status: created")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessResourceDetailsWithPath_includes_timing() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	configDuration := float64(1500)
	totalDuration := float64(3000)
	res := &ResourceDeployItem{
		Name:   "testResource",
		Status: core.ResourceStatusCreated,
		Durations: &state.ResourceCompletionDurations{
			ConfigCompleteDuration: &configDuration,
			TotalDuration:          &totalDuration,
		},
	}

	m.printHeadlessResourceDetailsWithPath(res, "testResource", "testResource")

	output := buf.String()
	s.Contains(output, "Timing:")
	s.Contains(output, "Config Complete:")
	s.Contains(output, "Total:")
}

// printHeadlessChildDetailsWithPath tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessChildDetailsWithPath_nil_child() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	m.printHeadlessChildDetailsWithPath(nil, "childBlueprint")

	s.Empty(buf.String())
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessChildDetailsWithPath_outputs_details() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	child := &ChildDeployItem{
		Name:            "childBlueprint",
		ChildInstanceID: "child-instance-456",
		Status:          core.InstanceStatusDeployed,
	}

	m.printHeadlessChildDetailsWithPath(child, "childBlueprint")

	output := buf.String()
	s.Contains(output, "child")
	s.Contains(output, "childBlueprint")
	s.Contains(output, "Instance ID: child-instance-456")
	s.Contains(output, "Status: deployed")
}

// printHeadlessLinkDetailsWithPath tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessLinkDetailsWithPath_nil_link() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	m.printHeadlessLinkDetailsWithPath(nil, "resA::resB")

	s.Empty(buf.String())
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessLinkDetailsWithPath_outputs_details() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	link := &LinkDeployItem{
		LinkID:        "link-789",
		LinkName:      "resourceA::resourceB",
		ResourceAName: "resourceA",
		ResourceBName: "resourceB",
		Status:        core.LinkStatusCreated,
	}

	m.printHeadlessLinkDetailsWithPath(link, "resourceA.resourceB")

	output := buf.String()
	s.Contains(output, "link")
	s.Contains(output, "Link ID: link-789")
	s.Contains(output, "Status: created")
	s.Contains(output, "Connection: resourceA -> resourceB")
}

// printHeadlessError tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessError_generic_error() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	m.printHeadlessError(errors.New("something went wrong"))

	output := buf.String()
	s.Contains(output, "ERR Deployment failed")
	s.Contains(output, "Error: something went wrong")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessError_validation_error() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	clientErr := &engineerrors.ClientError{
		StatusCode: 400,
		Message:    "validation failed",
		ValidationErrors: []*engineerrors.ValidationError{
			{Location: "resources.bucket", Message: "invalid bucket name"},
		},
	}

	m.printHeadlessError(clientErr)

	output := buf.String()
	s.Contains(output, "ERR Failed to start deployment")
	s.Contains(output, "Validation Errors:")
	s.Contains(output, "resources.bucket")
	s.Contains(output, "invalid bucket name")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessError_validation_error_with_diagnostics() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	clientErr := &engineerrors.ClientError{
		StatusCode: 400,
		Message:    "blueprint errors",
		ValidationDiagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "Invalid resource reference",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{Position: source.Position{Line: 10, Column: 5}},
				},
			},
		},
	}

	m.printHeadlessError(clientErr)

	output := buf.String()
	s.Contains(output, "Blueprint Diagnostics:")
	s.Contains(output, "[ERROR]")
	s.Contains(output, "line 10, col 5")
	s.Contains(output, "Invalid resource reference")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessError_stream_error() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	streamErr := &engineerrors.StreamError{
		Event: &types.StreamErrorMessageEvent{
			Message: "deployment failed mid-stream",
			Diagnostics: []*core.Diagnostic{
				{
					Level:   core.DiagnosticLevelWarning,
					Message: "Resource took too long",
				},
			},
		},
	}

	m.printHeadlessError(streamErr)

	output := buf.String()
	s.Contains(output, "ERR Error during deployment")
	s.Contains(output, "deployment failed mid-stream")
	s.Contains(output, "Diagnostics:")
	s.Contains(output, "[WARNING]")
	s.Contains(output, "Resource took too long")
}

// printHeadlessDestroyChangesetError tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessDestroyChangesetError_outputs_guidance() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	m.printHeadlessDestroyChangesetError()

	output := buf.String()
	s.Contains(output, "ERR Cannot deploy using a destroy changeset")
	s.Contains(output, "destroy operation")
	s.Contains(output, "bluelink destroy --instance-name test-instance --change-set-id changeset-456")
	s.Contains(output, "bluelink stage --instance-name test-instance")
}

// printHeadlessSkippedRollbackItems tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessSkippedRollbackItems_empty() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)
	m.skippedRollbackItems = nil

	m.printHeadlessSkippedRollbackItems()

	s.Empty(buf.String())
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessSkippedRollbackItems_outputs_items() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)
	m.skippedRollbackItems = []container.SkippedRollbackItem{
		{
			Name:      "failedResource",
			Type:      "resource",
			ChildPath: "childBlueprint",
			Status:    "create_failed",
			Reason:    "resource was in failed state",
		},
	}

	m.printHeadlessSkippedRollbackItems()

	output := buf.String()
	s.Contains(output, "Skipped Rollback Items")
	s.Contains(output, "not rolled back because they were not in a safe state")
	s.Contains(output, "childBlueprint.failedResource")
	s.Contains(output, "resource")
	s.Contains(output, "Status: create_failed")
	s.Contains(output, "Reason: resource was in failed state")
}

// printHeadlessDiagnostic tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessDiagnostic_error_with_location() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	diag := &core.Diagnostic{
		Level:   core.DiagnosticLevelError,
		Message: "Undefined variable",
		Range: &core.DiagnosticRange{
			Start: &source.Meta{Position: source.Position{Line: 15, Column: 8}},
		},
	}

	m.printHeadlessDiagnostic(diag)

	output := buf.String()
	s.Contains(output, "[ERROR]")
	s.Contains(output, "line 15, col 8")
	s.Contains(output, "Undefined variable")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessDiagnostic_warning_without_location() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	diag := &core.Diagnostic{
		Level:   core.DiagnosticLevelWarning,
		Message: "Deprecated feature used",
	}

	m.printHeadlessDiagnostic(diag)

	output := buf.String()
	s.Contains(output, "[WARNING]")
	s.Contains(output, "Deprecated feature used")
	s.NotContains(output, "line")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessDiagnostic_info_level() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	diag := &core.Diagnostic{
		Level:   core.DiagnosticLevelInfo,
		Message: "Informational message",
	}

	m.printHeadlessDiagnostic(diag)

	output := buf.String()
	s.Contains(output, "[INFO]")
	s.Contains(output, "Informational message")
}

// printHeadlessField tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessField_single_line() {
	var buf bytes.Buffer
	prefixedWriter := headless.NewPrefixedWriter(&buf, "[test] ")
	w := headless.NewPrinter(prefixedWriter, 80).Writer()

	printHeadlessField(w, "fieldName", "fieldValue")

	output := buf.String()
	s.Contains(output, "fieldName: fieldValue")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessField_multi_line() {
	var buf bytes.Buffer
	prefixedWriter := headless.NewPrefixedWriter(&buf, "[test] ")
	w := headless.NewPrinter(prefixedWriter, 80).Writer()

	printHeadlessField(w, "config", "line1\nline2\nline3")

	output := buf.String()
	s.Contains(output, "config:")
	s.Contains(output, "line1")
	s.Contains(output, "line2")
	s.Contains(output, "line3")
}

// printResourceOutputs tests

func (s *HeadlessOutputTestSuite) Test_printResourceOutputs_nil_state() {
	var buf bytes.Buffer
	prefixedWriter := headless.NewPrefixedWriter(&buf, "[test] ")
	w := headless.NewPrinter(prefixedWriter, 80).Writer()

	printResourceOutputs(w, nil)

	s.Empty(buf.String())
}

func (s *HeadlessOutputTestSuite) Test_printResourceOutputs_no_computed_fields() {
	var buf bytes.Buffer
	prefixedWriter := headless.NewPrefixedWriter(&buf, "[test] ")
	w := headless.NewPrinter(prefixedWriter, 80).Writer()

	resourceState := &state.ResourceState{
		SpecData:       nil, // No spec data
		ComputedFields: []string{},
	}

	printResourceOutputs(w, resourceState)

	s.Empty(buf.String())
}

// printResourceSpec tests

func (s *HeadlessOutputTestSuite) Test_printResourceSpec_nil_state() {
	var buf bytes.Buffer
	prefixedWriter := headless.NewPrefixedWriter(&buf, "[test] ")
	w := headless.NewPrinter(prefixedWriter, 80).Writer()

	printResourceSpec(w, nil)

	s.Empty(buf.String())
}

func (s *HeadlessOutputTestSuite) Test_printResourceSpec_nil_spec_data() {
	var buf bytes.Buffer
	prefixedWriter := headless.NewPrefixedWriter(&buf, "[test] ")
	w := headless.NewPrinter(prefixedWriter, 80).Writer()

	resourceState := &state.ResourceState{
		SpecData: nil,
	}

	printResourceSpec(w, resourceState)

	s.Empty(buf.String())
}

// printHeadlessExportField tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessExportField_with_all_fields() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	field := outpututil.ExportField{
		Name:        "apiEndpoint",
		Type:        "string",
		Field:       "api.endpoint",
		Description: "The API endpoint URL",
		Value:       "https://api.example.com",
	}

	m.printHeadlessExportField(field)

	output := buf.String()
	s.Contains(output, "apiEndpoint:")
	s.Contains(output, "Type: string")
	s.Contains(output, "Field: api.endpoint")
	s.Contains(output, "Description: The API endpoint URL")
	s.Contains(output, "Value:")
	s.Contains(output, "https://api.example.com")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessExportField_null_value() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	field := outpututil.ExportField{
		Name:  "nullField",
		Value: "",
	}

	m.printHeadlessExportField(field)

	output := buf.String()
	s.Contains(output, "null")
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessExportField_multiline_value() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	field := outpututil.ExportField{
		Name:  "config",
		Value: "line1\nline2\nline3",
	}

	m.printHeadlessExportField(field)

	output := buf.String()
	s.Contains(output, "line1")
	s.Contains(output, "line2")
	s.Contains(output, "line3")
}

// resolveResourceState tests

func (s *HeadlessOutputTestSuite) Test_resolveResourceState_from_post_deploy_state() {
	m := &DeployModel{
		postDeployInstanceState: &state.InstanceState{
			ResourceIDs: map[string]string{
				"myResource": "res-from-state",
			},
			Resources: map[string]*state.ResourceState{
				"res-from-state": {
					ResourceID: "res-from-state",
					Name:       "myResource",
				},
			},
		},
	}

	res := &ResourceDeployItem{Name: "myResource"}
	result := m.resolveResourceState(res, "myResource")

	s.NotNil(result)
	s.Equal("res-from-state", result.ResourceID)
}

func (s *HeadlessOutputTestSuite) Test_resolveResourceState_falls_back_to_item_state() {
	m := &DeployModel{
		postDeployInstanceState: nil,
	}

	itemState := &state.ResourceState{
		ResourceID: "res-from-item",
		Name:       "myResource",
	}
	res := &ResourceDeployItem{
		Name:          "myResource",
		ResourceState: itemState,
	}
	result := m.resolveResourceState(res, "myResource")

	s.NotNil(result)
	s.Equal("res-from-item", result.ResourceID)
}

func (s *HeadlessOutputTestSuite) Test_resolveResourceState_returns_nil_when_not_found() {
	m := &DeployModel{
		postDeployInstanceState: &state.InstanceState{
			Resources: map[string]*state.ResourceState{},
		},
	}

	res := &ResourceDeployItem{Name: "unknownResource"}
	result := m.resolveResourceState(res, "unknownResource")

	s.Nil(result)
}

// printHeadlessPreRollbackState tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessPreRollbackState_outputs_state() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)

	data := &container.PreRollbackStateMessage{
		InstanceID:     "instance-123",
		InstanceName:   "my-instance",
		Status:         core.InstanceStatusDeployFailed,
		FailureReasons: []string{"Resource creation failed", "Timeout exceeded"},
		Resources: []container.ResourceSnapshot{
			{
				ResourceName: "failedResource",
				ResourceType: "aws/lambda/function",
				Status:       core.ResourceStatusCreateFailed,
			},
		},
		Links: []container.LinkSnapshot{
			{
				LinkName: "resA::resB",
				Status:   core.LinkStatusCreateFailed,
			},
		},
		Children: []container.ChildSnapshot{
			{
				ChildName: "childBlueprint",
				Status:    core.InstanceStatusDeployFailed,
			},
		},
	}

	m.printHeadlessPreRollbackState(data)

	output := buf.String()
	s.Contains(output, "Pre-Rollback State Captured")
	s.Contains(output, "Instance ID: instance-123")
	s.Contains(output, "Instance Name: my-instance")
	s.Contains(output, "Failure Reasons:")
	s.Contains(output, "Resource creation failed")
	s.Contains(output, "Resources (1):")
	s.Contains(output, "failedResource")
	s.Contains(output, "Links (1):")
	s.Contains(output, "resA::resB")
	s.Contains(output, "Children (1):")
	s.Contains(output, "childBlueprint")
	s.Contains(output, "Auto-rollback is starting...")
}

// printHeadlessResourceSnapshot tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessResourceSnapshot_basic() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)
	w := m.printer.Writer()

	snapshot := &container.ResourceSnapshot{
		ResourceName: "testResource",
		ResourceType: "aws/s3/bucket",
		Status:       core.ResourceStatusCreated,
	}

	m.printHeadlessResourceSnapshot(w, snapshot, "  ")

	output := buf.String()
	s.Contains(output, "testResource")
	s.Contains(output, "aws/s3/bucket")
	s.Contains(output, "CREATED") // status.String() returns uppercase
}

func (s *HeadlessOutputTestSuite) Test_printHeadlessResourceSnapshot_without_outputs() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)
	w := m.printer.Writer()

	// Test a snapshot without any computed fields to output
	snapshot := &container.ResourceSnapshot{
		ResourceName:   "testResource",
		ResourceType:   "aws/s3/bucket",
		Status:         core.ResourceStatusCreated,
		SpecData:       nil,
		ComputedFields: nil,
	}

	m.printHeadlessResourceSnapshot(w, snapshot, "  ")

	output := buf.String()
	s.Contains(output, "testResource")
	s.NotContains(output, "Outputs:")
}

// printHeadlessChildSnapshot tests

func (s *HeadlessOutputTestSuite) Test_printHeadlessChildSnapshot_with_nested_items() {
	var buf bytes.Buffer
	m := s.createTestModel(&buf)
	w := m.printer.Writer()

	snapshot := &container.ChildSnapshot{
		ChildName: "childBlueprint",
		Status:    core.InstanceStatusDeployed,
		Resources: []container.ResourceSnapshot{
			{
				ResourceName: "nestedResource",
				ResourceType: "aws/lambda/function",
				Status:       core.ResourceStatusCreated,
			},
		},
		Links: []container.LinkSnapshot{
			{
				LinkName: "linkA::linkB",
				Status:   core.LinkStatusCreated,
			},
		},
		Children: []container.ChildSnapshot{
			{
				ChildName: "grandChild",
				Status:    core.InstanceStatusDeployed,
			},
		},
	}

	m.printHeadlessChildSnapshot(w, snapshot, "  ")

	output := buf.String()
	s.Contains(output, "childBlueprint")
	s.Contains(output, "Resources (1):")
	s.Contains(output, "nestedResource")
	s.Contains(output, "Links (1):")
	s.Contains(output, "linkA::linkB")
	s.Contains(output, "Children (1):")
	s.Contains(output, "grandChild")
}
