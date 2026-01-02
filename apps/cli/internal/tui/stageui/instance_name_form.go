package stageui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// InstanceNameSelectedMsg is sent when the user provides an instance name.
type InstanceNameSelectedMsg struct {
	InstanceName string
}

// InstanceNameFormModel provides a form for entering the instance name.
type InstanceNameFormModel struct {
	form         *huh.Form
	styles       *stylespkg.Styles
	instanceName string
	submitted    bool
}

// NewInstanceNameFormModel creates a new instance name form model.
func NewInstanceNameFormModel(styles *stylespkg.Styles) *InstanceNameFormModel {
	model := &InstanceNameFormModel{
		styles: styles,
	}

	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("instanceName").
				Title("Instance Name").
				Description("Enter the name of a new or existing instance.").
				Placeholder("my-app-production").
				Value(&model.instanceName).
				Validate(func(value string) error {
					trimmed := strings.TrimSpace(value)
					if trimmed == "" {
						return errors.New("instance name cannot be empty")
					}
					if len(trimmed) < 3 {
						return errors.New("instance name must be at least 3 characters")
					}
					if len(trimmed) > 128 {
						return errors.New("instance name must be at most 128 characters")
					}
					return nil
				}),
		),
	).WithTheme(stylespkg.NewHuhTheme(styles.Palette))

	return model
}

func (m *InstanceNameFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *InstanceNameFormModel) Update(msg tea.Msg) (*InstanceNameFormModel, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted && !m.submitted {
		m.submitted = true
		m.instanceName = strings.TrimSpace(m.form.GetString("instanceName"))
		return m, func() tea.Msg {
			return InstanceNameSelectedMsg{
				InstanceName: m.instanceName,
			}
		}
	}

	return m, cmd
}

func (m *InstanceNameFormModel) View() string {
	return m.form.View()
}
