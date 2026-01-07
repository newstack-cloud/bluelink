package deployui

import (
	"context"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DeployOptionsFormSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDeployOptionsFormSuite(t *testing.T) {
	suite.Run(t, new(DeployOptionsFormSuite))
}

func (s *DeployOptionsFormSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// --- TUI Integration Tests ---

func (s *DeployOptionsFormSuite) Test_deploy_options_form_stage_first_flow() {
	// Test the flow where user selects "Stage changes first" in the deploy options form
	mockEngine := newMockDeployEngineWithFullFlow(
		testStagingEventsForDeploy(stagingSuccessCreateDeploy),
		testDeployEvents(deploySuccessCreate),
		"test-changeset-stage",
		"test-instance-id",
	)

	model, err := NewDeployApp(
		mockEngine,
		zap.NewNop(),
		"",    // changesetID (none, will stage first)
		"",    // instanceID
		"",    // instanceName (will be entered via form)
		"",    // blueprintFile (will be selected via message)
		true,  // isDefaultBlueprintFile
		false, // autoRollback
		false, // force
		true,  // stageFirst (default in form)
		false, // autoApprove
		false, // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select blueprint to trigger the form
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        "file",
	})

	// Wait for deploy options form to appear (Instance Name field is visible)
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")

	// Fill in instance name
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("my-test-instance")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Stage first toggle - should default to Yes, just press enter
	testutils.WaitForContains(s.T(), testModel.Output(), "Stage changes first")
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Auto-approve toggle - select No (default)
	testutils.WaitForContains(s.T(), testModel.Output(), "Auto-approve")
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Auto-rollback toggle - select No (default)
	testutils.WaitForContains(s.T(), testModel.Output(), "auto-rollback")
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for staging to complete and show confirmation prompt
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-changeset-stage",
		"Apply these changes",
	)

	// Confirm deployment with 'y'
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Wait for deployment to complete (interactive mode shows "complete", not "completed")
	testutils.WaitForContains(s.T(), testModel.Output(), "Deployment complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
	s.Equal("my-test-instance", finalModel.InstanceName())
	s.True(finalModel.StageFirst())
}

func (s *DeployOptionsFormSuite) Test_deploy_options_form_existing_changeset_flow() {
	// Test the flow where user provides an existing changeset ID instead of staging
	mockEngine := newMockDeployEngineWithFullFlow(
		nil, // No staging events needed
		testDeployEvents(deploySuccessCreate),
		"",
		"test-instance-id",
	)

	model, err := NewDeployApp(
		mockEngine,
		zap.NewNop(),
		"",    // changesetID (will be entered via form)
		"",    // instanceID
		"",    // instanceName (will be entered via form)
		"",    // blueprintFile
		true,  // isDefaultBlueprintFile
		false, // autoRollback
		false, // force
		false, // stageFirst (will be set to No)
		false, // autoApprove
		false, // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select a blueprint
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        "file",
	})

	// Wait for deploy options form and fill in instance name
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("existing-changeset-instance")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Stage first toggle - select No (this will show changeset ID input)
	testutils.WaitForContains(s.T(), testModel.Output(), "Stage changes first")
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	// Changeset ID input should appear - fill it in
	testutils.WaitForContains(s.T(), testModel.Output(), "Changeset ID")
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("existing-changeset-123")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Auto-rollback toggle - keep default (No)
	testutils.WaitForContains(s.T(), testModel.Output(), "auto-rollback")
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for deployment to complete (interactive mode shows "complete", not "completed")
	testutils.WaitForContains(s.T(), testModel.Output(), "Deployment complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
	s.Equal("existing-changeset-instance", finalModel.InstanceName())
	s.Equal("existing-changeset-123", finalModel.ChangesetID())
	s.False(finalModel.StageFirst())
}

func (s *DeployOptionsFormSuite) Test_deploy_options_form_with_instance_id_shows_note() {
	// When instance ID is provided, the form should show a note instead of instance name input
	mockEngine := newMockDeployEngineWithFullFlow(
		testStagingEventsForDeploy(stagingSuccessUpdateDeploy),
		testDeployEvents(deploySuccessCreate),
		"test-changeset-update",
		"existing-instance-123",
	)

	model, err := NewDeployApp(
		mockEngine,
		zap.NewNop(),
		"",                      // changesetID
		"existing-instance-123", // instanceID provided
		"my-instance",           // instanceName
		"",                      // blueprintFile
		true,                    // isDefaultBlueprintFile
		false,                   // autoRollback
		false,                   // force
		true,                    // stageFirst
		false,                   // autoApprove
		false,                   // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select a blueprint to start the form
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        "file",
	})

	// Wait for form to appear - should show Instance ID as a note (not an input)
	// along with the Stage changes first toggle. We wait for both to verify
	// the Instance ID note is rendered correctly.
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Instance ID",
		"existing-instance-123",
		"Stage changes first",
	)

	// Stage first toggle - select Yes (default)
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Auto-approve toggle - select No (default)
	testutils.WaitForContains(s.T(), testModel.Output(), "Auto-approve")
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Auto-rollback toggle
	testutils.WaitForContains(s.T(), testModel.Output(), "auto-rollback")
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for staging to complete and show confirmation prompt
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-changeset-update",
		"Apply these changes",
	)

	// Confirm deployment
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Wait for deployment to complete (interactive mode shows "complete", not "completed")
	testutils.WaitForContains(s.T(), testModel.Output(), "Deployment complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
	s.Equal("existing-instance-123", finalModel.instanceID)
}

func (s *DeployOptionsFormSuite) Test_deploy_options_form_skipped_in_headless_mode() {
	// In headless mode, deploy options form should be skipped
	headlessOutput := testutils.NewSaveBuffer()
	mockEngine := newMockDeployEngineWithFullFlow(
		testStagingEventsForDeploy(stagingSuccessCreateDeploy),
		testDeployEvents(deploySuccessCreate),
		"test-changeset-headless",
		"test-instance-id",
	)

	model, err := NewDeployApp(
		mockEngine,
		zap.NewNop(),
		"",                   // changesetID
		"",                   // instanceID
		"headless-instance",  // instanceName
		"test.blueprint.yaml",
		false, // isDefaultBlueprintFile
		false, // autoRollback
		false, // force
		true,  // stageFirst
		true,  // autoApprove (required for headless)
		false, // skipPrompts
		s.styles,
		true, // headless
		headlessOutput,
		false, // jsonMode
	)
	s.Require().NoError(err)

	// In headless mode with stageFirst, should start in deployStaging state
	s.Equal(deployStaging, model.sessionState)
}

func (s *DeployOptionsFormSuite) Test_deploy_options_form_skipped_with_skip_prompts() {
	// With skipPrompts and all required values, form should be skipped
	mockEngine := newMockDeployEngineWithFullFlow(
		nil, // No staging events needed when using existing changeset
		testDeployEvents(deploySuccessCreate),
		"",
		"test-instance-id",
	)

	model, err := NewDeployApp(
		mockEngine,
		zap.NewNop(),
		"existing-changeset", // changesetID provided
		"",                   // instanceID
		"skip-prompts-inst",  // instanceName provided
		"test.blueprint.yaml",
		false, // isDefaultBlueprintFile
		false, // autoRollback
		false, // force
		false, // stageFirst
		false, // autoApprove
		true,  // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
	)
	s.Require().NoError(err)

	// With skipPrompts and all values provided, should skip to deployExecute
	s.Equal(deployExecute, model.sessionState)
}

func (s *DeployOptionsFormSuite) Test_deploy_options_form_instance_name_validation() {
	mockEngine := newMockDeployEngineWithFullFlow(
		testStagingEventsForDeploy(stagingSuccessCreateDeploy),
		testDeployEvents(deploySuccessCreate),
		"test-changeset",
		"test-instance-id",
	)

	model, err := NewDeployApp(
		mockEngine,
		zap.NewNop(),
		"",
		"",
		"",
		"",
		true,
		false,
		false,
		true,
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		false,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select blueprint
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        "file",
	})

	// Wait for deploy options form
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")

	// Try to submit empty name
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Should show validation error
	testutils.WaitForContains(s.T(), testModel.Output(), "cannot be empty")

	// Type a short name (less than 3 chars)
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ab")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Should show validation error for short name
	testutils.WaitForContains(s.T(), testModel.Output(), "at least 3 characters")

	// Clean up
	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *DeployOptionsFormSuite) Test_deploy_options_form_auto_approve_enabled() {
	// Test auto-approve flow where deployment proceeds immediately after staging
	mockEngine := newMockDeployEngineWithFullFlow(
		testStagingEventsForDeploy(stagingSuccessCreateDeploy),
		testDeployEvents(deploySuccessCreate),
		"test-changeset-auto",
		"test-instance-id",
	)

	model, err := NewDeployApp(
		mockEngine,
		zap.NewNop(),
		"",
		"",
		"",
		"",
		true,
		false,
		false,
		true,
		false, // autoApprove starts as false, will be set via form
		false,
		s.styles,
		false,
		os.Stdout,
		false,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select blueprint
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        "file",
	})

	// Wait for deploy options form
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")

	// Fill in instance name
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("auto-approve-instance")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Stage first - Yes (default)
	testutils.WaitForContains(s.T(), testModel.Output(), "Stage changes first")
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Auto-approve - select Yes
	testutils.WaitForContains(s.T(), testModel.Output(), "Auto-approve")
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Auto-rollback - No (default)
	testutils.WaitForContains(s.T(), testModel.Output(), "auto-rollback")
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// With auto-approve, should go directly to deployment after staging
	// Wait for deployment to complete (interactive mode shows "complete", not "completed")
	testutils.WaitForContains(s.T(), testModel.Output(), "Deployment complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
	s.True(finalModel.autoApprove)
}

func (s *DeployOptionsFormSuite) Test_deploy_options_form_auto_rollback_enabled() {
	// Test that auto-rollback setting is properly captured
	mockEngine := newMockDeployEngineWithFullFlow(
		testStagingEventsForDeploy(stagingSuccessCreateDeploy),
		testDeployEvents(deploySuccessCreate),
		"test-changeset-rollback",
		"test-instance-id",
	)

	model, err := NewDeployApp(
		mockEngine,
		zap.NewNop(),
		"",
		"",
		"",
		"",
		true,
		false, // autoRollback starts as false
		false,
		true,
		true,  // autoApprove to skip confirmation
		false,
		s.styles,
		false,
		os.Stdout,
		false,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select blueprint
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        "file",
	})

	// Wait for deploy options form
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")

	// Fill in instance name
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("rollback-instance")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Stage first - Yes
	testutils.WaitForContains(s.T(), testModel.Output(), "Stage changes first")
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Auto-approve - Yes (to skip confirmation)
	testutils.WaitForContains(s.T(), testModel.Output(), "Auto-approve")
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Auto-rollback - select Yes
	testutils.WaitForContains(s.T(), testModel.Output(), "auto-rollback")
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Wait for deployment to complete (interactive mode shows "complete", not "completed")
	testutils.WaitForContains(s.T(), testModel.Output(), "Deployment complete")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
	s.True(finalModel.autoRollback)
}

// --- Test Event Factories ---

type testStagingDeployType string

const (
	stagingSuccessCreateDeploy testStagingDeployType = "success_create"
	stagingSuccessUpdateDeploy testStagingDeployType = "success_update"
)

func testStagingEventsForDeploy(stagingType testStagingDeployType) []*types.ChangeStagingEvent {
	switch stagingType {
	case stagingSuccessCreateDeploy:
		return []*types.ChangeStagingEvent{
			resourceCreateEventForDeploy("test-resource"),
			completeChangesEventForDeploy(),
		}
	case stagingSuccessUpdateDeploy:
		return []*types.ChangeStagingEvent{
			resourceUpdateEventForDeploy("test-resource"),
			completeChangesEventForDeploy(),
		}
	default:
		return []*types.ChangeStagingEvent{completeChangesEventForDeploy()}
	}
}

func resourceCreateEventForDeploy(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          true,
				Changes: provider.Changes{
					NewFields: []provider.FieldChange{
						{FieldPath: "spec.name", NewValue: stringMappingNodeForDeploy(name)},
					},
				},
			},
		},
	}
}

func resourceUpdateEventForDeploy(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          false,
				Changes: provider.Changes{
					ModifiedFields: []provider.FieldChange{
						{
							FieldPath: "spec.config",
							PrevValue: stringMappingNodeForDeploy("old"),
							NewValue:  stringMappingNodeForDeploy("new"),
						},
					},
				},
			},
		},
	}
}

func completeChangesEventForDeploy() *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		CompleteChanges: &types.CompleteChangesEventData{
			Changes: &changes.BlueprintChanges{
				NewResources: map[string]provider.Changes{
					"test-resource": {},
				},
			},
		},
	}
}

func stringMappingNodeForDeploy(s string) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &s},
	}
}

// --- Mock Engine with Full Flow Support ---

type mockDeployEngineWithFullFlow struct {
	engine.DeployEngine
	stagingEvents    []*types.ChangeStagingEvent
	deploymentEvents []*types.BlueprintInstanceEvent
	changesetID      string
	instanceID       string
}

func newMockDeployEngineWithFullFlow(
	stagingEvents []*types.ChangeStagingEvent,
	deploymentEvents []*types.BlueprintInstanceEvent,
	changesetID string,
	instanceID string,
) engine.DeployEngine {
	baseEngine := testutils.NewTestDeployEngineWithDeployment(
		deploymentEvents,
		instanceID,
		&state.InstanceState{
			InstanceID: instanceID,
			Status:     core.InstanceStatusDeployed,
		},
	)
	return &mockDeployEngineWithFullFlow{
		DeployEngine:     baseEngine,
		stagingEvents:    stagingEvents,
		deploymentEvents: deploymentEvents,
		changesetID:      changesetID,
		instanceID:       instanceID,
	}
}

func (m *mockDeployEngineWithFullFlow) CreateChangeset(
	ctx context.Context,
	payload *types.CreateChangesetPayload,
) (*types.ChangesetResponse, error) {
	return &types.ChangesetResponse{
		Data: &manage.Changeset{
			ID:                m.changesetID,
			Status:            manage.ChangesetStatusStagingChanges,
			BlueprintLocation: payload.BlueprintFile,
		},
		LastEventID: "",
	}, nil
}

func (m *mockDeployEngineWithFullFlow) StreamChangeStagingEvents(
	ctx context.Context,
	changesetID string,
	lastEventID string,
	streamTo chan<- types.ChangeStagingEvent,
	errChan chan<- error,
) error {
	go func() {
		for _, event := range m.stagingEvents {
			streamTo <- *event
		}
	}()
	return nil
}

func (m *mockDeployEngineWithFullFlow) GetBlueprintInstance(
	ctx context.Context,
	instanceID string,
) (*state.InstanceState, error) {
	if m.instanceID != "" && instanceID == m.instanceID {
		return &state.InstanceState{
			InstanceID: m.instanceID,
			Status:     core.InstanceStatusDeployed,
		}, nil
	}
	return nil, nil
}

func (m *mockDeployEngineWithFullFlow) CreateBlueprintInstance(
	ctx context.Context,
	payload *types.BlueprintInstancePayload,
) (*types.BlueprintInstanceResponse, error) {
	return &types.BlueprintInstanceResponse{
		Data: state.InstanceState{
			InstanceID: m.instanceID,
			Status:     core.InstanceStatusDeploying,
		},
		LastEventID: "",
	}, nil
}

func (m *mockDeployEngineWithFullFlow) UpdateBlueprintInstance(
	ctx context.Context,
	instanceID string,
	payload *types.BlueprintInstancePayload,
) (*types.BlueprintInstanceResponse, error) {
	return &types.BlueprintInstanceResponse{
		Data: state.InstanceState{
			InstanceID: instanceID,
			Status:     core.InstanceStatusUpdating,
		},
		LastEventID: "",
	}, nil
}

func (m *mockDeployEngineWithFullFlow) StreamBlueprintInstanceEvents(
	ctx context.Context,
	instanceID string,
	lastEventID string,
	streamTo chan<- types.BlueprintInstanceEvent,
	errChan chan<- error,
) error {
	go func() {
		for _, event := range m.deploymentEvents {
			streamTo <- *event
		}
	}()
	return nil
}
