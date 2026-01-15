package registries

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type APIKeyFlowSuite struct {
	suite.Suite
	tempDir         string
	authConfigStore *AuthConfigStore
}

func (s *APIKeyFlowSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "api-key-flow-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
	s.authConfigStore = NewAuthConfigStoreWithPath(filepath.Join(tempDir, "plugins.auth.json"))
}

func (s *APIKeyFlowSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *APIKeyFlowSuite) TestNewAPIKeyCredentialStore_creates_store() {
	auth := NewAPIKeyCredentialStore(s.authConfigStore)
	s.NotNil(auth)
	s.NotNil(auth.httpClient)
	s.Equal(s.authConfigStore, auth.authConfigStore)
}

func (s *APIKeyFlowSuite) TestStore_returns_error_for_empty_api_key() {
	auth := NewAPIKeyCredentialStore(s.authConfigStore)
	authConfig := &AuthV1Config{APIKeyHeader: "X-API-Key"}

	err := auth.Store(context.Background(), "registry.example.com", authConfig, "")

	s.Error(err)
	s.True(errors.Is(err, ErrCredentialsRequired))
}

func (s *APIKeyFlowSuite) TestStore_saves_api_key() {
	auth := NewAPIKeyCredentialStore(s.authConfigStore)
	authConfig := &AuthV1Config{APIKeyHeader: "X-API-Key"}
	host := "registry.example.com"

	err := auth.Store(context.Background(), host, authConfig, "any-api-key")

	s.NoError(err)

	// Verify the API key was saved
	savedAuth, err := s.authConfigStore.GetRegistryAuth(host)
	s.NoError(err)
	s.NotNil(savedAuth)
	s.Equal("any-api-key", savedAuth.APIKey)
}

func (s *APIKeyFlowSuite) TestVerify_returns_error_when_auth_config_nil() {
	auth := NewAPIKeyCredentialStore(s.authConfigStore)

	err := auth.Verify(context.Background(), "registry.example.com", nil, "api-key")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "API key header not configured")
}

func (s *APIKeyFlowSuite) TestVerify_returns_error_when_api_key_header_empty() {
	auth := NewAPIKeyCredentialStore(s.authConfigStore)
	authConfig := &AuthV1Config{APIKeyHeader: ""}

	err := auth.Verify(context.Background(), "registry.example.com", authConfig, "api-key")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "API key header not configured")
}

func (s *APIKeyFlowSuite) TestVerify_sends_api_key_in_correct_header() {
	var receivedHeader string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("Custom-API-Header")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	auth := NewAPIKeyCredentialStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{APIKeyHeader: "Custom-API-Header"}
	host := strings.TrimPrefix(server.URL, "https://")

	err := auth.Verify(context.Background(), host, authConfig, "test-api-key")

	s.NoError(err)
	s.Equal("test-api-key", receivedHeader)
}

func (s *APIKeyFlowSuite) TestVerify_accepts_2xx_status_codes() {
	statusCodes := []int{200, 201, 202, 204}

	for _, statusCode := range statusCodes {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		}))

		auth := NewAPIKeyCredentialStoreWithHTTPClient(server.Client(), s.authConfigStore)
		authConfig := &AuthV1Config{APIKeyHeader: "X-API-Key"}
		host := strings.TrimPrefix(server.URL, "https://")

		err := auth.Verify(context.Background(), host, authConfig, "api-key")

		s.NoError(err, "Expected no error for status code %d", statusCode)
		server.Close()
	}
}

func (s *APIKeyFlowSuite) TestVerify_returns_error_for_401() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	auth := NewAPIKeyCredentialStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{APIKeyHeader: "X-API-Key"}
	host := strings.TrimPrefix(server.URL, "https://")

	err := auth.Verify(context.Background(), host, authConfig, "api-key")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "invalid API key")
}

func (s *APIKeyFlowSuite) TestVerify_returns_error_for_403() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	auth := NewAPIKeyCredentialStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{APIKeyHeader: "X-API-Key"}
	host := strings.TrimPrefix(server.URL, "https://")

	err := auth.Verify(context.Background(), host, authConfig, "api-key")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "invalid API key")
}

func (s *APIKeyFlowSuite) TestVerify_returns_error_for_500() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	auth := NewAPIKeyCredentialStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{APIKeyHeader: "X-API-Key"}
	host := strings.TrimPrefix(server.URL, "https://")

	err := auth.Verify(context.Background(), host, authConfig, "api-key")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "unexpected response status 500")
}

func TestAPIKeyFlowSuite(t *testing.T) {
	suite.Run(t, new(APIKeyFlowSuite))
}
