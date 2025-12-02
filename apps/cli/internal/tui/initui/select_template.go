package initui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/sharedui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
)

type SelectTemplateModel struct {
	selectTemplate   tea.Model
	selectedTemplate string
	autoSelect       bool
	quitting         bool
	err              error
}

type SelectTemplateMsg struct {
	Template string
}

func selectTemplateCmd(template string) tea.Cmd {
	return func() tea.Msg {
		return SelectTemplateMsg{
			Template: template,
		}
	}
}

func (m SelectTemplateModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	cmd := m.selectTemplate.Init()
	cmds = append(cmds, cmd)

	if m.autoSelect {
		// Dispatch command to select the template to move on to the next stage.
		cmds = append(cmds, selectTemplateCmd(m.selectedTemplate))
	}

	return tea.Batch(cmds...)
}

func (m SelectTemplateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case SelectTemplateMsg:
		m.selectedTemplate = msg.Template
	}

	var cmd tea.Cmd
	m.selectTemplate, cmd = m.selectTemplate.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m SelectTemplateModel) View() string {
	return m.selectTemplate.View()
}

func NewSelectTemplateModel(
	bluelinkStyles *styles.BluelinkStyles,
	autoSelect bool,
) (*SelectTemplateModel, error) {

	selectTemplate := sharedui.NewSelectWithPreview(
		"Choose a template",
		selectTemplateListItems(),
		bluelinkStyles,
		selectTemplateCmd,
		true, // enableFiltering
	)

	return &SelectTemplateModel{
		selectTemplate: selectTemplate,
		autoSelect:     autoSelect,
	}, nil
}
