package registries

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	serviceDiscoveryPath    = "/.well-known/bluelink-services.json"
	defaultDiscoveryTimeout = 30 * time.Second
)

// ServiceDiscoveryClient fetches and parses service discovery documents from registries.
type ServiceDiscoveryClient struct {
	httpClient *http.Client
}

// NewServiceDiscoveryClient creates a new service discovery client with default settings.
func NewServiceDiscoveryClient() *ServiceDiscoveryClient {
	return &ServiceDiscoveryClient{
		httpClient: &http.Client{
			Timeout: defaultDiscoveryTimeout,
		},
	}
}

// NewServiceDiscoveryClientWithHTTPClient creates a new service discovery client with a custom HTTP client.
// This is primarily useful for testing.
func NewServiceDiscoveryClientWithHTTPClient(client *http.Client) *ServiceDiscoveryClient {
	return &ServiceDiscoveryClient{
		httpClient: client,
	}
}

// Discover fetches and parses the service discovery document from a registry host.
// The host should be the registry hostname without scheme (e.g., "registry.example.com").
// HTTPS is enforced for security unless:
//   - The host already includes a scheme (for testing with local HTTP registries)
//   - The host is localhost or 127.0.0.1 (for local development/testing)
func (c *ServiceDiscoveryClient) Discover(ctx context.Context, registryHost string) (*ServiceDiscoveryDocument, error) {
	var url string
	if strings.HasPrefix(registryHost, "http://") || strings.HasPrefix(registryHost, "https://") {
		// Host already includes scheme (useful for testing with local HTTP registries)
		url = registryHost + serviceDiscoveryPath
	} else if isLocalhost(registryHost) {
		// Allow HTTP for localhost (local development/testing)
		url = "http://" + registryHost + serviceDiscoveryPath
	} else {
		// Default to HTTPS for security
		url = "https://" + registryHost + serviceDiscoveryPath
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrServiceDiscoveryFailed, err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrServiceDiscoveryFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrServiceDiscoveryFailed, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", ErrServiceDiscoveryFailed, err)
	}

	var doc ServiceDiscoveryDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("%w: failed to parse response: %v", ErrServiceDiscoveryFailed, err)
	}

	return &doc, nil
}

// NormalizeRegistryHost strips the scheme from a registry host for use as a storage key.
// This ensures consistent lookups regardless of whether the user provided http:// or https://.
// Examples:
//   - "http://localhost:8080" -> "localhost:8080"
//   - "https://registry.example.com" -> "registry.example.com"
//   - "localhost:8080" -> "localhost:8080" (unchanged)
func NormalizeRegistryHost(host string) string {
	if withoutHTTP, ok := strings.CutPrefix(host, "http://"); ok {
		return withoutHTTP
	}
	if withoutHTTPS, ok := strings.CutPrefix(host, "https://"); ok {
		return withoutHTTPS
	}
	return host
}

// isLocalhost checks if the host is a localhost address.
// Handles "localhost", "127.0.0.1", and "::1" (IPv6) with optional port.
func isLocalhost(host string) bool {
	// First normalize to remove any scheme
	host = NormalizeRegistryHost(host)

	// Handle IPv6 addresses in brackets (e.g., "[::1]:8080" or "[::1]")
	if strings.HasPrefix(host, "[") {
		if idx := strings.Index(host, "]"); idx != -1 {
			ipv6 := host[1:idx]
			return ipv6 == "::1"
		}
		return false
	}

	// Handle plain ::1 (IPv6 without brackets or port)
	if host == "::1" {
		return true
	}

	// Remove port if present (for non-bracketed addresses)
	hostOnly := host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		hostOnly = host[:idx]
	}

	return hostOnly == "localhost" || hostOnly == "127.0.0.1"
}

// GetPluginType determines the plugin type served by a registry host
// by checking which service endpoint is configured in the discovery document.
func (c *ServiceDiscoveryClient) GetPluginType(
	ctx context.Context,
	registryHost string,
) (PluginType, error) {
	doc, err := c.Discover(ctx, registryHost)
	if err != nil {
		return PluginTypeUnknown, err
	}

	if doc.ProviderV1 != nil && doc.ProviderV1.Endpoint != "" {
		return PluginTypeProvider, nil
	}

	if doc.TransformerV1 != nil && doc.TransformerV1.Endpoint != "" {
		return PluginTypeTransformer, nil
	}

	return PluginTypeUnknown, nil
}

// DiscoverAuthConfig fetches the service discovery document and returns just the auth configuration.
// Returns ErrNoAuthMethodsSupported if the registry has no auth.v1 configuration.
func (c *ServiceDiscoveryClient) DiscoverAuthConfig(ctx context.Context, registryHost string) (*AuthV1Config, error) {
	doc, err := c.Discover(ctx, registryHost)
	if err != nil {
		return nil, err
	}

	if doc.Auth == nil {
		return nil, ErrNoAuthMethodsSupported
	}

	return doc.Auth, nil
}
