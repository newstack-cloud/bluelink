package listui

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type ListTUISuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestListTUISuite(t *testing.T) {
	suite.Run(t, new(ListTUISuite))
}

func (s *ListTUISuite) SetupTest() {
	s.styles = stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)
}

// --- Test Helper Functions ---

func testInstances() []state.InstanceSummary {
	return []state.InstanceSummary{
		{
			InstanceID:            "inst-001",
			InstanceName:          "production-api",
			Status:                core.InstanceStatusDeployed,
			LastDeployedTimestamp: time.Now().Add(-2 * time.Hour).Unix(),
		},
		{
			InstanceID:            "inst-002",
			InstanceName:          "staging-api",
			Status:                core.InstanceStatusUpdated,
			LastDeployedTimestamp: time.Now().Add(-24 * time.Hour).Unix(),
		},
		{
			InstanceID:            "inst-003",
			InstanceName:          "dev-api",
			Status:                core.InstanceStatusDeployFailed,
			LastDeployedTimestamp: time.Now().Add(-72 * time.Hour).Unix(),
		},
	}
}

func (s *ListTUISuite) newTestModel(instances []state.InstanceSummary) MainModel {
	model, _ := NewListApp(
		testutils.NewTestDeployEngineForList(instances),
		zap.NewNop(),
		"",
		s.styles,
		false,
		os.Stdout,
		false,
	)
	return *model
}

func (s *ListTUISuite) newHeadlessTestModel(
	instances []state.InstanceSummary,
	output *bytes.Buffer,
	jsonMode bool,
) MainModel {
	model, _ := NewListApp(
		testutils.NewTestDeployEngineForList(instances),
		zap.NewNop(),
		"",
		s.styles,
		true,
		output,
		jsonMode,
	)
	return *model
}

// --- Interactive Mode Tests ---

func (s *ListTUISuite) Test_list_displays_instances_after_load() {
	instances := testInstances()
	model := s.newTestModel(instances)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  instances,
		TotalCount: len(instances),
		Page:       0,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"production-api",
		"staging-api",
		"dev-api",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)
	s.Equal(3, finalModel.totalCount)
}

func (s *ListTUISuite) Test_list_handles_empty_instances() {
	model := s.newTestModel(nil)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  nil,
		TotalCount: 0,
		Page:       0,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"No instances found",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *ListTUISuite) Test_list_navigation_up_down() {
	instances := testInstances()
	model := s.newTestModel(instances)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  instances,
		TotalCount: len(instances),
		Page:       0,
	})

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "production-api")

	// Move down
	testutils.KeyDown(testModel)
	testutils.KeyDown(testModel)

	// Quit and verify cursor position
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Equal(2, finalModel.cursor)
}

func (s *ListTUISuite) Test_list_search_mode_enter_exit() {
	instances := testInstances()
	model := s.newTestModel(instances)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  instances,
		TotalCount: len(instances),
		Page:       0,
	})

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "production-api")

	// Enter search mode
	testModel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "Search Instances")

	// Exit search mode
	testutils.KeyEscape(testModel)

	// Should be back in viewing mode
	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Equal(listViewing, finalModel.sessionState)
}

func (s *ListTUISuite) Test_list_select_navigates_to_inspect() {
	instances := testInstances()
	model := s.newTestModel(instances)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  instances,
		TotalCount: len(instances),
		Page:       0,
	})

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "production-api")

	// Select first item
	testutils.KeyEnter(testModel)

	// Should transition to inspect mode - wait for inspect model to render
	testutils.WaitFor(s.T(), testModel.Output(), func(output []byte) bool {
		return bytes.Contains(output, []byte("Instance Inspector"))
	})

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Equal(listInspecting, finalModel.sessionState)
	s.Equal("inst-001", finalModel.SelectedInstanceID)
	s.Equal("production-api", finalModel.SelectedInstanceName)
}

func (s *ListTUISuite) Test_list_handles_load_error() {
	model := s.newTestModel(nil)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadErrorMsg{
		Err: fmt.Errorf("connection failed"),
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Error",
		"connection failed",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.NotNil(finalModel.Error)
}

func (s *ListTUISuite) Test_list_pagination_footer_shows_counts() {
	instances := testInstances()
	model := s.newTestModel(instances)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  instances,
		TotalCount: len(instances),
		Page:       0,
	})

	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Page 1",
		"3 total",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *ListTUISuite) Test_list_quit_exits_cleanly() {
	instances := testInstances()
	model := s.newTestModel(instances)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  instances,
		TotalCount: len(instances),
		Page:       0,
	})

	testutils.WaitForContainsAll(s.T(), testModel.Output(), "production-api")

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.True(finalModel.quitting)
}

// --- Headless Mode Tests ---

func (s *ListTUISuite) Test_headless_outputs_instance_list() {
	headlessOutput := &bytes.Buffer{}
	instances := testInstances()
	model := s.newHeadlessTestModel(instances, headlessOutput, false)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  instances,
		TotalCount: len(instances),
		Page:       0,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "Blueprint Instances")
	s.Contains(output, "production-api")
	s.Contains(output, "staging-api")
	s.Contains(output, "dev-api")
	s.Contains(output, "Total: 3 instance(s)")
}

func (s *ListTUISuite) Test_headless_outputs_empty_list() {
	headlessOutput := &bytes.Buffer{}
	model := s.newHeadlessTestModel(nil, headlessOutput, false)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  nil,
		TotalCount: 0,
		Page:       0,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "No instances found")
}

func (s *ListTUISuite) Test_headless_outputs_error() {
	headlessOutput := &bytes.Buffer{}
	// Use the error mock so the engine returns an error on list
	model, _ := NewListApp(
		testutils.NewTestDeployEngineForListError(),
		zap.NewNop(),
		"",
		s.styles,
		true,
		headlessOutput,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		*model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "ERR")
	s.Contains(output, "List instances failed")
}

func (s *ListTUISuite) Test_headless_includes_search_filter() {
	headlessOutput := &bytes.Buffer{}
	instances := testInstances()[:1] // Just production-api
	model, _ := NewListApp(
		testutils.NewTestDeployEngineForList(instances),
		zap.NewNop(),
		"prod",
		s.styles,
		true,
		headlessOutput,
		false,
	)

	testModel := teatest.NewTestModel(
		s.T(),
		*model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(PageLoadedMsg{
		Instances:  instances,
		TotalCount: len(instances),
		Page:       0,
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	s.Contains(output, "prod")
	s.Contains(output, "production-api")
}
