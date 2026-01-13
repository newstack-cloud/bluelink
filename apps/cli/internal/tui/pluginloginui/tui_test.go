package pluginloginui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/registries"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/testutils"
	"github.com/stretchr/testify/suite"
)

type PluginLoginTUISuite struct {
	suite.Suite
	tempDir string
}

func (s *PluginLoginTUISuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "plugin-login-tui-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *PluginLoginTUISuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *PluginLoginTUISuite) newTestStyles() *stylespkg.Styles {
	return stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *PluginLoginTUISuite) newTestAuthConfigStore() *registries.AuthConfigStore {
	return registries.NewAuthConfigStoreWithPath(filepath.Join(s.tempDir, "plugins.auth.json"))
}

func (s *PluginLoginTUISuite) Test_successful_api_key_login() {
	// Create mock registry server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/bluelink-services.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"auth.v1": {
					"apiKeyHeader": "X-API-Key"
				}
			}`))
			return
		}
		// API key verification
		if r.Header.Get("X-API-Key") == "test-api-key" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	// Pass full URL with scheme - Discover respects explicit schemes
	authConfigStore := s.newTestAuthConfigStore()

	model, err := NewLoginApp(
		context.Background(),
		LoginAppOptions{
			RegistryHost:        server.URL,
			Styles:              s.newTestStyles(),
			Headless:            false,
			DiscoveryClient:     registries.NewServiceDiscoveryClientWithHTTPClient(server.Client()),
			APIKeyAuthenticator: registries.NewAPIKeyAuthenticatorWithHTTPClient(server.Client(), authConfigStore),
		},
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for API key form to appear
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"API Key",
		server.URL,
	)

	// Type the API key
	testModel.Type("test-api-key")
	testutils.KeyEnter(testModel)

	// Wait for success
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Successfully logged in",
		server.URL,
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)

	// Verify credentials were saved (normalized host without scheme)
	savedAuth, err := authConfigStore.GetRegistryAuth(server.URL)
	s.NoError(err)
	s.NotNil(savedAuth)
	s.Equal("test-api-key", savedAuth.APIKey)
}

func (s *PluginLoginTUISuite) Test_successful_oauth2_client_creds_login() {
	// Create mock registry server - use a pointer to capture server URL
	var serverURL string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/bluelink-services.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"auth.v1": {
					"endpoint": "` + serverURL + `",
					"grantTypes": ["client_credentials"],
					"token": "/token"
				}
			}`))
			return
		}
		if r.URL.Path == "/token" {
			// Accept any client credentials for this test
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	serverURL = server.URL
	defer server.Close()

	// Pass full URL with scheme - Discover respects explicit schemes
	authConfigStore := s.newTestAuthConfigStore()

	model, err := NewLoginApp(
		context.Background(),
		LoginAppOptions{
			RegistryHost:          server.URL,
			Styles:                s.newTestStyles(),
			Headless:              false,
			DiscoveryClient:       registries.NewServiceDiscoveryClientWithHTTPClient(server.Client()),
			OAuth2ClientCredsAuth: registries.NewOAuth2ClientCredsAuthenticatorWithHTTPClient(server.Client(), authConfigStore),
		},
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for OAuth2 client credentials form to appear
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Client ID",
	)

	// Type client ID and submit
	testModel.Type("myclient")
	testutils.KeyEnter(testModel)

	// Wait for client secret field
	testutils.WaitForContains(s.T(), testModel.Output(), "Client Secret")

	// Type client secret and submit
	testModel.Type("mysecret")
	testutils.KeyEnter(testModel)

	// Wait for success
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Successfully logged in",
		server.URL,
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.Nil(finalModel.Error)

	// Verify credentials were saved (normalized host without scheme)
	savedAuth, err := authConfigStore.GetRegistryAuth(server.URL)
	s.NoError(err)
	s.NotNil(savedAuth)
	s.NotNil(savedAuth.OAuth2)
	s.Equal("myclient", savedAuth.OAuth2.ClientId)
}

func (s *PluginLoginTUISuite) Test_service_discovery_error() {
	// Create mock registry server that returns an error
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Pass full URL with scheme - Discover respects explicit schemes
	model, err := NewLoginApp(
		context.Background(),
		LoginAppOptions{
			RegistryHost:    server.URL,
			Styles:          s.newTestStyles(),
			Headless:        false,
			DiscoveryClient: registries.NewServiceDiscoveryClientWithHTTPClient(server.Client()),
		},
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for error display
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Login failed",
		"HTTP 500",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.NotNil(finalModel.Error)
}

func (s *PluginLoginTUISuite) Test_no_auth_methods_supported_error() {
	// Create mock registry server with no auth config
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/bluelink-services.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Empty auth.v1 config - no auth methods
			_, _ = w.Write([]byte(`{"auth.v1": {}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Pass full URL with scheme - Discover respects explicit schemes
	model, err := NewLoginApp(
		context.Background(),
		LoginAppOptions{
			RegistryHost:    server.URL,
			Styles:          s.newTestStyles(),
			Headless:        false,
			DiscoveryClient: registries.NewServiceDiscoveryClientWithHTTPClient(server.Client()),
		},
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for error display
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Login failed",
		"does not support any known authentication methods",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *PluginLoginTUISuite) Test_authentication_failed_error() {
	// Create mock registry server that rejects credentials
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/bluelink-services.json" {
			// Check if there's an API key header - if so, reject it
			if r.Header.Get("X-API-Key") != "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"auth.v1": {
					"apiKeyHeader": "X-API-Key"
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	// Pass full URL with scheme - Discover respects explicit schemes
	authConfigStore := s.newTestAuthConfigStore()

	model, err := NewLoginApp(
		context.Background(),
		LoginAppOptions{
			RegistryHost:        server.URL,
			Styles:              s.newTestStyles(),
			Headless:            false,
			DiscoveryClient:     registries.NewServiceDiscoveryClientWithHTTPClient(server.Client()),
			APIKeyAuthenticator: registries.NewAPIKeyAuthenticatorWithHTTPClient(server.Client(), authConfigStore),
		},
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for API key form
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"API Key",
	)

	// Type an invalid API key
	testModel.Type("invalid-key")
	testutils.KeyEnter(testModel)

	// Wait for error display
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Login failed",
		"invalid API key",
	)

	testutils.KeyQ(testModel)
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *PluginLoginTUISuite) Test_headless_service_discovery_error() {
	// Create mock registry server that returns an error
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Pass full URL with scheme - Discover respects explicit schemes
	headlessOutput := testutils.NewSaveBuffer()

	model, err := NewLoginApp(
		context.Background(),
		LoginAppOptions{
			RegistryHost:    server.URL,
			Styles:          s.newTestStyles(),
			Headless:        true,
			HeadlessWriter:  headlessOutput,
			DiscoveryClient: registries.NewServiceDiscoveryClientWithHTTPClient(server.Client()),
		},
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// In headless mode, errors are written to the headless output
	testutils.WaitForContainsAll(
		s.T(),
		headlessOutput,
		"Discovering authentication methods",
		"Error:",
	)

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func (s *PluginLoginTUISuite) Test_ctrl_c_cancellation() {
	// Create mock registry server with slow response
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/bluelink-services.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"auth.v1": {
					"apiKeyHeader": "X-API-Key"
				}
			}`))
			return
		}
	}))
	defer server.Close()

	// Pass full URL with scheme - Discover respects explicit schemes
	model, err := NewLoginApp(
		context.Background(),
		LoginAppOptions{
			RegistryHost:    server.URL,
			Styles:          s.newTestStyles(),
			Headless:        false,
			DiscoveryClient: registries.NewServiceDiscoveryClientWithHTTPClient(server.Client()),
		},
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for API key form to appear
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"API Key",
	)

	// Press Ctrl+C to cancel
	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))

	finalModel := testModel.FinalModel(s.T()).(MainModel)
	s.True(finalModel.quitting)
}

func (s *PluginLoginTUISuite) Test_auth_type_selection_shown_when_multiple_supported() {
	// Create mock registry server that supports multiple auth types
	var serverURL string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/bluelink-services.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"auth.v1": {
					"apiKeyHeader": "X-API-Key",
					"endpoint": "` + serverURL + `",
					"grantTypes": ["client_credentials"],
					"token": "/token"
				}
			}`))
			return
		}
	}))
	serverURL = server.URL
	defer server.Close()

	// Pass full URL with scheme - Discover respects explicit schemes
	model, err := NewLoginApp(
		context.Background(),
		LoginAppOptions{
			RegistryHost:    server.URL,
			Styles:          s.newTestStyles(),
			Headless:        false,
			DiscoveryClient: registries.NewServiceDiscoveryClientWithHTTPClient(server.Client()),
		},
	)
	s.Require().NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		model,
		teatest.WithInitialTermSize(300, 100),
	)

	// Wait for auth type selection form
	testutils.WaitForContainsAll(
		s.T(),
		testModel.Output(),
		"Authentication Method",
		"API Key",
		"OAuth2 Client Credentials",
	)

	testModel.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	testModel.WaitFinished(s.T(), teatest.WithFinalTimeout(5*time.Second))
}

func TestPluginLoginTUISuite(t *testing.T) {
	suite.Run(t, new(PluginLoginTUISuite))
}
