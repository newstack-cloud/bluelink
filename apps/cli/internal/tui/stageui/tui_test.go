package stageui

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
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

type StageTUISuite struct {
	suite.Suite
}

func TestStageTUISuite(t *testing.T) {
	suite.Run(t, new(StageTUISuite))
}

// --- Test Event Factories ---

// testStagingType determines the test scenario
type testStagingType string

const (
	stagingSuccessCreate   testStagingType = "success_create"
	stagingSuccessUpdate   testStagingType = "success_update"
	stagingSuccessDelete   testStagingType = "success_delete"
	stagingSuccessRecreate testStagingType = "success_recreate"
	stagingSuccessMixed    testStagingType = "success_mixed"
	stagingSuccessChild    testStagingType = "success_child"
	stagingSuccessLink     testStagingType = "success_link"
)

func testStagingEvents(stagingType testStagingType) []*types.ChangeStagingEvent {
	switch stagingType {
	case stagingSuccessCreate:
		return []*types.ChangeStagingEvent{
			resourceCreateEvent("test-resource"),
			completeChangesEvent(),
		}
	case stagingSuccessUpdate:
		return []*types.ChangeStagingEvent{
			resourceUpdateEvent("test-resource"),
			completeChangesEvent(),
		}
	case stagingSuccessDelete:
		return []*types.ChangeStagingEvent{
			resourceDeleteEvent("test-resource"),
			completeChangesEvent(),
		}
	case stagingSuccessRecreate:
		return []*types.ChangeStagingEvent{
			resourceRecreateEvent("test-resource"),
			completeChangesEvent(),
		}
	case stagingSuccessMixed:
		return []*types.ChangeStagingEvent{
			resourceCreateEvent("resource-a"),
			resourceUpdateEvent("resource-b"),
			childChangesEvent("child-blueprint"),
			linkChangesEvent("resource-a", "resource-b"),
			completeChangesEvent(),
		}
	case stagingSuccessChild:
		return []*types.ChangeStagingEvent{
			childChangesEvent("child-blueprint"),
			completeChangesEvent(),
		}
	case stagingSuccessLink:
		return []*types.ChangeStagingEvent{
			linkChangesEvent("resource-a", "resource-b"),
			completeChangesEvent(),
		}
	default:
		return []*types.ChangeStagingEvent{completeChangesEvent()}
	}
}

func resourceCreateEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          true,
				Changes: provider.Changes{
					NewFields: []provider.FieldChange{
						{FieldPath: "spec.field1", NewValue: stringMappingNode("value1")},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func resourceUpdateEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          false,
				Removed:      false,
				Changes: provider.Changes{
					ModifiedFields: []provider.FieldChange{
						{
							FieldPath: "spec.replicas",
							PrevValue: intMappingNode(2),
							NewValue:  intMappingNode(4),
						},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func resourceDeleteEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          false,
				Removed:      true,
				Changes:      provider.Changes{},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func resourceRecreateEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ResourceChanges: &types.ResourceChangesEventData{
			ResourceChangesMessage: container.ResourceChangesMessage{
				ResourceName: name,
				New:          false,
				Removed:      false,
				Changes: provider.Changes{
					MustRecreate: true,
					ModifiedFields: []provider.FieldChange{
						{
							FieldPath: "spec.immutableField",
							PrevValue: stringMappingNode("old"),
							NewValue:  stringMappingNode("new"),
						},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func childChangesEvent(name string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		ChildChanges: &types.ChildChangesEventData{
			ChildChangesMessage: container.ChildChangesMessage{
				ChildBlueprintName: name,
				New:                true,
				Changes: changes.BlueprintChanges{
					NewResources: map[string]provider.Changes{
						"child-resource-1": {},
						"child-resource-2": {},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func linkChangesEvent(resourceA, resourceB string) *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		LinkChanges: &types.LinkChangesEventData{
			LinkChangesMessage: container.LinkChangesMessage{
				ResourceAName: resourceA,
				ResourceBName: resourceB,
				New:           true,
				Changes: provider.LinkChanges{
					NewFields: []*provider.FieldChange{
						{FieldPath: "link.field1", NewValue: stringMappingNode("linked")},
					},
				},
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

func completeChangesEvent() *types.ChangeStagingEvent {
	return &types.ChangeStagingEvent{
		CompleteChanges: &types.CompleteChangesEventData{
			Changes: &changes.BlueprintChanges{},
		},
	}
}

// Helper functions for MappingNode creation
func stringMappingNode(s string) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &s},
	}
}

func intMappingNode(i int) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{IntValue: &i},
	}
}

// --- Interactive Mode Tests ---

func (s *StageTUISuite) Test_successful_staging_with_resource_create() {
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessCreate),
			"test-changeset-123",
		),
		zap.NewNop(),
		"",              // instanceID
		"test-instance", // instanceName
		false,           // destroy
		false,           // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false, // headless
		os.Stdout,
		false, // jsonMode
	)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Send blueprint selection to trigger staging
	testModel.Send(sharedui.SelectBlueprintMsg{
		BlueprintFile: "test.blueprint.yaml",
		Source:        consts.BlueprintSourceFile,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"CREATE",
		"test-changeset-123",
		"bluelink deploy",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.err)
	s.True(finalModel.finished)
	s.Equal("test-changeset-123", finalModel.changesetID)
}

func (s *StageTUISuite) Test_successful_staging_with_resource_update() {
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessUpdate),
			"test-changeset-456",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false,
		os.Stdout,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"UPDATE",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.err)
}

func (s *StageTUISuite) Test_successful_staging_with_resource_delete() {
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessDelete),
			"test-changeset-789",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false,
		os.Stdout,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"DELETE",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.err)
}

func (s *StageTUISuite) Test_successful_staging_with_resource_recreate() {
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessRecreate),
			"test-changeset-recreate",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false,
		os.Stdout,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-resource",
		"RECREATE",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.err)
}

func (s *StageTUISuite) Test_successful_staging_with_child_blueprint() {
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessChild),
			"test-changeset-child",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false,
		os.Stdout,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"child-blueprint",
		"Child Blueprints",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.err)
}

func (s *StageTUISuite) Test_successful_staging_with_link() {
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessLink),
			"test-changeset-link",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false,
		os.Stdout,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-a::resource-b",
		"Links",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.err)
}

func (s *StageTUISuite) Test_successful_staging_with_mixed_items() {
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessMixed),
			"test-changeset-mixed",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false,
		os.Stdout,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"resource-a",
		"resource-b",
		"child-blueprint",
		"resource-a::resource-b",
		"Resources",
		"Child Blueprints",
		"Links",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.Nil(finalModel.err)
	s.Len(finalModel.items, 4)
}

// --- Headless Mode Tests ---

func (s *StageTUISuite) Test_successful_staging_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessCreate),
			"test-changeset-headless",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true, // headless
		headlessOutput,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"[stage]",
		"test-resource",
		"CREATE",
		"test-changeset-headless",
		"bluelink deploy",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_shows_progress() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessMixed),
			"test-changeset-progress",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		headlessOutput,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"✓ resource: resource-a - CREATE",
		"✓ resource: resource-b - UPDATE",
		"✓ child: child-blueprint - CREATE",
		"✓ link: resource-a::resource-b - CREATE",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_shows_summary() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessMixed),
			"test-changeset-summary",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		headlessOutput,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Change staging complete",
		"2 resources",
		"1 child",
		"1 link",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_headless_shows_field_changes() {
	headlessOutput := testutils.NewSaveBuffer()
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessUpdate),
			"test-changeset-fields",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		headlessOutput,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"~",           // Modified field indicator
		"spec.replicas",
		"2",
		"4",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

// --- Error Handling Tests ---

func (s *StageTUISuite) Test_staging_validation_error() {
	validationErr := &engineerrors.ClientError{
		StatusCode: 400,
		Message:    "Validation failed",
		ValidationDiagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "Resource 'test-resource' is invalid",
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
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false,
		os.Stdout,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Failed to create changeset",
		"Resource 'test-resource' is invalid",
		"[line 10, col 5]",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.NotNil(finalModel.err)
}

func (s *StageTUISuite) Test_staging_validation_error_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	validationErr := &engineerrors.ClientError{
		StatusCode: 400,
		Message:    "Validation failed",
		ValidationDiagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "Blueprint validation failed",
			},
		},
	}

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStagingError(validationErr),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true,
		headlessOutput,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"[stage]",
		"Failed to create changeset",
		"Blueprint validation failed",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StageTUISuite) Test_staging_generic_error() {
	genericErr := &engineerrors.ClientError{
		StatusCode: 500,
		Message:    "Internal server error",
	}

	model := NewStageModel(
		testutils.NewTestDeployEngineWithStagingError(genericErr),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false,
		os.Stdout,
		false, // jsonMode
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

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Error during change staging",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(StageModel)
	s.NotNil(finalModel.err)
}

// --- Navigation Tests ---

func (s *StageTUISuite) Test_quit_with_q() {
	model := NewStageModel(
		testutils.NewTestDeployEngineWithStaging(
			testStagingEvents(stagingSuccessCreate),
			"test-changeset-quit",
		),
		zap.NewNop(),
		"",
		"test-instance",
		false,
		false, // skipDriftCheck
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false,
		os.Stdout,
		false, // jsonMode
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

	// Wait for staging to complete
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"test-changeset-quit",
	)

	// Press q to quit
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}
