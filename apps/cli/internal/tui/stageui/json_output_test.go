package stageui

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type JSONOutputTestSuite struct {
	suite.Suite
}

func TestJSONOutputTestSuite(t *testing.T) {
	suite.Run(t, new(JSONOutputTestSuite))
}

func (s *JSONOutputTestSuite) Test_outputJSON_includes_changeset_id() {
	jsonOutput := &bytes.Buffer{}
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			[]*types.ChangeStagingEvent{
				resourceCreateEvent("test-resource"),
				completeChangesEvent(),
			},
			"test-changeset-json-123",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		jsonOutput,
		true, // jsonMode
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.StageOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)
	s.True(output.Success)
	s.Equal("test-changeset-json-123", output.ChangesetID)
	s.Equal("test-instance", output.InstanceName)
}

func (s *JSONOutputTestSuite) Test_outputJSON_includes_resource_summary() {
	jsonOutput := &bytes.Buffer{}
	events := []*types.ChangeStagingEvent{
		resourceCreateEvent("resource-1"),
		resourceCreateEvent("resource-2"),
		resourceUpdateEvent("resource-3"),
		resourceDeleteEvent("resource-4"),
		completeChangesEvent(),
	}

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(events, "test-changeset-summary"),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.StageOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal(4, output.Summary.Resources.Total)
	s.Equal(2, output.Summary.Resources.Create)
	s.Equal(1, output.Summary.Resources.Update)
	s.Equal(1, output.Summary.Resources.Delete)
}

func (s *JSONOutputTestSuite) Test_outputJSON_includes_child_summary() {
	jsonOutput := &bytes.Buffer{}
	events := []*types.ChangeStagingEvent{
		childChangesEvent("child-1"),
		childChangesEvent("child-2"),
		completeChangesEvent(),
	}

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(events, "test-changeset-children"),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.StageOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal(2, output.Summary.Children.Total)
	s.Equal(2, output.Summary.Children.Create)
}

func (s *JSONOutputTestSuite) Test_outputJSON_includes_link_summary() {
	jsonOutput := &bytes.Buffer{}
	events := []*types.ChangeStagingEvent{
		linkChangesEvent("resource-a", "resource-b"),
		linkChangesEvent("resource-c", "resource-d"),
		completeChangesEvent(),
	}

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(events, "test-changeset-links"),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.StageOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal(2, output.Summary.Links.Total)
	s.Equal(2, output.Summary.Links.Create)
}

func (s *JSONOutputTestSuite) Test_outputJSON_includes_export_summary() {
	jsonOutput := &bytes.Buffer{}

	completeEvent := &types.ChangeStagingEvent{
		CompleteChanges: &types.CompleteChangesEventData{
			Changes: &changes.BlueprintChanges{
				NewExports: map[string]provider.FieldChange{
					"export1": {NewValue: stringMappingNode("value1")},
					"export2": {NewValue: stringMappingNode("value2")},
				},
				ExportChanges: map[string]provider.FieldChange{
					"export3": {
						PrevValue: stringMappingNode("old"),
						NewValue:  stringMappingNode("new"),
					},
				},
				RemovedExports: []string{"export4"},
			},
		},
	}

	events := []*types.ChangeStagingEvent{
		resourceCreateEvent("test-resource"),
		completeEvent,
	}

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(events, "test-changeset-exports"),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.StageOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal(2, output.Summary.Exports.New)
	s.Equal(1, output.Summary.Exports.Modified)
	s.Equal(1, output.Summary.Exports.Removed)
}

func (s *JSONOutputTestSuite) Test_outputJSONError_formats_validation_errors() {
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

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStagingError(validationErr),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.ErrorOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.False(output.Success)
	s.Equal("validation", output.Error.Type)
	s.Len(output.Error.Diagnostics, 1)
	s.Equal("Name is required", output.Error.Diagnostics[0].Message)
}

func (s *JSONOutputTestSuite) Test_outputJSONError_formats_stream_errors() {
	jsonOutput := &bytes.Buffer{}

	streamErr := &engineerrors.StreamError{
		Event: &types.StreamErrorMessageEvent{
			Message: "Resource deployment failed",
			Diagnostics: []*core.Diagnostic{
				{
					Level:   core.DiagnosticLevelError,
					Message: "Missing required field",
					Range: &core.DiagnosticRange{
						Start: &source.Meta{Position: source.Position{Line: 10, Column: 5}},
					},
				},
			},
		},
	}

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStagingError(streamErr),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.ErrorOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.False(output.Success)
	s.Equal("stream", output.Error.Type)
	s.Len(output.Error.Diagnostics, 1)
	s.Equal("Missing required field", output.Error.Diagnostics[0].Message)
}

func (s *JSONOutputTestSuite) Test_outputJSON_includes_recreate_count() {
	jsonOutput := &bytes.Buffer{}
	events := []*types.ChangeStagingEvent{
		resourceRecreateEvent("resource-1"),
		resourceRecreateEvent("resource-2"),
		completeChangesEvent(),
	}

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(events, "test-changeset-recreate"),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.StageOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal(2, output.Summary.Resources.Recreate)
}

func (s *JSONOutputTestSuite) Test_outputJSONDrift_includes_reconciliation_result() {
	jsonOutput := &bytes.Buffer{}

	driftEvent := &types.ChangeStagingEvent{
		DriftDetected: &types.DriftDetectedEventData{
			ReconciliationResult: &container.ReconciliationCheckResult{
				Resources: []container.ResourceReconcileResult{
					{
						ResourceName: "drifted-resource",
						ResourceType: "aws/sqs/queue",
						Type:         container.ReconciliationTypeDrift,
					},
				},
			},
			Message: "Drift detected in resource state",
		},
	}

	events := []*types.ChangeStagingEvent{driftEvent}

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(events, "test-changeset-drift"),
		zap.NewNop(),
		"test-instance-id",
		"test-instance",
		false,
		false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		jsonOutput,
		true,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	var output jsonout.StageDriftOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.True(output.Success)
	s.True(output.DriftDetected)
	s.Equal("test-instance-id", output.InstanceID)
	s.Equal("Drift detected in resource state", output.Message)
	s.NotNil(output.Reconciliation)
	s.Len(output.Reconciliation.Resources, 1)
	s.Equal("drifted-resource", output.Reconciliation.Resources[0].ResourceName)
}
