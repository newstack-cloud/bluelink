// Package main provides a combined plugin registry and OAuth2/OIDC test server
// for manual testing of bluelink plugin commands (login, install, uninstall).
//
// Features:
// - Service Discovery (/.well-known/bluelink-services.json)
// - API Key authentication
// - OAuth2 Client Credentials flow
// - OAuth2 Authorization Code flow with PKCE
// - Plugin catalog and download endpoints
//
// This server is intended for local development and testing only.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/gorilla/mux"
	"github.com/lestrrat-go/jwx/jwk"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

const (
	defaultPort = "8080"

	// Test credentials
	defaultClientID     = "test-client-id"
	defaultClientSecret = "test-client-secret"
	defaultAPIKey       = "test-api-key-12345"

	// Token settings
	tokenExpiry = time.Hour
	issuer      = "plugin-registry-test-server"
	keyID       = "test-key-1"

	// Authorization code settings
	authCodeExpiry = 5 * time.Minute
)

var (
	clientID     string
	clientSecret string
	apiKey       string
	privateKey   interface{}
	publicJWKS   []byte

	// Store for authorization codes (in-memory, for testing only)
	authCodes     = make(map[string]*authCodeData)
	authCodeMutex sync.RWMutex
)

type authCodeData struct {
	ClientID     string
	RedirectURI  string
	CodeVerifier string // For PKCE
	ExpiresAt    time.Time
}

// Plugin registry types
type pluginRegistry struct {
	gpgKey        *openpgp.Entity
	publicKeyPEM  string
	plugins       []testPlugin
	pluginArchive []byte
	archiveShasum string
}

type testPlugin struct {
	namespace    string
	name         string
	versions     []string
	dependencies map[string]string // plugin ID -> version constraint
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

var pluginReg *pluginRegistry

// TokenResponse represents an OAuth2 token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// ErrorResponse represents an OAuth2 error response.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func main() {
	// Load configuration from environment
	clientID = getEnv("OAUTH2_CLIENT_ID", defaultClientID)
	clientSecret = getEnv("OAUTH2_CLIENT_SECRET", defaultClientSecret)
	apiKey = getEnv("OAUTH2_API_KEY", defaultAPIKey)
	port := getEnv("PORT", defaultPort)

	// Load keys
	if err := loadKeys(); err != nil {
		log.Fatalf("Failed to load keys: %v", err)
	}

	// Initialize plugin registry
	var err error
	pluginReg, err = initPluginRegistry()
	if err != nil {
		log.Fatalf("Failed to initialize plugin registry: %v", err)
	}

	// Setup router
	r := mux.NewRouter()

	// Service discovery endpoint (Bluelink-specific)
	r.HandleFunc("/.well-known/bluelink-services.json", handleBluelinkServiceDiscovery).Methods("GET")

	// Standard OIDC/OAuth2 discovery endpoints
	r.HandleFunc("/.well-known/openid-configuration", handleOpenIDConfiguration).Methods("GET")
	r.HandleFunc("/.well-known/jwks.json", handleJWKS).Methods("GET")

	// OAuth2 endpoints
	r.HandleFunc("/oauth2/authorize", handleAuthorize).Methods("GET")
	r.HandleFunc("/oauth2/authorize/consent", handleAuthorizeConsent).Methods("POST")
	r.HandleFunc("/oauth2/token", handleToken).Methods("POST")

	// API key verification endpoint
	r.HandleFunc("/auth/verify", handleAPIKeyVerify).Methods("GET", "POST")

	// Health check
	r.HandleFunc("/health", handleHealth).Methods("GET")

	// Plugin registry endpoints
	r.HandleFunc("/v1/plugins/{namespace}/{name}/versions", handleListVersions).Methods("GET")
	r.HandleFunc("/v1/plugins/{namespace}/{name}/{version}/package/{os}/{arch}", handleGetPackageMetadata).Methods("GET")
	r.HandleFunc("/download/{namespace}/{name}/{version}/{filename}", handleDownload).Methods("GET")

	// Start server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      logMiddleware(r),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Plugin Registry Test Server starting on port %s", port)
	log.Printf("")
	log.Printf("Endpoints:")
	log.Printf("  Service Discovery: http://localhost:%s/.well-known/bluelink-services.json", port)
	log.Printf("  OpenID Config:     http://localhost:%s/.well-known/openid-configuration", port)
	log.Printf("  Health Check:      http://localhost:%s/health", port)
	log.Printf("")
	log.Printf("Test Credentials:")
	log.Printf("  Client ID:     %s", clientID)
	log.Printf("  Client Secret: %s", clientSecret)
	log.Printf("  API Key:       %s", apiKey)
	log.Printf("")
	log.Printf("Usage:")
	log.Printf("  bluelink plugins login http://localhost:%s", port)
	log.Printf("  bluelink plugins install localhost:%s/bluelink/test-provider@1.0.0", port)
	log.Printf("")
	log.Printf("Test Plugins:")
	log.Printf("  bluelink/test-provider (1.0.0, 1.1.0, 2.0.0)")
	log.Printf("  bluelink/test-transformer (1.0.0)")
	log.Printf("  bluelink/bad-signature (1.0.0) - returns invalid GPG signature")
	log.Printf("  bluelink/unsigned (1.0.0) - no signature URLs")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loadKeys() error {
	// Try to load private key from file
	keyPath := getEnv("OAUTH2_PRIVATE_KEY_PATH", "keys/private.json")
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		// Generate a new key pair for testing
		log.Printf("No private key found at %s, generating new key pair...", keyPath)
		return generateKeyPair()
	}

	// Parse private key
	keySet, err := jwk.Parse(keyData)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	key, ok := keySet.Get(0)
	if !ok {
		return fmt.Errorf("no key found in key set")
	}

	if err := key.Raw(&privateKey); err != nil {
		return fmt.Errorf("failed to get raw private key: %w", err)
	}

	// Load public JWKS
	jwksPath := getEnv("OAUTH2_JWKS_PATH", "keys/jwks.json")
	publicJWKS, err = os.ReadFile(jwksPath)
	if err != nil {
		return fmt.Errorf("failed to load JWKS: %w", err)
	}

	return nil
}

func generateKeyPair() error {
	// For simplicity in testing, we'll generate an RSA key pair
	// In production, you would load pre-generated keys
	log.Println("Generating RSA key pair for testing...")

	// Generate RSA key pair
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privateKey = rsaKey

	// Create JWKS from the public key
	jwkKey, err := jwk.New(&rsaKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to create JWK: %w", err)
	}
	if err := jwkKey.Set(jwk.KeyIDKey, keyID); err != nil {
		return fmt.Errorf("failed to set key ID: %w", err)
	}
	if err := jwkKey.Set(jwk.AlgorithmKey, "RS256"); err != nil {
		return fmt.Errorf("failed to set algorithm: %w", err)
	}
	if err := jwkKey.Set(jwk.KeyUsageKey, "sig"); err != nil {
		return fmt.Errorf("failed to set key usage: %w", err)
	}

	keySet := jwk.NewSet()
	keySet.Add(jwkKey)

	publicJWKS, err = json.MarshalIndent(keySet, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	return nil
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.String())
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func handleBluelinkServiceDiscovery(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	// Return a service discovery document that supports all auth types
	doc := map[string]interface{}{
		"auth.v1": map[string]interface{}{
			"apiKeyHeader": "X-API-Key",
			"downloadAuth": "bearer",
			"endpoint":     baseURL + "/oauth2",
			"token":        "/token",
			"authorize":    "/authorize",
			"clientId":     clientID,
			"grantTypes":   []string{"client_credentials", "authorization_code"},
			"pkce":         true,
		},
		"provider.v1": map[string]interface{}{
			"endpoint": "/v1/plugins",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func handleOpenIDConfiguration(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	config := map[string]interface{}{
		"issuer":                                issuer,
		"authorization_endpoint":                baseURL + "/oauth2/authorize",
		"token_endpoint":                        baseURL + "/oauth2/token",
		"jwks_uri":                              baseURL + "/.well-known/jwks.json",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "client_credentials"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"code_challenge_methods_supported":      []string{"S256"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func handleJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(publicJWKS)
}

func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Parse authorization request parameters
	responseType := r.URL.Query().Get("response_type")
	reqClientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")
	codeChallenge := r.URL.Query().Get("code_challenge")
	codeChallengeMethod := r.URL.Query().Get("code_challenge_method")

	// Validate request
	if responseType != "code" {
		errorRedirect(w, r, redirectURI, "unsupported_response_type", "Only 'code' response type is supported", state)
		return
	}

	if reqClientID != clientID {
		errorRedirect(w, r, redirectURI, "invalid_client", "Unknown client_id", state)
		return
	}

	if redirectURI == "" {
		http.Error(w, "Missing redirect_uri", http.StatusBadRequest)
		return
	}

	// For PKCE, we need S256
	if codeChallenge != "" && codeChallengeMethod != "S256" {
		errorRedirect(w, r, redirectURI, "invalid_request", "Only S256 code_challenge_method is supported", state)
		return
	}

	// Show a simple consent page
	consentHTML := `<!DOCTYPE html>
<html>
<head>
    <title>OAuth2 Test Server - Authorize</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 500px; margin: 50px auto; padding: 20px; }
        .card { border: 1px solid #ddd; border-radius: 8px; padding: 24px; }
        h1 { color: #333; margin-top: 0; }
        .info { background: #f5f5f5; padding: 12px; border-radius: 4px; margin: 16px 0; }
        .info code { background: #e0e0e0; padding: 2px 6px; border-radius: 3px; }
        button { background: #0066cc; color: white; border: none; padding: 12px 24px; border-radius: 4px; cursor: pointer; font-size: 16px; margin-right: 8px; }
        button:hover { background: #0052a3; }
        button.secondary { background: #666; }
        button.secondary:hover { background: #555; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Authorization Request</h1>
        <p>An application is requesting access to your account.</p>
        <div class="info">
            <strong>Client ID:</strong> <code>{{.ClientID}}</code><br>
            <strong>Redirect URI:</strong> <code>{{.RedirectURI}}</code>
        </div>
        <form method="POST" action="/oauth2/authorize/consent">
            <input type="hidden" name="client_id" value="{{.ClientID}}">
            <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
            <input type="hidden" name="state" value="{{.State}}">
            <input type="hidden" name="code_challenge" value="{{.CodeChallenge}}">
            <button type="submit" name="action" value="approve">Approve</button>
            <button type="submit" name="action" value="deny" class="secondary">Deny</button>
        </form>
    </div>
</body>
</html>`

	tmpl, err := template.New("consent").Parse(consentHTML)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, map[string]string{
		"ClientID":      reqClientID,
		"RedirectURI":   redirectURI,
		"State":         state,
		"CodeChallenge": codeChallenge,
	})
}

func handleAuthorizeConsent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	action := r.FormValue("action")
	reqClientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	state := r.FormValue("state")
	codeChallenge := r.FormValue("code_challenge")

	if action == "deny" {
		errorRedirect(w, r, redirectURI, "access_denied", "User denied the request", state)
		return
	}

	// Generate authorization code
	code := generateRandomString(32)

	// Store the code with associated data
	authCodeMutex.Lock()
	authCodes[code] = &authCodeData{
		ClientID:     reqClientID,
		RedirectURI:  redirectURI,
		CodeVerifier: codeChallenge, // Store the challenge, we'll verify against verifier
		ExpiresAt:    time.Now().Add(authCodeExpiry),
	}
	authCodeMutex.Unlock()

	// Redirect back to client with code
	redirectURL, _ := url.Parse(redirectURI)
	q := redirectURL.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()

	log.Printf("Authorization code issued: %s (redirect: %s)", code[:8]+"...", redirectURI)
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	grantType := r.Form.Get("grant_type")

	switch grantType {
	case "client_credentials":
		handleClientCredentialsGrant(w, r)
	case "authorization_code":
		handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		handleRefreshTokenGrant(w, r)
	default:
		writeError(w, http.StatusBadRequest, "unsupported_grant_type",
			fmt.Sprintf("Grant type '%s' is not supported", grantType))
	}
}

func handleClientCredentialsGrant(w http.ResponseWriter, r *http.Request) {
	// Check credentials via Basic Auth or form params
	reqClientID, reqClientSecret, ok := r.BasicAuth()
	if !ok {
		reqClientID = r.Form.Get("client_id")
		reqClientSecret = r.Form.Get("client_secret")
	}

	if !validateClientCredentials(reqClientID, reqClientSecret) {
		writeError(w, http.StatusUnauthorized, "invalid_client", "Invalid client credentials")
		return
	}

	// Generate tokens
	accessToken, err := generateToken(reqClientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to generate token")
		return
	}

	refreshToken := generateRandomString(32)

	writeTokenResponse(w, accessToken, refreshToken)
}

func handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.Form.Get("code")
	redirectURI := r.Form.Get("redirect_uri")
	codeVerifier := r.Form.Get("code_verifier")

	// Check credentials
	reqClientID, reqClientSecret, ok := r.BasicAuth()
	if !ok {
		reqClientID = r.Form.Get("client_id")
		reqClientSecret = r.Form.Get("client_secret")
	}

	// For auth code flow, client secret may be empty if using PKCE
	if reqClientSecret != "" && !validateClientCredentials(reqClientID, reqClientSecret) {
		writeError(w, http.StatusUnauthorized, "invalid_client", "Invalid client credentials")
		return
	}

	// Validate authorization code
	authCodeMutex.Lock()
	codeData, exists := authCodes[code]
	if exists {
		delete(authCodes, code) // One-time use
	}
	authCodeMutex.Unlock()

	if !exists {
		writeError(w, http.StatusBadRequest, "invalid_grant", "Invalid or expired authorization code")
		return
	}

	if time.Now().After(codeData.ExpiresAt) {
		writeError(w, http.StatusBadRequest, "invalid_grant", "Authorization code has expired")
		return
	}

	if codeData.RedirectURI != redirectURI {
		writeError(w, http.StatusBadRequest, "invalid_grant", "Redirect URI mismatch")
		return
	}

	if codeData.ClientID != reqClientID {
		writeError(w, http.StatusBadRequest, "invalid_grant", "Client ID mismatch")
		return
	}

	// Validate PKCE if code_challenge was provided during authorization
	if codeData.CodeVerifier != "" {
		if codeVerifier == "" {
			writeError(w, http.StatusBadRequest, "invalid_grant", "PKCE code_verifier required")
			return
		}
		if !validatePKCE(codeData.CodeVerifier, codeVerifier) {
			writeError(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
			return
		}
	}

	// Generate tokens
	accessToken, err := generateToken(reqClientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to generate token")
		return
	}

	refreshToken := generateRandomString(32)

	writeTokenResponse(w, accessToken, refreshToken)
}

func handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.Form.Get("refresh_token")
	if refreshToken == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Missing refresh_token")
		return
	}

	// Check credentials
	reqClientID, reqClientSecret, ok := r.BasicAuth()
	if !ok {
		reqClientID = r.Form.Get("client_id")
		reqClientSecret = r.Form.Get("client_secret")
	}

	if reqClientSecret != "" && !validateClientCredentials(reqClientID, reqClientSecret) {
		writeError(w, http.StatusUnauthorized, "invalid_client", "Invalid client credentials")
		return
	}

	// For testing, accept any refresh token and issue new tokens
	accessToken, err := generateToken(reqClientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to generate token")
		return
	}

	newRefreshToken := generateRandomString(32)

	writeTokenResponse(w, accessToken, newRefreshToken)
}

func handleAPIKeyVerify(w http.ResponseWriter, r *http.Request) {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Also check X-API-Key header
		authHeader = r.Header.Get("X-API-Key")
	}

	if authHeader == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing Authorization header")
		return
	}

	// Support "Bearer <key>" or just "<key>"
	providedKey := strings.TrimPrefix(authHeader, "Bearer ")

	if !constantTimeCompare(providedKey, apiKey) {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid API key")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "valid"})
}

func validateClientCredentials(reqClientID, reqClientSecret string) bool {
	return constantTimeCompare(reqClientID, clientID) &&
		constantTimeCompare(reqClientSecret, clientSecret)
}

func constantTimeCompare(a, b string) bool {
	aHash := sha256.Sum256([]byte(a))
	bHash := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(aHash[:], bHash[:]) == 1
}

func validatePKCE(codeChallenge, codeVerifier string) bool {
	// S256: code_challenge = BASE64URL(SHA256(code_verifier))
	h := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(h[:])
	return constantTimeCompare(codeChallenge, computed)
}

func generateToken(subject string) (string, error) {
	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create signer: %w", err)
	}

	now := time.Now()
	claims := jwt.Claims{
		Subject:   subject,
		Issuer:    issuer,
		Audience:  jwt.Audience{clientID},
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Expiry:    jwt.NewNumericDate(now.Add(tokenExpiry)),
	}

	token, err := jwt.Signed(sig).Claims(claims).CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return token, nil
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		log.Printf("Warning: failed to generate random bytes: %v", err)
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}

func writeTokenResponse(w http.ResponseWriter, accessToken, refreshToken string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(tokenExpiry.Seconds()),
		RefreshToken: refreshToken,
	})
}

func writeError(w http.ResponseWriter, status int, errCode, errDesc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:            errCode,
		ErrorDescription: errDesc,
	})
}

func errorRedirect(w http.ResponseWriter, r *http.Request, redirectURI, errCode, errDesc, state string) {
	if redirectURI == "" {
		writeError(w, http.StatusBadRequest, errCode, errDesc)
		return
	}

	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid redirect_uri")
		return
	}

	q := redirectURL.Query()
	q.Set("error", errCode)
	q.Set("error_description", errDesc)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// Plugin registry functions

func initPluginRegistry() (*pluginRegistry, error) {
	// Generate GPG key for signing
	config := &packet.Config{
		DefaultHash: crypto.SHA256,
		Time:        func() time.Time { return time.Now() },
	}

	entity, err := openpgp.NewEntity("Test Registry", "Plugin Testing", "test@example.com", config)
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
		gpgKey:       entity,
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
					"bluelink/test-provider":     "1.0.0",
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

func handleListVersions(w http.ResponseWriter, r *http.Request) {
	if !verifyPluginAuth(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	plugin := pluginReg.findPlugin(namespace, name)
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

func handleGetPackageMetadata(w http.ResponseWriter, r *http.Request) {
	if !verifyPluginAuth(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	version := vars["version"]
	osName := vars["os"]
	arch := vars["arch"]

	plugin := pluginReg.findPlugin(namespace, name)
	if plugin == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !containsVersion(plugin.versions, version) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	filename := fmt.Sprintf("%s_%s_%s_%s.tar.gz", name, version, osName, arch)

	metadata := PluginPackageMetadata{
		Filename:     filename,
		DownloadURL:  fmt.Sprintf("%s/download/%s/%s/%s/%s", baseURL, namespace, name, version, filename),
		OS:           osName,
		Arch:         arch,
		Shasum:       pluginReg.archiveShasum,
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
		metadata.SigningKeys = map[string]string{"gpg_public_key": pluginReg.publicKeyPEM}
	default:
		metadata.ShasumsURL = fmt.Sprintf("%s/download/%s/%s/%s/SHA256SUMS", baseURL, namespace, name, version)
		metadata.ShasumsSignatureURL = fmt.Sprintf(
			"%s/download/%s/%s/%s/SHA256SUMS.sig",
			baseURL, namespace, name, version,
		)
		metadata.SigningKeys = map[string]string{"gpg_public_key": pluginReg.publicKeyPEM}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	version := vars["version"]
	filename := vars["filename"]

	plugin := pluginReg.findPlugin(namespace, name)
	if plugin == nil || !containsVersion(plugin.versions, version) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch {
	case strings.HasSuffix(filename, ".tar.gz"):
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(pluginReg.pluginArchive)

	case filename == "SHA256SUMS":
		shasums := createShasumsContentAllPlatforms(name, version, pluginReg.archiveShasum)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(shasums))

	case filename == "SHA256SUMS.sig":
		// Check for bad signature query param
		if r.URL.Query().Get("bad") == "true" {
			w.Header().Set("Content-Type", "application/pgp-signature")
			w.Write([]byte("-----BEGIN PGP SIGNATURE-----\ninvalid\n-----END PGP SIGNATURE-----"))
			return
		}

		// Generate valid signature
		shasums := createShasumsContentAllPlatforms(name, version, pluginReg.archiveShasum)
		sig, err := signData([]byte(shasums), pluginReg.gpgKey)
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

func (pr *pluginRegistry) findPlugin(namespace, name string) *testPlugin {
	for i := range pr.plugins {
		if pr.plugins[i].namespace == namespace && pr.plugins[i].name == name {
			return &pr.plugins[i]
		}
	}
	return nil
}

func containsVersion(versions []string, version string) bool {
	for _, v := range versions {
		if v == version {
			return true
		}
	}
	return false
}

func verifyPluginAuth(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// Accept "Bearer <token>" format
	token := strings.TrimPrefix(authHeader, "Bearer ")
	// Accept test API key or any JWT tokens (they start with "ey")
	return token == apiKey || strings.HasPrefix(token, "ey")
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
