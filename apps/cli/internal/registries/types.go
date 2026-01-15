package registries

import (
	"slices"
	"time"
)

// AuthType represents the type of authentication supported by a registry.
type AuthType string

const (
	AuthTypeAPIKey            AuthType = "api_key"
	AuthTypeOAuth2ClientCreds AuthType = "oauth2_client_credentials"
	AuthTypeOAuth2AuthCode    AuthType = "oauth2_authorization_code"
)

// ServiceDiscoveryDocument represents the .well-known/bluelink-services.json response.
type ServiceDiscoveryDocument struct {
	Auth          *AuthV1Config        `json:"auth.v1,omitempty"`
	ProviderV1    *PluginServiceConfig `json:"provider.v1,omitempty"`
	TransformerV1 *PluginServiceConfig `json:"transformer.v1,omitempty"`
}

// PluginServiceConfig represents the configuration for plugin services (provider.v1, transformer.v1).
type PluginServiceConfig struct {
	// Endpoint is the base URL for the plugin API (e.g., "/v1/plugins").
	Endpoint string `json:"endpoint"`

	// DownloadAcceptContentType specifies the expected content type for downloads.
	// Defaults to "application/octet-stream" if not specified.
	DownloadAcceptContentType string `json:"downloadAcceptContentType,omitempty"`
}

// AuthV1Config represents the auth.v1 configuration from service discovery.
type AuthV1Config struct {
	// APIKeyHeader is the HTTP header name for API key authentication.
	// When present, indicates API key auth is supported.
	APIKeyHeader string `json:"apiKeyHeader,omitempty"`

	// DownloadAuth specifies the authentication scheme for downloading artifacts.
	// Values: "bearer", "basic", "digest". Defaults to "bearer".
	DownloadAuth string `json:"downloadAuth,omitempty"`

	// Endpoint is the OAuth2 server base URL (e.g., "https://github.com/login/oauth").
	Endpoint string `json:"endpoint,omitempty"`

	// ClientId is the OAuth2 client ID for the authorization code flow.
	// This is provided by the server for auth code flow, not by the user.
	ClientId string `json:"clientId,omitempty"`

	// GrantTypes lists the supported OAuth2 grant types.
	// Supported values: "client_credentials", "authorization_code".
	GrantTypes []string `json:"grantTypes,omitempty"`

	// Authorize is the path for the authorization endpoint (defaults to "/authorize").
	Authorize string `json:"authorize,omitempty"`

	// Token is the path for the token endpoint (required for OAuth2).
	Token string `json:"token,omitempty"`

	// PKCE indicates whether PKCE (Proof Key for Code Exchange) is supported.
	PKCE bool `json:"pkce,omitempty"`
}

// SupportsAuthType checks if the config supports a given authentication type.
func (c *AuthV1Config) SupportsAuthType(authType AuthType) bool {
	if c == nil {
		return false
	}

	switch authType {
	case AuthTypeAPIKey:
		return c.APIKeyHeader != ""
	case AuthTypeOAuth2ClientCreds:
		return c.Endpoint != "" && c.Token != "" && slices.Contains(c.GrantTypes, "client_credentials")
	case AuthTypeOAuth2AuthCode:
		hasAuthCodeGrant := slices.Contains(c.GrantTypes, "authorization_code")
		// If no grant types specified, default to authorization_code
		if len(c.GrantTypes) == 0 && c.Endpoint != "" && c.Token != "" {
			hasAuthCodeGrant = true
		}
		return c.Endpoint != "" && c.Token != "" && hasAuthCodeGrant
	}
	return false
}

// GetSupportedAuthTypes returns all supported authentication types.
func (c *AuthV1Config) GetSupportedAuthTypes() []AuthType {
	if c == nil {
		return nil
	}

	var types []AuthType
	if c.SupportsAuthType(AuthTypeAPIKey) {
		types = append(types, AuthTypeAPIKey)
	}
	if c.SupportsAuthType(AuthTypeOAuth2ClientCreds) {
		types = append(types, AuthTypeOAuth2ClientCreds)
	}
	if c.SupportsAuthType(AuthTypeOAuth2AuthCode) {
		types = append(types, AuthTypeOAuth2AuthCode)
	}
	return types
}

// SupportsPKCE returns whether PKCE is enabled for this config.
func (c *AuthV1Config) SupportsPKCE() bool {
	if c == nil {
		return false
	}
	return c.PKCE
}

// GetAuthorizeURL returns the full authorization URL.
func (c *AuthV1Config) GetAuthorizeURL() string {
	if c == nil || c.Endpoint == "" {
		return ""
	}
	authorize := c.Authorize
	if authorize == "" {
		authorize = "/authorize"
	}
	return c.Endpoint + authorize
}

// GetTokenURL returns the full token URL.
func (c *AuthV1Config) GetTokenURL() string {
	if c == nil || c.Endpoint == "" || c.Token == "" {
		return ""
	}
	return c.Endpoint + c.Token
}

// RegistryAuthConfig holds authentication configuration for a single registry.
// This is stored in plugins.auth.json.
type RegistryAuthConfig struct {
	// APIKey is the API key for API key authentication.
	APIKey string `json:"apiKey,omitempty"`

	// OAuth2 holds OAuth2 client credentials.
	OAuth2 *OAuth2ClientConfig `json:"oauth2,omitempty"`
}

// OAuth2ClientConfig holds OAuth2 client credentials for client_credentials flow.
type OAuth2ClientConfig struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

// AuthConfigFile represents the plugins.auth.json file structure.
// Map of registry host to auth configuration.
type AuthConfigFile map[string]*RegistryAuthConfig

// RegistryTokens holds OAuth2 tokens for authorization code flow.
// This is stored in plugins.tokens.json.
type RegistryTokens struct {
	// ClientId is the OAuth2 client ID (server-provided for auth code flow).
	ClientId string `json:"clientId"`

	// AccessToken is the current access token.
	AccessToken string `json:"accessToken"`

	// RefreshToken is the refresh token for obtaining new access tokens.
	RefreshToken string `json:"refreshToken,omitempty"`

	// TokenExpiry is when the access token expires.
	TokenExpiry *time.Time `json:"tokenExpiry,omitempty"`
}

// IsExpired returns true if the access token is expired or about to expire.
// Considers a token expired if it expires within the next 30 seconds.
func (t *RegistryTokens) IsExpired() bool {
	if t == nil || t.TokenExpiry == nil {
		return false
	}
	return time.Now().Add(30 * time.Second).After(*t.TokenExpiry)
}

// TokensFile represents the plugins.tokens.json file structure.
// Map of registry host to OAuth2 tokens.
type TokensFile map[string]*RegistryTokens
