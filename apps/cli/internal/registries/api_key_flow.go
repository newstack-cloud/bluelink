package registries

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultAPIKeyVerifyTimeout = 30 * time.Second

// APIKeyAuthenticator handles API key authentication with registries.
type APIKeyAuthenticator struct {
	httpClient      *http.Client
	authConfigStore *AuthConfigStore
}

// NewAPIKeyAuthenticator creates a new API key authenticator with default settings.
func NewAPIKeyAuthenticator(authConfigStore *AuthConfigStore) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		httpClient: &http.Client{
			Timeout: defaultAPIKeyVerifyTimeout,
		},
		authConfigStore: authConfigStore,
	}
}

// NewAPIKeyAuthenticatorWithHTTPClient creates a new API key authenticator with a custom HTTP client.
// This is primarily useful for testing.
func NewAPIKeyAuthenticatorWithHTTPClient(
	httpClient *http.Client,
	authConfigStore *AuthConfigStore,
) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		httpClient:      httpClient,
		authConfigStore: authConfigStore,
	}
}

// Authenticate verifies the API key and stores it in the auth config if valid.
func (a *APIKeyAuthenticator) Authenticate(
	ctx context.Context,
	registryHost string,
	authConfig *AuthV1Config,
	apiKey string,
) error {
	if apiKey == "" {
		return ErrCredentialsRequired
	}

	if err := a.Verify(ctx, registryHost, authConfig, apiKey); err != nil {
		return err
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
func (a *APIKeyAuthenticator) Verify(
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
