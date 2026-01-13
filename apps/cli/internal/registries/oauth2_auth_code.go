package registries

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
)

const (
	defaultAuthCodeTimeout = 5 * time.Minute
	callbackPath           = "/callback"
)

// BrowserOpener is a function that opens a URL in the default browser.
type BrowserOpener func(url string) error

// DefaultBrowserOpener opens a URL in the default system browser.
// Uses absolute paths to system utilities to avoid PATH-based attacks.
func DefaultBrowserOpener(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// /usr/bin/open is the standard location on macOS
		cmd = exec.Command("/usr/bin/open", url)
	case "windows":
		// cmd.exe is always in the system directory, accessed via ComSpec or fixed path
		cmd = exec.Command("C:\\Windows\\System32\\cmd.exe", "/c", "start", url)
	default:
		// xdg-open is typically in /usr/bin on most Linux distributions
		xdgPath := findXdgOpen()
		if xdgPath == "" {
			return fmt.Errorf("xdg-open not found in standard locations")
		}
		cmd = exec.Command(xdgPath, url)
	}

	return cmd.Start()
}

// findXdgOpen searches for xdg-open in standard system directories.
// Returns the absolute path if found, empty string otherwise.
func findXdgOpen() string {
	// Standard locations for xdg-open on Linux systems
	standardPaths := []string{
		"/usr/bin/xdg-open",
		"/usr/local/bin/xdg-open",
		"/bin/xdg-open",
	}

	for _, path := range standardPaths {
		if fileExists(path) {
			return path
		}
	}
	return ""
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// OAuth2AuthCodeAuthenticator handles OAuth2 authorization code authentication.
type OAuth2AuthCodeAuthenticator struct {
	httpClient    *http.Client
	tokenStore    *TokenStore
	browserOpener BrowserOpener
	timeout       time.Duration
}

// NewOAuth2AuthCodeAuthenticator creates a new OAuth2 authorization code authenticator.
func NewOAuth2AuthCodeAuthenticator(tokenStore *TokenStore) *OAuth2AuthCodeAuthenticator {
	return &OAuth2AuthCodeAuthenticator{
		httpClient: &http.Client{
			Timeout: defaultOAuth2Timeout,
		},
		tokenStore:    tokenStore,
		browserOpener: DefaultBrowserOpener,
		timeout:       defaultAuthCodeTimeout,
	}
}

// NewOAuth2AuthCodeAuthenticatorWithOptions creates a new authenticator with custom options.
// This is primarily useful for testing.
func NewOAuth2AuthCodeAuthenticatorWithOptions(
	httpClient *http.Client,
	tokenStore *TokenStore,
	browserOpener BrowserOpener,
	timeout time.Duration,
) *OAuth2AuthCodeAuthenticator {
	return &OAuth2AuthCodeAuthenticator{
		httpClient:    httpClient,
		tokenStore:    tokenStore,
		browserOpener: browserOpener,
		timeout:       timeout,
	}
}

// AuthCodeResult contains the result of an authorization code flow.
type AuthCodeResult struct {
	AccessToken  string
	RefreshToken string
	TokenExpiry  *time.Time
	ClientId     string
}

// Authenticate performs the authorization code flow and stores the resulting tokens.
func (a *OAuth2AuthCodeAuthenticator) Authenticate(
	ctx context.Context,
	registryHost string,
	authConfig *AuthV1Config,
) (*AuthCodeResult, error) {
	if authConfig == nil {
		return nil, fmt.Errorf("%w: auth config is nil", ErrAuthenticationFailed)
	}

	if authConfig.ClientId == "" {
		return nil, fmt.Errorf("%w: client ID not configured in service discovery", ErrAuthenticationFailed)
	}

	// Start the callback server
	callbackServer, err := newCallbackServer()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start callback server: %v", ErrAuthenticationFailed, err)
	}
	defer callbackServer.Close()

	// Create the oauth2 config
	oauthConfig := &oauth2.Config{
		ClientID:    authConfig.ClientId,
		RedirectURL: callbackServer.RedirectURI(),
		Endpoint: oauth2.Endpoint{
			AuthURL:  authConfig.GetAuthorizeURL(),
			TokenURL: authConfig.GetTokenURL(),
		},
	}

	if oauthConfig.Endpoint.AuthURL == "" {
		return nil, fmt.Errorf("%w: authorization URL not configured", ErrAuthenticationFailed)
	}
	if oauthConfig.Endpoint.TokenURL == "" {
		return nil, fmt.Errorf("%w: token URL not configured", ErrAuthenticationFailed)
	}

	// Generate state for CSRF protection
	state, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to generate state: %v", ErrAuthenticationFailed, err)
	}

	// Build authorization URL options
	var authCodeOpts []oauth2.AuthCodeOption

	// Add PKCE if supported
	var codeVerifier string
	if authConfig.SupportsPKCE() {
		codeVerifier, err = generateRandomString(64)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to generate PKCE verifier: %v", ErrAuthenticationFailed, err)
		}
		authCodeOpts = append(authCodeOpts, oauth2.S256ChallengeOption(codeVerifier))
	}

	// Build the authorization URL
	authURL := oauthConfig.AuthCodeURL(state, authCodeOpts...)

	// Open the browser
	if err := a.browserOpener(authURL); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBrowserOpenFailed, err)
	}

	// Wait for the callback
	callbackCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	callbackResult, err := callbackServer.WaitForCallback(callbackCtx)
	if err != nil {
		return nil, err
	}

	// Validate state
	if callbackResult.State != state {
		return nil, ErrStateMismatch
	}

	// Exchange the code for tokens using the oauth2 package
	ctx = context.WithValue(ctx, oauth2.HTTPClient, a.httpClient)

	var exchangeOpts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		exchangeOpts = append(exchangeOpts, oauth2.VerifierOption(codeVerifier))
	}

	token, err := oauthConfig.Exchange(ctx, callbackResult.Code, exchangeOpts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenExchangeFailed, err)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("%w: no access token in response", ErrTokenExchangeFailed)
	}

	// Calculate token expiry
	var tokenExpiry *time.Time
	if !token.Expiry.IsZero() {
		tokenExpiry = &token.Expiry
	}

	// Store the tokens
	tokens := &RegistryTokens{
		ClientId:     authConfig.ClientId,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  tokenExpiry,
	}
	if err := a.tokenStore.SaveRegistryTokens(registryHost, tokens); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigSaveFailed, err)
	}

	return &AuthCodeResult{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  tokenExpiry,
		ClientId:     authConfig.ClientId,
	}, nil
}

// RefreshTokens refreshes expired tokens using the refresh token.
func (a *OAuth2AuthCodeAuthenticator) RefreshTokens(
	ctx context.Context,
	registryHost string,
	authConfig *AuthV1Config,
) (*AuthCodeResult, error) {
	tokens, err := a.tokenStore.GetRegistryTokens(registryHost)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenRefreshFailed, err)
	}

	if tokens == nil || tokens.RefreshToken == "" {
		return nil, fmt.Errorf("%w: no refresh token available", ErrTokenRefreshFailed)
	}

	tokenURL := authConfig.GetTokenURL()
	if tokenURL == "" {
		return nil, fmt.Errorf("%w: token URL not configured", ErrTokenRefreshFailed)
	}

	// Create the oauth2 config for token refresh
	oauthConfig := &oauth2.Config{
		ClientID: tokens.ClientId,
		Endpoint: oauth2.Endpoint{
			TokenURL: tokenURL,
		},
	}

	// Create a token source that will refresh the token
	oldToken := &oauth2.Token{
		RefreshToken: tokens.RefreshToken,
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, a.httpClient)
	tokenSource := oauthConfig.TokenSource(ctx, oldToken)

	// Get the new token (this will trigger a refresh)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenRefreshFailed, err)
	}

	// Calculate new token expiry
	var tokenExpiry *time.Time
	if !newToken.Expiry.IsZero() {
		tokenExpiry = &newToken.Expiry
	}

	// Use the new refresh token if provided, otherwise keep the old one
	refreshToken := newToken.RefreshToken
	if refreshToken == "" {
		refreshToken = tokens.RefreshToken
	}

	// Update stored tokens
	newTokens := &RegistryTokens{
		ClientId:     tokens.ClientId,
		AccessToken:  newToken.AccessToken,
		RefreshToken: refreshToken,
		TokenExpiry:  tokenExpiry,
	}
	if err := a.tokenStore.SaveRegistryTokens(registryHost, newTokens); err != nil {
		return nil, fmt.Errorf("%w: failed to save tokens: %v", ErrTokenRefreshFailed, err)
	}

	return &AuthCodeResult{
		AccessToken:  newToken.AccessToken,
		RefreshToken: refreshToken,
		TokenExpiry:  tokenExpiry,
		ClientId:     tokens.ClientId,
	}, nil
}

// callbackServer handles the OAuth2 callback.
type callbackServer struct {
	listener   net.Listener
	server     *http.Server
	resultChan chan callbackResult
	errChan    chan error
}

type callbackResult struct {
	Code  string
	State string
}

func newCallbackServer() (*callbackServer, error) {
	// Bind to localhost only for security
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	cs := &callbackServer{
		listener:   listener,
		resultChan: make(chan callbackResult, 1),
		errChan:    make(chan error, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, cs.handleCallback)

	cs.server = &http.Server{
		Handler: mux,
	}

	go func() {
		if err := cs.server.Serve(listener); err != http.ErrServerClosed {
			cs.errChan <- err
		}
	}()

	return cs, nil
}

func (cs *callbackServer) RedirectURI() string {
	return fmt.Sprintf("http://%s%s", cs.listener.Addr().String(), callbackPath)
}

func (cs *callbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	errorDesc := r.URL.Query().Get("error_description")

	if errorParam != "" {
		errMsg := errorParam
		if errorDesc != "" {
			errMsg = errorParam + ": " + errorDesc
		}
		cs.errChan <- fmt.Errorf("%w: %s", ErrAuthenticationFailed, errMsg)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(authFailureHTML(errMsg)))
		return
	}

	if code == "" {
		cs.errChan <- fmt.Errorf("%w: no authorization code received", ErrAuthenticationFailed)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(authFailureHTML("No authorization code received")))
		return
	}

	cs.resultChan <- callbackResult{Code: code, State: state}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(authSuccessHTML()))
}

func (cs *callbackServer) WaitForCallback(ctx context.Context) (*callbackResult, error) {
	select {
	case result := <-cs.resultChan:
		return &result, nil
	case err := <-cs.errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ErrAuthorizationTimeout
	}
}

func (cs *callbackServer) Close() error {
	return cs.server.Close()
}

// generateRandomString generates a cryptographically random string.
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func authSuccessHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Authorization Successful - Bluelink</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
            display: flex; justify-content: center; align-items: center;
            min-height: 100vh; margin: 0; padding: 20px;
            background: #f8fafc;
        }
        @media (prefers-color-scheme: dark) {
            body { background: #0f172a; }
            .container { background: #1e293b; border-color: #334155; }
            h1 { color: #f1f5f9; }
            p { color: #94a3b8; }
        }
        .container {
            text-align: center; background: white;
            padding: 48px 56px; border-radius: 16px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -2px rgba(0,0,0,0.1);
            max-width: 420px; width: 100%;
        }
        .logo {
            width: 48px; height: 48px; margin: 0 auto 24px;
            background: #2563eb; border-radius: 12px;
            display: flex; align-items: center; justify-content: center;
        }
        .logo svg { width: 28px; height: 28px; }
        .icon-circle {
            width: 64px; height: 64px; margin: 0 auto 20px;
            background: #dcfce7; border-radius: 50%;
            display: flex; align-items: center; justify-content: center;
        }
        .icon-circle svg { width: 32px; height: 32px; color: #16a34a; }
        h1 { color: #0f172a; font-size: 22px; font-weight: 600; margin: 0 0 8px; }
        p { color: #64748b; font-size: 15px; margin: 0; line-height: 1.5; }
        .brand { color: #64748b; font-size: 13px; margin-top: 32px; }
        .brand a { color: #2563eb; text-decoration: none; font-weight: 500; }
        .brand a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">
            <svg viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
            </svg>
        </div>
        <div class="icon-circle">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"/>
            </svg>
        </div>
        <h1>Authorization Successful</h1>
        <p>You can close this window and return to the terminal.</p>
        <p class="brand">Powered by <a href="https://bluelink.dev">Bluelink</a></p>
    </div>
</body>
</html>`
}

func authFailureHTML(errMsg string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Authorization Failed - Bluelink</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
            display: flex; justify-content: center; align-items: center;
            min-height: 100vh; margin: 0; padding: 20px;
            background: #f8fafc;
        }
        @media (prefers-color-scheme: dark) {
            body { background: #0f172a; }
            .container { background: #1e293b; border-color: #334155; }
            h1 { color: #f1f5f9; }
            p { color: #94a3b8; }
            .error-detail { background: #450a0a; border-color: #7f1d1d; color: #fca5a5; }
        }
        .container {
            text-align: center; background: white;
            padding: 48px 56px; border-radius: 16px;
            border: 1px solid #e2e8f0;
            box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -2px rgba(0,0,0,0.1);
            max-width: 420px; width: 100%%;
        }
        .logo {
            width: 48px; height: 48px; margin: 0 auto 24px;
            background: #2563eb; border-radius: 12px;
            display: flex; align-items: center; justify-content: center;
        }
        .logo svg { width: 28px; height: 28px; }
        .icon-circle {
            width: 64px; height: 64px; margin: 0 auto 20px;
            background: #fee2e2; border-radius: 50%%;
            display: flex; align-items: center; justify-content: center;
        }
        .icon-circle svg { width: 32px; height: 32px; color: #dc2626; }
        h1 { color: #0f172a; font-size: 22px; font-weight: 600; margin: 0 0 8px; }
        p { color: #64748b; font-size: 15px; margin: 0; line-height: 1.5; }
        .error-detail {
            background: #fef2f2; border: 1px solid #fecaca;
            border-radius: 8px; padding: 12px 16px;
            color: #b91c1c; font-size: 13px; margin-top: 20px;
            font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, monospace;
            word-break: break-word;
        }
        .brand { color: #64748b; font-size: 13px; margin-top: 32px; }
        .brand a { color: #2563eb; text-decoration: none; font-weight: 500; }
        .brand a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">
            <svg viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
            </svg>
        </div>
        <div class="icon-circle">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
        </div>
        <h1>Authorization Failed</h1>
        <p>An error occurred during authorization.</p>
        <p class="error-detail">%s</p>
        <p class="brand">Powered by <a href="https://bluelink.dev">Bluelink</a></p>
    </div>
</body>
</html>`, errMsg)
}
