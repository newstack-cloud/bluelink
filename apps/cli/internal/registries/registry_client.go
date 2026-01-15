package registries

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultRegistryTimeout = 60 * time.Second
)

// RegistryClient handles authenticated requests to plugin registries.
type RegistryClient struct {
	httpClient      *http.Client
	authConfigStore *AuthConfigStore
	tokenStore      *TokenStore
	discoveryClient *ServiceDiscoveryClient
}

// NewRegistryClient creates a new registry client with default settings.
func NewRegistryClient(
	authConfigStore *AuthConfigStore,
	tokenStore *TokenStore,
	discoveryClient *ServiceDiscoveryClient,
) *RegistryClient {
	return &RegistryClient{
		httpClient: &http.Client{
			Timeout: defaultRegistryTimeout,
		},
		authConfigStore: authConfigStore,
		tokenStore:      tokenStore,
		discoveryClient: discoveryClient,
	}
}

// NewRegistryClientWithHTTPClient creates a new registry client with a custom HTTP client.
func NewRegistryClientWithHTTPClient(
	httpClient *http.Client,
	authConfigStore *AuthConfigStore,
	tokenStore *TokenStore,
	discoveryClient *ServiceDiscoveryClient,
) *RegistryClient {
	return &RegistryClient{
		httpClient:      httpClient,
		authConfigStore: authConfigStore,
		tokenStore:      tokenStore,
		discoveryClient: discoveryClient,
	}
}

// PluginVersionsResponse represents the response from the list versions endpoint.
type PluginVersionsResponse struct {
	Versions []PluginVersionInfo `json:"versions"`
}

// PluginVersionInfo contains information about a single plugin version.
type PluginVersionInfo struct {
	Version            string   `json:"version"`
	SupportedProtocols []string `json:"supportedProtocols,omitempty"`
}

// PluginPackageMetadata contains metadata about a downloadable plugin package.
type PluginPackageMetadata struct {
	Filename            string            `json:"filename"`
	DownloadURL         string            `json:"downloadUrl"`
	OS                  string            `json:"os"`
	Arch                string            `json:"arch"`
	Shasum              string            `json:"shasum"`
	ShasumsURL          string            `json:"shasumsUrl,omitempty"`
	ShasumsSignatureURL string            `json:"shasumsSignatureUrl,omitempty"`
	SigningKeys         map[string]string `json:"signingKeys,omitempty"`
	Dependencies        map[string]string `json:"dependencies,omitempty"`
}

// ListVersions fetches available versions for a plugin from the registry.
func (c *RegistryClient) ListVersions(
	ctx context.Context,
	registryHost, namespace, pluginName string,
) (*PluginVersionsResponse, error) {
	doc, err := c.discoveryClient.Discover(ctx, registryHost)
	if err != nil {
		return nil, err
	}

	endpoint := c.getPluginEndpoint(doc)
	if endpoint == "" {
		return nil, fmt.Errorf("no plugin service configured for registry %s", registryHost)
	}

	url := c.buildURL(registryHost, endpoint, namespace, pluginName, "versions")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if err := c.addAuthHeader(ctx, req, registryHost, doc.Auth); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrPluginNotFound
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrNoCredentials
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var versionsResp PluginVersionsResponse
	if err := json.Unmarshal(body, &versionsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &versionsResp, nil
}

// GetPackageMetadata fetches metadata for a specific plugin version package.
func (c *RegistryClient) GetPackageMetadata(
	ctx context.Context,
	registryHost, namespace, pluginName, version, osName, arch string,
) (*PluginPackageMetadata, error) {
	doc, err := c.discoveryClient.Discover(ctx, registryHost)
	if err != nil {
		return nil, err
	}

	endpoint := c.getPluginEndpoint(doc)
	if endpoint == "" {
		return nil, fmt.Errorf("no plugin service configured for registry %s", registryHost)
	}

	url := c.buildURL(registryHost, endpoint, namespace, pluginName, version, "package", osName, arch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if err := c.addAuthHeader(ctx, req, registryHost, doc.Auth); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrVersionNotFound
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrNoCredentials
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var metadata PluginPackageMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &metadata, nil
}

// ProgressFunc is called during download to report progress.
type ProgressFunc func(downloaded, total int64)

// DownloadPackage downloads a plugin package to the specified destination.
func (c *RegistryClient) DownloadPackage(
	ctx context.Context,
	registryHost string,
	metadata *PluginPackageMetadata,
	destPath string,
	progressFn ProgressFunc,
) error {
	doc, err := c.discoveryClient.Discover(ctx, registryHost)
	if err != nil {
		return err
	}

	downloadURL := metadata.DownloadURL
	if !strings.HasPrefix(downloadURL, "http://") && !strings.HasPrefix(downloadURL, "https://") {
		downloadURL = c.buildBaseURL(registryHost) + downloadURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to create request: %v", ErrDownloadFailed, err)
	}

	contentType := "application/octet-stream"
	if pluginConfig := c.getPluginServiceConfig(doc); pluginConfig != nil && pluginConfig.DownloadAcceptContentType != "" {
		contentType = pluginConfig.DownloadAcceptContentType
	}
	req.Header.Set("Accept", contentType)

	if err := c.addDownloadAuthHeader(ctx, req, registryHost, doc.Auth); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrNoCredentials
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: HTTP %d", ErrDownloadFailed, resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("%w: failed to create file: %v", ErrDownloadFailed, err)
	}
	defer file.Close()

	total := resp.ContentLength
	var downloaded int64

	if progressFn != nil {
		progressFn(0, total)
	}

	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("%w: failed to write file: %v", ErrDownloadFailed, writeErr)
			}
			downloaded += int64(n)
			if progressFn != nil {
				progressFn(downloaded, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("%w: failed to read response: %v", ErrDownloadFailed, readErr)
		}
	}

	return nil
}

// DownloadShasums downloads the shasums file from the given URL.
func (c *RegistryClient) DownloadShasums(
	ctx context.Context,
	registryHost, shasumsURL string,
) ([]byte, error) {
	return c.downloadFile(ctx, registryHost, shasumsURL)
}

// DownloadSignature downloads the GPG signature file from the given URL.
func (c *RegistryClient) DownloadSignature(
	ctx context.Context,
	registryHost, signatureURL string,
) ([]byte, error) {
	return c.downloadFile(ctx, registryHost, signatureURL)
}

func (c *RegistryClient) downloadFile(ctx context.Context, registryHost, fileURL string) ([]byte, error) {
	doc, err := c.discoveryClient.Discover(ctx, registryHost)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(fileURL, "http://") && !strings.HasPrefix(fileURL, "https://") {
		fileURL = c.buildBaseURL(registryHost) + fileURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.addDownloadAuthHeader(ctx, req, registryHost, doc.Auth); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrNoCredentials
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *RegistryClient) addAuthHeader(
	ctx context.Context,
	req *http.Request,
	registryHost string,
	authConfig *AuthV1Config,
) error {
	token, err := c.getAuthToken(ctx, registryHost, authConfig)
	if err != nil {
		return err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return nil
}

func (c *RegistryClient) addDownloadAuthHeader(
	ctx context.Context,
	req *http.Request,
	registryHost string,
	authConfig *AuthV1Config,
) error {
	token, err := c.getAuthToken(ctx, registryHost, authConfig)
	if err != nil {
		return err
	}

	if token == "" {
		return nil
	}

	scheme := "bearer"
	if authConfig != nil && authConfig.DownloadAuth != "" {
		scheme = strings.ToLower(authConfig.DownloadAuth)
	}

	switch scheme {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+token)
	case "basic":
		req.Header.Set("Authorization", "Basic "+token)
	default:
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return nil
}

func (c *RegistryClient) getAuthToken(
	ctx context.Context,
	registryHost string,
	authConfig *AuthV1Config,
) (string, error) {
	// First, check for OAuth2 auth code tokens
	tokens, err := c.tokenStore.GetRegistryTokens(registryHost)
	if err != nil {
		return "", fmt.Errorf("failed to load tokens: %w", err)
	}

	if tokens != nil {
		if !tokens.IsExpired() {
			return tokens.AccessToken, nil
		}

		// Try to refresh expired token
		if tokens.RefreshToken != "" && authConfig != nil {
			refreshedTokens, err := c.refreshToken(ctx, registryHost, authConfig)
			if err == nil {
				return refreshedTokens.AccessToken, nil
			}
			// If refresh fails, fall through to other auth methods
		}
	}

	// Check for API key
	registryAuth, err := c.authConfigStore.GetRegistryAuth(registryHost)
	if err != nil {
		return "", fmt.Errorf("failed to load auth config: %w", err)
	}

	if registryAuth != nil {
		if registryAuth.APIKey != "" {
			return registryAuth.APIKey, nil
		}

		// OAuth2 client credentials - exchange for token
		if registryAuth.OAuth2 != nil && authConfig != nil {
			token, err := c.exchangeClientCredentials(ctx, registryAuth.OAuth2, authConfig)
			if err != nil {
				return "", err
			}
			return token, nil
		}
	}

	return "", nil
}

func (c *RegistryClient) refreshToken(
	ctx context.Context,
	registryHost string,
	authConfig *AuthV1Config,
) (*RegistryTokens, error) {
	authenticator := NewOAuth2AuthCodeAuthenticatorWithOptions(
		c.httpClient,
		c.tokenStore,
		nil,
		0,
	)

	result, err := authenticator.RefreshTokens(ctx, registryHost, authConfig)
	if err != nil {
		return nil, err
	}

	return &RegistryTokens{
		ClientId:     result.ClientId,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenExpiry:  result.TokenExpiry,
	}, nil
}

func (c *RegistryClient) exchangeClientCredentials(
	ctx context.Context,
	oauth2Config *OAuth2ClientConfig,
	authConfig *AuthV1Config,
) (string, error) {
	store := NewOAuth2ClientCredsStoreWithHTTPClient(c.httpClient, nil)
	token, err := store.ObtainToken(ctx, authConfig, oauth2Config.ClientId, oauth2Config.ClientSecret)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func (c *RegistryClient) getPluginEndpoint(doc *ServiceDiscoveryDocument) string {
	if doc.ProviderV1 != nil && doc.ProviderV1.Endpoint != "" {
		return doc.ProviderV1.Endpoint
	}
	if doc.TransformerV1 != nil && doc.TransformerV1.Endpoint != "" {
		return doc.TransformerV1.Endpoint
	}
	return ""
}

func (c *RegistryClient) getPluginServiceConfig(doc *ServiceDiscoveryDocument) *PluginServiceConfig {
	if doc.ProviderV1 != nil {
		return doc.ProviderV1
	}
	return doc.TransformerV1
}

func (c *RegistryClient) buildBaseURL(registryHost string) string {
	if strings.HasPrefix(registryHost, "http://") || strings.HasPrefix(registryHost, "https://") {
		return registryHost
	}
	// Allow HTTP for localhost (local development/testing)
	if isLocalhost(registryHost) {
		return "http://" + registryHost
	}
	return "https://" + registryHost
}

func (c *RegistryClient) buildURL(registryHost, endpoint string, pathParts ...string) string {
	base := c.buildBaseURL(registryHost)
	path := endpoint
	if len(pathParts) > 0 {
		path = path + "/" + strings.Join(pathParts, "/")
	}
	return base + path
}
