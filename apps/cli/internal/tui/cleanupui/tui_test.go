package cleanupui

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type CleanupTUISuite struct {
	suite.Suite
}

func (s *CleanupTUISuite) Test_successful_cleanup_all_types() {
	mainModel, err := NewCleanupApp(
		testutils.NewTestDeployEngine(nil),
		zap.NewNop(),
		true,  // cleanupValidations
		true,  // cleanupChangesets
		true,  // cleanupReconciliationResults
		true,  // cleanupEvents
		false, // showOptionsForm
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false, // headless
		os.Stdout,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Cleanup complete",
		"validations",
		"changesets",
		"reconciliation results",
		"events",
		"items deleted",
		"Press q to quit",
	)

	testutils.KeyQ(testModel)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
}

func (s *CleanupTUISuite) Test_successful_cleanup_specific_types() {
	mainModel, err := NewCleanupApp(
		testutils.NewTestDeployEngine(nil),
		zap.NewNop(),
		true,  // cleanupValidations
		false, // cleanupChangesets
		false, // cleanupReconciliationResults
		true,  // cleanupEvents
		false, // showOptionsForm
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false, // headless
		os.Stdout,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Cleanup complete",
		"validations",
		"events",
		"Press q to quit",
	)

	testutils.KeyQ(testModel)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
}

func (s *CleanupTUISuite) Test_successful_cleanup_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewCleanupApp(
		testutils.NewTestDeployEngine(nil),
		zap.NewNop(),
		true,  // cleanupValidations
		true,  // cleanupChangesets
		true,  // cleanupReconciliationResults
		true,  // cleanupEvents
		false, // showOptionsForm
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		true, // headless
		headlessOutput,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Cleanup complete",
		"validations",
		"changesets",
		"reconciliation results",
		"events",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *CleanupTUISuite) Test_no_cleanup_types_selected() {
	mainModel, err := NewCleanupApp(
		testutils.NewTestDeployEngine(nil),
		zap.NewNop(),
		false, // cleanupValidations
		false, // cleanupChangesets
		false, // cleanupReconciliationResults
		false, // cleanupEvents
		false, // showOptionsForm
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false, // headless
		os.Stdout,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Cleanup complete",
		"Press q to quit",
	)

	testutils.KeyQ(testModel)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
}

func (s *CleanupTUISuite) Test_options_form_shown_when_configured() {
	mainModel, err := NewCleanupApp(
		testutils.NewTestDeployEngine(nil),
		zap.NewNop(),
		false, // cleanupValidations (will be set by form)
		false, // cleanupChangesets (will be set by form)
		false, // cleanupReconciliationResults (will be set by form)
		false, // cleanupEvents (will be set by form)
		true,  // showOptionsForm
		stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		false, // headless
		os.Stdout,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Select resource types to clean up",
		"Cleanup Validations",
		"Cleanup Changesets",
		"Cleanup Reconciliation Results",
		"Cleanup Events",
	)

	testutils.KeyQ(testModel)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func TestCleanupTUISuite(t *testing.T) {
	suite.Run(t, new(CleanupTUISuite))
}
