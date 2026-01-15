package pluginloginui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/registries"
)

// ServiceDiscoveryCompleteMsg signals completion of service discovery.
type ServiceDiscoveryCompleteMsg struct {
	AuthConfig *registries.AuthV1Config
}

// ServiceDiscoveryErrorMsg signals an error during service discovery.
type ServiceDiscoveryErrorMsg struct {
	Err error
}

func discoverServiceCmd(
	ctx context.Context,
	client *registries.ServiceDiscoveryClient,
	registryHost string,
) tea.Cmd {
	return func() tea.Msg {
		authConfig, err := client.DiscoverAuthConfig(ctx, registryHost)
		if err != nil {
			return ServiceDiscoveryErrorMsg{Err: err}
		}
		return ServiceDiscoveryCompleteMsg{AuthConfig: authConfig}
	}
}

// APIKeyAuthCompleteMsg signals completion of API key authentication.
type APIKeyAuthCompleteMsg struct{}

// APIKeyAuthErrorMsg signals an error during API key authentication.
type APIKeyAuthErrorMsg struct {
	Err error
}

func storeAPIKeyCmd(
	ctx context.Context,
	store *registries.APIKeyCredentialStore,
	registryHost string,
	authConfig *registries.AuthV1Config,
	apiKey string,
) tea.Cmd {
	return func() tea.Msg {
		if err := store.Store(ctx, registryHost, authConfig, apiKey); err != nil {
			return APIKeyAuthErrorMsg{Err: err}
		}
		return APIKeyAuthCompleteMsg{}
	}
}

// OAuth2ClientCredsAuthCompleteMsg signals completion of OAuth2 client credentials authentication.
type OAuth2ClientCredsAuthCompleteMsg struct{}

// OAuth2ClientCredsAuthErrorMsg signals an error during OAuth2 client credentials authentication.
type OAuth2ClientCredsAuthErrorMsg struct {
	Err error
}

func storeOAuth2ClientCredsCmd(
	ctx context.Context,
	store *registries.OAuth2ClientCredsStore,
	registryHost string,
	authConfig *registries.AuthV1Config,
	clientId string,
	clientSecret string,
) tea.Cmd {
	return func() tea.Msg {
		if err := store.Store(ctx, registryHost, authConfig, clientId, clientSecret); err != nil {
			return OAuth2ClientCredsAuthErrorMsg{Err: err}
		}
		return OAuth2ClientCredsAuthCompleteMsg{}
	}
}

// OAuth2AuthCodeCompleteMsg signals completion of OAuth2 authorization code authentication.
type OAuth2AuthCodeCompleteMsg struct {
	Result *registries.AuthCodeResult
}

// OAuth2AuthCodeErrorMsg signals an error during OAuth2 authorization code authentication.
type OAuth2AuthCodeErrorMsg struct {
	Err error
}

func authenticateOAuth2AuthCodeCmd(
	ctx context.Context,
	authenticator *registries.OAuth2AuthCodeAuthenticator,
	registryHost string,
	authConfig *registries.AuthV1Config,
) tea.Cmd {
	return func() tea.Msg {
		result, err := authenticator.Authenticate(ctx, registryHost, authConfig)
		if err != nil {
			return OAuth2AuthCodeErrorMsg{Err: err}
		}
		return OAuth2AuthCodeCompleteMsg{Result: result}
	}
}

// AuthTypeSelectedMsg signals the user selected an auth type.
type AuthTypeSelectedMsg struct {
	AuthType registries.AuthType
}

// APIKeyInputCompleteMsg signals the user completed API key input.
type APIKeyInputCompleteMsg struct {
	APIKey string
}

// OAuth2ClientCredsInputCompleteMsg signals the user completed OAuth2 client credentials input.
type OAuth2ClientCredsInputCompleteMsg struct {
	ClientId     string
	ClientSecret string
}
