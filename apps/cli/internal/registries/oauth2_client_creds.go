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

// OAuth2ClientCredsStore handles storing OAuth2 client credentials for registries.
// Unlike OAuth2 Authorization Code flow, this does not perform authentication
// during login - it only stores credentials for later use.
type OAuth2ClientCredsStore struct {
	httpClient      *http.Client
	authConfigStore *AuthConfigStore
}

// NewOAuth2ClientCredsStore creates a new OAuth2 client credentials store.
func NewOAuth2ClientCredsStore(authConfigStore *AuthConfigStore) *OAuth2ClientCredsStore {
	return &OAuth2ClientCredsStore{
		httpClient: &http.Client{
			Timeout: defaultOAuth2Timeout,
		},
		authConfigStore: authConfigStore,
	}
}

// NewOAuth2ClientCredsStoreWithHTTPClient creates a new credential store with a custom HTTP client.
// This is primarily useful for testing.
func NewOAuth2ClientCredsStoreWithHTTPClient(
	httpClient *http.Client,
	authConfigStore *AuthConfigStore,
) *OAuth2ClientCredsStore {
	return &OAuth2ClientCredsStore{
		httpClient:      httpClient,
		authConfigStore: authConfigStore,
	}
}

// Store saves the client credentials in the auth config for later use.
// The credentials are not verified during storage - validation happens when making
// authenticated requests to the registry.
func (a *OAuth2ClientCredsStore) Store(
	_ context.Context,
	registryHost string,
	_ *AuthV1Config,
	clientId string,
	clientSecret string,
) error {
	if clientId == "" || clientSecret == "" {
		return ErrCredentialsRequired
	}

	// Store the client credentials for future use
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
func (a *OAuth2ClientCredsStore) ObtainToken(
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
