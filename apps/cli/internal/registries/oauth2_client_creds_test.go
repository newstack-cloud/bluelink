package registries

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type OAuth2ClientCredsSuite struct {
	suite.Suite
	tempDir         string
	authConfigStore *AuthConfigStore
}

func (s *OAuth2ClientCredsSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "oauth2-client-creds-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
	s.authConfigStore = NewAuthConfigStoreWithPath(filepath.Join(tempDir, "plugins.auth.json"))
}

func (s *OAuth2ClientCredsSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *OAuth2ClientCredsSuite) TestNewOAuth2ClientCredsStore_creates_store() {
	auth := NewOAuth2ClientCredsStore(s.authConfigStore)
	s.NotNil(auth)
	s.NotNil(auth.httpClient)
	s.Equal(s.authConfigStore, auth.authConfigStore)
}

func (s *OAuth2ClientCredsSuite) TestStore_returns_error_for_empty_client_id() {
	auth := NewOAuth2ClientCredsStore(s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: "https://auth.example.com",
		Token:    "/token",
	}

	err := auth.Store(context.Background(), "registry.example.com", authConfig, "", "secret")

	s.Error(err)
	s.True(errors.Is(err, ErrCredentialsRequired))
}

func (s *OAuth2ClientCredsSuite) TestStore_returns_error_for_empty_client_secret() {
	auth := NewOAuth2ClientCredsStore(s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: "https://auth.example.com",
		Token:    "/token",
	}

	err := auth.Store(context.Background(), "registry.example.com", authConfig, "client-id", "")

	s.Error(err)
	s.True(errors.Is(err, ErrCredentialsRequired))
}

func (s *OAuth2ClientCredsSuite) TestStore_saves_credentials() {
	// Authenticate no longer verifies credentials - it just stores them for later use
	auth := NewOAuth2ClientCredsStore(s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: "https://auth.example.com",
		Token:    "/token",
	}
	host := "registry.example.com"

	err := auth.Store(context.Background(), host, authConfig, "test-client-id", "test-client-secret")

	s.NoError(err)

	// Verify the credentials were saved
	savedAuth, err := s.authConfigStore.GetRegistryAuth(host)
	s.NoError(err)
	s.NotNil(savedAuth)
	s.NotNil(savedAuth.OAuth2)
	s.Equal("test-client-id", savedAuth.OAuth2.ClientId)
	s.Equal("test-client-secret", savedAuth.OAuth2.ClientSecret)
}

func (s *OAuth2ClientCredsSuite) TestObtainToken_returns_error_for_nil_auth_config() {
	auth := NewOAuth2ClientCredsStore(s.authConfigStore)

	_, err := auth.ObtainToken(context.Background(), nil, "client", "secret")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "auth config is nil")
}

func (s *OAuth2ClientCredsSuite) TestObtainToken_returns_error_for_missing_token_url() {
	auth := NewOAuth2ClientCredsStore(s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: "",
		Token:    "",
	}

	_, err := auth.ObtainToken(context.Background(), authConfig, "client", "secret")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "token URL not configured")
}

func (s *OAuth2ClientCredsSuite) TestObtainToken_returns_token_response() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"access_token": "test-token",
			"token_type": "Bearer",
			"expires_in": 7200,
			"refresh_token": "test-refresh"
		}`))
	}))
	defer server.Close()

	auth := NewOAuth2ClientCredsStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: server.URL,
		Token:    "/token",
	}

	token, err := auth.ObtainToken(context.Background(), authConfig, "client", "secret")

	s.NoError(err)
	s.NotNil(token)
	s.Equal("test-token", token.AccessToken)
	s.Equal("Bearer", token.TokenType)
	s.Equal("test-refresh", token.RefreshToken)
	// oauth2.Token uses Expiry (time.Time) not ExpiresIn (int)
	s.False(token.Expiry.IsZero())
}

func (s *OAuth2ClientCredsSuite) TestObtainToken_returns_error_for_empty_access_token() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"token_type": "Bearer"
		}`))
	}))
	defer server.Close()

	auth := NewOAuth2ClientCredsStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: server.URL,
		Token:    "/token",
	}

	_, err := auth.ObtainToken(context.Background(), authConfig, "client", "secret")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	// The oauth2 library returns "oauth2: server response missing access_token"
	s.Contains(err.Error(), "access_token")
}

func (s *OAuth2ClientCredsSuite) TestObtainToken_returns_error_for_invalid_json() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	auth := NewOAuth2ClientCredsStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: server.URL,
		Token:    "/token",
	}

	_, err := auth.ObtainToken(context.Background(), authConfig, "client", "secret")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
}

func (s *OAuth2ClientCredsSuite) TestObtainToken_parses_oauth2_error_with_description() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{
			"error": "invalid_grant",
			"error_description": "The grant type is not supported"
		}`))
	}))
	defer server.Close()

	auth := NewOAuth2ClientCredsStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: server.URL,
		Token:    "/token",
	}

	_, err := auth.ObtainToken(context.Background(), authConfig, "client", "secret")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "invalid_grant")
}

func (s *OAuth2ClientCredsSuite) TestObtainToken_parses_oauth2_error_without_description() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{
			"error": "unsupported_grant_type"
		}`))
	}))
	defer server.Close()

	auth := NewOAuth2ClientCredsStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: server.URL,
		Token:    "/token",
	}

	_, err := auth.ObtainToken(context.Background(), authConfig, "client", "secret")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "unsupported_grant_type")
}

func (s *OAuth2ClientCredsSuite) TestObtainToken_handles_401_error() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{
			"error": "invalid_client",
			"error_description": "The client credentials are invalid"
		}`))
	}))
	defer server.Close()

	auth := NewOAuth2ClientCredsStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: server.URL,
		Token:    "/token",
	}

	_, err := auth.ObtainToken(context.Background(), authConfig, "client", "secret")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
}

func (s *OAuth2ClientCredsSuite) TestObtainToken_handles_500_error() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{
			"error": "server_error",
			"error_description": "Internal server error"
		}`))
	}))
	defer server.Close()

	auth := NewOAuth2ClientCredsStoreWithHTTPClient(server.Client(), s.authConfigStore)
	authConfig := &AuthV1Config{
		Endpoint: server.URL,
		Token:    "/token",
	}

	_, err := auth.ObtainToken(context.Background(), authConfig, "client", "secret")

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
}

func TestOAuth2ClientCredsSuite(t *testing.T) {
	suite.Run(t, new(OAuth2ClientCredsSuite))
}
