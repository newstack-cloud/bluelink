package inspectui

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type InspectTUISuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestInspectTUISuite(t *testing.T) {
	suite.Run(t, new(InspectTUISuite))
}

func (s *InspectTUISuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// Test helper functions

func testInstanceState(status core.InstanceStatus) *state.InstanceState {
	return &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: "test-instance",
		Status:       status,
	}
}

func testInstanceStateWithResources(status core.InstanceStatus) *state.InstanceState {
	return &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: "test-instance",
		Status:       status,
		Resources: map[string]*state.ResourceState{
			"res-resource-1": {
				ResourceID: "res-resource-1",
				Name:       "resource-1",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
			},
			"res-resource-2": {
				ResourceID: "res-resource-2",
				Name:       "resource-2",
				Type:       "aws/lambda/function",
				Status:     core.ResourceStatusCreated,
			},
		},
		ResourceIDs: map[string]string{
			"resource-1": "res-resource-1",
			"resource-2": "res-resource-2",
		},
	}
}

func testInstanceStateWithChild(status core.InstanceStatus) *state.InstanceState {
	return &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: "test-instance",
		Status:       status,
		ChildBlueprints: map[string]*state.InstanceState{
			"child-blueprint": {
				InstanceID:   "child-instance-id",
				InstanceName: "child-blueprint",
				Status:       core.InstanceStatusDeployed,
			},
		},
	}
}

func testInstanceStateWithLink(status core.InstanceStatus) *state.InstanceState {
	return &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: "test-instance",
		Status:       status,
		Links: map[string]*state.LinkState{
			"resource-a::resource-b": {
				LinkID: "link-123",
				Status: core.LinkStatusCreated,
			},
		},
	}
}

func testDeploymentEvents(finalStatus core.InstanceStatus) []*types.BlueprintInstanceEvent {
	return []*types.BlueprintInstanceEvent{
		resourceEvent("test-resource", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
		resourceEvent("test-resource", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
		finishEvent(finalStatus),
	}
}

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

// --- Static View Tests ---

func (s *InspectTUISuite) Test_inspect_completed_deployment_shows_resources() {
	instanceState := testInstanceStateWithResources(core.InstanceStatusDeployed)

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-1",
		"resource-2",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.Nil(finalModel.err)
	s.NotNil(finalModel.instanceState)
	s.Equal("test-instance-id", finalModel.instanceState.InstanceID)
}

func (s *InspectTUISuite) Test_inspect_completed_deployment_shows_child_blueprints() {
	instanceState := testInstanceStateWithChild(core.InstanceStatusDeployed)

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"child-blueprint",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *InspectTUISuite) Test_inspect_completed_deployment_shows_links() {
	instanceState := testInstanceStateWithLink(core.InstanceStatusDeployed)

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-a::resource-b",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *InspectTUISuite) Test_inspect_failed_deployment_shows_error_status() {
	instanceState := testInstanceState(core.InstanceStatusDeployFailed)

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Deploy Failed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// --- Streaming Mode Tests ---

func (s *InspectTUISuite) Test_inspect_in_progress_deployment_streams_events() {
	instanceState := testInstanceState(core.InstanceStatusDeploying)
	events := testDeploymentEvents(core.InstanceStatusDeployed)

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *InspectTUISuite) Test_inspect_in_progress_destroy_streams_events() {
	instanceState := testInstanceState(core.InstanceStatusDestroying)
	events := []*types.BlueprintInstanceEvent{
		resourceEvent("test-resource", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
		resourceEvent("test-resource", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
		finishEvent(core.InstanceStatusDestroyed),
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *InspectTUISuite) Test_inspect_in_progress_with_child_streams_events() {
	instanceState := testInstanceState(core.InstanceStatusDeploying)
	events := []*types.BlueprintInstanceEvent{
		childEvent("child-blueprint", core.InstanceStatusDeploying),
		childEvent("child-blueprint", core.InstanceStatusDeployed),
		finishEvent(core.InstanceStatusDeployed),
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"child-blueprint",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *InspectTUISuite) Test_inspect_in_progress_with_link_streams_events() {
	instanceState := testInstanceState(core.InstanceStatusDeploying)
	events := []*types.BlueprintInstanceEvent{
		linkEvent("resource-a::resource-b", core.LinkStatusCreating, core.PreciseLinkStatusUpdatingResourceA),
		linkEvent("resource-a::resource-b", core.LinkStatusCreated, core.PreciseLinkStatusResourceBUpdated),
		finishEvent(core.InstanceStatusDeployed),
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-a::resource-b",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// --- Error Cases ---

func (s *InspectTUISuite) Test_inspect_instance_not_found_shows_error() {
	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspectNotFound(),
		zap.NewNop(),
		"non-existent-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceNotFoundMsg{
		Err: errInstanceNotFound("non-existent-id", ""),
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"instance not found",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.NotNil(finalModel.err)
}

// --- Headless Mode Tests ---

func (s *InspectTUISuite) Test_headless_mode_outputs_instance_state() {
	headlessOutput := &bytes.Buffer{}
	instanceState := testInstanceStateWithResources(core.InstanceStatusDeployed)

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		true, // headless
		headlessOutput,
		false,
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

	output := headlessOutput.String()
	s.Contains(output, "Instance ID")
	s.Contains(output, "test-instance-id")
}

func (s *InspectTUISuite) Test_headless_mode_outputs_streaming_events() {
	headlessOutput := &bytes.Buffer{}
	instanceState := testInstanceState(core.InstanceStatusDeploying)
	events := testDeploymentEvents(core.InstanceStatusDeployed)

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		true, // headless
		headlessOutput,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "test-resource")
}

func (s *InspectTUISuite) Test_headless_mode_outputs_error() {
	headlessOutput := &bytes.Buffer{}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspectNotFound(),
		zap.NewNop(),
		"non-existent-id",
		"",
		s.styles,
		true, // headless
		headlessOutput,
		false,
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

	output := headlessOutput.String()
	s.Contains(output, "instance not found")
}

// --- Additional Status Rendering Tests ---

func (s *InspectTUISuite) Test_inspect_resource_failure_shows_create_failed_status() {
	instanceState := testInstanceState(core.InstanceStatusDeploying)
	events := []*types.BlueprintInstanceEvent{
		resourceEvent("failing-resource", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
		resourceEventFailed("failing-resource", core.ResourceStatusCreateFailed, core.PreciseResourceStatusCreateFailed, []string{"Connection timeout"}),
		finishEvent(core.InstanceStatusDeployFailed),
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"failing-resource",
		"Create Failed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.Equal(core.InstanceStatusDeployFailed, finalModel.footerRenderer.CurrentStatus)
}

func (s *InspectTUISuite) Test_inspect_rollback_shows_rolling_back_status() {
	instanceState := testInstanceState(core.InstanceStatusDeploying)
	events := []*types.BlueprintInstanceEvent{
		resourceEvent("resource-1", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
		resourceEvent("resource-1", core.ResourceStatusCreated, core.PreciseResourceStatusCreated),
		resourceEvent("resource-2", core.ResourceStatusCreating, core.PreciseResourceStatusCreating),
		resourceEventFailed("resource-2", core.ResourceStatusCreateFailed, core.PreciseResourceStatusCreateFailed, []string{"Failed to create"}),
		instanceStatusEvent(core.InstanceStatusDeployRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollingBack, core.PreciseResourceStatusCreateRollingBack),
		resourceEvent("resource-1", core.ResourceStatusRollbackComplete, core.PreciseResourceStatusCreateRollbackComplete),
		finishEvent(core.InstanceStatusDeployRollbackComplete),
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})

	// Wait for the rollback to complete - "Rolled Back" indicates the final status
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-1",
		"Rolled Back",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.Equal(core.InstanceStatusDeployRollbackComplete, finalModel.footerRenderer.CurrentStatus)
}

func (s *InspectTUISuite) Test_inspect_update_status_shows_updated() {
	instanceState := testInstanceState(core.InstanceStatusUpdating)
	events := []*types.BlueprintInstanceEvent{
		resourceEvent("test-resource", core.ResourceStatusUpdating, core.PreciseResourceStatusUpdating),
		resourceEvent("test-resource", core.ResourceStatusUpdated, core.PreciseResourceStatusUpdated),
		finishEvent(core.InstanceStatusUpdated),
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"Updated",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InspectModel)
	s.Equal(core.InstanceStatusUpdated, finalModel.footerRenderer.CurrentStatus)
}

func (s *InspectTUISuite) Test_inspect_nested_child_events_during_streaming() {
	instanceState := testInstanceState(core.InstanceStatusDeploying)
	events := []*types.BlueprintInstanceEvent{
		childEvent("parent-child", core.InstanceStatusDeploying),
		nestedChildEvent("parent-child", "nested-child", core.InstanceStatusDeploying),
		nestedChildEvent("parent-child", "nested-child", core.InstanceStatusDeployed),
		childEvent("parent-child", core.InstanceStatusDeployed),
		finishEvent(core.InstanceStatusDeployed),
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"parent-child",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *InspectTUISuite) Test_inspect_link_failure_shows_create_failed() {
	instanceState := testInstanceState(core.InstanceStatusDeploying)
	events := []*types.BlueprintInstanceEvent{
		linkEvent("resource-a::resource-b", core.LinkStatusCreating, core.PreciseLinkStatusUpdatingResourceA),
		linkEventFailed("resource-a::resource-b", core.LinkStatusCreateFailed, core.PreciseLinkStatusResourceAUpdateFailed, []string{"Link failed"}),
		finishEvent(core.InstanceStatusDeployFailed),
	}

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, events),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(InstanceStateFetchedMsg{
		InstanceState: instanceState,
		IsInProgress:  true,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-a::resource-b",
		"Create Failed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// --- Additional Headless Mode Tests ---

func (s *InspectTUISuite) Test_headless_mode_shows_nested_children() {
	headlessOutput := &bytes.Buffer{}
	instanceState := testInstanceStateWithNestedChildren(core.InstanceStatusDeployed)

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		true, // headless
		headlessOutput,
		false,
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

	output := headlessOutput.String()
	s.Contains(output, "child-blueprint")
	s.Contains(output, "nested-child")
}

func (s *InspectTUISuite) Test_headless_mode_shows_link_status() {
	headlessOutput := &bytes.Buffer{}
	instanceState := testInstanceStateWithLink(core.InstanceStatusDeployed)

	model := *NewInspectModel(
		testutils.NewTestDeployEngineForInspect(instanceState, nil),
		zap.NewNop(),
		"test-instance-id",
		"",
		s.styles,
		true, // headless
		headlessOutput,
		false,
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

	output := headlessOutput.String()
	s.Contains(output, "resource-a::resource-b")
	s.Contains(output, "CREATED")
}

// --- Additional Test Helpers ---

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

func linkEventFailed(linkName string, status core.LinkStatus, preciseStatus core.PreciseLinkStatus, reasons []string) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			LinkUpdateEvent: &container.LinkDeployUpdateMessage{
				LinkName:       linkName,
				Status:         status,
				PreciseStatus:  preciseStatus,
				FailureReasons: reasons,
			},
		},
	}
}

func instanceStatusEvent(status core.InstanceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
				Status: status,
			},
		},
	}
}

func nestedChildEvent(parentChild, childName string, status core.InstanceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			ChildUpdateEvent: &container.ChildDeployUpdateMessage{
				ChildName:        childName,
				Status:           status,
				ParentInstanceID: "parent-instance-id",
				ChildInstanceID:  "nested-instance-id",
			},
		},
	}
}

func testInstanceStateWithNestedChildren(status core.InstanceStatus) *state.InstanceState {
	return &state.InstanceState{
		InstanceID:   "test-instance-id",
		InstanceName: "test-instance",
		Status:       status,
		ChildBlueprints: map[string]*state.InstanceState{
			"child-blueprint": {
				InstanceID:   "child-instance-id",
				InstanceName: "child-blueprint",
				Status:       core.InstanceStatusDeployed,
				ChildBlueprints: map[string]*state.InstanceState{
					"nested-child": {
						InstanceID:   "nested-instance-id",
						InstanceName: "nested-child",
						Status:       core.InstanceStatusDeployed,
					},
				},
			},
		},
	}
}
