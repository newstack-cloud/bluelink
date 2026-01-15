package pluginloginui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/registries"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

type loginStage int

const (
	// Stage where the service discovery is being performed.
	discoveringStage loginStage = iota

	// Stage where the user selects the auth type (if multiple are supported).
	selectAuthTypeStage

	// Stage where the user enters API key.
	apiKeyInputStage

	// Stage where the user enters OAuth2 client credentials.
	oauth2ClientCredsInputStage

	// Stage where OAuth2 auth code flow is in progress (browser).
	oauth2AuthCodeWaitingStage

	// Stage where authentication is being performed.
	authenticatingStage

	// Stage where authentication completed successfully.
	loginCompleteStage

	// Stage where an error occurred.
	errorStage
)

// MainModel is the main model for the plugin login TUI.
type MainModel struct {
	ctx            context.Context
	stage          loginStage
	registryHost   string
	authConfig     *registries.AuthV1Config
	selectedAuth   registries.AuthType
	spinner        spinner.Model
	styles         *stylespkg.Styles
	headless       bool
	headlessWriter io.Writer
	quitting       bool
	width          int
	Error          error

	// Sub-models
	authTypeForm          tea.Model
	apiKeyForm            tea.Model
	oauth2ClientCredsForm tea.Model
	oauth2AuthCodeView    tea.Model

	// Dependencies (for testing)
	discoveryClient       *registries.ServiceDiscoveryClient
	apiKeyStore   *registries.APIKeyCredentialStore
	oauth2ClientCredsStore *registries.OAuth2ClientCredsStore
	oauth2AuthCodeAuth    *registries.OAuth2AuthCodeAuthenticator
}

func (m MainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "Discovering authentication methods for %s...\n", m.registryHost)
	}

	// Start service discovery
	cmds = append(cmds, discoverServiceCmd(m.ctx, m.discoveryClient, m.registryHost))

	return tea.Batch(cmds...)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			if m.stage == loginCompleteStage || m.stage == errorStage {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case ServiceDiscoveryCompleteMsg:
		m.authConfig = msg.AuthConfig
		return m.handleServiceDiscoveryComplete()

	case ServiceDiscoveryErrorMsg:
		m.Error = msg.Err
		m.stage = errorStage
		if m.headless {
			fmt.Fprintf(m.headlessWriter, "Error: %v\n", msg.Err)
			return m, tea.Quit
		}
		return m, nil

	case AuthTypeSelectedMsg:
		m.selectedAuth = msg.AuthType
		return m.handleAuthTypeSelected()

	case APIKeyInputCompleteMsg:
		m.stage = authenticatingStage
		if m.headless {
			fmt.Fprintln(m.headlessWriter, "Authenticating...")
		}
		return m, storeAPIKeyCmd(
			m.ctx,
			m.apiKeyStore,
			m.registryHost,
			m.authConfig,
			msg.APIKey,
		)

	case OAuth2ClientCredsInputCompleteMsg:
		m.stage = authenticatingStage
		if m.headless {
			fmt.Fprintln(m.headlessWriter, "Authenticating...")
		}
		return m, storeOAuth2ClientCredsCmd(
			m.ctx,
			m.oauth2ClientCredsStore,
			m.registryHost,
			m.authConfig,
			msg.ClientId,
			msg.ClientSecret,
		)

	case APIKeyAuthCompleteMsg:
		return m.handleAuthComplete()

	case APIKeyAuthErrorMsg:
		return m.handleAuthError(msg.Err)

	case OAuth2ClientCredsAuthCompleteMsg:
		return m.handleAuthComplete()

	case OAuth2ClientCredsAuthErrorMsg:
		return m.handleAuthError(msg.Err)

	case OAuth2AuthCodeCompleteMsg:
		return m.handleAuthComplete()

	case OAuth2AuthCodeErrorMsg:
		return m.handleAuthError(msg.Err)
	}

	// Route updates to the current stage's model
	var cmd tea.Cmd
	switch m.stage {
	case discoveringStage, authenticatingStage:
		m.spinner, cmd = m.spinner.Update(msg)
	case selectAuthTypeStage:
		m.authTypeForm, cmd = m.authTypeForm.Update(msg)
	case apiKeyInputStage:
		m.apiKeyForm, cmd = m.apiKeyForm.Update(msg)
	case oauth2ClientCredsInputStage:
		m.oauth2ClientCredsForm, cmd = m.oauth2ClientCredsForm.Update(msg)
	case oauth2AuthCodeWaitingStage:
		m.oauth2AuthCodeView, cmd = m.oauth2AuthCodeView.Update(msg)
	}
	return m, cmd
}

func (m MainModel) View() string {
	if m.headless {
		return ""
	}

	if m.quitting {
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("Login cancelled.")
	}

	switch m.stage {
	case discoveringStage:
		return m.renderDiscovering()
	case selectAuthTypeStage:
		return "\n" + m.authTypeForm.View()
	case apiKeyInputStage:
		return "\n" + m.apiKeyForm.View()
	case oauth2ClientCredsInputStage:
		return "\n" + m.oauth2ClientCredsForm.View()
	case oauth2AuthCodeWaitingStage:
		return m.oauth2AuthCodeView.View()
	case authenticatingStage:
		return m.renderAuthenticating()
	case loginCompleteStage:
		return m.renderComplete()
	case errorStage:
		return m.renderError()
	}

	return "\n"
}

func (m *MainModel) handleServiceDiscoveryComplete() (tea.Model, tea.Cmd) {
	supportedTypes := m.authConfig.GetSupportedAuthTypes()

	if len(supportedTypes) == 0 {
		m.Error = registries.ErrNoAuthMethodsSupported
		m.stage = errorStage
		if m.headless {
			fmt.Fprintf(m.headlessWriter, "Error: %v\n", m.Error)
			return m, tea.Quit
		}
		return m, nil
	}

	if m.headless {
		fmt.Fprintf(m.headlessWriter, "Found %d authentication method(s)\n", len(supportedTypes))
	}

	// If only one auth type is supported, use it directly
	if len(supportedTypes) == 1 {
		m.selectedAuth = supportedTypes[0]
		return m.handleAuthTypeSelected()
	}

	// Multiple auth types supported - in headless mode we can't show selection form
	if m.headless {
		m.Error = fmt.Errorf("multiple authentication methods available; interactive terminal required to select")
		m.stage = errorStage
		fmt.Fprintf(m.headlessWriter, "Error: %v\n", m.Error)
		return m, tea.Quit
	}

	// Multiple auth types supported, show selection form
	m.stage = selectAuthTypeStage
	m.authTypeForm = NewAuthTypeFormModel(m.registryHost, supportedTypes, m.styles)
	return m, m.authTypeForm.Init()
}

func (m *MainModel) handleAuthTypeSelected() (tea.Model, tea.Cmd) {
	switch m.selectedAuth {
	case registries.AuthTypeAPIKey:
		// In headless mode, we can't prompt for API key input
		if m.headless {
			m.Error = fmt.Errorf("API key authentication requires an interactive terminal")
			m.stage = errorStage
			fmt.Fprintf(m.headlessWriter, "Error: %v\n", m.Error)
			return m, tea.Quit
		}
		m.stage = apiKeyInputStage
		m.apiKeyForm = NewAPIKeyFormModel(m.registryHost, m.styles)
		return m, m.apiKeyForm.Init()

	case registries.AuthTypeOAuth2ClientCreds:
		// In headless mode, we can't prompt for client credentials input
		if m.headless {
			m.Error = fmt.Errorf("OAuth2 client credentials authentication requires an interactive terminal")
			m.stage = errorStage
			fmt.Fprintf(m.headlessWriter, "Error: %v\n", m.Error)
			return m, tea.Quit
		}
		m.stage = oauth2ClientCredsInputStage
		m.oauth2ClientCredsForm = NewOAuth2ClientCredsFormModel(m.registryHost, m.styles)
		return m, m.oauth2ClientCredsForm.Init()

	case registries.AuthTypeOAuth2AuthCode:
		m.stage = oauth2AuthCodeWaitingStage
		m.oauth2AuthCodeView = NewOAuth2AuthCodeViewModel(m.registryHost, m.styles)
		if m.headless {
			fmt.Fprintln(m.headlessWriter, "Opening browser for authorization...")
		}
		return m, tea.Batch(
			m.oauth2AuthCodeView.Init(),
			authenticateOAuth2AuthCodeCmd(m.ctx, m.oauth2AuthCodeAuth, m.registryHost, m.authConfig),
		)
	}

	return m, nil
}

func (m *MainModel) handleAuthComplete() (tea.Model, tea.Cmd) {
	m.stage = loginCompleteStage
	if m.headless {
		msg := m.getSuccessMessage()
		fmt.Fprintf(m.headlessWriter, "%s %s\n", msg, m.registryHost)
		return m, tea.Quit
	}
	return m, nil
}

// getSuccessMessage returns the appropriate success message based on auth type.
// OAuth2 Authorization Code flow performs actual authentication, while API key
// and client credentials just store credentials for later use.
func (m *MainModel) getSuccessMessage() string {
	if m.selectedAuth == registries.AuthTypeOAuth2AuthCode {
		return "Successfully logged in to"
	}
	return "Credentials saved for"
}

func (m *MainModel) handleAuthError(err error) (tea.Model, tea.Cmd) {
	m.Error = err
	m.stage = errorStage
	if m.headless {
		fmt.Fprintf(m.headlessWriter, "Error: %v\n", err)
		return m, tea.Quit
	}
	return m, nil
}

func (m MainModel) renderDiscovering() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  ")
	sb.WriteString(m.spinner.View())
	sb.WriteString(" Discovering authentication methods for ")
	sb.WriteString(m.styles.Selected.Render(m.registryHost))
	sb.WriteString("...")
	sb.WriteString("\n")
	return sb.String()
}

func (m MainModel) renderAuthenticating() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  ")
	sb.WriteString(m.spinner.View())
	sb.WriteString(" Authenticating with ")
	sb.WriteString(m.styles.Selected.Render(m.registryHost))
	sb.WriteString("...")
	sb.WriteString("\n")
	return sb.String()
}

func (m MainModel) renderComplete() string {
	var sb strings.Builder

	successStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Success())

	sb.WriteString("\n")
	// Show different message based on auth type
	if m.selectedAuth == registries.AuthTypeOAuth2AuthCode {
		sb.WriteString(successStyle.Render("  ✓ Successfully logged in!"))
	} else {
		sb.WriteString(successStyle.Render("  ✓ Credentials saved!"))
	}
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Registry: "))
	sb.WriteString(m.styles.Selected.Render(m.registryHost))
	sb.WriteString("\n")

	sb.WriteString(m.styles.Muted.Render("  Auth method: "))
	sb.WriteString(m.styles.Selected.Render(authTypeDisplayName(m.selectedAuth)))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Press q to quit."))
	sb.WriteString("\n\n")

	return sb.String()
}

func (m MainModel) renderError() string {
	var sb strings.Builder

	errorStyle := lipgloss.NewStyle().Foreground(m.styles.Palette.Error())

	sb.WriteString("\n")
	sb.WriteString(errorStyle.Render("  ✗ Login failed"))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Registry: "))
	sb.WriteString(m.styles.Selected.Render(m.registryHost))
	sb.WriteString("\n\n")

	// Calculate max width for error text wrapping
	// Account for "  Error: " prefix (9 chars)
	maxWidth := min(max(m.width-12, 40), 100)
	errorWrapStyle := errorStyle.Width(maxWidth)

	sb.WriteString(m.styles.Muted.Render("  Error: "))
	sb.WriteString(errorWrapStyle.Render(m.Error.Error()))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.Muted.Render("  Press q to quit."))
	sb.WriteString("\n\n")

	return sb.String()
}

// LoginAppOptions contains options for creating a new login app.
type LoginAppOptions struct {
	RegistryHost           string
	Styles                 *stylespkg.Styles
	Headless               bool
	HeadlessWriter         io.Writer
	DiscoveryClient        *registries.ServiceDiscoveryClient
	APIKeyStore            *registries.APIKeyCredentialStore
	OAuth2ClientCredsStore *registries.OAuth2ClientCredsStore
	OAuth2AuthCodeAuth     *registries.OAuth2AuthCodeAuthenticator
}

// NewLoginApp creates a new plugin login TUI application.
func NewLoginApp(ctx context.Context, opts LoginAppOptions) (*MainModel, error) {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = opts.Styles.Selected

	// Use provided dependencies or create defaults
	discoveryClient := opts.DiscoveryClient
	if discoveryClient == nil {
		discoveryClient = registries.NewServiceDiscoveryClient()
	}

	apiKeyStore := opts.APIKeyStore
	if apiKeyStore == nil {
		apiKeyStore = registries.NewAPIKeyCredentialStore(registries.NewAuthConfigStore())
	}

	oauth2ClientCredsStore := opts.OAuth2ClientCredsStore
	if oauth2ClientCredsStore == nil {
		oauth2ClientCredsStore = registries.NewOAuth2ClientCredsStore(registries.NewAuthConfigStore())
	}

	oauth2AuthCodeAuth := opts.OAuth2AuthCodeAuth
	if oauth2AuthCodeAuth == nil {
		oauth2AuthCodeAuth = registries.NewOAuth2AuthCodeAuthenticator(registries.NewTokenStore())
	}

	return &MainModel{
		ctx:                   ctx,
		stage:                 discoveringStage,
		registryHost:          opts.RegistryHost,
		spinner:               s,
		styles:                opts.Styles,
		headless:              opts.Headless,
		headlessWriter:        opts.HeadlessWriter,
		width:                 80, // Default width until WindowSizeMsg is received
		discoveryClient:       discoveryClient,
		apiKeyStore:           apiKeyStore,
		oauth2ClientCredsStore: oauth2ClientCredsStore,
		oauth2AuthCodeAuth:    oauth2AuthCodeAuth,
	}, nil
}
