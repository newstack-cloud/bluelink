// Package main provides a mock plugin registry server for e2e testing.
// It serves service discovery documents and handles authentication requests.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

const (
	defaultPort = "8080"

	// Test credentials
	testAPIKey       = "test-api-key-12345"
	testClientID     = "test-client-id"
	testClientSecret = "test-client-secret"
)

// ServiceDiscoveryDocument represents the .well-known/bluelink-services.json response.
type ServiceDiscoveryDocument struct {
	Auth       *AuthV1Config        `json:"auth.v1,omitempty"`
	ProviderV1 *PluginServiceConfig `json:"provider.v1,omitempty"`
}

// PluginServiceConfig represents plugin service configuration.
type PluginServiceConfig struct {
	Endpoint string `json:"endpoint"`
}

// AuthV1Config represents authentication configuration.
// Mirrors the CLI's registries.AuthV1Config for compatibility.
type AuthV1Config struct {
	// APIKeyHeader indicates API key auth is supported (e.g., "X-API-Key")
	APIKeyHeader string `json:"apiKeyHeader,omitempty"`
	// DownloadAuth specifies download auth scheme
	DownloadAuth string `json:"downloadAuth,omitempty"`
	// Endpoint is the OAuth2 server base URL
	Endpoint string `json:"endpoint,omitempty"`
	// ClientId is for OAuth2 auth code flow
	ClientId string `json:"clientId,omitempty"`
	// GrantTypes lists supported OAuth2 grant types
	GrantTypes []string `json:"grantTypes,omitempty"`
	// Authorize is the path for authorization endpoint
	Authorize string `json:"authorize,omitempty"`
	// Token is the path for token endpoint
	Token string `json:"token,omitempty"`
	// PKCE indicates whether PKCE is supported
	PKCE bool `json:"pkce,omitempty"`
}

// OAuth2TokenResponse represents the response from an OAuth2 token endpoint.
type OAuth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// OAuth2Error represents an OAuth2 error response.
type OAuth2Error struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// PluginVersionsResponse represents the list versions API response.
type PluginVersionsResponse struct {
	Versions []PluginVersionInfo `json:"versions"`
}

// PluginVersionInfo represents a single version in the versions list.
type PluginVersionInfo struct {
	Version string `json:"version"`
}

// PluginPackageMetadata represents metadata for a plugin package.
type PluginPackageMetadata struct {
	Filename            string            `json:"filename"`
	DownloadURL         string            `json:"downloadUrl"`
	OS                  string            `json:"os"`
	Arch                string            `json:"arch"`
	Shasum              string            `json:"shasum"`
	ShasumsURL          string            `json:"shasumsUrl"`
	ShasumsSignatureURL string            `json:"shasumsSignatureUrl"`
	SigningKeys         map[string]string `json:"signingKeys"`
	Dependencies        map[string]string `json:"dependencies,omitempty"`
}

// testPlugin represents a test plugin configuration.
type testPlugin struct {
	namespace    string
	name         string
	versions     []string
	dependencies map[string]string // plugin ID -> version constraint
}

// pluginRegistry holds test plugin data and signing keys.
type pluginRegistry struct {
	signingKey    *openpgp.Entity
	publicKeyPEM  string
	plugins       []testPlugin
	pluginArchive []byte
	archiveShasum string
}

var registry *pluginRegistry

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Initialize plugin registry with test data
	var err error
	registry, err = initPluginRegistry()
	if err != nil {
		log.Fatalf("Failed to initialize plugin registry: %v", err)
	}

	mux := http.NewServeMux()

	// Service discovery endpoint
	mux.HandleFunc("/.well-known/bluelink-services.json", handleServiceDiscovery)

	// API key verification endpoint
	mux.HandleFunc("/auth/verify", handleAPIKeyVerify)

	// OAuth2 token endpoint (for client credentials flow)
	mux.HandleFunc("/auth/token", handleOAuth2Token)

	// OAuth2 authorization endpoint (for authorization code flow)
	mux.HandleFunc("/auth/authorize", handleOAuth2Authorize)

	// Health check endpoint
	mux.HandleFunc("/health", handleHealth)

	// Plugin registry endpoints
	mux.HandleFunc("/v1/plugins/", handlePlugins)

	// Download endpoint for plugin archives and signatures
	mux.HandleFunc("/download/", handleDownload)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      logMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Mock registry server starting on port %s", port)
	log.Printf("Service discovery: http://localhost:%s/.well-known/bluelink-services.json", port)
	log.Printf("Test API Key: %s", testAPIKey)
	log.Printf("Test OAuth2 Client ID: %s", testClientID)
	log.Printf("Test OAuth2 Client Secret: %s", testClientSecret)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func handleServiceDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for auth type query parameter to customize response
	authType := r.URL.Query().Get("auth_type")

	var doc ServiceDiscoveryDocument

	// Plugin service config - always include
	pluginService := &PluginServiceConfig{
		Endpoint: "/v1/plugins",
	}

	switch authType {
	case "api_key":
		// API key auth only - set apiKeyHeader, no OAuth2 fields
		doc = ServiceDiscoveryDocument{
			Auth: &AuthV1Config{
				APIKeyHeader: "X-API-Key",
				DownloadAuth: "bearer",
			},
			ProviderV1: pluginService,
		}
	case "oauth2_client_credentials":
		// OAuth2 client credentials only
		doc = ServiceDiscoveryDocument{
			Auth: &AuthV1Config{
				Endpoint:   fmt.Sprintf("http://%s/auth", r.Host),
				Token:      "/token",
				GrantTypes: []string{"client_credentials"},
			},
			ProviderV1: pluginService,
		}
	case "oauth2_authorization_code":
		// OAuth2 authorization code only
		doc = ServiceDiscoveryDocument{
			Auth: &AuthV1Config{
				Endpoint:   fmt.Sprintf("http://%s/auth", r.Host),
				Token:      "/token",
				Authorize:  "/authorize",
				ClientId:   testClientID,
				GrantTypes: []string{"authorization_code"},
				PKCE:       true,
			},
			ProviderV1: pluginService,
		}
	case "all":
		// All auth types supported
		doc = ServiceDiscoveryDocument{
			Auth: &AuthV1Config{
				APIKeyHeader: "X-API-Key",
				DownloadAuth: "bearer",
				Endpoint:     fmt.Sprintf("http://%s/auth", r.Host),
				Token:        "/token",
				Authorize:    "/authorize",
				ClientId:     testClientID,
				GrantTypes:   []string{"client_credentials", "authorization_code"},
				PKCE:         true,
			},
			ProviderV1: pluginService,
		}
	default:
		// Default: API key only (simplest for testing)
		doc = ServiceDiscoveryDocument{
			Auth: &AuthV1Config{
				APIKeyHeader: "X-API-Key",
				DownloadAuth: "bearer",
			},
			ProviderV1: pluginService,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func handleAPIKeyVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check Authorization header for API key
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(OAuth2Error{
			Error:            "unauthorized",
			ErrorDescription: "Missing Authorization header",
		})
		return
	}

	// Support "Bearer <key>" or just "<key>"
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")

	if apiKey != testAPIKey {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(OAuth2Error{
			Error:            "invalid_token",
			ErrorDescription: "Invalid API key",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "valid"}`)
}

func handleOAuth2Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(OAuth2Error{
			Error:            "invalid_request",
			ErrorDescription: "Failed to parse form",
		})
		return
	}

	grantType := r.Form.Get("grant_type")

	switch grantType {
	case "client_credentials":
		handleClientCredentials(w, r)
	case "authorization_code":
		handleAuthorizationCode(w, r)
	case "refresh_token":
		handleRefreshToken(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(OAuth2Error{
			Error:            "unsupported_grant_type",
			ErrorDescription: fmt.Sprintf("Grant type '%s' is not supported", grantType),
		})
	}
}

func handleClientCredentials(w http.ResponseWriter, r *http.Request) {
	// Check credentials - support both form params and basic auth
	clientID := r.Form.Get("client_id")
	clientSecret := r.Form.Get("client_secret")

	if clientID == "" || clientSecret == "" {
		// Try basic auth
		var ok bool
		clientID, clientSecret, ok = r.BasicAuth()
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(OAuth2Error{
				Error:            "invalid_client",
				ErrorDescription: "Missing client credentials",
			})
			return
		}
	}

	if clientID != testClientID || clientSecret != testClientSecret {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(OAuth2Error{
			Error:            "invalid_client",
			ErrorDescription: "Invalid client credentials",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(OAuth2TokenResponse{
		AccessToken:  "test-access-token-client-creds",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "test-refresh-token-client-creds",
	})
}

func handleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	code := r.Form.Get("code")
	if code == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(OAuth2Error{
			Error:            "invalid_request",
			ErrorDescription: "Missing authorization code",
		})
		return
	}

	// Accept any code for testing purposes
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(OAuth2TokenResponse{
		AccessToken:  "test-access-token-auth-code",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "test-refresh-token-auth-code",
	})
}

func handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.Form.Get("refresh_token")
	if refreshToken == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(OAuth2Error{
			Error:            "invalid_request",
			ErrorDescription: "Missing refresh token",
		})
		return
	}

	// Accept any refresh token for testing
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(OAuth2TokenResponse{
		AccessToken:  "test-access-token-refreshed",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "test-refresh-token-new",
	})
}

func handleOAuth2Authorize(w http.ResponseWriter, r *http.Request) {
	// This endpoint would normally redirect to a login page.
	// For e2e testing, we can simulate auto-approval by redirecting back immediately.
	// However, the actual authorization code flow requires browser interaction,
	// so this endpoint is mainly here for completeness.

	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	if redirectURI == "" {
		http.Error(w, "Missing redirect_uri", http.StatusBadRequest)
		return
	}

	// Auto-approve and redirect with a test code
	redirectURL := fmt.Sprintf("%s?code=test-auth-code&state=%s", redirectURI, state)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// initPluginRegistry creates test plugins and GPG signing keys.
func initPluginRegistry() (*pluginRegistry, error) {
	// Generate GPG key for signing
	config := &packet.Config{
		DefaultHash: crypto.SHA256,
		Time:        func() time.Time { return time.Now() },
	}

	entity, err := openpgp.NewEntity("Test Registry", "E2E Testing", "test@example.com", config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GPG key: %w", err)
	}

	// Export public key to armored PEM
	var pubKeyBuf bytes.Buffer
	armorWriter, err := armor.Encode(&pubKeyBuf, openpgp.PublicKeyType, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create armor encoder: %w", err)
	}
	if err := entity.Serialize(armorWriter); err != nil {
		return nil, fmt.Errorf("failed to serialize public key: %w", err)
	}
	armorWriter.Close()

	// Create a test plugin archive
	archive, shasum, err := createTestPluginArchive()
	if err != nil {
		return nil, fmt.Errorf("failed to create test archive: %w", err)
	}

	return &pluginRegistry{
		signingKey:   entity,
		publicKeyPEM: pubKeyBuf.String(),
		plugins: []testPlugin{
			// Base provider with no dependencies
			{namespace: "bluelink", name: "test-provider", versions: []string{"1.0.0", "1.1.0", "2.0.0"}},
			// Transformer with no dependencies
			{namespace: "bluelink", name: "test-transformer", versions: []string{"1.0.0"}},
			// Provider that depends on test-provider (e.g., implements cross-provider links)
			{
				namespace: "bluelink",
				name:      "aws-link-provider",
				versions:  []string{"1.0.0"},
				dependencies: map[string]string{
					"bluelink/test-provider": "1.0.0",
				},
			},
			// Transformer that depends on both providers
			{
				namespace: "bluelink",
				name:      "multi-cloud-transformer",
				versions:  []string{"1.0.0"},
				dependencies: map[string]string{
					"bluelink/test-provider":    "1.0.0",
					"bluelink/aws-link-provider": "1.0.0",
				},
			},
			// Test cases for error scenarios
			{namespace: "bluelink", name: "bad-signature", versions: []string{"1.0.0"}},
			{namespace: "bluelink", name: "unsigned", versions: []string{"1.0.0"}},
		},
		pluginArchive: archive,
		archiveShasum: shasum,
	}, nil
}

// createTestPluginArchive creates a minimal tar.gz archive for testing.
func createTestPluginArchive() ([]byte, string, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Add a simple executable script
	content := []byte("#!/bin/sh\necho 'test plugin'\n")
	header := &tar.Header{
		Name: "plugin",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return nil, "", err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, "", err
	}

	if err := tw.Close(); err != nil {
		return nil, "", err
	}
	if err := gzw.Close(); err != nil {
		return nil, "", err
	}

	data := buf.Bytes()
	hash := sha256.Sum256(data)
	return data, hex.EncodeToString(hash[:]), nil
}

// handlePlugins routes plugin API requests.
func handlePlugins(w http.ResponseWriter, r *http.Request) {
	// Verify authentication
	if !verifyAuth(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(OAuth2Error{
			Error:            "unauthorized",
			ErrorDescription: "Authentication required",
		})
		return
	}

	// Parse path: /v1/plugins/{namespace}/{name}/...
	path := strings.TrimPrefix(r.URL.Path, "/v1/plugins/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "Invalid plugin path", http.StatusBadRequest)
		return
	}

	namespace := parts[0]
	name := parts[1]

	// Route based on remaining path
	switch {
	case len(parts) == 3 && parts[2] == "versions":
		handleListVersions(w, r, namespace, name)
	case len(parts) == 6 && parts[3] == "package":
		// /v1/plugins/{namespace}/{name}/{version}/package/{os}/{arch}
		version := parts[2]
		osName := parts[4]
		arch := parts[5]
		handleGetPackageMetadata(w, r, namespace, name, version, osName, arch)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func handleListVersions(w http.ResponseWriter, _ *http.Request, namespace, name string) {
	plugin := registry.findPlugin(namespace, name)
	if plugin == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	versions := make([]PluginVersionInfo, len(plugin.versions))
	for i, v := range plugin.versions {
		versions[i] = PluginVersionInfo{Version: v}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PluginVersionsResponse{Versions: versions})
}

func handleGetPackageMetadata(
	w http.ResponseWriter,
	r *http.Request,
	namespace, name, version, osName, arch string,
) {
	plugin := registry.findPlugin(namespace, name)
	if plugin == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !containsVersion(plugin.versions, version) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	filename := fmt.Sprintf("%s_%s_%s_%s.tar.gz", name, version, osName, arch)
	baseURL := fmt.Sprintf("http://%s", r.Host)

	metadata := PluginPackageMetadata{
		Filename:     filename,
		DownloadURL:  fmt.Sprintf("%s/download/%s/%s/%s/%s", baseURL, namespace, name, version, filename),
		OS:           osName,
		Arch:         arch,
		Shasum:       registry.archiveShasum,
		Dependencies: plugin.dependencies,
	}

	// Handle special test cases
	switch name {
	case "unsigned":
		// No signature URLs for unsigned plugin
	case "bad-signature":
		metadata.ShasumsURL = fmt.Sprintf("%s/download/%s/%s/%s/SHA256SUMS", baseURL, namespace, name, version)
		metadata.ShasumsSignatureURL = fmt.Sprintf(
			"%s/download/%s/%s/%s/SHA256SUMS.sig?bad=true",
			baseURL, namespace, name, version,
		)
		metadata.SigningKeys = map[string]string{"gpg_public_key": registry.publicKeyPEM}
	default:
		metadata.ShasumsURL = fmt.Sprintf("%s/download/%s/%s/%s/SHA256SUMS", baseURL, namespace, name, version)
		metadata.ShasumsSignatureURL = fmt.Sprintf(
			"%s/download/%s/%s/%s/SHA256SUMS.sig",
			baseURL, namespace, name, version,
		)
		metadata.SigningKeys = map[string]string{"gpg_public_key": registry.publicKeyPEM}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}

func (r *pluginRegistry) findPlugin(namespace, name string) *testPlugin {
	for i := range r.plugins {
		if r.plugins[i].namespace == namespace && r.plugins[i].name == name {
			return &r.plugins[i]
		}
	}
	return nil
}

func containsVersion(versions []string, version string) bool {
	return slices.Contains(versions, version)
}

func verifyAuth(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// Accept "Bearer <token>" format
	token := strings.TrimPrefix(authHeader, "Bearer ")
	// Accept test API key or any OAuth2 tokens
	return token == testAPIKey ||
		strings.HasPrefix(token, "test-access-token")
}

// handleDownload serves plugin archives and signature files.
func handleDownload(w http.ResponseWriter, r *http.Request) {
	// Path: /download/{namespace}/{name}/{version}/{filename}
	path := strings.TrimPrefix(r.URL.Path, "/download/")
	parts := strings.Split(path, "/")

	if len(parts) < 4 {
		http.Error(w, "Invalid download path", http.StatusBadRequest)
		return
	}

	namespace := parts[0]
	name := parts[1]
	version := parts[2]
	filename := parts[3]

	plugin := registry.findPlugin(namespace, name)
	if plugin == nil || !containsVersion(plugin.versions, version) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch {
	case strings.HasSuffix(filename, ".tar.gz"):
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(registry.pluginArchive)

	case filename == "SHA256SUMS":
		shasums := createShasumsContentAllPlatforms(name, version, registry.archiveShasum)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(shasums))

	case filename == "SHA256SUMS.sig":
		// Check for bad signature query param
		if r.URL.Query().Get("bad") == "true" {
			// Return invalid signature
			w.Header().Set("Content-Type", "application/pgp-signature")
			w.Write([]byte("-----BEGIN PGP SIGNATURE-----\ninvalid\n-----END PGP SIGNATURE-----"))
			return
		}

		// Generate valid signature
		shasums := createShasumsContentAllPlatforms(name, version, registry.archiveShasum)
		sig, err := signData([]byte(shasums), registry.signingKey)
		if err != nil {
			http.Error(w, "Failed to sign", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pgp-signature")
		w.Write(sig)

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// createShasumsContentAllPlatforms generates a SHA256SUMS file with entries for all common platforms.
// This ensures clients on any platform can find their checksum entry.
func createShasumsContentAllPlatforms(name, version, shasum string) string {
	platforms := []struct {
		os   string
		arch string
	}{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"windows", "amd64"},
		{"windows", "arm64"},
	}

	var result strings.Builder
	for _, p := range platforms {
		filename := fmt.Sprintf("%s_%s_%s_%s.tar.gz", name, version, p.os, p.arch)
		result.WriteString(fmt.Sprintf("%s  %s\n", shasum, filename))
	}
	return result.String()
}

func signData(data []byte, entity *openpgp.Entity) ([]byte, error) {
	var buf bytes.Buffer
	// ArmoredDetachSign writes armored output directly to the writer
	if err := openpgp.ArmoredDetachSign(&buf, entity, bytes.NewReader(data), nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
