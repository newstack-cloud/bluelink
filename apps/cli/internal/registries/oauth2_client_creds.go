package registries

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const defaultOAuth2Timeout = 30 * time.Second

// OAuth2ClientCredsAuthenticator handles OAuth2 client credentials authentication.
type OAuth2ClientCredsAuthenticator struct {
	httpClient      *http.Client
	authConfigStore *AuthConfigStore
}

// NewOAuth2ClientCredsAuthenticator creates a new OAuth2 client credentials authenticator.
func NewOAuth2ClientCredsAuthenticator(authConfigStore *AuthConfigStore) *OAuth2ClientCredsAuthenticator {
	return &OAuth2ClientCredsAuthenticator{
		httpClient: &http.Client{
			Timeout: defaultOAuth2Timeout,
		},
		authConfigStore: authConfigStore,
	}
}

// NewOAuth2ClientCredsAuthenticatorWithHTTPClient creates a new authenticator with a custom HTTP client.
// This is primarily useful for testing.
func NewOAuth2ClientCredsAuthenticatorWithHTTPClient(
	httpClient *http.Client,
	authConfigStore *AuthConfigStore,
) *OAuth2ClientCredsAuthenticator {
	return &OAuth2ClientCredsAuthenticator{
		httpClient:      httpClient,
		authConfigStore: authConfigStore,
	}
}

// Authenticate obtains a token using client credentials and stores the credentials if successful.
func (a *OAuth2ClientCredsAuthenticator) Authenticate(
	ctx context.Context,
	registryHost string,
	authConfig *AuthV1Config,
	clientId string,
	clientSecret string,
) error {
	if clientId == "" || clientSecret == "" {
		return ErrCredentialsRequired
	}

	// Verify by obtaining a token
	_, err := a.ObtainToken(ctx, authConfig, clientId, clientSecret)
	if err != nil {
		return err
	}

	// Store the client credentials (not the token) for future use
	auth := &RegistryAuthConfig{
		OAuth2: &OAuth2ClientConfig{
			ClientId:     clientId,
			ClientSecret: clientSecret,
		},
	}
	if err := a.authConfigStore.SaveRegistryAuth(registryHost, auth); err != nil {
		return fmt.Errorf("%w: %v", ErrConfigSaveFailed, err)
	}

	return nil
}

// ObtainToken exchanges client credentials for an access token using the oauth2 package.
func (a *OAuth2ClientCredsAuthenticator) ObtainToken(
	ctx context.Context,
	authConfig *AuthV1Config,
	clientId string,
	clientSecret string,
) (*oauth2.Token, error) {
	if authConfig == nil {
		return nil, fmt.Errorf("%w: auth config is nil", ErrAuthenticationFailed)
	}

	tokenURL := authConfig.GetTokenURL()
	if tokenURL == "" {
		return nil, fmt.Errorf("%w: token URL not configured", ErrAuthenticationFailed)
	}

	// Create the client credentials config
	config := &clientcredentials.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}

	// Use the custom HTTP client with the oauth2 context
	ctx = context.WithValue(ctx, oauth2.HTTPClient, a.httpClient)

	// Obtain the token
	token, err := config.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthenticationFailed, err)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("%w: no access token in response", ErrAuthenticationFailed)
	}

	return token, nil
}
