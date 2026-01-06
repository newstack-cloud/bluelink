package deployui

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DeployJSONOutputTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDeployJSONOutputTestSuite(t *testing.T) {
	suite.Run(t, new(DeployJSONOutputTestSuite))
}

func (s *DeployJSONOutputTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *DeployJSONOutputTestSuite) Test_outputJSON_includes_instance_info() {
	jsonOutput := &bytes.Buffer{}

	instanceState := &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDeployed,
	}

	events := []*types.BlueprintInstanceEvent{
		resourceDeployEvent("resource-1", core.ResourceStatusCreated),
		deployFinishEvent(core.InstanceStatusDeployed),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(events, "test-instance-id", instanceState),
		zap.NewNop(),
		"test-changeset-123",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		jsonOutput,
		nil,
		true, // jsonMode
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.DeployOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.True(output.Success)
	s.Equal("test-instance-id", output.InstanceID)
	s.Equal("test-instance", output.InstanceName)
	s.Equal("test-changeset-123", output.ChangesetID)
}

func (s *DeployJSONOutputTestSuite) Test_outputJSON_includes_deployment_summary() {
	jsonOutput := &bytes.Buffer{}

	instanceState := &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDeployed,
	}

	events := []*types.BlueprintInstanceEvent{
		resourceDeployEvent("resource-1", core.ResourceStatusCreated),
		resourceDeployEvent("resource-2", core.ResourceStatusUpdated),
		resourceDeployEventFailed("resource-3", core.ResourceStatusCreateFailed, []string{"timeout"}),
		deployFinishEvent(core.InstanceStatusDeployFailed),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(events, "test-instance-id", instanceState),
		zap.NewNop(),
		"test-changeset-123",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		jsonOutput,
		nil,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.DeployOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal(2, output.Summary.Successful)
	s.Equal(1, output.Summary.Failed)
}

func (s *DeployJSONOutputTestSuite) Test_outputJSON_includes_instance_state() {
	jsonOutput := &bytes.Buffer{}

	instanceState := &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "myResource",
				Type:       "aws/s3/bucket",
			},
		},
	}

	events := []*types.BlueprintInstanceEvent{
		resourceDeployEvent("myResource", core.ResourceStatusCreated),
		deployFinishEvent(core.InstanceStatusDeployed),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(events, "test-instance-id", instanceState),
		zap.NewNop(),
		"test-changeset-123",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		jsonOutput,
		nil,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.DeployOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.NotNil(output.InstanceState)
	s.Equal("test-instance-id", output.InstanceState.InstanceID)
	s.Len(output.InstanceState.Resources, 1)
}

func (s *DeployJSONOutputTestSuite) Test_outputJSONError_formats_validation_errors() {
	jsonOutput := &bytes.Buffer{}

	validationErr := &engineerrors.ClientError{
		Message:    "Validation failed",
		StatusCode: 400,
		ValidationDiagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "Name is required",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{Position: source.Position{Line: 10, Column: 5}},
				},
			},
		},
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeploymentError(validationErr),
		zap.NewNop(),
		"test-changeset-123",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		jsonOutput,
		nil,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.ErrorOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.False(output.Success)
	s.Equal("validation", output.Error.Type)
	s.Equal(400, output.Error.StatusCode)
	s.Len(output.Error.Diagnostics, 1)
	s.Equal("Name is required", output.Error.Diagnostics[0].Message)
}

func (s *DeployJSONOutputTestSuite) Test_outputJSONDrift_includes_reconciliation_result() {
	jsonOutput := &bytes.Buffer{}

	driftErr := &engineerrors.ClientError{
		StatusCode: 409,
		Message:    "Drift detected in resource state",
		DriftBlockedResponse: &types.DriftBlockedResponse{
			Message:     "Drift detected in resource state",
			InstanceID:  "test-instance-id",
			ChangesetID: "blocked-changeset-123",
			ReconciliationResult: &container.ReconciliationCheckResult{
				InstanceID: "test-instance-id",
				Resources: []container.ResourceReconcileResult{
					{
						ResourceName: "drifted-resource",
						ResourceType: "aws/sqs/queue",
						Type:         container.ReconciliationTypeDrift,
					},
				},
			},
		},
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeploymentError(driftErr),
		zap.NewNop(),
		"test-changeset-123",
		"test-instance-id",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		jsonOutput,
		nil,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.DeployDriftOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.True(output.Success)
	s.True(output.DriftDetected)
	s.Equal("test-instance-id", output.InstanceID)
	s.Equal("test-instance", output.InstanceName)
	s.Equal("Drift detected in resource state", output.Message)
	s.NotNil(output.Reconciliation)
	s.Len(output.Reconciliation.Resources, 1)
	s.Equal("drifted-resource", output.Reconciliation.Resources[0].ResourceName)
}

// Helper functions to create deployment events

func resourceDeployEvent(resourceName string, status core.ResourceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			ResourceUpdateEvent: &container.ResourceDeployUpdateMessage{
				ResourceName:  resourceName,
				ResourceID:    "res-" + resourceName,
				Status:        status,
				PreciseStatus: core.PreciseResourceStatusCreated,
			},
		},
	}
}

func resourceDeployEventFailed(
	resourceName string,
	status core.ResourceStatus,
	failureReasons []string,
) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			ResourceUpdateEvent: &container.ResourceDeployUpdateMessage{
				ResourceName:   resourceName,
				ResourceID:     "res-" + resourceName,
				Status:         status,
				PreciseStatus:  core.PreciseResourceStatusCreateFailed,
				FailureReasons: failureReasons,
			},
		},
	}
}

func deployFinishEvent(status core.InstanceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			FinishEvent: &container.DeploymentFinishedMessage{
				Status:      status,
				EndOfStream: true,
			},
		},
	}
}
