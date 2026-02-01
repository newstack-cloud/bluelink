package destroyui

import (
	"context"
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
	"github.com/stretchr/testify/suite"
)

type DestroyConfigFormSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestDestroyConfigFormSuite(t *testing.T) {
	suite.Run(t, new(DestroyConfigFormSuite))
}

func (s *DestroyConfigFormSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// --- Initial State Tests ---
// These tests verify that NewDestroyApp correctly determines initial session state
// based on the provided flags and parameters.

func (s *DestroyConfigFormSuite) Test_stage_first_without_blueprint_goes_to_blueprint_select() {
	mockEngine := testutils.NewTestDeployEngineWithDeployment(
		testDestroyEvents(destroySuccess),
		"test-instance-id",
		nil,
	)

	model, err := NewDestroyApp(
		mockEngine,
		nil,
		"",    // no changesetID
		"",    // no instanceID
		"",    // no instanceName
		"",    // no blueprintFile
		true,  // isDefaultBlueprintFile (means use default)
		false, // force
		true,  // stageFirst
		false, // autoApprove
		false, // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// When staging without a blueprint file, should go to blueprint select
	s.Equal(destroyBlueprintSelect, model.sessionState)
}

func (s *DestroyConfigFormSuite) Test_stage_first_with_blueprint_goes_to_config_input() {
	mockEngine := testutils.NewTestDeployEngineWithDeployment(
		testDestroyEvents(destroySuccess),
		"test-instance-id",
		nil,
	)

	model, err := NewDestroyApp(
		mockEngine,
		nil,
		"",                    // no changesetID
		"",                    // no instanceID
		"",                    // no instanceName
		"test.blueprint.yaml", // blueprintFile provided
		false,                 // isDefaultBlueprintFile (explicit file)
		false,                 // force
		true,                  // stageFirst
		false,                 // autoApprove
		false,                 // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// When staging with a blueprint file but no instance name, should go to config input
	s.Equal(destroyConfigInput, model.sessionState)
}

func (s *DestroyConfigFormSuite) Test_skip_prompts_with_changeset_skips_to_execute() {
	mockEngine := testutils.NewTestDeployEngineWithDeployment(
		testDestroyEvents(destroySuccess),
		"test-instance-id",
		nil,
	)

	model, err := NewDestroyApp(
		mockEngine,
		nil,
		"existing-changeset", // changesetID provided
		"",                   // instanceID
		"skip-prompts-inst",  // instanceName provided
		"test.blueprint.yaml",
		false, // isDefaultBlueprintFile
		false, // force
		false, // stageFirst
		false, // autoApprove
		true,  // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// With skipPrompts and all values provided, should skip to destroyExecute
	s.Equal(destroyExecute, model.sessionState)
}

func (s *DestroyConfigFormSuite) Test_skip_prompts_with_stage_first_goes_to_staging() {
	mockEngine := testutils.NewTestDeployEngineWithDeployment(
		testDestroyEvents(destroySuccess),
		"test-instance-id",
		nil,
	)

	model, err := NewDestroyApp(
		mockEngine,
		nil,
		"",                    // no changesetID
		"",                    // no instanceID
		"skip-prompts-inst",   // instanceName provided
		"test.blueprint.yaml",
		false, // isDefaultBlueprintFile
		false, // force
		true,  // stageFirst
		false, // autoApprove
		true,  // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// With skipPrompts + stageFirst, should go to staging
	s.Equal(destroyStaging, model.sessionState)
}

func (s *DestroyConfigFormSuite) Test_headless_with_changeset_skips_to_execute() {
	headlessOutput := testutils.NewSaveBuffer()
	mockEngine := testutils.NewTestDeployEngineWithDeployment(
		testDestroyEvents(destroySuccess),
		"test-instance-id",
		nil,
	)

	model, err := NewDestroyApp(
		mockEngine,
		nil,
		"test-changeset",    // changesetID provided
		"",                  // instanceID
		"headless-instance", // instanceName provided
		"",                  // no blueprint file needed with changeset
		true,                // isDefaultBlueprintFile
		false,               // force
		false,               // stageFirst
		false,               // autoApprove
		false,               // skipPrompts
		s.styles,
		true, // headless
		headlessOutput,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// In headless mode with changeset, should go directly to execute
	s.Equal(destroyExecute, model.sessionState)
}

func (s *DestroyConfigFormSuite) Test_headless_with_stage_first_goes_to_staging() {
	headlessOutput := testutils.NewSaveBuffer()
	mockEngine := newMockDestroyEngineWithFullFlow(
		testStagingEventsForDestroy(stagingSuccessDeleteDestroy),
		testDestroyEvents(destroySuccess),
		"test-changeset-headless",
		"test-instance-id",
	)

	model, err := NewDestroyApp(
		mockEngine,
		nil,
		"",                    // changesetID
		"",                    // instanceID
		"headless-instance",   // instanceName
		"test.blueprint.yaml",
		false, // isDefaultBlueprintFile
		false, // force
		true,  // stageFirst
		true,  // autoApprove (required for headless)
		false, // skipPrompts
		s.styles,
		true, // headless
		headlessOutput,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// In headless mode with stageFirst, should start in destroyStaging state
	s.Equal(destroyStaging, model.sessionState)
}

func (s *DestroyConfigFormSuite) Test_interactive_mode_without_required_values_goes_to_config_input() {
	mockEngine := testutils.NewTestDeployEngineWithDeployment(
		testDestroyEvents(destroySuccess),
		"test-instance-id",
		nil,
	)

	model, err := NewDestroyApp(
		mockEngine,
		nil,
		"",                    // no changesetID
		"",                    // no instanceID
		"",                    // no instanceName
		"test.blueprint.yaml", // blueprintFile provided
		false,                 // isDefaultBlueprintFile
		false,                 // force
		false,                 // stageFirst (not staging)
		false,                 // autoApprove
		false,                 // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// In interactive mode without all values, should go to config input
	s.Equal(destroyConfigInput, model.sessionState)
}

func (s *DestroyConfigFormSuite) Test_with_instance_id_preserves_value() {
	mockEngine := testutils.NewTestDeployEngineWithDeployment(
		testDestroyEvents(destroySuccess),
		"existing-instance-123",
		nil,
	)

	model, err := NewDestroyApp(
		mockEngine,
		nil,
		"",                      // changesetID
		"existing-instance-123", // instanceID provided
		"my-instance",           // instanceName
		"test.blueprint.yaml",
		false, // isDefaultBlueprintFile
		false, // force
		true,  // stageFirst
		false, // autoApprove
		false, // skipPrompts
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// Should preserve the instance ID
	s.Equal("existing-instance-123", model.instanceID)
	s.Equal("my-instance", model.instanceName)
}

// --- Unit Tests for Form Initialization ---

func (s *DestroyConfigFormSuite) Test_form_has_instance_id_flag_when_provided() {
	form := NewDestroyConfigFormModel(
		DestroyConfigFormInitialValues{
			InstanceName: "my-instance",
			InstanceID:   "instance-123",
			ChangesetID:  "",
			StageFirst:   true,
			AutoApprove:  false,
		},
		s.styles,
	)

	s.True(form.hasInstanceID)
	s.Equal("instance-123", form.instanceID)
}

func (s *DestroyConfigFormSuite) Test_form_has_no_instance_id_flag_when_empty() {
	form := NewDestroyConfigFormModel(
		DestroyConfigFormInitialValues{
			InstanceName: "",
			InstanceID:   "",
			ChangesetID:  "",
			StageFirst:   true,
			AutoApprove:  false,
		},
		s.styles,
	)

	s.False(form.hasInstanceID)
}

func (s *DestroyConfigFormSuite) Test_form_preserves_initial_values() {
	form := NewDestroyConfigFormModel(
		DestroyConfigFormInitialValues{
			InstanceName: "test-instance",
			InstanceID:   "",
			ChangesetID:  "changeset-abc",
			StageFirst:   false,
			AutoApprove:  true,
		},
		s.styles,
	)

	s.Equal("test-instance", form.instanceName)
	s.Equal("changeset-abc", form.changesetID)
	s.False(form.stageFirst)
	s.True(form.autoApprove)
}

func (s *DestroyConfigFormSuite) Test_form_view_not_empty() {
	form := NewDestroyConfigFormModel(
		DestroyConfigFormInitialValues{
			InstanceName: "",
			InstanceID:   "",
			ChangesetID:  "",
			StageFirst:   true,
			AutoApprove:  false,
		},
		s.styles,
	)

	view := form.View()
	s.NotEmpty(view)
	s.Contains(view, "Destroy Options")
}

func (s *DestroyConfigFormSuite) Test_form_with_instance_id_sets_internal_state() {
	form := NewDestroyConfigFormModel(
		DestroyConfigFormInitialValues{
			InstanceName: "",
			InstanceID:   "my-instance-id",
			ChangesetID:  "",
			StageFirst:   true,
			AutoApprove:  false,
		},
		s.styles,
	)

	// The form stores the instance ID internally when provided
	// The view rendering depends on huh form lifecycle initialization
	s.True(form.hasInstanceID)
	s.Equal("my-instance-id", form.instanceID)
}

// --- Test Event Factories ---

type testStagingDestroyType string

const (
	stagingSuccessDeleteDestroy testStagingDestroyType = "success_delete"
)

func testStagingEventsForDestroy(stagingType testStagingDestroyType) []*types.ChangeStagingEvent {
	switch stagingType {
	case stagingSuccessDeleteDestroy:
		return []*types.ChangeStagingEvent{
			resourceDeleteEventForDestroy("test-resource"),
			completeChangesEventForDestroy(),
		}
	default:
		return []*types.ChangeStagingEvent{completeChangesEventForDestroy()}
	}
}

func resourceDeleteEventForDestroy(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          false,
				Removed:      true,
				Changes:      provider.Changes{},
			},
		},
	}
}

func completeChangesEventForDestroy() *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		CompleteChanges: &types.CompleteChangesEventData{
			Changes: &changes.BlueprintChanges{
				RemovedResources: []string{"test-resource"},
			},
		},
	}
}

// --- Mock Engine with Full Flow Support ---

type mockDestroyEngineWithFullFlow struct {
	engine.DeployEngine
	stagingEvents []*types.ChangeStagingEvent
	destroyEvents []*types.BlueprintInstanceEvent
	changesetID   string
	instanceID    string
}

func newMockDestroyEngineWithFullFlow(
	stagingEvents []*types.ChangeStagingEvent,
	destroyEvents []*types.BlueprintInstanceEvent,
	changesetID string,
	instanceID string,
) engine.DeployEngine {
	baseEngine := testutils.NewTestDeployEngineWithDeployment(
		destroyEvents,
		instanceID,
		&state.InstanceState{
			InstanceID: instanceID,
			Status:     core.InstanceStatusDestroyed,
		},
	)
	return &mockDestroyEngineWithFullFlow{
		DeployEngine:  baseEngine,
		stagingEvents: stagingEvents,
		destroyEvents: destroyEvents,
		changesetID:   changesetID,
		instanceID:    instanceID,
	}
}

func (m *mockDestroyEngineWithFullFlow) CreateChangeset(
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

func (m *mockDestroyEngineWithFullFlow) StreamChangeStagingEvents(
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

func (m *mockDestroyEngineWithFullFlow) GetBlueprintInstance(
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

func (m *mockDestroyEngineWithFullFlow) DestroyBlueprintInstance(
	ctx context.Context,
	instanceID string,
	payload *types.DestroyBlueprintInstancePayload,
) (*types.BlueprintInstanceResponse, error) {
	return &types.BlueprintInstanceResponse{
		Data: state.InstanceState{
			InstanceID: instanceID,
			Status:     core.InstanceStatusDestroying,
		},
		LastEventID: "",
	}, nil
}

func (m *mockDestroyEngineWithFullFlow) StreamBlueprintInstanceEvents(
	ctx context.Context,
	instanceID string,
	lastEventID string,
	streamTo chan<- types.BlueprintInstanceEvent,
	errChan chan<- error,
) error {
	go func() {
		for _, event := range m.destroyEvents {
			streamTo <- *event
		}
	}()
	return nil
}
