package initui

import (
	"errors"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/project"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
)

// mockGit is a mock implementation of the git.Git interface for testing.
type mockGit struct {
	cloneErr   error
	initErr    error
	initCalled bool
}

func (m *mockGit) Clone(repoURL string, directory string) error {
	return m.cloneErr
}

func (m *mockGit) Init(directory string) error {
	m.initCalled = true
	return m.initErr
}

// mockPreparer is a mock implementation of the project.Preparer interface for testing.
type mockPreparer struct {
	removeGitHistoryErr       error
	removeMaintainerFilesErr  error
	selectBlueprintFormatErr  error
	substitutePlaceholdersErr error
}

func (m *mockPreparer) RemoveGitHistory(directory string) error {
	return m.removeGitHistoryErr
}

func (m *mockPreparer) RemoveMaintainerFiles(directory string) error {
	return m.removeMaintainerFilesErr
}

func (m *mockPreparer) SelectBlueprintFormat(directory string, format string) error {
	return m.selectBlueprintFormatErr
}

func (m *mockPreparer) SubstitutePlaceholders(directory string, values project.TemplateValues) error {
	return m.substitutePlaceholdersErr
}

// Helper functions

func defaultInitialState() InitialState {
	noGit := true
	return InitialState{
		Template:                 "scaffold",
		IsDefaultTemplate:        false,
		ProjectName:              "TestProject",
		BlueprintFormat:          "yaml",
		IsDefaultBlueprintFormat: false,
		NoGit:                    &noGit,
		IsDefaultNoGit:           false,
		Directory:                "./test-project",
	}
}

func newTestStyles() *stylespkg.Styles {
	return stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)
}

// InitTUISuite is the test suite for the init TUI.
type InitTUISuite struct {
	suite.Suite
}

func (s *InitTUISuite) Test_successful_init() {
	mainModel, err := NewInitApp(
		defaultInitialState(),
		newTestStyles(),
		&mockGit{},
		&mockPreparer{},
		/* headless */ false,
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
		"Project initialized successfully",
		"TestProject",
		"./test-project",
	)

	// Press any key to exit completion screen
	testutils.KeyQ(testModel)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InitModel)
	s.Nil(finalModel.Error)
}

func (s *InitTUISuite) Test_successful_init_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	mainModel, err := NewInitApp(
		defaultInitialState(),
		newTestStyles(),
		&mockGit{},
		&mockPreparer{},
		/* headless */ true,
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
		"Project initialized successfully",
		"TestProject",
		"./test-project",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InitModel)
	s.Nil(finalModel.Error)
}

func (s *InitTUISuite) Test_download_error() {
	mockGitService := &mockGit{
		cloneErr: errors.New("git clone failed: network error"),
	}

	mainModel, err := NewInitApp(
		defaultInitialState(),
		newTestStyles(),
		mockGitService,
		&mockPreparer{},
		/* headless */ false,
		os.Stdout,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InitModel)
	s.NotNil(finalModel.Error)
	s.Contains(finalModel.Error.Error(), "git clone failed")
}

func (s *InitTUISuite) Test_download_error_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	mockGitService := &mockGit{
		cloneErr: errors.New("git clone failed: network error"),
	}

	mainModel, err := NewInitApp(
		defaultInitialState(),
		newTestStyles(),
		mockGitService,
		&mockPreparer{},
		/* headless */ true,
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
		"Error:",
		"git clone failed",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *InitTUISuite) Test_prepare_error() {
	mockPreparerService := &mockPreparer{
		removeGitHistoryErr: errors.New("failed to remove .git directory"),
	}

	mainModel, err := NewInitApp(
		defaultInitialState(),
		newTestStyles(),
		&mockGit{},
		mockPreparerService,
		/* headless */ false,
		os.Stdout,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InitModel)
	s.NotNil(finalModel.Error)
	s.Contains(finalModel.Error.Error(), "failed to remove .git directory")
}

func (s *InitTUISuite) Test_prepare_error_headless() {
	headlessOutput := testutils.NewSaveBuffer()
	mockPreparerService := &mockPreparer{
		removeGitHistoryErr: errors.New("failed to remove .git directory"),
	}

	mainModel, err := NewInitApp(
		defaultInitialState(),
		newTestStyles(),
		&mockGit{},
		mockPreparerService,
		/* headless */ true,
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
		"Error:",
		"failed to remove .git directory",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *InitTUISuite) Test_init_without_git() {
	mockGitService := &mockGit{}
	noGit := true
	initialState := InitialState{
		Template:                 "scaffold",
		IsDefaultTemplate:        false,
		ProjectName:              "TestProject",
		BlueprintFormat:          "yaml",
		IsDefaultBlueprintFormat: false,
		NoGit:                    &noGit,
		IsDefaultNoGit:           false,
		Directory:                "./test-project",
	}

	mainModel, err := NewInitApp(
		initialState,
		newTestStyles(),
		mockGitService,
		&mockPreparer{},
		/* headless */ false,
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
		"Project initialized successfully",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	// Verify git.Init was NOT called
	s.False(mockGitService.initCalled, "git.Init should not be called when noGit is true")
}

func (s *InitTUISuite) Test_init_with_git() {
	mockGitService := &mockGit{}
	noGit := false
	initialState := InitialState{
		Template:                 "scaffold",
		IsDefaultTemplate:        false,
		ProjectName:              "TestProject",
		BlueprintFormat:          "yaml",
		IsDefaultBlueprintFormat: false,
		NoGit:                    &noGit,
		IsDefaultNoGit:           false,
		Directory:                "./test-project",
	}

	mainModel, err := NewInitApp(
		initialState,
		newTestStyles(),
		mockGitService,
		&mockPreparer{},
		/* headless */ false,
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
		"Project initialized successfully",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	// Verify git.Init WAS called
	s.True(mockGitService.initCalled, "git.Init should be called when noGit is false")
}

func (s *InitTUISuite) Test_ctrl_c_cancellation() {
	// Use an initial state that requires user input (no project name)
	// so the TUI doesn't complete immediately
	initialState := InitialState{
		Template:                 "scaffold",
		IsDefaultTemplate:        false,
		ProjectName:              "", // No project name - will show input form
		BlueprintFormat:          "yaml",
		IsDefaultBlueprintFormat: false,
		NoGit:                    nil,
		IsDefaultNoGit:           true,
		Directory:                "",
	}

	mainModel, err := NewInitApp(
		initialState,
		newTestStyles(),
		&mockGit{},
		&mockPreparer{},
		/* headless */ false,
		os.Stdout,
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		mainModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for the input form to render
	testutils.WaitForContains(s.T(), testModel.Output(), "Project Name")

	// Send Ctrl+C to cancel
	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(InitModel)
	s.True(finalModel.quitting)
}

func (s *InitTUISuite) Test_completion_screen_any_key_exits() {
	mainModel, err := NewInitApp(
		defaultInitialState(),
		newTestStyles(),
		&mockGit{},
		&mockPreparer{},
		/* headless */ false,
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
		"Project initialized successfully",
		"Press any key to exit",
	)

	// Press space (any key should work)
	testModel.Send(tea.KeyMsg{Type: tea.KeySpace})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func TestInitTUISuite(t *testing.T) {
	suite.Run(t, new(InitTUISuite))
}
