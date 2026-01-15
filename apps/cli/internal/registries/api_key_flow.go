package registries

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultAPIKeyVerifyTimeout = 30 * time.Second

// APIKeyCredentialStore handles storing and verifying API keys for registries.
// Unlike OAuth2 Authorization Code flow, this does not perform authentication
// during login - it only stores credentials for later use.
type APIKeyCredentialStore struct {
	httpClient      *http.Client
	authConfigStore *AuthConfigStore
}

// NewAPIKeyCredentialStore creates a new API key credential store with default settings.
func NewAPIKeyCredentialStore(authConfigStore *AuthConfigStore) *APIKeyCredentialStore {
	return &APIKeyCredentialStore{
		httpClient: &http.Client{
			Timeout: defaultAPIKeyVerifyTimeout,
		},
		authConfigStore: authConfigStore,
	}
}

// NewAPIKeyCredentialStoreWithHTTPClient creates a new API key credential store with a custom HTTP client.
// This is primarily useful for testing.
func NewAPIKeyCredentialStoreWithHTTPClient(
	httpClient *http.Client,
	authConfigStore *AuthConfigStore,
) *APIKeyCredentialStore {
	return &APIKeyCredentialStore{
		httpClient:      httpClient,
		authConfigStore: authConfigStore,
	}
}

// Store saves the API key in the auth config for later use.
// The key is not verified during storage - validation happens when making
// authenticated requests to the registry.
func (a *APIKeyCredentialStore) Store(
	_ context.Context,
	registryHost string,
	_ *AuthV1Config,
	apiKey string,
) error {
	if apiKey == "" {
		return ErrCredentialsRequired
	}

	auth := &RegistryAuthConfig{
		APIKey: apiKey,
	}
	if err := a.authConfigStore.SaveRegistryAuth(registryHost, auth); err != nil {
		return fmt.Errorf("%w: %v", ErrConfigSaveFailed, err)
	}

	return nil
}

// Verify checks if the API key is valid by making a test request to the registry.
func (a *APIKeyCredentialStore) Verify(
	ctx context.Context,
	registryHost string,
	authConfig *AuthV1Config,
	apiKey string,
) error {
	if authConfig == nil || authConfig.APIKeyHeader == "" {
		return fmt.Errorf("%w: API key header not configured", ErrAuthenticationFailed)
	}

	// Make a test request to the service discovery endpoint with the API key
	var url string
	if strings.HasPrefix(registryHost, "http://") || strings.HasPrefix(registryHost, "https://") {
		url = registryHost + serviceDiscoveryPath
	} else {
		url = "https://" + registryHost + serviceDiscoveryPath
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAuthenticationFailed, err)
	}

	req.Header.Set(authConfig.APIKeyHeader, apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAuthenticationFailed, err)
	}
	defer resp.Body.Close()

	// Accept 2xx status codes as success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	if resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w: invalid API key", ErrAuthenticationFailed)
	}

	return fmt.Errorf(
		"%w: unexpected response status %d",
		ErrAuthenticationFailed,
		resp.StatusCode,
	)
}
