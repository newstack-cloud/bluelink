package pluginloginui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// OAuth2AuthCodeViewModel shows the browser authorization progress.
type OAuth2AuthCodeViewModel struct {
	spinner      spinner.Model
	styles       *stylespkg.Styles
	registryHost string
}

func (m OAuth2AuthCodeViewModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m OAuth2AuthCodeViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m OAuth2AuthCodeViewModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(m.styles.Selected.Render("  OAuth2 Authorization Code Flow"))
	sb.WriteString("\n\n")

	sb.WriteString("  ")
	sb.WriteString(m.spinner.View())
	sb.WriteString(" Opening browser for authorization...")
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Please complete the authorization in your browser."))
	sb.WriteString("\n")
	sb.WriteString(m.styles.Muted.Render("  This window will update automatically once complete."))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Press Ctrl+C to cancel."))
	sb.WriteString("\n")

	return sb.String()
}

// NewOAuth2AuthCodeViewModel creates a new OAuth2 authorization code view model.
func NewOAuth2AuthCodeViewModel(registryHost string, styles *stylespkg.Styles) *OAuth2AuthCodeViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Selected

	return &OAuth2AuthCodeViewModel{
		spinner:      s,
		styles:       styles,
		registryHost: registryHost,
	}
}
