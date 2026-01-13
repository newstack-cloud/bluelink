package registries

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ServiceDiscoverySuite struct {
	suite.Suite
}

func (s *ServiceDiscoverySuite) TestNewServiceDiscoveryClient_creates_client_with_default_timeout() {
	client := NewServiceDiscoveryClient()
	s.NotNil(client)
	s.NotNil(client.httpClient)
	s.Equal(defaultDiscoveryTimeout, client.httpClient.Timeout)
}

func (s *ServiceDiscoverySuite) TestNewServiceDiscoveryClientWithHTTPClient_uses_custom_client() {
	customClient := &http.Client{Timeout: 5 * time.Second}
	client := NewServiceDiscoveryClientWithHTTPClient(customClient)
	s.Equal(customClient, client.httpClient)
}

func (s *ServiceDiscoverySuite) TestDiscover_returns_valid_document() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/.well-known/bluelink-services.json", r.URL.Path)
		s.Equal("application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"auth.v1": {
				"apiKeyHeader": "X-API-Key",
				"downloadAuth": "bearer"
			}
		}`))
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())

	// Pass full URL with scheme - Discover respects explicit schemes
	doc, err := client.Discover(context.Background(), server.URL)

	s.NoError(err)
	s.NotNil(doc)
	s.NotNil(doc.Auth)
	s.Equal("X-API-Key", doc.Auth.APIKeyHeader)
	s.Equal("bearer", doc.Auth.DownloadAuth)
}

func (s *ServiceDiscoverySuite) TestDiscover_returns_oauth2_config() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"auth.v1": {
				"endpoint": "https://auth.example.com",
				"clientId": "server-client-id",
				"grantTypes": ["authorization_code"],
				"authorize": "/authorize",
				"token": "/token",
				"pkce": true
			}
		}`))
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())
	doc, err := client.Discover(context.Background(), server.URL)

	s.NoError(err)
	s.NotNil(doc.Auth)
	s.Equal("https://auth.example.com", doc.Auth.Endpoint)
	s.Equal("server-client-id", doc.Auth.ClientId)
	s.Equal([]string{"authorization_code"}, doc.Auth.GrantTypes)
	s.Equal("/authorize", doc.Auth.Authorize)
	s.Equal("/token", doc.Auth.Token)
	s.True(doc.Auth.PKCE)
}

func (s *ServiceDiscoverySuite) TestDiscover_returns_error_on_http_error() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())
	_, err := client.Discover(context.Background(), server.URL)

	s.Error(err)
	s.True(errors.Is(err, ErrServiceDiscoveryFailed))
	s.Contains(err.Error(), "HTTP 500")
}

func (s *ServiceDiscoverySuite) TestDiscover_returns_error_on_not_found() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())
	_, err := client.Discover(context.Background(), server.URL)

	s.Error(err)
	s.True(errors.Is(err, ErrServiceDiscoveryFailed))
	s.Contains(err.Error(), "HTTP 404")
}

func (s *ServiceDiscoverySuite) TestDiscover_returns_error_on_invalid_json() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())
	_, err := client.Discover(context.Background(), server.URL)

	s.Error(err)
	s.True(errors.Is(err, ErrServiceDiscoveryFailed))
	s.Contains(err.Error(), "failed to parse response")
}

func (s *ServiceDiscoverySuite) TestDiscover_returns_error_on_network_failure() {
	// Use a custom HTTP client with a short timeout to avoid long waits
	client := NewServiceDiscoveryClientWithHTTPClient(&http.Client{
		Timeout: 1 * time.Second,
	})

	// Use a non-routable IP (TEST-NET-1 per RFC 5737) to simulate network failure
	// Use a short context timeout to ensure the test doesn't hang
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Discover(ctx, "192.0.2.1:12345")

	s.Error(err)
	s.True(errors.Is(err, ErrServiceDiscoveryFailed))
}

func (s *ServiceDiscoverySuite) TestDiscover_respects_context_cancellation() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Discover(ctx, server.URL)

	s.Error(err)
	s.True(errors.Is(err, ErrServiceDiscoveryFailed))
}

func (s *ServiceDiscoverySuite) TestDiscover_returns_document_with_no_auth() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())
	doc, err := client.Discover(context.Background(), server.URL)

	s.NoError(err)
	s.NotNil(doc)
	s.Nil(doc.Auth)
}

func (s *ServiceDiscoverySuite) TestDiscoverAuthConfig_returns_auth_config() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"auth.v1": {
				"apiKeyHeader": "X-API-Key"
			}
		}`))
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())
	authConfig, err := client.DiscoverAuthConfig(context.Background(), server.URL)

	s.NoError(err)
	s.NotNil(authConfig)
	s.Equal("X-API-Key", authConfig.APIKeyHeader)
}

func (s *ServiceDiscoverySuite) TestDiscoverAuthConfig_returns_error_when_no_auth_config() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())
	_, err := client.DiscoverAuthConfig(context.Background(), server.URL)

	s.Error(err)
	s.True(errors.Is(err, ErrNoAuthMethodsSupported))
}

func (s *ServiceDiscoverySuite) TestDiscoverAuthConfig_propagates_discovery_error() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewServiceDiscoveryClientWithHTTPClient(server.Client())
	_, err := client.DiscoverAuthConfig(context.Background(), server.URL)

	s.Error(err)
	s.True(errors.Is(err, ErrServiceDiscoveryFailed))
}

func TestServiceDiscoverySuite(t *testing.T) {
	suite.Run(t, new(ServiceDiscoverySuite))
}
