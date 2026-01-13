package pluginloginui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// OAuth2ClientCredsFormModel handles the OAuth2 client credentials input form.
type OAuth2ClientCredsFormModel struct {
	form         *huh.Form
	styles       *stylespkg.Styles
	registryHost string

	// Bound form values
	clientId     string
	clientSecret string
}

func (m OAuth2ClientCredsFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m OAuth2ClientCredsFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	formModel, cmd := m.form.Update(msg)
	if form, ok := formModel.(*huh.Form); ok {
		m.form = form
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		cmds = append(cmds, func() tea.Msg {
			return OAuth2ClientCredsInputCompleteMsg{
				ClientId:     m.form.GetString("clientId"),
				ClientSecret: m.form.GetString("clientSecret"),
			}
		})
	}

	return m, tea.Batch(cmds...)
}

func (m OAuth2ClientCredsFormModel) View() string {
	return m.form.View()
}

// NewOAuth2ClientCredsFormModel creates a new OAuth2 client credentials input form.
func NewOAuth2ClientCredsFormModel(registryHost string, styles *stylespkg.Styles) *OAuth2ClientCredsFormModel {
	model := &OAuth2ClientCredsFormModel{
		styles:       styles,
		registryHost: registryHost,
	}

	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("clientId").
				Title("Client ID").
				Description("Enter your OAuth2 client ID for " + registryHost).
				Value(&model.clientId).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("client ID cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Key("clientSecret").
				Title("Client Secret").
				Description("Enter your OAuth2 client secret").
				EchoMode(huh.EchoModePassword).
				Value(&model.clientSecret).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("client secret cannot be empty")
					}
					return nil
				}),
		),
	).WithTheme(stylespkg.NewBluelinkHuhTheme())

	return model
}
