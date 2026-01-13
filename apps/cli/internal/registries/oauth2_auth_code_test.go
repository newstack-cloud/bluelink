package registries

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type OAuth2AuthCodeSuite struct {
	suite.Suite
	tempDir    string
	tokenStore *TokenStore
}

func (s *OAuth2AuthCodeSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "oauth2-auth-code-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
	s.tokenStore = NewTokenStoreWithPath(filepath.Join(tempDir, "plugins.tokens.json"))
}

func (s *OAuth2AuthCodeSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *OAuth2AuthCodeSuite) TestNewOAuth2AuthCodeAuthenticator_creates_authenticator() {
	auth := NewOAuth2AuthCodeAuthenticator(s.tokenStore)
	s.NotNil(auth)
	s.NotNil(auth.httpClient)
	s.NotNil(auth.browserOpener)
	s.Equal(s.tokenStore, auth.tokenStore)
	s.Equal(defaultAuthCodeTimeout, auth.timeout)
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_returns_error_for_nil_auth_config() {
	auth := NewOAuth2AuthCodeAuthenticator(s.tokenStore)

	_, err := auth.Authenticate(context.Background(), "registry.example.com", nil)

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "auth config is nil")
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_returns_error_for_missing_client_id() {
	auth := NewOAuth2AuthCodeAuthenticator(s.tokenStore)
	authConfig := &AuthV1Config{
		Endpoint: "https://auth.example.com",
		Token:    "/token",
	}

	_, err := auth.Authenticate(context.Background(), "registry.example.com", authConfig)

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "client ID not configured")
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_performs_full_flow() {
	// Create a mock OAuth2 server
	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			err := r.ParseForm()
			s.NoError(err)

			s.Equal("authorization_code", r.Form.Get("grant_type"))
			s.NotEmpty(r.Form.Get("code"))
			s.NotEmpty(r.Form.Get("redirect_uri"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"access_token": "test-access-token",
				"token_type": "Bearer",
				"expires_in": 3600,
				"refresh_token": "test-refresh-token"
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	var capturedAuthURL string
	mockBrowserOpener := func(authURL string) error {
		capturedAuthURL = authURL

		// Simulate the OAuth provider calling back with a code
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			return err
		}

		redirectURI := parsedURL.Query().Get("redirect_uri")
		state := parsedURL.Query().Get("state")

		// Make a callback request
		callbackURL := redirectURI + "?code=test-auth-code&state=" + state
		resp, err := http.Get(callbackURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		tokenServer.Client(),
		s.tokenStore,
		mockBrowserOpener,
		5*time.Second,
	)

	authConfig := &AuthV1Config{
		Endpoint: tokenServer.URL,
		ClientId: "test-client-id",
		Token:    "/token",
	}

	result, err := auth.Authenticate(context.Background(), "registry.example.com", authConfig)

	s.NoError(err)
	s.NotNil(result)
	s.Equal("test-access-token", result.AccessToken)
	s.Equal("test-refresh-token", result.RefreshToken)
	s.Equal("test-client-id", result.ClientId)
	s.NotNil(result.TokenExpiry)

	// Verify the auth URL was constructed correctly
	s.NotEmpty(capturedAuthURL)
	parsedAuthURL, err := url.Parse(capturedAuthURL)
	s.NoError(err)
	s.Equal("test-client-id", parsedAuthURL.Query().Get("client_id"))
	s.Equal("code", parsedAuthURL.Query().Get("response_type"))
	s.NotEmpty(parsedAuthURL.Query().Get("state"))
	s.NotEmpty(parsedAuthURL.Query().Get("redirect_uri"))

	// Verify tokens were saved
	savedTokens, err := s.tokenStore.GetRegistryTokens("registry.example.com")
	s.NoError(err)
	s.NotNil(savedTokens)
	s.Equal("test-access-token", savedTokens.AccessToken)
	s.Equal("test-refresh-token", savedTokens.RefreshToken)
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_includes_pkce_when_supported() {
	var receivedCodeChallenge string
	var receivedCodeVerifier string

	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			err := r.ParseForm()
			s.NoError(err)
			receivedCodeVerifier = r.Form.Get("code_verifier")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"access_token": "test-token",
				"token_type": "Bearer",
				"expires_in": 3600
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	mockBrowserOpener := func(authURL string) error {
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			return err
		}

		receivedCodeChallenge = parsedURL.Query().Get("code_challenge")
		redirectURI := parsedURL.Query().Get("redirect_uri")
		state := parsedURL.Query().Get("state")

		callbackURL := redirectURI + "?code=test-code&state=" + state
		resp, err := http.Get(callbackURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		tokenServer.Client(),
		s.tokenStore,
		mockBrowserOpener,
		5*time.Second,
	)

	authConfig := &AuthV1Config{
		Endpoint: tokenServer.URL,
		ClientId: "test-client",
		Token:    "/token",
		PKCE:     true,
	}

	_, err := auth.Authenticate(context.Background(), "registry.example.com", authConfig)

	s.NoError(err)
	s.NotEmpty(receivedCodeChallenge, "Code challenge should be included in auth URL")
	s.NotEmpty(receivedCodeVerifier, "Code verifier should be included in token request")
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_omits_pkce_when_not_supported() {
	var receivedCodeChallenge string
	var receivedCodeVerifier string

	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			err := r.ParseForm()
			s.NoError(err)
			receivedCodeVerifier = r.Form.Get("code_verifier")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	mockBrowserOpener := func(authURL string) error {
		parsedURL, _ := url.Parse(authURL)
		receivedCodeChallenge = parsedURL.Query().Get("code_challenge")
		redirectURI := parsedURL.Query().Get("redirect_uri")
		state := parsedURL.Query().Get("state")

		callbackURL := redirectURI + "?code=test-code&state=" + state
		resp, err := http.Get(callbackURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		tokenServer.Client(),
		s.tokenStore,
		mockBrowserOpener,
		5*time.Second,
	)

	authConfig := &AuthV1Config{
		Endpoint: tokenServer.URL,
		ClientId: "test-client",
		Token:    "/token",
		PKCE:     false,
	}

	_, err := auth.Authenticate(context.Background(), "registry.example.com", authConfig)

	s.NoError(err)
	s.Empty(receivedCodeChallenge, "Code challenge should NOT be included")
	s.Empty(receivedCodeVerifier, "Code verifier should NOT be included")
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_returns_error_on_state_mismatch() {
	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	mockBrowserOpener := func(authURL string) error {
		parsedURL, _ := url.Parse(authURL)
		redirectURI := parsedURL.Query().Get("redirect_uri")

		// Return with a different state
		callbackURL := redirectURI + "?code=test-code&state=wrong-state"
		resp, err := http.Get(callbackURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		tokenServer.Client(),
		s.tokenStore,
		mockBrowserOpener,
		5*time.Second,
	)

	authConfig := &AuthV1Config{
		Endpoint: tokenServer.URL,
		ClientId: "test-client",
		Token:    "/token",
	}

	_, err := auth.Authenticate(context.Background(), "registry.example.com", authConfig)

	s.Error(err)
	s.True(errors.Is(err, ErrStateMismatch))
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_returns_error_on_browser_open_failure() {
	mockBrowserOpener := func(url string) error {
		return errors.New("failed to open browser")
	}

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		http.DefaultClient,
		s.tokenStore,
		mockBrowserOpener,
		5*time.Second,
	)

	authConfig := &AuthV1Config{
		Endpoint:  "https://auth.example.com",
		ClientId:  "test-client",
		Token:     "/token",
		Authorize: "/authorize",
	}

	_, err := auth.Authenticate(context.Background(), "registry.example.com", authConfig)

	s.Error(err)
	s.True(errors.Is(err, ErrBrowserOpenFailed))
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_returns_error_on_timeout() {
	mockBrowserOpener := func(url string) error {
		// Don't simulate the callback, let it timeout
		return nil
	}

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		http.DefaultClient,
		s.tokenStore,
		mockBrowserOpener,
		100*time.Millisecond,
	)

	authConfig := &AuthV1Config{
		Endpoint:  "https://auth.example.com",
		ClientId:  "test-client",
		Token:     "/token",
		Authorize: "/authorize",
	}

	_, err := auth.Authenticate(context.Background(), "registry.example.com", authConfig)

	s.Error(err)
	s.True(errors.Is(err, ErrAuthorizationTimeout))
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_returns_error_on_oauth_error_callback() {
	mockBrowserOpener := func(authURL string) error {
		parsedURL, _ := url.Parse(authURL)
		redirectURI := parsedURL.Query().Get("redirect_uri")

		callbackURL := redirectURI + "?error=access_denied&error_description=User%20denied%20access"
		resp, err := http.Get(callbackURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		http.DefaultClient,
		s.tokenStore,
		mockBrowserOpener,
		5*time.Second,
	)

	authConfig := &AuthV1Config{
		Endpoint:  "https://auth.example.com",
		ClientId:  "test-client",
		Token:     "/token",
		Authorize: "/authorize",
	}

	_, err := auth.Authenticate(context.Background(), "registry.example.com", authConfig)

	s.Error(err)
	s.True(errors.Is(err, ErrAuthenticationFailed))
	s.Contains(err.Error(), "access_denied")
}

func (s *OAuth2AuthCodeSuite) TestAuthenticate_returns_error_on_token_exchange_failure() {
	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid_grant", "error_description": "Code expired"}`))
	}))
	defer tokenServer.Close()

	mockBrowserOpener := func(authURL string) error {
		parsedURL, _ := url.Parse(authURL)
		redirectURI := parsedURL.Query().Get("redirect_uri")
		state := parsedURL.Query().Get("state")

		callbackURL := redirectURI + "?code=test-code&state=" + state
		resp, err := http.Get(callbackURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		tokenServer.Client(),
		s.tokenStore,
		mockBrowserOpener,
		5*time.Second,
	)

	authConfig := &AuthV1Config{
		Endpoint: tokenServer.URL,
		ClientId: "test-client",
		Token:    "/token",
	}

	_, err := auth.Authenticate(context.Background(), "registry.example.com", authConfig)

	s.Error(err)
	s.True(errors.Is(err, ErrTokenExchangeFailed))
	s.Contains(err.Error(), "invalid_grant")
}

func (s *OAuth2AuthCodeSuite) TestRefreshTokens_refreshes_expired_tokens() {
	// First save some tokens
	expiredTime := time.Now().Add(-time.Hour)
	err := s.tokenStore.SaveRegistryTokens("registry.example.com", &RegistryTokens{
		ClientId:     "test-client",
		AccessToken:  "old-access-token",
		RefreshToken: "test-refresh-token",
		TokenExpiry:  &expiredTime,
	})
	s.Require().NoError(err)

	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		s.NoError(err)

		s.Equal("refresh_token", r.Form.Get("grant_type"))
		s.Equal("test-refresh-token", r.Form.Get("refresh_token"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"access_token": "new-access-token",
			"token_type": "Bearer",
			"expires_in": 3600,
			"refresh_token": "new-refresh-token"
		}`))
	}))
	defer tokenServer.Close()

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		tokenServer.Client(),
		s.tokenStore,
		nil,
		5*time.Second,
	)

	authConfig := &AuthV1Config{
		Endpoint: tokenServer.URL,
		Token:    "/token",
	}

	result, err := auth.RefreshTokens(context.Background(), "registry.example.com", authConfig)

	s.NoError(err)
	s.NotNil(result)
	s.Equal("new-access-token", result.AccessToken)
	s.Equal("new-refresh-token", result.RefreshToken)

	// Verify tokens were updated
	savedTokens, err := s.tokenStore.GetRegistryTokens("registry.example.com")
	s.NoError(err)
	s.Equal("new-access-token", savedTokens.AccessToken)
	s.Equal("new-refresh-token", savedTokens.RefreshToken)
}

func (s *OAuth2AuthCodeSuite) TestRefreshTokens_keeps_old_refresh_token_if_not_returned() {
	err := s.tokenStore.SaveRegistryTokens("registry.example.com", &RegistryTokens{
		ClientId:     "test-client",
		AccessToken:  "old-access",
		RefreshToken: "original-refresh",
	})
	s.Require().NoError(err)

	tokenServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// No refresh_token in response
		_, _ = w.Write([]byte(`{"access_token": "new-access", "token_type": "Bearer"}`))
	}))
	defer tokenServer.Close()

	auth := NewOAuth2AuthCodeAuthenticatorWithOptions(
		tokenServer.Client(),
		s.tokenStore,
		nil,
		5*time.Second,
	)

	authConfig := &AuthV1Config{
		Endpoint: tokenServer.URL,
		Token:    "/token",
	}

	result, err := auth.RefreshTokens(context.Background(), "registry.example.com", authConfig)

	s.NoError(err)
	s.Equal("new-access", result.AccessToken)
	s.Equal("original-refresh", result.RefreshToken)
}

func (s *OAuth2AuthCodeSuite) TestRefreshTokens_returns_error_when_no_tokens() {
	auth := NewOAuth2AuthCodeAuthenticator(s.tokenStore)
	authConfig := &AuthV1Config{
		Endpoint: "https://auth.example.com",
		Token:    "/token",
	}

	_, err := auth.RefreshTokens(context.Background(), "nonexistent.example.com", authConfig)

	s.Error(err)
	s.True(errors.Is(err, ErrTokenRefreshFailed))
	s.Contains(err.Error(), "no refresh token available")
}

func (s *OAuth2AuthCodeSuite) TestRefreshTokens_returns_error_when_no_refresh_token() {
	err := s.tokenStore.SaveRegistryTokens("registry.example.com", &RegistryTokens{
		ClientId:    "test-client",
		AccessToken: "access-token",
		// No refresh token
	})
	s.Require().NoError(err)

	auth := NewOAuth2AuthCodeAuthenticator(s.tokenStore)
	authConfig := &AuthV1Config{
		Endpoint: "https://auth.example.com",
		Token:    "/token",
	}

	_, err = auth.RefreshTokens(context.Background(), "registry.example.com", authConfig)

	s.Error(err)
	s.True(errors.Is(err, ErrTokenRefreshFailed))
	s.Contains(err.Error(), "no refresh token available")
}

func (s *OAuth2AuthCodeSuite) TestCallbackServer_binds_to_localhost() {
	cs, err := newCallbackServer()
	s.Require().NoError(err)
	defer cs.Close()

	redirectURI := cs.RedirectURI()
	s.True(strings.HasPrefix(redirectURI, "http://127.0.0.1:"))
	s.True(strings.HasSuffix(redirectURI, "/callback"))
}

func (s *OAuth2AuthCodeSuite) TestGenerateRandomString_generates_unique_strings() {
	str1, err := generateRandomString(32)
	s.NoError(err)
	s.NotEmpty(str1)

	str2, err := generateRandomString(32)
	s.NoError(err)
	s.NotEmpty(str2)

	s.NotEqual(str1, str2, "Random strings should be unique")
}

func TestOAuth2AuthCodeSuite(t *testing.T) {
	suite.Run(t, new(OAuth2AuthCodeSuite))
}
