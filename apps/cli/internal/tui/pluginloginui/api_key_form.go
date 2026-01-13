package pluginloginui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// APIKeyFormModel handles the API key input form.
type APIKeyFormModel struct {
	form         *huh.Form
	styles       *stylespkg.Styles
	registryHost string

	// Bound form value
	apiKey string
}

func (m APIKeyFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m APIKeyFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	formModel, cmd := m.form.Update(msg)
	if form, ok := formModel.(*huh.Form); ok {
		m.form = form
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		cmds = append(cmds, func() tea.Msg {
			return APIKeyInputCompleteMsg{
				APIKey: m.form.GetString("apiKey"),
			}
		})
	}

	return m, tea.Batch(cmds...)
}

func (m APIKeyFormModel) View() string {
	return m.form.View()
}

// NewAPIKeyFormModel creates a new API key input form.
func NewAPIKeyFormModel(registryHost string, styles *stylespkg.Styles) *APIKeyFormModel {
	model := &APIKeyFormModel{
		styles:       styles,
		registryHost: registryHost,
	}

	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("apiKey").
				Title("API Key").
				Description("Enter your API key for " + registryHost).
				EchoMode(huh.EchoModePassword).
				Value(&model.apiKey).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("API key cannot be empty")
					}
					return nil
				}),
		),
	).WithTheme(stylespkg.NewBluelinkHuhTheme())

	return model
}
