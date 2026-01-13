package pluginloginui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/registries"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// AuthTypeFormModel handles the auth type selection form when multiple auth types are supported.
type AuthTypeFormModel struct {
	form         *huh.Form
	styles       *stylespkg.Styles
	registryHost string

	// Bound form value
	selectedAuthType string
}

func (m AuthTypeFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m AuthTypeFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	formModel, cmd := m.form.Update(msg)
	if form, ok := formModel.(*huh.Form); ok {
		m.form = form
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		cmds = append(cmds, func() tea.Msg {
			return AuthTypeSelectedMsg{
				AuthType: registries.AuthType(m.form.GetString("authType")),
			}
		})
	}

	return m, tea.Batch(cmds...)
}

func (m AuthTypeFormModel) View() string {
	return m.form.View()
}

// NewAuthTypeFormModel creates a new auth type selection form.
func NewAuthTypeFormModel(
	registryHost string,
	supportedTypes []registries.AuthType,
	styles *stylespkg.Styles,
) *AuthTypeFormModel {
	model := &AuthTypeFormModel{
		styles:       styles,
		registryHost: registryHost,
	}

	// Build options from supported types
	options := make([]huh.Option[string], 0, len(supportedTypes))
	for _, authType := range supportedTypes {
		options = append(options, huh.NewOption(authTypeDisplayName(authType), string(authType)))
	}

	// Set default to first option
	if len(supportedTypes) > 0 {
		model.selectedAuthType = string(supportedTypes[0])
	}

	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("authType").
				Title("Authentication Method").
				Description("Select how you want to authenticate with " + registryHost).
				Options(options...).
				Value(&model.selectedAuthType),
		),
	).WithTheme(stylespkg.NewBluelinkHuhTheme())

	return model
}

func authTypeDisplayName(authType registries.AuthType) string {
	switch authType {
	case registries.AuthTypeAPIKey:
		return "API Key"
	case registries.AuthTypeOAuth2ClientCreds:
		return "OAuth2 Client Credentials"
	case registries.AuthTypeOAuth2AuthCode:
		return "OAuth2 Authorization Code (Browser)"
	default:
		return string(authType)
	}
}
