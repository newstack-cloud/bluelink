package stateexportui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/stateio"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
)

type StateExportTUISuite struct {
	suite.Suite
	tempDir      string
	stateDir     string
	outputFile   string
	engineConfig *stateio.EngineConfig
}

func (s *StateExportTUISuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "stateexport-tui-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	s.stateDir = filepath.Join(tempDir, "state")
	err = os.MkdirAll(s.stateDir, 0755)
	s.Require().NoError(err)

	s.engineConfig = &stateio.EngineConfig{
		State: stateio.StateConfig{
			StorageEngine:   stateio.StorageEngineMemfile,
			MemFileStateDir: s.stateDir,
		},
	}

	// Set up an output file path for export
	s.outputFile = filepath.Join(tempDir, "exported.json")

	// Pre-populate the memfile state with test data by importing
	inputFile := filepath.Join(tempDir, "input.json")
	instancesJSON := `[{"id":"inst-1","name":"Test Instance","status":2}]`
	err = os.WriteFile(inputFile, []byte(instancesJSON), 0644)
	s.Require().NoError(err)

	// Import the test data
	_, err = stateio.Import(stateio.ImportParams{
		FilePath:     inputFile,
		EngineConfig: s.engineConfig,
	})
	s.Require().NoError(err)
}

func (s *StateExportTUISuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *StateExportTUISuite) Test_successful_export() {
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       false,
		HeadlessWriter: os.Stdout,
		JSONMode:       false,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Export complete",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)

	// Verify the output file was created
	_, err = os.Stat(s.outputFile)
	s.NoError(err)
}

func (s *StateExportTUISuite) Test_successful_export_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       true,
		HeadlessWriter: headlessOutput,
		JSONMode:       false,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Successfully exported",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateExportTUISuite) Test_json_output_mode() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       true,
		HeadlessWriter: headlessOutput,
		JSONMode:       true,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		`"success": true`,
		`"instancesCount"`,
		`"mode": "export"`,
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateExportTUISuite) Test_export_with_instance_filter() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:        s.outputFile,
		InstanceFilters: []string{"inst-1"},
		EngineConfig:    s.engineConfig,
		Styles:          stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:        true,
		HeadlessWriter:  headlessOutput,
		JSONMode:        true,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		`"success": true`,
		`"instancesCount": 1`,
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateExportTUISuite) Test_export_error_invalid_instance() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:        s.outputFile,
		InstanceFilters: []string{"nonexistent-instance"},
		EngineConfig:    s.engineConfig,
		Styles:          stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:        true,
		HeadlessWriter:  headlessOutput,
		JSONMode:        true,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		`"success": false`,
		`"error"`,
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *StateExportTUISuite) Test_ctrl_c_quits() {
	mainModel, err := NewStateExportApp(StateExportAppConfig{
		FilePath:       s.outputFile,
		EngineConfig:   s.engineConfig,
		Styles:         stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette()),
		Headless:       false,
		HeadlessWriter: os.Stdout,
		JSONMode:       false,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for some output to ensure the model is running
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Export",
	)

	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.True(finalModel.quitting)
}

func TestStateExportTUISuite(t *testing.T) {
	suite.Run(t, new(StateExportTUISuite))
}
