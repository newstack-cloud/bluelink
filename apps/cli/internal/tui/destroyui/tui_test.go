package destroyui

import (
	"bytes"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
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

type DestroyTUISuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDestroyTUISuite(t *testing.T) {
	suite.Run(t, new(DestroyTUISuite))
}

func (s *DestroyTUISuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// --- Test Event Factories ---

type testDestroyType string

const (
	destroySuccess     testDestroyType = "success"
	destroyFailure     testDestroyType = "failure"
	destroyRollback    testDestroyType = "rollback"
	destroyInterrupted testDestroyType = "interrupted"
	destroyWithChild   testDestroyType = "with_child"
	destroyWithLink    testDestroyType = "with_link"
	destroyMultiple    testDestroyType = "multiple"
)

func testDestroyEvents(destroyType testDestroyType) []*types.BlueprintInstanceEvent {
	switch destroyType {
	case destroySuccess:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("test-resource", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
			resourceEvent("test-resource", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
			finishEvent(core.InstanceStatusDestroyed),
		}
	case destroyFailure:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("test-resource", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
			resourceEventFailed("test-resource", core.ResourceStatusDestroyFailed, core.PreciseResourceStatusDestroyFailed, []string{"Connection timeout", "Resource in use"}),
			finishEvent(core.InstanceStatusDestroyFailed),
		}
	case destroyRollback:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("resource-1", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
			resourceEvent("resource-1", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
			resourceEvent("resource-2", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
			resourceEventFailed("resource-2", core.ResourceStatusDestroyFailed, core.PreciseResourceStatusDestroyFailed, []string{"Failed to destroy"}),
			deploymentStatusEvent(core.InstanceStatusDestroyRollingBack),
			resourceEvent("resource-1", core.ResourceStatusRollingBack, core.PreciseResourceStatusDestroyRollingBack),
			resourceEvent("resource-1", core.ResourceStatusRollbackComplete, core.PreciseResourceStatusDestroyRollbackComplete),
			finishEvent(core.InstanceStatusDestroyRollbackComplete),
		}
	case destroyInterrupted:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("test-resource", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
			resourceEvent("test-resource", core.ResourceStatusDestroyInterrupted, core.PreciseResourceStatusDestroyInterrupted),
			finishEvent(core.InstanceStatusDestroyInterrupted),
		}
	case destroyWithChild:
		return []*types.BlueprintInstanceEvent{
			childEvent("child-blueprint", core.InstanceStatusDestroying),
			childEvent("child-blueprint", core.InstanceStatusDestroyed),
			finishEvent(core.InstanceStatusDestroyed),
		}
	case destroyWithLink:
		return []*types.BlueprintInstanceEvent{
			linkEvent("resource-a::resource-b", core.LinkStatusDestroying, core.PreciseLinkStatusUpdatingResourceA),
			linkEvent("resource-a::resource-b", core.LinkStatusDestroyed, core.PreciseLinkStatusResourceBUpdated),
			resourceEvent("resource-a", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
			resourceEvent("resource-b", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
			finishEvent(core.InstanceStatusDestroyed),
		}
	case destroyMultiple:
		return []*types.BlueprintInstanceEvent{
			resourceEvent("resource-1", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
			resourceEvent("resource-1", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
			resourceEvent("resource-2", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
			resourceEvent("resource-2", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
			resourceEvent("resource-3", core.ResourceStatusDestroying, core.PreciseResourceStatusDestroying),
			resourceEvent("resource-3", core.ResourceStatusDestroyed, core.PreciseResourceStatusDestroyed),
			finishEvent(core.InstanceStatusDestroyed),
		}
	default:
		return []*types.BlueprintInstanceEvent{finishEvent(core.InstanceStatusDestroyed)}
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

func (s *DestroyTUISuite) Test_successful_destroy_single_resource() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-123",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	// Wait for destroy to complete - "complete" appears in footer when finished
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"Destroyed",
		"complete", // destroy finished
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.Nil(finalModel.err)
	s.Equal(core.InstanceStatusDestroyed, finalModel.finalStatus)
}

func (s *DestroyTUISuite) Test_successful_destroy_multiple_resources() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyMultiple),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-multi",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	// Wait for destroy to complete - "complete" appears in footer when finished
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-1",
		"resource-2",
		"resource-3",
		"complete", // destroy finished
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.Nil(finalModel.err)
	s.Equal(core.InstanceStatusDestroyed, finalModel.finalStatus)
}

func (s *DestroyTUISuite) Test_destroy_with_child_blueprints() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyWithChild),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-child",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"child-blueprint",
		"Destroyed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.Equal(core.InstanceStatusDestroyed, finalModel.finalStatus)
}

func (s *DestroyTUISuite) Test_destroy_with_links() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyWithLink),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-link",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	// Wait for destroy to complete - "complete" appears in footer when finished
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-a::resource-b",
		"Destroyed",
		"complete", // destroy finished
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.Equal(core.InstanceStatusDestroyed, finalModel.finalStatus)
}

func (s *DestroyTUISuite) Test_destroy_failure_shows_error() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyFailure),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyFailed),
		),
		zap.NewNop(),
		"test-changeset-fail",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"Destroy Failed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.Equal(core.InstanceStatusDestroyFailed, finalModel.finalStatus)
}

func (s *DestroyTUISuite) Test_destroy_rollback_sets_final_status() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyRollback),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyRollbackComplete),
		),
		zap.NewNop(),
		"test-changeset-rollback",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-1",
		"Destroy rolled back",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.Equal(core.InstanceStatusDestroyRollbackComplete, finalModel.finalStatus)
	s.Equal(core.ResourceStatusRollbackComplete, finalModel.resourcesByName["resource-1"].Status)
}

func (s *DestroyTUISuite) Test_destroy_interrupted() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyInterrupted),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyInterrupted),
		),
		zap.NewNop(),
		"test-changeset-interrupted",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"Interrupted",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.Equal(core.InstanceStatusDestroyInterrupted, finalModel.finalStatus)
}

func (s *DestroyTUISuite) Test_quit_with_q() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-quit",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// --- Headless Mode Tests ---

func (s *DestroyTUISuite) Test_headless_mode_outputs_progress() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-headless",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "test-resource")
	s.Contains(output, "Destroy completed")
	s.Contains(output, "test-instance-id")
}

func (s *DestroyTUISuite) Test_headless_mode_shows_failure_details() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyFailure),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyFailed),
		),
		zap.NewNop(),
		"test-changeset-fail",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "test-resource")
	s.Contains(output, "destroy failed")
}

// --- JSON Mode Tests ---

func (s *DestroyTUISuite) Test_json_mode_outputs_result() {
	jsonOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-json",
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

	output := jsonOutput.String()
	s.Contains(output, `"success": true`)
	s.Contains(output, `"instanceId": "test-instance-id"`)
}

// Test that items pre-created from changeset get updated correctly when events arrive.
func (s *DestroyTUISuite) Test_precreated_items_from_changeset_get_updated() {
	// Create changeset with a resource that will be destroyed
	changesetChanges := &changes.BlueprintChanges{
		RemovedResources: []string{"test-resource"},
	}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-precreated",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		changesetChanges, // Pass changeset changes to pre-create items
		false,
	)

	// Verify item was pre-created with pending status
	s.Require().Len(model.items, 1)
	s.Require().NotNil(model.items[0].Resource)
	s.Equal("test-resource", model.items[0].Resource.Name)
	s.Equal(core.ResourceStatusUnknown, model.items[0].Resource.Status) // Should be pending initially

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"Destroyed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.Nil(finalModel.err)
	s.Equal(core.InstanceStatusDestroyed, finalModel.finalStatus)

	// Verify the pre-created item was updated with final status
	s.Require().Len(finalModel.items, 1)
	s.Require().NotNil(finalModel.items[0].Resource)
	s.Equal(core.ResourceStatusDestroyed, finalModel.items[0].Resource.Status)
}

func (s *DestroyTUISuite) Test_destroy_force_mode() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-force",
		"",
		"test-instance",
		true, // force mode
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

	testModel.Send(StartDestroyMsg{})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"Destroyed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.True(finalModel.force)
	s.Equal(core.InstanceStatusDestroyed, finalModel.finalStatus)
}

// --- Pre-Destroy State View Tests ---

func testPreDestroyInstanceState() *state.InstanceState {
	exportValue1 := "export-value-1"
	exportValue2 := "export-value-2"
	specFieldValue := "my-bucket-name"
	outputFieldValue := "arn:aws:s3:::my-bucket-name"
	return &state.InstanceState{
		InstanceID: "pre-destroy-instance-id",
		Status:     core.InstanceStatusDeployed,
		ResourceIDs: map[string]string{
			"resource-1": "res-id-1",
			"resource-2": "res-id-2",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID: "res-id-1",
				Type:       "test/resource",
				Status:     core.ResourceStatusCreated,
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"bucketName": {Scalar: &core.ScalarValue{StringValue: &specFieldValue}},
						"arn":        {Scalar: &core.ScalarValue{StringValue: &outputFieldValue}},
					},
				},
				ComputedFields: []string{"arn"},
			},
			"res-id-2": {
				ResourceID: "res-id-2",
				Type:       "test/other-resource",
				Status:     core.ResourceStatusCreated,
			},
		},
		Links: map[string]*state.LinkState{
			"resource-1::resource-2": {
				LinkID: "link-id-1",
				Status: core.LinkStatusCreated,
			},
		},
		Exports: map[string]*state.ExportState{
			"export-1": {Value: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &exportValue1}}},
			"export-2": {Value: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &exportValue2}}},
		},
		ChildBlueprints: map[string]*state.InstanceState{
			"child-bp": {
				InstanceID: "child-instance-id",
				Status:     core.InstanceStatusDeployed,
				ResourceIDs: map[string]string{
					"child-resource": "child-res-id",
				},
				Resources: map[string]*state.ResourceState{
					"child-res-id": {
						ResourceID: "child-res-id",
						Type:       "test/child-resource",
						Status:     core.ResourceStatusCreated,
					},
				},
			},
		},
	}
}

func (s *DestroyTUISuite) Test_pre_destroy_state_view_shows_on_s_key() {
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-state-view",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	// Set pre-destroy state before starting
	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})

	// Wait for destroy to complete
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	// Press 's' to open pre-destroy state view
	testutils.Key(testModel, "s")

	// Verify pre-destroy state view content is shown
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Pre-Destroy Instance State",
		"pre-destroy-instance-id",
	)

	// Press 's' again to close the view
	testutils.Key(testModel, "s")

	// Wait for main view to be back
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroy complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *DestroyTUISuite) Test_pre_destroy_state_view_shows_resources() {
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-resources",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	testutils.Key(testModel, "s")

	// Verify resources are shown with their IDs and types
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Resources:",
		"resource-1",
		"resource-2",
		"res-id-1",
		"res-id-2",
		"test/resource",
		"test/other-resource",
	)

	testutils.Key(testModel, "esc")
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *DestroyTUISuite) Test_pre_destroy_state_view_shows_links() {
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-links",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	testutils.Key(testModel, "s")

	// Verify links are shown with their IDs
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Links:",
		"resource-1::resource-2",
		"link-id-1",
	)

	testutils.Key(testModel, "esc")
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *DestroyTUISuite) Test_pre_destroy_state_view_shows_exports() {
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-exports",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	testutils.Key(testModel, "s")

	// Verify exports are shown with their values
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Exports:",
		"export-1",
		"export-2",
		"export-value-1",
		"export-value-2",
	)

	testutils.Key(testModel, "esc")
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *DestroyTUISuite) Test_pre_destroy_state_view_shows_child_blueprints() {
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-children",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	testutils.Key(testModel, "s")

	// Verify child blueprints are shown with their instance IDs and nested resources
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Child Blueprints:",
		"child-bp",
		"child-instance-id",
		"child-resource",
		"child-res-id",
	)

	testutils.Key(testModel, "esc")
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *DestroyTUISuite) Test_pre_destroy_state_view_closes_on_esc() {
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-esc",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	// Open pre-destroy state view
	testutils.Key(testModel, "s")
	testutils.WaitForContains(s.T(), testModel.Output(), "Pre-Destroy Instance State")

	// Close with esc
	testutils.Key(testModel, "esc")

	// Verify we're back to main view
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroy complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.False(finalModel.showingPreDestroyState)
}

func (s *DestroyTUISuite) Test_pre_destroy_state_hint_shown_when_state_available() {
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-hint",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})

	// Wait for destroy to complete and verify hint is shown
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Destroyed",
		"for pre-destroy state",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *DestroyTUISuite) Test_pre_destroy_state_view_quit_with_q() {
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-quit-state",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	// Open pre-destroy state view
	testutils.Key(testModel, "s")
	testutils.WaitForContains(s.T(), testModel.Output(), "Pre-Destroy Instance State")

	// Quit directly from the state view
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *DestroyTUISuite) Test_s_key_does_nothing_without_pre_destroy_state() {
	// Use a deploy engine that returns an error when fetching instance state
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithNoInstanceState(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
		),
		zap.NewNop(),
		"test-changeset-no-state",
		"",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	// Press 's' - should not open state view since no pre-destroy state
	testutils.Key(testModel, "s")

	// Should still be on main view (not showing "Pre-Destroy Instance State")
	// Wait a bit and verify the main view is still shown
	time.Sleep(100 * time.Millisecond)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.False(finalModel.showingPreDestroyState)
	s.Nil(finalModel.preDestroyInstanceState)
}

func (s *DestroyTUISuite) Test_pre_destroy_state_view_shows_spec_and_outputs_separately() {
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-spec-outputs",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil,
		false,
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testutils.WaitForContains(s.T(), testModel.Output(), "Destroyed")

	testutils.Key(testModel, "s")

	// Verify spec and outputs are shown separately for resource-1
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Spec:",
		`bucketName: "my-bucket-name"`,
		"Outputs:",
		`arn: "arn:aws:s3:::my-bucket-name"`,
	)

	testutils.Key(testModel, "esc")
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// Test that changeset changes are fetched when destroying with a changeset ID directly.
// This verifies that child blueprint navigation works when changes are fetched from
// the engine rather than pre-provided.
func (s *DestroyTUISuite) Test_changeset_changes_fetched_when_destroying_with_changeset_id() {
	// Create changeset changes that will be returned by GetChangeset
	changesetChanges := &changes.BlueprintChanges{
		RemovedResources: []string{"resource-1"},
		RemovedChildren:  []string{"child-1"},
	}

	// Create instance state that includes the child blueprint for navigation
	preDestroyState := &state.InstanceState{
		InstanceID: "test-instance-id",
		ResourceIDs: map[string]string{
			"resource-1": "resource-1-id",
		},
		Resources: map[string]*state.ResourceState{
			"resource-1-id": {
				ResourceID: "resource-1-id",
				Name:       "resource-1",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
			},
		},
		ChildBlueprints: map[string]*state.InstanceState{
			"child-1": {
				InstanceID: "child-1-instance-id",
				ResourceIDs: map[string]string{
					"nested-resource": "nested-resource-id",
				},
				Resources: map[string]*state.ResourceState{
					"nested-resource-id": {
						ResourceID: "nested-resource-id",
						Name:       "nested-resource",
						Type:       "aws/s3/bucket",
						Status:     core.ResourceStatusCreated,
					},
				},
			},
		},
	}

	// Use the new constructor that supports changeset changes being fetched
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeploymentAndChangeset(
			testDestroyEvents(destroyWithChild),
			"test-instance-id",
			preDestroyState,
			changesetChanges,
		),
		zap.NewNop(),
		"test-changeset-fetch",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil, // No pre-provided changeset changes - should be fetched
		false,
	)

	// Set pre-destroy state for resource type lookups
	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Start destroy - this should trigger fetching changeset changes
	testModel.Send(StartDestroyMsg{})

	// Wait for both resource and child to appear in the UI
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-1",
		"child-1",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)

	// Verify changeset changes were fetched and items were built
	s.NotNil(finalModel.changesetChanges)
	s.Contains(finalModel.changesetChanges.RemovedResources, "resource-1")
	s.Contains(finalModel.changesetChanges.RemovedChildren, "child-1")

	// Verify items were created from the fetched changeset
	s.GreaterOrEqual(len(finalModel.items), 2)
}

// Test destroy with child showing nested resource changes for navigation.
func (s *DestroyTUISuite) Test_child_blueprint_navigation_available_with_fetched_changeset() {
	// Create nested changeset changes
	changesetChanges := &changes.BlueprintChanges{
		RemovedChildren: []string{"notifications"},
		ChildChanges: map[string]changes.BlueprintChanges{
			"notifications": {
				RemovedResources: []string{"queue", "topic"},
			},
		},
	}

	// Create instance state with nested child
	preDestroyState := &state.InstanceState{
		InstanceID: "test-instance-id",
		ChildBlueprints: map[string]*state.InstanceState{
			"notifications": {
				InstanceID: "notifications-instance-id",
				ResourceIDs: map[string]string{
					"queue": "queue-id",
					"topic": "topic-id",
				},
				Resources: map[string]*state.ResourceState{
					"queue-id": {
						ResourceID: "queue-id",
						Name:       "queue",
						Type:       "aws/sqs/queue",
						Status:     core.ResourceStatusCreated,
					},
					"topic-id": {
						ResourceID: "topic-id",
						Name:       "topic",
						Type:       "aws/sns/topic",
						Status:     core.ResourceStatusCreated,
					},
				},
			},
		},
	}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeploymentAndChangeset(
			testDestroyEvents(destroyWithChild),
			"test-instance-id",
			preDestroyState,
			changesetChanges,
		),
		zap.NewNop(),
		"test-changeset-nav",
		"",
		"test-instance",
		false,
		s.styles,
		false,
		os.Stdout,
		nil, // Changeset changes will be fetched
		false,
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})

	// Wait for child to appear
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"notifications",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)

	// Verify the child item was created with its nested changes for navigation
	var childItem *DestroyItem
	for i := range finalModel.items {
		if finalModel.items[i].Type == ItemTypeChild && finalModel.items[i].Child != nil {
			if finalModel.items[i].Child.Name == "notifications" {
				childItem = &finalModel.items[i]
				break
			}
		}
	}

	s.NotNil(childItem, "Child item 'notifications' should exist")
	s.NotNil(childItem.Changes, "Child item should have nested changes for navigation")
	s.Contains(childItem.Changes.RemovedResources, "queue")
	s.Contains(childItem.Changes.RemovedResources, "topic")
}

// --- Drift Handling Tests ---

func createDriftBlockedError() error {
	return &engineerrors.ClientError{
		StatusCode: http.StatusConflict,
		Message:    "Drift detected: resources have changed externally",
		DriftBlockedResponse: &types.DriftBlockedResponse{
			Message:     "Drift detected: resources have changed externally",
			InstanceID:  "test-instance-id",
			ChangesetID: "test-changeset-drift",
			ReconciliationResult: &container.ReconciliationCheckResult{
				InstanceID: "test-instance-id",
				Resources: []container.ResourceReconcileResult{
					{
						ResourceID:        "drifted-resource-id",
						ResourceName:      "drifted-resource",
						ResourceType:      "aws/s3/bucket",
						Type:              container.ReconciliationTypeDrift,
						RecommendedAction: container.ReconciliationActionAcceptExternal,
					},
				},
				HasDrift: true,
			},
		},
	}
}

func (s *DestroyTUISuite) Test_drift_detected_during_destroy_shows_drift_view() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDestroyError(createDriftBlockedError()),
		zap.NewNop(),
		"test-changeset-drift",
		"test-instance-id",
		"test-instance",
		false, // force = false
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

	testModel.Send(StartDestroyMsg{})

	// Wait for drift view to appear (title is "⚠ Drift Detected")
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Drift Detected",
		"drifted-resource",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.True(finalModel.driftReviewMode)
}

func (s *DestroyTUISuite) Test_force_flag_bypasses_drift_check() {
	// With force=true, even if drift is detected, the operation proceeds
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-force-drift",
		"test-instance-id",
		"test-instance",
		true, // force = true
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

	testModel.Send(StartDestroyMsg{})

	// Should succeed and not show drift view
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Destroyed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.True(finalModel.force)
	s.False(finalModel.driftReviewMode)
	s.Equal(core.InstanceStatusDestroyed, finalModel.finalStatus)
}

func (s *DestroyTUISuite) Test_drift_detected_headless_mode() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDestroyError(createDriftBlockedError()),
		zap.NewNop(),
		"test-changeset-drift-headless",
		"test-instance-id",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Drift detected")
	s.Contains(output, "drifted-resource")
}

// --- Error Scenario Tests ---

func createDeployChangesetError() error {
	return &engineerrors.ClientError{
		StatusCode: http.StatusBadRequest,
		Message:    "cannot destroy using a deploy changeset",
		Code:       engineerrors.ErrorCodeDeployChangeset,
	}
}

func (s *DestroyTUISuite) Test_deploy_changeset_error_shows_message() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDestroyError(createDeployChangesetError()),
		zap.NewNop(),
		"test-changeset-deploy",
		"test-instance-id",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	// Wait for error message to appear
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"deploy changeset",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.True(finalModel.deployChangesetError)
}

func (s *DestroyTUISuite) Test_deploy_changeset_error_headless_mode() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDestroyError(createDeployChangesetError()),
		zap.NewNop(),
		"test-changeset-deploy-headless",
		"test-instance-id",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "deploy changeset")
}

func createNetworkError() error {
	return &engineerrors.RequestError{
		Err: &testNetworkError{message: "connection refused"},
	}
}

type testNetworkError struct {
	message string
}

func (e *testNetworkError) Error() string {
	return e.message
}

func (s *DestroyTUISuite) Test_network_error_during_destroy() {
	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDestroyError(createNetworkError()),
		zap.NewNop(),
		"test-changeset-network",
		"test-instance-id",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})

	// Wait for error to appear
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Destroy failed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(DestroyModel)
	s.NotNil(finalModel.err)
}

func (s *DestroyTUISuite) Test_network_error_headless_mode() {
	headlessOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDestroyError(createNetworkError()),
		zap.NewNop(),
		"test-changeset-network-headless",
		"test-instance-id",
		"test-instance",
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

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "error")
}

// --- JSON Mode Tests with Pre-Destroy State ---

func (s *DestroyTUISuite) Test_json_mode_includes_pre_destroy_state() {
	jsonOutput := &bytes.Buffer{}
	preDestroyState := testPreDestroyInstanceState()

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroySuccess),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyed),
		),
		zap.NewNop(),
		"test-changeset-json-predestroy",
		"test-instance-id",
		"test-instance",
		false,
		s.styles,
		true, // headless
		jsonOutput,
		nil,
		true, // jsonMode
	)

	model.SetPreDestroyInstanceState(preDestroyState)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(StartDestroyMsg{})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := jsonOutput.String()
	s.Contains(output, `"success": true`)
	s.Contains(output, `"preDestroyState"`)
	s.Contains(output, `"pre-destroy-instance-id"`)
}

func (s *DestroyTUISuite) Test_json_mode_with_failures() {
	jsonOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyFailure),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyFailed),
		),
		zap.NewNop(),
		"test-changeset-json-failure",
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

	output := jsonOutput.String()
	s.Contains(output, `"success": true`)
	s.Contains(output, `"status": "DESTROY FAILED"`)
	s.Contains(output, `"failed": 1`)
}

func (s *DestroyTUISuite) Test_json_mode_with_rollback() {
	jsonOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyRollback),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyRollbackComplete),
		),
		zap.NewNop(),
		"test-changeset-json-rollback",
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

	output := jsonOutput.String()
	s.Contains(output, `"success": true`)
	s.Contains(output, `"status": "DESTROY ROLLBACK COMPLETE"`)
}

func (s *DestroyTUISuite) Test_json_mode_with_interrupted() {
	jsonOutput := &bytes.Buffer{}

	model := NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			testDestroyEvents(destroyInterrupted),
			"test-instance-id",
			testInstanceState(core.InstanceStatusDestroyInterrupted),
		),
		zap.NewNop(),
		"test-changeset-json-interrupted",
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

	output := jsonOutput.String()
	s.Contains(output, `"success": true`)
	s.Contains(output, `"status": "DESTROY INTERRUPTED"`)
	s.Contains(output, `"interrupted": 1`)
}
