package registries

import "errors"

var (
	// ErrServiceDiscoveryFailed indicates the service discovery document could not be fetched.
	ErrServiceDiscoveryFailed = errors.New("failed to fetch service discovery document")

	// ErrNoAuthMethodsSupported indicates the registry has no supported authentication methods.
	ErrNoAuthMethodsSupported = errors.New("registry does not support any known authentication methods")

	// ErrCredentialsRequired indicates credentials are required but not provided.
	ErrCredentialsRequired = errors.New("credentials required")

	// ErrAuthenticationFailed indicates the authentication attempt failed.
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrTokenExchangeFailed indicates the OAuth2 token exchange failed.
	ErrTokenExchangeFailed = errors.New("failed to exchange authorization code for token")

	// ErrStateMismatch indicates the OAuth2 state parameter didn't match.
	ErrStateMismatch = errors.New("OAuth2 state parameter mismatch")

	// ErrAuthorizationTimeout indicates the user didn't complete auth in time.
	ErrAuthorizationTimeout = errors.New("authorization timeout - please try again")

	// ErrBrowserOpenFailed indicates the browser could not be opened.
	ErrBrowserOpenFailed = errors.New("failed to open browser for authorization")

	// ErrConfigSaveFailed indicates the auth config could not be saved.
	ErrConfigSaveFailed = errors.New("failed to save authentication configuration")

	// ErrTokenRefreshFailed indicates the OAuth2 token refresh failed.
	ErrTokenRefreshFailed = errors.New("failed to refresh access token")

	// ErrPluginNotFound indicates the requested plugin was not found in the registry.
	ErrPluginNotFound = errors.New("plugin not found")

	// ErrVersionNotFound indicates the requested version was not found for the plugin.
	ErrVersionNotFound = errors.New("version not found")

	// ErrDownloadFailed indicates the plugin package download failed.
	ErrDownloadFailed = errors.New("failed to download plugin")

	// ErrChecksumMismatch indicates the downloaded file's checksum doesn't match expected.
	ErrChecksumMismatch = errors.New("checksum verification failed")

	// ErrSignatureInvalid indicates the GPG signature verification failed.
	ErrSignatureInvalid = errors.New("signature verification failed")

	// ErrSignatureMissing indicates a signature is required but not provided by registry.
	ErrSignatureMissing = errors.New("signature required but not provided by registry")

	// ErrSigningKeysMissing indicates signing keys are required but not provided by registry.
	ErrSigningKeysMissing = errors.New("signing keys required but not provided by registry")

	// ErrExtractionFailed indicates the plugin archive extraction failed.
	ErrExtractionFailed = errors.New("failed to extract plugin archive")

	// ErrInvalidPluginID indicates the plugin ID format is invalid.
	ErrInvalidPluginID = errors.New("invalid plugin ID format")

	// ErrNoCredentials indicates no credentials are stored for the registry.
	ErrNoCredentials = errors.New("no credentials found - run 'bluelink plugins login' first")
)
