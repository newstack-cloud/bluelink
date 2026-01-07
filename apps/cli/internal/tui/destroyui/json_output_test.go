package destroyui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
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

// --- JSON mode tests through public interface ---

func (s *JSONOutputSuite) Test_json_output_success() {
	jsonOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		jsonResourceEvent("resource-1", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
		jsonResourceEvent("resource-1", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
		jsonResourceEvent("resource-2", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
		jsonResourceEvent("resource-2", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
		jsonFinishEvent(core.InstanceStatusDestroyed),
	}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			&state.InstanceState{InstanceID: "test-instance-id", Status: core.InstanceStatusDestroyed},
		),
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

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

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
}

func (s *JSONOutputSuite) Test_json_output_with_failures() {
	jsonOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		jsonResourceEvent("resource-1", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
		jsonResourceEvent("resource-1", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
		jsonResourceEventFailed("resource-2", core.ResourceStatusDestroyFailed, core.PreciseResourceStatusDestroyFailed, []string{"Resource is in use"}),
		jsonFinishEvent(core.InstanceStatusDestroyFailed),
	}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			&state.InstanceState{InstanceID: "test-instance-id", Status: core.InstanceStatusDestroyFailed},
		),
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

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	// Parse and validate JSON output
	var output jsonout.DestroyOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.True(output.Success) // Still true because operation completed
	s.Equal("DESTROY FAILED", output.Status)
	s.Equal(1, output.Summary.Destroyed)
	s.Equal(1, output.Summary.Failed)
}

func (s *JSONOutputSuite) Test_json_output_with_interrupted() {
	jsonOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		jsonResourceEvent("resource-1", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
		jsonResourceEvent("resource-1", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
		jsonResourceEvent("resource-2", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
		jsonResourceEvent("resource-2", core.ResourceStatusDestroyInterrupted, core.PreciseResourceStatusDestroyInterrupted),
		jsonFinishEvent(core.InstanceStatusDestroyInterrupted),
	}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			&state.InstanceState{InstanceID: "test-instance-id", Status: core.InstanceStatusDestroyInterrupted},
		),
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

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	// Parse and validate JSON output
	var output jsonout.DestroyOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal("DESTROY INTERRUPTED", output.Status)
	s.Equal(1, output.Summary.Destroyed)
	s.Equal(0, output.Summary.Failed)
	s.Equal(1, output.Summary.Interrupted)
}

func (s *JSONOutputSuite) Test_json_output_drift_detected() {
	jsonOutput := &bytes.Buffer{}

	driftError := &engineerrors.ClientError{
		StatusCode: http.StatusConflict,
		Message:    "Drift detected: external changes found",
		DriftBlockedResponse: &types.DriftBlockedResponse{
			Message:     "Drift detected: external changes found",
			InstanceID:  "test-instance-id",
			ChangesetID: "test-changeset-drift",
			ReconciliationResult: &container.ReconciliationCheckResult{
				InstanceID: "test-instance-id",
				Resources: []container.ResourceReconcileResult{
					{
						ResourceID:        "res-1-id",
						ResourceName:      "resource-1",
						ResourceType:      "aws/ec2/instance",
						Type:              container.ReconciliationTypeDrift,
						RecommendedAction: container.ReconciliationActionAcceptExternal,
					},
				},
				HasDrift: true,
			},
		},
	}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDestroyError(driftError),
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

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	// Parse and validate JSON output
	var output jsonout.DestroyDriftOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.True(output.Success)
	s.True(output.DriftDetected)
	s.Equal("test-instance-id", output.InstanceID)
	s.Equal("test-instance", output.InstanceName)
	s.Contains(output.Message, "Drift detected")
	s.NotNil(output.Reconciliation)
	s.Len(output.Reconciliation.Resources, 1)
}

func (s *JSONOutputSuite) Test_json_output_error() {
	jsonOutput := &bytes.Buffer{}

	networkErr := &engineerrors.RequestError{
		Err: &jsonTestNetworkError{message: "failed to connect to deploy engine"},
	}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDestroyError(networkErr),
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

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	// Parse and validate JSON output
	var output jsonout.ErrorOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.False(output.Success)
	s.Contains(output.Error.Message, "failed to connect to deploy engine")
}

// --- Helper functions for creating test events ---

func jsonResourceEvent(name string, status core.ResourceStatus, preciseStatus core.PreciseResourceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			ResourceUpdateEvent: &container.ResourceDeployUpdateMessage{
				ResourceName:  name,
				ResourceID:    "res-" + name,
				Status:        status,
				PreciseStatus: preciseStatus,
			},
		},
	}
}

func jsonResourceEventFailed(name string, status core.ResourceStatus, preciseStatus core.PreciseResourceStatus, reasons []string) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			ResourceUpdateEvent: &container.ResourceDeployUpdateMessage{
				ResourceName:   name,
				ResourceID:     "res-" + name,
				Status:         status,
				PreciseStatus:  preciseStatus,
				FailureReasons: reasons,
			},
		},
	}
}

func jsonFinishEvent(status core.InstanceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			FinishEvent: &container.DeploymentFinishedMessage{
				Status:      status,
				EndOfStream: true,
			},
		},
	}
}

// Helper type for testing
type jsonTestNetworkError struct {
	message string
}

func (e *jsonTestNetworkError) Error() string {
	return e.message
}
