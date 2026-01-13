package registries

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// GetTokenStorePath returns the platform-specific path for plugins.tokens.json.
func GetTokenStorePath() string {
	if runtime.GOOS == "windows" {
		return os.ExpandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\clients\\plugins.tokens.json")
	}
	return os.ExpandEnv("$HOME/.bluelink/clients/plugins.tokens.json")
}

// TokenStore manages the plugins.tokens.json file for OAuth2 auth code flow tokens.
type TokenStore struct {
	path string
}

// NewTokenStore creates a new token store using the default path.
func NewTokenStore() *TokenStore {
	return &TokenStore{
		path: GetTokenStorePath(),
	}
}

// NewTokenStoreWithPath creates a new token store with a custom path.
// This is primarily useful for testing.
func NewTokenStoreWithPath(path string) *TokenStore {
	return &TokenStore{
		path: path,
	}
}

// Path returns the path to the token store file.
func (s *TokenStore) Path() string {
	return s.path
}

// Load loads the token store file.
func (s *TokenStore) Load() (TokensFile, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return make(TokensFile), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read token store: %w", err)
	}

	var tokens TokensFile
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse token store: %w", err)
	}

	if tokens == nil {
		tokens = make(TokensFile)
	}

	return tokens, nil
}

// Save saves the token store file with restrictive permissions.
func (s *TokenStore) Save(tokens TokensFile) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token store directory: %w", err)
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token store: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write token store: %w", err)
	}

	return nil
}

// SaveRegistryTokens saves tokens for a specific registry.
// The registry host is normalized (scheme stripped) for consistent storage.
func (s *TokenStore) SaveRegistryTokens(registryHost string, tokens *RegistryTokens) error {
	allTokens, err := s.Load()
	if err != nil {
		return err
	}

	normalizedHost := NormalizeRegistryHost(registryHost)
	allTokens[normalizedHost] = tokens
	return s.Save(allTokens)
}

// GetRegistryTokens retrieves tokens for a specific registry.
// The registry host is normalized (scheme stripped) for consistent lookups.
func (s *TokenStore) GetRegistryTokens(registryHost string) (*RegistryTokens, error) {
	tokens, err := s.Load()
	if err != nil {
		return nil, err
	}

	normalizedHost := NormalizeRegistryHost(registryHost)
	return tokens[normalizedHost], nil
}

// RemoveRegistryTokens removes tokens for a specific registry.
// The registry host is normalized (scheme stripped) for consistent lookups.
func (s *TokenStore) RemoveRegistryTokens(registryHost string) error {
	tokens, err := s.Load()
	if err != nil {
		return err
	}

	normalizedHost := NormalizeRegistryHost(registryHost)
	delete(tokens, normalizedHost)
	return s.Save(tokens)
}

// HasRegistryTokens checks if tokens exist for a specific registry.
// The registry host is normalized (scheme stripped) for consistent lookups.
func (s *TokenStore) HasRegistryTokens(registryHost string) (bool, error) {
	tokens, err := s.Load()
	if err != nil {
		return false, err
	}

	normalizedHost := NormalizeRegistryHost(registryHost)
	_, exists := tokens[normalizedHost]
	return exists, nil
}

// GetValidTokens retrieves tokens for a registry only if they are not expired.
// Returns nil if tokens don't exist or are expired.
func (s *TokenStore) GetValidTokens(registryHost string) (*RegistryTokens, error) {
	tokens, err := s.GetRegistryTokens(registryHost)
	if err != nil {
		return nil, err
	}

	if tokens == nil || tokens.IsExpired() {
		return nil, nil
	}

	return tokens, nil
}
