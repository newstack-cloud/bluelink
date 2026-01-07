package destroyui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type ViewTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestViewTestSuite(t *testing.T) {
	suite.Run(t, new(ViewTestSuite))
}

func (s *ViewTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		styles.NewBluelinkPalette(),
	)
}

// createTestModel creates a model using the public constructor.
func (s *ViewTestSuite) createTestModel() DestroyModel {
	return NewDestroyModel(
		testutils.NewTestDeployEngineWithDeployment(
			[]*types.BlueprintInstanceEvent{},
			"inst-123",
			&state.InstanceState{InstanceID: "inst-123"},
		),
		zap.NewNop(),
		"cs-456",
		"inst-123",
		"test-instance",
		false,
		s.testStyles,
		false,
		os.Stdout,
		nil,
		false,
	)
}

// createTestModelWithWindowSize creates a model and sets the window size.
func (s *ViewTestSuite) createTestModelWithWindowSize() DestroyModel {
	model := s.createTestModel()
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return updatedModel.(DestroyModel)
}

// --- View with error state tests ---

func (s *ViewTestSuite) Test_View_with_validation_error_shows_validation_details() {
	model := s.createTestModelWithWindowSize()

	err := &engineerrors.ClientError{
		StatusCode: 422,
		Message:    "Validation failed",
		ValidationErrors: []*engineerrors.ValidationError{
			{Location: "resources.myRes", Message: "invalid field"},
		},
	}

	updatedModel, _ := model.Update(DestroyErrorMsg{Err: err})
	resultModel := updatedModel.(DestroyModel)
	output := resultModel.View()

	s.Contains(output, "Validation")
}

func (s *ViewTestSuite) Test_View_with_stream_error_shows_error_message() {
	model := s.createTestModelWithWindowSize()

	err := &engineerrors.StreamError{
		Event: &types.StreamErrorMessageEvent{
			Message: "Stream error occurred",
		},
	}

	updatedModel, _ := model.Update(DestroyErrorMsg{Err: err})
	resultModel := updatedModel.(DestroyModel)
	output := resultModel.View()

	s.Contains(output, "Stream error")
}

func (s *ViewTestSuite) Test_View_with_generic_error_shows_error_message() {
	model := s.createTestModelWithWindowSize()

	updatedModel, _ := model.Update(DestroyErrorMsg{Err: &viewTestError{message: "Something went wrong"}})
	resultModel := updatedModel.(DestroyModel)
	output := resultModel.View()

	s.Contains(output, "Something went wrong")
}

// --- View with deploy changeset error tests ---

func (s *ViewTestSuite) Test_View_with_deploy_changeset_error_shows_mismatch_message() {
	model := s.createTestModelWithWindowSize()

	updatedModel, _ := model.Update(DeployChangesetErrorMsg{})
	resultModel := updatedModel.(DestroyModel)
	output := resultModel.View()

	s.NotEmpty(output)
	s.Contains(output, "deploy changeset")
}

// --- View with overview showing tests ---

func (s *ViewTestSuite) Test_View_overview_shows_destroyed_header() {
	model := s.createTestModelWithWindowSize()

	// Send finish event to complete destroy
	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	// Open overview with 'o' key
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "Destroy Summary")
}

func (s *ViewTestSuite) Test_View_overview_shows_failed_header() {
	model := s.createTestModelWithWindowSize()

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyFailed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "Destroy Failed")
}

func (s *ViewTestSuite) Test_View_overview_shows_rollback_header() {
	model := s.createTestModelWithWindowSize()

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyRollbackComplete))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "Destroy Rolled Back")
}

func (s *ViewTestSuite) Test_View_overview_shows_interrupted_header() {
	model := s.createTestModelWithWindowSize()

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyInterrupted))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "Destroy Interrupted")
}

func (s *ViewTestSuite) Test_View_overview_shows_instance_info() {
	model := s.createTestModelWithWindowSize()

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "test-instance")
	s.Contains(output, "inst-123")
	s.Contains(output, "cs-456")
}

// --- View with pre-destroy state tests ---

func (s *ViewTestSuite) Test_View_pre_destroy_state_handles_nil_state() {
	model := s.createTestModelWithWindowSize()

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	// Try to open pre-destroy state view without setting state
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model = updatedModel.(DestroyModel)

	// View should not show pre-destroy state header since state is nil
	output := model.View()
	s.NotContains(output, "Pre-Destroy Instance State")
}

func (s *ViewTestSuite) Test_View_pre_destroy_state_shows_header() {
	model := s.createTestModelWithWindowSize()

	model.SetPreDestroyInstanceState(&state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
	})

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "Pre-Destroy Instance State")
}

func (s *ViewTestSuite) Test_View_pre_destroy_state_shows_resources() {
	model := s.createTestModelWithWindowSize()

	model.SetPreDestroyInstanceState(&state.InstanceState{
		InstanceID: "inst-123",
		ResourceIDs: map[string]string{
			"myBucket": "res-bucket-123",
		},
		Resources: map[string]*state.ResourceState{
			"res-bucket-123": {
				ResourceID: "res-bucket-123",
				Name:       "myBucket",
				Type:       "aws/s3/bucket",
			},
		},
	})

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "Resources")
	s.Contains(output, "myBucket")
}

func (s *ViewTestSuite) Test_View_pre_destroy_state_shows_links() {
	model := s.createTestModelWithWindowSize()

	model.SetPreDestroyInstanceState(&state.InstanceState{
		InstanceID: "inst-123",
		Links: map[string]*state.LinkState{
			"resA::resB": {LinkID: "link-123"},
		},
	})

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "Links")
	s.Contains(output, "resA::resB")
}

func (s *ViewTestSuite) Test_View_pre_destroy_state_shows_exports() {
	model := s.createTestModelWithWindowSize()

	val := "exported-value"
	model.SetPreDestroyInstanceState(&state.InstanceState{
		InstanceID: "inst-123",
		Exports: map[string]*state.ExportState{
			"myExport": {
				Value: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &val}},
			},
		},
	})

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "Exports")
	s.Contains(output, "myExport")
}

func (s *ViewTestSuite) Test_View_pre_destroy_state_shows_children() {
	model := s.createTestModelWithWindowSize()

	model.SetPreDestroyInstanceState(&state.InstanceState{
		InstanceID: "inst-123",
		ChildBlueprints: map[string]*state.InstanceState{
			"childBlueprint": {
				InstanceID:   "child-inst-456",
				InstanceName: "childBlueprint",
			},
		},
	})

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	s.Contains(output, "Child Blueprints")
	s.Contains(output, "childBlueprint")
	s.Contains(output, "child-inst-456")
}

// --- View shows spec and computed fields separately ---

func (s *ViewTestSuite) Test_View_pre_destroy_state_shows_spec_and_outputs_separately() {
	model := s.createTestModelWithWindowSize()

	inputVal := "input-value"
	outputVal := "output-value"
	model.SetPreDestroyInstanceState(&state.InstanceState{
		InstanceID: "inst-123",
		ResourceIDs: map[string]string{
			"myResource": "res-123",
		},
		Resources: map[string]*state.ResourceState{
			"res-123": {
				ResourceID: "res-123",
				Name:       "myResource",
				Type:       "aws/lambda/function",
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"inputField":  {Scalar: &core.ScalarValue{StringValue: &inputVal}},
						"outputField": {Scalar: &core.ScalarValue{StringValue: &outputVal}},
					},
				},
				ComputedFields: []string{"outputField"},
			},
		},
	})

	finishEvt := DestroyEventMsg(*createViewFinishEvent(core.InstanceStatusDestroyed))
	updatedModel, _ := model.Update(finishEvt)
	model = updatedModel.(DestroyModel)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model = updatedModel.(DestroyModel)

	output := model.View()
	// Both Spec and Outputs sections should appear
	s.Contains(output, "Spec")
	s.Contains(output, "Outputs")
	// Both fields should appear in output (spec filtering happens internally)
	s.Contains(output, "inputField")
	s.Contains(output, "outputField")
}

// --- Helper functions for creating test events ---

func createViewFinishEvent(status core.InstanceStatus) *types.BlueprintInstanceEvent {
	return &types.BlueprintInstanceEvent{
		DeployEvent: container.DeployEvent{
			FinishEvent: &container.DeploymentFinishedMessage{
				Status:      status,
				EndOfStream: true,
			},
		},
	}
}

// Helper type for testing (unique name to avoid redeclaration)
type viewTestError struct {
	message string
}

func (e *viewTestError) Error() string {
	return e.message
}
