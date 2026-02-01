package templatelistui

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/templates"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
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

func (s *ListTUISuite) extractFinalModel(model tea.Model) *MainModel {
	switch m := model.(type) {
	case *MainModel:
		return m
	case MainModel:
		return &m
	default:
		s.FailNow("unexpected model type")
		return nil
	}
}

// --- Headless Mode Tests ---

func (s *ListTUISuite) Test_headless_outputs_template_list() {
	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.Nil(finalModel.Error)

	output := headlessOutput.String()
	s.Contains(output, "Available Templates")
	s.Contains(output, "Scaffold")
	s.Contains(output, "scaffold")
	s.Contains(output, "AWS Simple API")
	s.Contains(output, "aws-simple-api")
	s.Contains(output, "Total: 2 template(s)")
}

func (s *ListTUISuite) Test_headless_outputs_template_details() {
	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	output := headlessOutput.String()
	allTemplates := templates.GetTemplates()
	for _, t := range allTemplates {
		s.Contains(output, t.Description,
			"Headless output should include description for template %s", t.Key)
	}
}

func (s *ListTUISuite) Test_headless_outputs_empty_list() {
	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		Search:         "nonexistent-template-xyz",
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.Nil(finalModel.Error)

	output := headlessOutput.String()
	s.Contains(output, "No templates found")
	s.Contains(output, "Total: 0 template(s)")
}

func (s *ListTUISuite) Test_headless_outputs_error() {
	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(TemplatesLoadErrorMsg{
		Err: fmt.Errorf("failed to load templates"),
	})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.NotNil(finalModel.Error)

	output := headlessOutput.String()
	s.Contains(output, "ERR")
	s.Contains(output, "List templates failed")
}

func (s *ListTUISuite) Test_headless_search_filters_output() {
	headlessOutput := &bytes.Buffer{}
	model, err := NewListApp(ListAppOptions{
		Search:         "api",
		Styles:         s.styles,
		Headless:       true,
		HeadlessWriter: headlessOutput,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.Nil(finalModel.Error)

	output := headlessOutput.String()
	s.Contains(output, "AWS Simple API")
	s.Contains(output, "aws-simple-api")
	s.Contains(output, `search: "api"`)
	s.NotContains(output, "Scaffold (scaffold)")
	s.Contains(output, "Total: 1 template(s)")
}

// --- Interactive Mode Tests ---

func (s *ListTUISuite) Test_list_loads_templates() {
	model, err := NewListApp(ListAppOptions{
		Styles:         s.styles,
		Headless:       false,
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(TemplatesLoadedMsg{
		Templates: templates.GetTemplates(),
	})

	// Wait for the view to update then quit
	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.Nil(finalModel.Error)
	s.Equal(listViewing, finalModel.sessionState)
	s.Len(finalModel.allTemplates, 2)
}

func (s *ListTUISuite) Test_list_handles_load_error() {
	model, err := NewListApp(ListAppOptions{
		Styles:         s.styles,
		Headless:       false,
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(TemplatesLoadErrorMsg{
		Err: fmt.Errorf("load error"),
	})

	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.NotNil(finalModel.Error)
	s.Equal("load error", finalModel.Error.Error())
}

func (s *ListTUISuite) Test_list_quit_exits_cleanly() {
	model, err := NewListApp(ListAppOptions{
		Styles:         s.styles,
		Headless:       false,
		HeadlessWriter: os.Stdout,
	})
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	testModel.Send(TemplatesLoadedMsg{
		Templates: templates.GetTemplates(),
	})

	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := s.extractFinalModel(testModel.FinalModel(s.T()))
	s.True(finalModel.quitting)
}
