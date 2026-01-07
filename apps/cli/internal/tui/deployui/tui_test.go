package deployui

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DeployTUISuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDeployTUISuite(t *testing.T) {
	suite.Run(t, new(DeployTUISuite))
}

func (s *DeployTUISuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// --- Test Event Factories ---

type testDeployType string

const (
	deploySuccessCreate   testDeployType = "success_create"
	deploySuccessUpdate   testDeployType = "success_update"
	deployFailure         testDeployType = "failure"
	deployRollback        testDeployType = "rollback"
	deployInterrupted     testDeployType = "interrupted"
	deployWithChild       testDeployType = "with_child"
	deployWithLink        testDeployType = "with_link"
	deployMultipleSuccess testDeployType = "multiple_success"
)

func testDeployEvents(deployType testDeployType) []*types.BlueprintInstanceEvent {
	switch deployType {
	case deploySuccessCreate:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("test-resource", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			resourceEvent("test-resource", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
			finishEvent(core.InstanceStatusDeployed),
		}
	case deploySuccessUpdate:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("test-resource", core.ResourceStatusUpdating, core.PreciseResourceStatusUpdating),
			resourceEvent("test-resource", core.ResourceStatusUpdated, core.PreciseResourceStatusUpdated),
			finishEvent(core.InstanceStatusUpdated),
		}
	case deployFailure:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("test-resource", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			resourceEventFailed("test-resource", core.ResourceStatusCreateFailed, core.PreciseResourceStatusCreateFailed, []string{"Connection timeout", "Retry limit exceeded"}),
			finishEvent(core.InstanceStatusDeployFailed),
		}
	case deployRollback:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("resource-1", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			resourceEvent("resource-1", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
			resourceEvent("resource-2", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			resourceEventFailed("resource-2", core.ResourceStatusCreateFailed, core.PreciseResourceStatusCreateFailed, []string{"Failed to create"}),
			deploymentStatusEvent(core.InstanceStatusDeployRollingBack),
			resourceEvent("resource-1", core.ResourceStatusRollingBack, core.PreciseResourceStatusCreateRollingBack),
			resourceEvent("resource-1", core.ResourceStatusRollbackComplete, core.PreciseResourceStatusCreateRollbackComplete),
			finishEvent(core.InstanceStatusDeployRollbackComplete),
		}
	case deployInterrupted:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("test-resource", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			resourceEvent("test-resource", core.ResourceStatusCreateInterrupted, core.PreciseResourceStatusCreateInterrupted),
			finishEvent(core.InstanceStatusDeployInterrupted),
		}
	case deployWithChild:
		return []*types.BlueprintInstanceEvent{
			childEvent("child-blueprint", core.InstanceStatusDeploying),
			childEvent("child-blueprint", core.InstanceStatusDeployed),
			finishEvent(core.InstanceStatusDeployed),
		}
	case deployWithLink:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("resource-a", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
			resourceEvent("resource-b", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
			linkEvent("resource-a::resource-b", core.LinkStatusCreating, core.PreciseLinkStatusUpdatingResourceA),
			linkEvent("resource-a::resource-b", core.LinkStatusCreated, core.PreciseLinkStatusResourceBUpdated),
			finishEvent(core.InstanceStatusDeployed),
		}
	case deployMultipleSuccess:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("resource-1", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			resourceEvent("resource-1", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
			resourceEvent("resource-2", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
			resourceEvent("resource-2", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
			resourceEvent("resource-3", core.ResourceStatusUpdating, core.PreciseResourceStatusUpdating),
			resourceEvent("resource-3", core.ResourceStatusUpdated, core.PreciseResourceStatusUpdated),
			finishEvent(core.InstanceStatusDeployed),
		}
	default:
		return []*types.BlueprintInstanceEvent{finishEvent(core.InstanceStatusDeployed)}
	}
}

func testInstanceState(status core.InstanceStatus) *state.InstanceState {
	return &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     status,
	}
}

// Event factory functions

func resourceEvent(name string, status core.ResourceStatus, preciseStatus core.PreciseResourceStatus) *types.BlueprintInstanceEvent {
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

func resourceEventFailed(name string, status core.ResourceStatus, preciseStatus core.PreciseResourceStatus, reasons []string) *types.BlueprintInstanceEvent {
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

func childEvent(name string, status core.InstanceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			ChildUpdateEvent: &container.ChildDeployUpdateMessage{
				ChildName: name,
				Status:    status,
			},
		},
	}
}

func linkEvent(linkName string, status core.LinkStatus, preciseStatus core.PreciseLinkStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			LinkUpdateEvent: &container.LinkDeployUpdateMessage{
				LinkName:      linkName,
				Status:        status,
				PreciseStatus: preciseStatus,
			},
		},
	}
}

func deploymentStatusEvent(status core.InstanceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
				Status: status,
			},
		},
	}
}

func finishEvent(status core.InstanceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			FinishEvent: &container.DeploymentFinishedMessage{
				Status:      status,
				EndOfStream: true,
			},
		},
	}
}

// --- Interactive Mode Tests ---

func (s *DeployTUISuite) Test_successful_deployment_with_resource_create() {
	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deploySuccessCreate),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		zap.NewNop(),
		"test-changeset-123",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		false, // headless
		os.Stdout,
		nil,
		false, // jsonMode
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"Created",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DeployModel)
	s.Nil(finalModel.Err())
	s.Equal(core.InstanceStatusDeployed, finalModel.FinalStatus())
}

func (s *DeployTUISuite) Test_successful_deployment_with_resource_update() {
	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deploySuccessUpdate),
			"test-instance-id",
			testInstanceState(core.InstanceStatusUpdated),
		),
		zap.NewNop(),
		"test-changeset-456",
		"existing-instance-id",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"Updated",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DeployModel)
	s.Nil(finalModel.Err())
	s.Equal(core.InstanceStatusUpdated, finalModel.FinalStatus())
}

func (s *DeployTUISuite) Test_deployment_failure_shows_error() {
	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployFailure),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployFailed),
		),
		zap.NewNop(),
		"test-changeset-fail",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"Create Failed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DeployModel)
	s.Equal(core.InstanceStatusDeployFailed, finalModel.FinalStatus())
}

func (s *DeployTUISuite) Test_deployment_rollback_sets_final_status() {
	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployRollback),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployRollbackComplete),
		),
		zap.NewNop(),
		"test-changeset-rollback",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	// Wait for the rollback to complete - this ensures the finish event has been processed
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-1",
		"Deployment rolled back",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DeployModel)
	s.Equal(core.InstanceStatusDeployRollbackComplete, finalModel.FinalStatus())
	s.Equal(core.ResourceStatusRollbackComplete, finalModel.ResourcesByName()["resource-1"].Status)
}

func (s *DeployTUISuite) Test_deployment_with_child_blueprints() {
	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployWithChild),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		zap.NewNop(),
		"test-changeset-child",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"child-blueprint",
		"Deployed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DeployModel)
	s.Equal(core.InstanceStatusDeployed, finalModel.FinalStatus())
}

func (s *DeployTUISuite) Test_deployment_with_links() {
	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployWithLink),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		zap.NewNop(),
		"test-changeset-link",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-a::resource-b",
		"Created",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DeployModel)
	s.Equal(core.InstanceStatusDeployed, finalModel.FinalStatus())
}

func (s *DeployTUISuite) Test_deployment_with_multiple_resources() {
	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployMultipleSuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		zap.NewNop(),
		"test-changeset-multi",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	// Wait for deployment to complete - "complete" appears in footer when finished
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-1",
		"resource-2",
		"resource-3",
		"complete", // deployment finished
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DeployModel)
	s.Equal(core.InstanceStatusDeployed, finalModel.FinalStatus())
}

// --- Headless Mode Tests ---

func (s *DeployTUISuite) Test_headless_mode_outputs_deployment_progress() {
	headlessOutput := &bytes.Buffer{}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deploySuccessCreate),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		zap.NewNop(),
		"test-changeset-headless",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true, // headless
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "test-resource")
	s.Contains(output, "Deployment completed")
	s.Contains(output, "test-instance-id")
}

func (s *DeployTUISuite) Test_headless_mode_shows_failure_details() {
	headlessOutput := &bytes.Buffer{}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployFailure),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployFailed),
		),
		zap.NewNop(),
		"test-changeset-fail",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "test-resource")
	s.Contains(output, "create failed")
}

// --- Changeset Integration Tests ---

func (s *DeployTUISuite) Test_deployment_uses_changeset_for_initial_items() {
	changesetChanges := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"new-resource": {},
		},
		ResourceChanges: map[string]provider.Changes{
			"changed-resource": {},
		},
	}

	events := []*types.BlueprintInstanceEvent{
		resourceEvent("new-resource", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
		resourceEvent("changed-resource", core.ResourceStatusUpdated, core.PreciseResourceStatusUpdated),
		finishEvent(core.InstanceStatusDeployed),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(events, "test-instance-id", testInstanceState(core.InstanceStatusDeployed)),
		zap.NewNop(),
		"test-changeset",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		changesetChanges,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"new-resource",
		"changed-resource",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DeployModel)
	s.Contains(finalModel.ResourcesByName(), "new-resource")
	s.Contains(finalModel.ResourcesByName(), "changed-resource")
}

// --- Headless Mode Tests with Children and Links ---

func (s *DeployTUISuite) Test_headless_mode_outputs_child_and_link_events() {
	headlessOutput := &bytes.Buffer{}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployWithChild),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		zap.NewNop(),
		"test-changeset-child",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true, // headless
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "child-blueprint")
	s.Contains(output, "deployed")
	s.Contains(output, "Deployment completed")
}

func (s *DeployTUISuite) Test_headless_mode_outputs_link_events() {
	headlessOutput := &bytes.Buffer{}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployWithLink),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		zap.NewNop(),
		"test-changeset-link",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true, // headless
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "resource-a::resource-b")
	s.Contains(output, "created")
}

func (s *DeployTUISuite) Test_headless_mode_outputs_exports() {
	headlessOutput := &bytes.Buffer{}

	instanceStateWithExports := &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDeployed,
		Exports: map[string]*state.ExportState{
			"apiEndpoint": {
				Value: core.MappingNodeFromString("https://api.example.com"),
				Type:  "string",
				Field: "api.endpoint",
			},
		},
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deploySuccessCreate),
			"test-instance-id",
			instanceStateWithExports,
		),
		zap.NewNop(),
		"test-changeset-exports",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true, // headless
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "apiEndpoint")
	s.Contains(output, "exports")
}

func (s *DeployTUISuite) Test_headless_mode_outputs_nested_child_exports() {
	headlessOutput := &bytes.Buffer{}

	instanceStateWithNestedExports := &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDeployed,
		ChildBlueprints: map[string]*state.InstanceState{
			"child-blueprint": {
				InstanceID: "child-instance-id",
				Status:     core.InstanceStatusDeployed,
				Exports: map[string]*state.ExportState{
					"childExport": {
						Value: core.MappingNodeFromString("child-value"),
						Type:  "string",
					},
				},
			},
		},
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployWithChild),
			"test-instance-id",
			instanceStateWithNestedExports,
		),
		zap.NewNop(),
		"test-changeset-nested",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true, // headless
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "child-blueprint")
	s.Contains(output, "childExport")
}

func (s *DeployTUISuite) Test_headless_mode_outputs_rollback_status() {
	headlessOutput := &bytes.Buffer{}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployRollback),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployRollbackComplete),
		),
		zap.NewNop(),
		"test-changeset-rollback",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true, // headless
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Deployment rolled back")
}

func (s *DeployTUISuite) Test_headless_mode_outputs_resource_with_durations() {
	headlessOutput := &bytes.Buffer{}

	configDuration := float64(1500)
	totalDuration := float64(3000)
	instanceStateWithDurations := &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDeployed,
		ResourceIDs: map[string]string{
			"test-resource": "res-test-resource",
		},
		Resources: map[string]*state.ResourceState{
			"res-test-resource": {
				ResourceID: "res-test-resource",
				Name:       "test-resource",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
				Durations: &state.ResourceCompletionDurations{
					ConfigCompleteDuration: &configDuration,
					TotalDuration:          &totalDuration,
				},
			},
		},
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deploySuccessCreate),
			"test-instance-id",
			instanceStateWithDurations,
		),
		zap.NewNop(),
		"test-changeset-durations",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true, // headless
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "test-resource")
	s.Contains(output, "Deployment completed")
}

// --- Additional Headless Mode Tests for Edge Cases ---

func (s *DeployTUISuite) Test_headless_mode_outputs_header_info() {
	headlessOutput := &bytes.Buffer{}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deploySuccessCreate),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployed),
		),
		zap.NewNop(),
		"test-changeset-header",
		"",
		"my-test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true, // headless
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Starting deployment")
	s.Contains(output, "Instance ID:")
	s.Contains(output, "Instance Name: my-test-instance")
	s.Contains(output, "Changeset: test-changeset-header")
}

func (s *DeployTUISuite) Test_headless_mode_update_rollback_complete() {
	headlessOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		resourceEvent("resource-1", core.ResourceStatusUpdating, core.PreciseResourceStatusUpdating),
		resourceEventFailed("resource-1", core.ResourceStatusUpdateFailed, core.PreciseResourceStatusUpdateFailed, []string{"Update failed"}),
		deploymentStatusEvent(core.InstanceStatusUpdateRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollingBack, core.PreciseResourceStatusUpdateRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollbackComplete, core.PreciseResourceStatusUpdateRollbackComplete),
		finishEvent(core.InstanceStatusUpdateRollbackComplete),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			testInstanceState(core.InstanceStatusUpdateRollbackComplete),
		),
		zap.NewNop(),
		"test-changeset-update-rollback",
		"existing-id",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Update rolled back")
}

func (s *DeployTUISuite) Test_headless_mode_destroy_rollback_complete() {
	headlessOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		resourceEvent("resource-1", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
		resourceEventFailed("resource-1", core.ResourceStatusDestroyFailed, core.PreciseResourceStatusDestroyFailed, []string{"Destroy failed"}),
		deploymentStatusEvent(core.InstanceStatusDestroyRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollingBack, core.PreciseResourceStatusDestroyRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollbackComplete, core.PreciseResourceStatusDestroyRollbackComplete),
		finishEvent(core.InstanceStatusDestroyRollbackComplete),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyRollbackComplete),
		),
		zap.NewNop(),
		"test-changeset-destroy-rollback",
		"existing-id",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Destroy rolled back")
}

func (s *DeployTUISuite) Test_headless_mode_rollback_failed() {
	headlessOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		resourceEvent("resource-1", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
		resourceEventFailed("resource-1", core.ResourceStatusCreateFailed, core.PreciseResourceStatusCreateFailed, []string{"Create failed"}),
		deploymentStatusEvent(core.InstanceStatusDeployRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollingBack, core.PreciseResourceStatusCreateRollingBack),
		resourceEventFailed("resource-1", core.ResourceStatusRollbackFailed, core.PreciseResourceStatusCreateRollbackFailed, []string{"Rollback failed"}),
		finishEvent(core.InstanceStatusDeployRollbackFailed),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployRollbackFailed),
		),
		zap.NewNop(),
		"test-changeset-rollback-failed",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Deployment rollback failed")
}

func (s *DeployTUISuite) Test_headless_mode_update_failed() {
	headlessOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		resourceEvent("resource-1", core.ResourceStatusUpdating, core.PreciseResourceStatusUpdating),
		resourceEventFailed("resource-1", core.ResourceStatusUpdateFailed, core.PreciseResourceStatusUpdateFailed, []string{"Update failed"}),
		finishEvent(core.InstanceStatusUpdateFailed),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			testInstanceState(core.InstanceStatusUpdateFailed),
		),
		zap.NewNop(),
		"test-changeset-update-failed",
		"existing-id",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Update failed")
}

func (s *DeployTUISuite) Test_headless_mode_destroy_failed() {
	headlessOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		resourceEvent("resource-1", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
		resourceEventFailed("resource-1", core.ResourceStatusDestroyFailed, core.PreciseResourceStatusDestroyFailed, []string{"Destroy failed"}),
		finishEvent(core.InstanceStatusDestroyFailed),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyFailed),
		),
		zap.NewNop(),
		"test-changeset-destroy-failed",
		"existing-id",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Destroy failed")
}

func (s *DeployTUISuite) Test_headless_mode_destroyed_status() {
	headlessOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		resourceEvent("resource-1", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
		resourceEvent("resource-1", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
		finishEvent(core.InstanceStatusDestroyed),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-destroyed",
		"existing-id",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Destroy completed")
}

func (s *DeployTUISuite) Test_headless_mode_resource_with_outputs() {
	headlessOutput := &bytes.Buffer{}

	bucketArn := "arn:aws:s3:::my-bucket"
	instanceStateWithOutputs := &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDeployed,
		ResourceIDs: map[string]string{
			"test-resource": "res-test-resource",
		},
		Resources: map[string]*state.ResourceState{
			"res-test-resource": {
				ResourceID: "res-test-resource",
				Name:       "test-resource",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"bucketArn": {Scalar: &core.ScalarValue{StringValue: &bucketArn}},
					},
				},
				ComputedFields: []string{"bucketArn"},
			},
		},
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deploySuccessCreate),
			"test-instance-id",
			instanceStateWithOutputs,
		),
		zap.NewNop(),
		"test-changeset-outputs",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "test-resource")
	s.Contains(output, "Deployment completed")
}

func (s *DeployTUISuite) Test_headless_mode_interrupted() {
	headlessOutput := &bytes.Buffer{}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deployInterrupted),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployInterrupted),
		),
		zap.NewNop(),
		"test-changeset-interrupted",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "interrupted")
}

func (s *DeployTUISuite) Test_headless_mode_with_pre_rollback_state() {
	headlessOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		resourceEvent("resource-1", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
		resourceEvent("resource-1", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
		resourceEvent("resource-2", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
		resourceEventFailed("resource-2", core.ResourceStatusCreateFailed, core.PreciseResourceStatusCreateFailed, []string{"Failed to create"}),
		{
			DeployEvent: container.DeployEvent{
				PreRollbackStateEvent: &container.PreRollbackStateMessage{
					InstanceID:     "test-instance-id",
					InstanceName:   "test-instance",
					Status:         core.InstanceStatusDeployFailed,
					FailureReasons: []string{"resource-2 failed to create"},
					Resources: []container.ResourceSnapshot{
						{
							ResourceName: "resource-1",
							ResourceType: "aws/s3/bucket",
							Status:       core.ResourceStatusCreated,
						},
						{
							ResourceName: "resource-2",
							ResourceType: "aws/lambda/function",
							Status:       core.ResourceStatusCreateFailed,
						},
					},
				},
			},
		},
		deploymentStatusEvent(core.InstanceStatusDeployRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollingBack, core.PreciseResourceStatusCreateRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollbackComplete, core.PreciseResourceStatusCreateRollbackComplete),
		finishEvent(core.InstanceStatusDeployRollbackComplete),
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployRollbackComplete),
		),
		zap.NewNop(),
		"test-changeset-pre-rollback",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Pre-Rollback State Captured")
	s.Contains(output, "resource-1")
	s.Contains(output, "resource-2")
	s.Contains(output, "Auto-rollback is starting")
}

func (s *DeployTUISuite) Test_headless_mode_with_skipped_rollback_items() {
	headlessOutput := &bytes.Buffer{}

	events := []*types.BlueprintInstanceEvent{
		resourceEvent("resource-1", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
		resourceEvent("resource-1", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
		resourceEvent("resource-2", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
		resourceEventFailed("resource-2", core.ResourceStatusCreateFailed, core.PreciseResourceStatusCreateFailed, []string{"Failed to create"}),
		deploymentStatusEvent(core.InstanceStatusDeployRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollingBack, core.PreciseResourceStatusCreateRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollbackComplete, core.PreciseResourceStatusCreateRollbackComplete),
		// Finish event with skipped rollback items
		{
			DeployEvent: container.DeployEvent{
				FinishEvent: &container.DeploymentFinishedMessage{
					Status:      core.InstanceStatusDeployRollbackComplete,
					EndOfStream: true,
					SkippedRollbackItems: []container.SkippedRollbackItem{
						{
							Name:      "resource-2",
							Type:      "resource",
							ChildPath: "",
							Status:    "create_failed",
							Reason:    "resource was in failed state",
						},
					},
				},
			},
		},
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			events,
			"test-instance-id",
			testInstanceState(core.InstanceStatusDeployRollbackComplete),
		),
		zap.NewNop(),
		"test-changeset-skipped",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Skipped Rollback Items")
	s.Contains(output, "resource-2")
	s.Contains(output, "resource was in failed state")
}

func (s *DeployTUISuite) Test_headless_mode_export_with_all_fields() {
	headlessOutput := &bytes.Buffer{}

	instanceStateWithExportFields := &state.InstanceState{
		InstanceID: "test-instance-id",
		Status:     core.InstanceStatusDeployed,
		Exports: map[string]*state.ExportState{
			"apiEndpoint": {
				Value:       core.MappingNodeFromString("https://api.example.com"),
				Type:        "string",
				Field:       "api.endpoint",
				Description: "The API endpoint URL",
			},
		},
	}

	model := NewDeployModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDeployEvents(deploySuccessCreate),
			"test-instance-id",
			instanceStateWithExportFields,
		),
		zap.NewNop(),
		"test-changeset-export-fields",
		"",
		"test-instance",
		"test.blueprint.yaml",
		"",
		false,
		false,
		s.styles,
		true,
		headlessOutput,
		nil,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDeployMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "apiEndpoint")
	s.Contains(output, "string")
	s.Contains(output, "https://api.example.com")
}
