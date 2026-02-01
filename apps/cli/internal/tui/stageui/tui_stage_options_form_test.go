package stageui

import (
	"context"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type StageOptionsFormSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestStageOptionsFormSuite(t *testing.T) {
	suite.Run(t, new(StageOptionsFormSuite))
}

func (s *StageOptionsFormSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

// --- TUI Integration Tests ---

func (s *StageOptionsFormSuite) Test_stage_options_flow_new_instance_skips_options() {
	// When instance doesn't exist, should skip options form and proceed directly to staging
	mockEngine := newMockEngineWithInstances(
		testStagingEvents(stagingSuccessCreate),
		"test-changeset-new",
		nil, // No existing instances
	)

	model, err := NewStageApp(
		mockEngine,
		zap.NewNop(),
		"",    // blueprintFile (will be selected)
		true,  // isDefaultBlueprintFile
		"",    // instanceID
		"",    // instanceName (triggers options form)
		false, // destroy
		false, // skipDriftCheck
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select a blueprint first
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        "file",
	})

	// Wait for instance name form to appear
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")

	// Type instance name and submit
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("new-instance-name")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for staging to complete (should skip options since instance doesn't exist)
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-changeset-new",
		"CREATE",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)

	// Verify the stage model received correct options (no destroy/skipDriftCheck for new instance)
	stageModel := finalModel.Stage().(StageModel)
	s.False(stageModel.Destroy())
	s.False(stageModel.SkipDriftCheck())
}

func (s *StageOptionsFormSuite) Test_stage_options_flow_existing_instance_shows_options() {
	// When instance exists, should show destroy/skipDriftCheck options
	existingInstances := map[string]*state.InstanceState{
		"existing-instance": {
			InstanceID:   "instance-123",
			InstanceName: "existing-instance",
		},
	}
	mockEngine := newMockEngineWithInstances(
		testStagingEvents(stagingSuccessUpdate),
		"test-changeset-existing",
		existingInstances,
	)

	model, err := NewStageApp(
		mockEngine,
		zap.NewNop(),
		"",    // blueprintFile
		true,  // isDefaultBlueprintFile
		"",    // instanceID
		"",    // instanceName (triggers options form)
		false, // destroy
		false, // skipDriftCheck
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select a blueprint first
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        "file",
	})

	// Wait for instance name form
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")

	// Type existing instance name and submit
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("existing-instance")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for instance check and options form
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"existing-instance",
		"exists",
	)

	// Submit options form with defaults (No for both)
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter}) // Destroy Mode - default No
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter}) // Skip Drift Check - default No

	// Wait for staging to complete
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-changeset-existing",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
}

func (s *StageOptionsFormSuite) Test_stage_options_flow_existing_instance_with_destroy_enabled() {
	existingInstances := map[string]*state.InstanceState{
		"destroy-instance": {
			InstanceID:   "instance-456",
			InstanceName: "destroy-instance",
		},
	}
	mockEngine := newMockEngineWithInstances(
		testStagingEvents(stagingSuccessDelete),
		"test-changeset-destroy",
		existingInstances,
	)

	model, err := NewStageApp(
		mockEngine,
		zap.NewNop(),
		"",    // blueprintFile
		true,  // isDefaultBlueprintFile
		"",    // instanceID
		"",    // instanceName
		false, // destroy (initial, will be changed via form)
		false, // skipDriftCheck
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
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

	// Wait for instance name form
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")

	// Type instance name and submit
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("destroy-instance")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for options form
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"destroy-instance",
		"exists",
	)

	// Select Yes for Destroy Mode
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter}) // Skip Drift Check - default No

	// Wait for staging to complete
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"test-changeset-destroy",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)

	// Verify destroy was set
	stageModel := finalModel.Stage().(StageModel)
	s.True(stageModel.Destroy())
}

func (s *StageOptionsFormSuite) Test_stage_options_skipped_when_instance_name_provided() {
	// When instanceName is provided via flag, skip the options form entirely
	mockEngine := newMockEngineWithInstances(
		testStagingEvents(stagingSuccessCreate),
		"test-changeset-direct",
		nil,
	)

	model, err := NewStageApp(
		mockEngine,
		zap.NewNop(),
		"test.blueprint.yaml", // blueprintFile provided
		false,                 // isDefaultBlueprintFile
		"",                    // instanceID
		"provided-instance",   // instanceName provided via flag
		false,                 // destroy
		false,                 // skipDriftCheck
		s.styles,
		false, // headless
		os.Stdout,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// Options form should not be created when instance name is provided
	s.Nil(model.StageOptionsForm())
	s.False(model.NeedsOptionsInput())
}

func (s *StageOptionsFormSuite) Test_stage_options_skipped_in_headless_mode() {
	// In headless mode, skip the options form
	headlessOutput := testutils.NewSaveBuffer()
	mockEngine := newMockEngineWithInstances(
		testStagingEvents(stagingSuccessCreate),
		"test-changeset-headless",
		nil,
	)

	model, err := NewStageApp(
		mockEngine,
		zap.NewNop(),
		"test.blueprint.yaml",
		false, // isDefaultBlueprintFile
		"",    // instanceID
		"",    // instanceName
		false, // destroy
		false, // skipDriftCheck
		s.styles,
		true, // headless
		headlessOutput,
		false, // jsonMode
		nil,
	)
	s.Require().NoError(err)

	// Options form should not be created in headless mode
	s.Nil(model.StageOptionsForm())
	s.False(model.NeedsOptionsInput())
}

func (s *StageOptionsFormSuite) Test_instance_name_validation_rejects_empty() {
	mockEngine := newMockEngineWithInstances(
		testStagingEvents(stagingSuccessCreate),
		"test-changeset",
		nil,
	)

	model, err := NewStageApp(
		mockEngine,
		zap.NewNop(),
		"",
		true,
		"",
		"",
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		false,
		nil,
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

	// Wait for instance name form
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")

	// Try to submit empty name
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Should show validation error
	testutils.WaitForContains(s.T(), testModel.Output(), "cannot be empty")

	// Clean up
	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageOptionsFormSuite) Test_instance_name_validation_rejects_short_names() {
	mockEngine := newMockEngineWithInstances(
		testStagingEvents(stagingSuccessCreate),
		"test-changeset",
		nil,
	)

	model, err := NewStageApp(
		mockEngine,
		zap.NewNop(),
		"",
		true,
		"",
		"",
		false,
		false,
		s.styles,
		false,
		os.Stdout,
		false,
		nil,
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

	// Wait for instance name form
	testutils.WaitForContains(s.T(), testModel.Output(), "Instance Name")

	// Type a short name (less than 3 chars)
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ab")})
	testModel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Should show validation error
	testutils.WaitForContains(s.T(), testModel.Output(), "at least 3 characters")

	// Clean up
	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// --- Mock Engine with Instance Support ---

// mockEngineWithInstances wraps a test deploy engine and adds configurable
// instance existence checks for testing the stage options form flow.
type mockEngineWithInstances struct {
	engine.DeployEngine
	instances map[string]*state.InstanceState
}

func newMockEngineWithInstances(
	stagingEvents []*types.ChangeStagingEvent,
	changesetID string,
	instances map[string]*state.InstanceState,
) engine.DeployEngine {
	baseEngine := testutils.NewTestDeployEngineWithStaging(stagingEvents, changesetID)
	return &mockEngineWithInstances{
		DeployEngine: baseEngine,
		instances:    instances,
	}
}

func (m *mockEngineWithInstances) GetBlueprintInstance(
	ctx context.Context,
	instanceID string,
) (*state.InstanceState, error) {
	if m.instances == nil {
		return nil, nil
	}
	if instance, ok := m.instances[instanceID]; ok {
		return instance, nil
	}
	return nil, nil
}

// Delegate other methods to the base engine while preserving staging behavior
func (m *mockEngineWithInstances) CreateChangeset(
	ctx context.Context,
	payload *types.CreateChangesetPayload,
) (*types.ChangesetResponse, error) {
	return m.DeployEngine.CreateChangeset(ctx, payload)
}

func (m *mockEngineWithInstances) StreamChangeStagingEvents(
	ctx context.Context,
	changesetID string,
	lastEventID string,
	streamTo chan<- types.ChangeStagingEvent,
	errChan chan<- error,
) error {
	return m.DeployEngine.StreamChangeStagingEvents(ctx, changesetID, lastEventID, streamTo, errChan)
}

func (m *mockEngineWithInstances) ApplyReconciliation(
	ctx context.Context,
	instanceID string,
	payload *types.ApplyReconciliationPayload,
) (*container.ApplyReconciliationResult, error) {
	return m.DeployEngine.ApplyReconciliation(ctx, instanceID, payload)
}
