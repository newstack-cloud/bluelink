package validateui

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type ValidateTUISuite struct {
	suite.Suite
}

func (s *ValidateTUISuite) Test_successful_validation() {
	mainModel, err := NewValidateApp(
		testutils.NewTestDeployEngine(testValidationEvents(validationSuccess)),
		zap.NewNop(),
		"test.blueprint.yaml",
		/* isDefaultBlueprintFile */ false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		/* headless */ false,
		os.Stdout,
	)
	if err != nil {
		s.FailNow("failed to create main model: %v", err)
	}

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"This is a test informational diagnostic",
		"This is a test warning diagnostic",
	)

	testutils.KeyQ(testModel)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Equal(finalModel.Error, nil)
}

func (s *ValidateTUISuite) Test_successful_validation_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewValidateApp(
		testutils.NewTestDeployEngine(testValidationEvents(validationSuccess)),
		zap.NewNop(),
		"test.blueprint.yaml",
		/* isDefaultBlueprintFile */ false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		/* headless */ true,
		headlessOutput,
	)
	if err != nil {
		s.FailNow("failed to create main model: %v", err)
	}

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"This is a test informational diagnostic",
		"This is a test warning diagnostic",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *ValidateTUISuite) Test_validation_failed() {
	mainModel, err := NewValidateApp(
		testutils.NewTestDeployEngine(testValidationEvents(validationFailed)),
		zap.NewNop(),
		"test.blueprint.yaml",
		/* isDefaultBlueprintFile */ false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		/* headless */ false,
		os.Stdout,
	)
	if err != nil {
		s.FailNow("failed to create main model: %v", err)
	}

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"This is a test informational diagnostic",
		"This is a test warning diagnostic",
		"function \"test-function\" not found",
		"Install provider",
		"Install the provider for the function",
		"Explore providers in the official registry",
		"https://registry.bluelink.dev/providers",
	)

	testutils.KeyQ(testModel)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.NotNil(finalModel.Error)
	s.Equal(finalModel.Error.Error(), "validation failed")
}

func (s *ValidateTUISuite) Test_validation_failed_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewValidateApp(
		testutils.NewTestDeployEngine(testValidationEvents(validationFailed)),
		zap.NewNop(),
		"test.blueprint.yaml",
		/* isDefaultBlueprintFile */ false,
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		/* headless */ true,
		headlessOutput,
	)
	if err != nil {
		s.FailNow("failed to create main model: %v", err)
	}

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"This is a test warning diagnostic",
		"This is a test informational diagnostic",
		"function \"test-function\" not found",
		"Install provider",
		"Install the provider for the function",
		"Explore providers in the official registry",
		"https://registry.bluelink.dev/providers",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

type testValidationType string

const (
	validationFailed  testValidationType = "failed"
	validationSuccess testValidationType = "success"
)

func testValidationEvents(validationType testValidationType) []*types.BlueprintValidationEvent {
	events := []*types.BlueprintValidationEvent{
		{
			ID: "test-diagnostic-id-1",
			Diagnostic: core.Diagnostic{
				Level:   core.DiagnosticLevelInfo,
				Message: "This is a test informational diagnostic",
			},
		},
		{
			ID: "test-diagnostic-id-2",
			Diagnostic: core.Diagnostic{
				Level:   core.DiagnosticLevelWarning,
				Message: "This is a test warning diagnostic",
			},
			End: validationType == validationSuccess,
		},
	}

	if validationType == validationFailed {
		events = append(events, &types.BlueprintValidationEvent{
			ID: "test-diagnostic-id-3",
			Diagnostic: core.Diagnostic{
				Level:   core.DiagnosticLevelError,
				Message: "function \"test-function\" not found",
				Context: &errors.ErrorContext{
					Category:   errors.ErrorCategoryProvider,
					ReasonCode: provider.ErrorReasonCodeFunctionNotFound,
					SuggestedActions: []errors.SuggestedAction{
						{
							Type:        string(errors.ActionTypeInstallProvider),
							Title:       "Install provider",
							Description: "Install the provider for the function",
							Priority:    1,
						},
					},
					Metadata: map[string]any{
						"functionName": "test-function",
					},
				},
			},
			End: true,
		})
	}

	return events
}

func TestValidateTUISuite(t *testing.T) {
	suite.Run(t, new(ValidateTUISuite))
}
