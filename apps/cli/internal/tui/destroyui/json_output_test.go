package destroyui

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type JSONOutputSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestJSONOutputSuite(t *testing.T) {
	suite.Run(t, new(JSONOutputSuite))
}

func (s *JSONOutputSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *JSONOutputSuite) Test_json_output_success() {
	jsonOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(nil, "test-instance-id", nil),
		zap.NewNop(),
		"test-changeset-123",
		"test-instance-id",
		"test-instance",
		false,
		s.styles,
		true, // headless
		jsonOutput,
		nil,
		true, // jsonMode
	)

	// Set up final state as if destroy completed successfully
	model.finalStatus = core.InstanceStatusDestroyed
	model.instanceID = "test-instance-id"
	model.instanceName = "test-instance"
	model.changesetID = "test-changeset-123"
	model.destroyedElements = []DestroyedElement{
		{ElementName: "resource-1", ElementPath: "resource-1", ElementType: "aws/ec2/instance"},
		{ElementName: "resource-2", ElementPath: "resource-2", ElementType: "aws/s3/bucket"},
	}
	model.postDestroyInstanceState = &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDestroyed,
	}

	model.outputJSON()

	// Parse and validate JSON output
	var output jsonout.DestroyOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.True(output.Success)
	s.Equal("test-instance-id", output.InstanceID)
	s.Equal("test-instance", output.InstanceName)
	s.Equal("test-changeset-123", output.ChangesetID)
	s.Equal("DESTROYED", output.Status)
	s.Equal(2, output.Summary.Destroyed)
	s.Equal(0, output.Summary.Failed)
	s.Equal(0, output.Summary.Interrupted)
	s.Len(output.Summary.Elements, 2)
}

func (s *JSONOutputSuite) Test_json_output_with_failures() {
	jsonOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(nil, "test-instance-id", nil),
		zap.NewNop(),
		"test-changeset-fail",
		"test-instance-id",
		"test-instance",
		false,
		s.styles,
		true, // headless
		jsonOutput,
		nil,
		true, // jsonMode
	)

	// Set up final state with failures
	model.finalStatus = core.InstanceStatusDestroyFailed
	model.instanceID = "test-instance-id"
	model.instanceName = "test-instance"
	model.changesetID = "test-changeset-fail"
	model.destroyedElements = []DestroyedElement{
		{ElementName: "resource-1", ElementPath: "resource-1", ElementType: "aws/ec2/instance"},
	}
	model.elementFailures = []ElementFailure{
		{
			ElementName:    "resource-2",
			ElementPath:    "resource-2",
			ElementType:    "aws/s3/bucket",
			FailureReasons: []string{"Resource is in use", "Cannot delete non-empty bucket"},
		},
	}
	model.postDestroyInstanceState = &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDestroyFailed,
	}

	model.outputJSON()

	// Parse and validate JSON output
	var output jsonout.DestroyOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.True(output.Success) // Still true because operation completed
	s.Equal("DESTROY FAILED", output.Status)
	s.Equal(1, output.Summary.Destroyed)
	s.Equal(1, output.Summary.Failed)
	s.Len(output.Summary.Elements, 2)

	// Check failure reasons are included
	var failedElement *jsonout.DestroyedElement
	for i := range output.Summary.Elements {
		if output.Summary.Elements[i].Status == "failed" {
			failedElement = &output.Summary.Elements[i]
			break
		}
	}
	s.Require().NotNil(failedElement)
	s.Equal("resource-2", failedElement.Name)
	s.Len(failedElement.FailureReasons, 2)
}

func (s *JSONOutputSuite) Test_json_output_with_interrupted() {
	jsonOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(nil, "test-instance-id", nil),
		zap.NewNop(),
		"test-changeset-int",
		"test-instance-id",
		"test-instance",
		false,
		s.styles,
		true, // headless
		jsonOutput,
		nil,
		true, // jsonMode
	)

	// Set up final state with interrupted elements
	model.finalStatus = core.InstanceStatusDestroyInterrupted
	model.instanceID = "test-instance-id"
	model.instanceName = "test-instance"
	model.changesetID = "test-changeset-int"
	model.destroyedElements = []DestroyedElement{
		{ElementName: "resource-1", ElementPath: "resource-1", ElementType: "aws/ec2/instance"},
	}
	model.interruptedElements = []InterruptedElement{
		{ElementName: "resource-2", ElementPath: "resource-2", ElementType: "aws/s3/bucket"},
		{ElementName: "resource-3", ElementPath: "resource-3", ElementType: "aws/rds/instance"},
	}
	model.postDestroyInstanceState = &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDestroyInterrupted,
	}

	model.outputJSON()

	// Parse and validate JSON output
	var output jsonout.DestroyOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal("DESTROY INTERRUPTED", output.Status)
	s.Equal(1, output.Summary.Destroyed)
	s.Equal(0, output.Summary.Failed)
	s.Equal(2, output.Summary.Interrupted)
	s.Len(output.Summary.Elements, 3)
}

func (s *JSONOutputSuite) Test_json_output_drift_detected() {
	jsonOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(nil, "test-instance-id", nil),
		zap.NewNop(),
		"test-changeset-drift",
		"test-instance-id",
		"test-instance",
		false,
		s.styles,
		true, // headless
		jsonOutput,
		nil,
		true, // jsonMode
	)

	// Set up drift state
	model.instanceID = "test-instance-id"
	model.instanceName = "test-instance"
	model.driftMessage = "Drift detected: external changes found"
	model.driftResult = &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "resource-1",
				ResourceType: "aws/ec2/instance",
				Type:         container.ReconciliationTypeDrift,
			},
		},
	}

	model.outputJSONDrift()

	// Parse and validate JSON output
	var output jsonout.DestroyDriftOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.True(output.Success)
	s.True(output.DriftDetected)
	s.Equal("test-instance-id", output.InstanceID)
	s.Equal("test-instance", output.InstanceName)
	s.Equal("Drift detected: external changes found", output.Message)
	s.NotNil(output.Reconciliation)
	s.Len(output.Reconciliation.Resources, 1)
}

func (s *JSONOutputSuite) Test_json_output_error() {
	jsonOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(nil, "test-instance-id", nil),
		zap.NewNop(),
		"test-changeset-err",
		"test-instance-id",
		"test-instance",
		false,
		s.styles,
		true, // headless
		jsonOutput,
		nil,
		true, // jsonMode
	)

	testErr := &testError{message: "failed to connect to deploy engine"}
	model.outputJSONError(testErr)

	// Parse and validate JSON output
	var output jsonout.ErrorOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.False(output.Success)
	s.Contains(output.Error.Message, "failed to connect to deploy engine")
}

func (s *JSONOutputSuite) Test_build_destroy_summary() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(nil, "test-instance-id", nil),
		zap.NewNop(),
		"test-changeset",
		"test-instance-id",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	model.destroyedElements = []DestroyedElement{
		{ElementName: "res-1", ElementPath: "res-1", ElementType: "type-a"},
		{ElementName: "res-2", ElementPath: "res-2", ElementType: "type-b"},
	}
	model.elementFailures = []ElementFailure{
		{ElementName: "res-3", ElementPath: "res-3", ElementType: "type-c", FailureReasons: []string{"error"}},
	}
	model.interruptedElements = []InterruptedElement{
		{ElementName: "res-4", ElementPath: "res-4", ElementType: "type-d"},
	}

	summary := model.buildDestroySummary()

	s.Equal(2, summary.Destroyed)
	s.Equal(1, summary.Failed)
	s.Equal(1, summary.Interrupted)
	s.Len(summary.Elements, 4)

	// Verify element statuses
	statuses := make(map[string]string)
	for _, elem := range summary.Elements {
		statuses[elem.Name] = elem.Status
	}
	s.Equal("destroyed", statuses["res-1"])
	s.Equal("destroyed", statuses["res-2"])
	s.Equal("failed", statuses["res-3"])
	s.Equal("interrupted", statuses["res-4"])
}

// testError is a simple error type for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
