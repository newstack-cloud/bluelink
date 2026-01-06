package inspectui

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type InspectJSONOutputTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestInspectJSONOutputTestSuite(t *testing.T) {
	suite.Run(t, new(InspectJSONOutputTestSuite))
}

func (s *InspectJSONOutputTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func strPtr(s string) *string {
	return &s
}

func (s *InspectJSONOutputTestSuite) Test_outputJSON_outputs_raw_instance_state() {
	jsonOutput := &bytes.Buffer{}

	instanceState := &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-1": {
				ResourceID: "res-1",
				Name:       "myResource",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
			},
		},
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		true, // headless
		jsonOutput,
		true, // jsonMode
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	// The JSON output should be the raw InstanceState
	var output state.InstanceState
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal("test-instance-id", output.InstanceID)
	s.Equal("test-instance", output.InstanceName)
	s.Equal(core.InstanceStatusDeployed, output.Status)
	s.Len(output.Resources, 1)
	s.NotNil(output.Resources["res-1"])
	s.Equal("myResource", output.Resources["res-1"].Name)
}

func (s *InspectJSONOutputTestSuite) Test_outputJSON_includes_child_blueprints() {
	jsonOutput := &bytes.Buffer{}

	instanceState := &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		ChildBlueprints: map[string]*state.InstanceState{
			"child-1": {
				InstanceID:   "child-instance-id",
				InstanceName: "child-blueprint",
				Status:       core.InstanceStatusDeployed,
			},
		},
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output state.InstanceState
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Len(output.ChildBlueprints, 1)
	s.NotNil(output.ChildBlueprints["child-1"])
	s.Equal("child-instance-id", output.ChildBlueprints["child-1"].InstanceID)
}

func (s *InspectJSONOutputTestSuite) Test_outputJSON_includes_links() {
	jsonOutput := &bytes.Buffer{}

	instanceState := &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Links: map[string]*state.LinkState{
			"resource-a::resource-b": {
				LinkID: "link-123",
				Status: core.LinkStatusCreated,
			},
		},
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output state.InstanceState
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Len(output.Links, 1)
	s.NotNil(output.Links["resource-a::resource-b"])
	s.Equal("link-123", output.Links["resource-a::resource-b"].LinkID)
}

func (s *InspectJSONOutputTestSuite) Test_outputJSONError_for_not_found() {
	jsonOutput := &bytes.Buffer{}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspectNotFound(),
		zap.NewNop(),
		"non-existent-id",
		"",
		s.styles,
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceNotFoundMsg{
		Err: errInstanceNotFound("non-existent-id", ""),
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.ErrorOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.False(output.Success)
	s.Contains(output.Error.Message, "instance not found")
}

func (s *InspectJSONOutputTestSuite) Test_outputJSON_preserves_exports() {
	jsonOutput := &bytes.Buffer{}

	instanceState := &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: "test-instance",
		Status:       core.InstanceStatusDeployed,
		Exports: map[string]*state.ExportState{
			"api_endpoint": {
				Value: &core.MappingNode{
					Scalar: &core.ScalarValue{StringValue: strPtr("https://api.example.com")},
				},
				Type:  "string",
				Field: "endpoint",
			},
		},
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  false,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output state.InstanceState
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Len(output.Exports, 1)
	s.NotNil(output.Exports["api_endpoint"])
	s.NotNil(output.Exports["api_endpoint"].Value)
}

func (s *InspectJSONOutputTestSuite) Test_outputJSON_includes_status_variations() {
	testCases := []struct {
		name   string
		status core.InstanceStatus
	}{
		{"deployed", core.InstanceStatusDeployed},
		{"deploy_failed", core.InstanceStatusDeployFailed},
		{"updated", core.InstanceStatusUpdated},
		{"update_failed", core.InstanceStatusUpdateFailed},
		{"destroyed", core.InstanceStatusDestroyed},
		{"destroy_failed", core.InstanceStatusDestroyFailed},
		{"rollback_complete", core.InstanceStatusDeployRollbackComplete},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			jsonOutput := &bytes.Buffer{}

			instanceState := &state.InstanceState{
				InstanceID:   "test-instance-id",
				InstanceName: "test-instance",
				Status:       tc.status,
			}

			model := *NewInspectModel(
				testutils.NewTestDeployEngineForInspect(instanceState, nil),
				zap.NewNop(),
				"test-instance-id",
				"",
				s.styles,
				true,
				jsonOutput,
				true,
			)

			testModel := teatest.NewTestModel(
				s.T(),
				model,
				teatest.WithInitialTermSize(300, 100),
			)

			testModel.Send(InstanceStateFetchedMsg{
				InstanceState: instanceState,
				IsInProgress:  false,
			})
			testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

			var output state.InstanceState
			err := json.Unmarshal(jsonOutput.Bytes(), &output)
			s.Require().NoError(err)
			s.Equal(tc.status, output.Status)
		})
	}
}
