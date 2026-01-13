package registries

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// GetAuthConfigPath returns the platform-specific path for plugins.auth.json.
func GetAuthConfigPath() string {
	if runtime.GOOS == "windows" {
		return os.ExpandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\clients\\plugins.auth.json")
	}
	return os.ExpandEnv("$HOME/.bluelink/clients/plugins.auth.json")
}

// AuthConfigStore manages the plugins.auth.json file.
type AuthConfigStore struct {
	path string
}

// NewAuthConfigStore creates a new auth config store using the default path.
func NewAuthConfigStore() *AuthConfigStore {
	return &AuthConfigStore{
		path: GetAuthConfigPath(),
	}
}

// NewAuthConfigStoreWithPath creates a new auth config store with a custom path.
// This is primarily useful for testing.
func NewAuthConfigStoreWithPath(path string) *AuthConfigStore {
	return &AuthConfigStore{
		path: path,
	}
}

// Path returns the path to the auth config file.
func (s *AuthConfigStore) Path() string {
	return s.path
}

// Load loads the auth config file.
func (s *AuthConfigStore) Load() (AuthConfigFile, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return make(AuthConfigFile), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read auth config: %w", err)
	}

	var config AuthConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse auth config: %w", err)
	}

	if config == nil {
		config = make(AuthConfigFile)
	}

	return config, nil
}

// Save saves the auth config file with restrictive permissions.
func (s *AuthConfigStore) Save(config AuthConfigFile) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth config: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write auth config: %w", err)
	}

	return nil
}

// SaveRegistryAuth saves authentication for a specific registry.
// The registry host is normalized (scheme stripped) for consistent storage.
func (s *AuthConfigStore) SaveRegistryAuth(registryHost string, auth *RegistryAuthConfig) error {
	config, err := s.Load()
	if err != nil {
		return err
	}

	normalizedHost := NormalizeRegistryHost(registryHost)
	config[normalizedHost] = auth
	return s.Save(config)
}

// GetRegistryAuth retrieves authentication for a specific registry.
// The registry host is normalized (scheme stripped) for consistent lookups.
func (s *AuthConfigStore) GetRegistryAuth(registryHost string) (*RegistryAuthConfig, error) {
	config, err := s.Load()
	if err != nil {
		return nil, err
	}

	normalizedHost := NormalizeRegistryHost(registryHost)
	return config[normalizedHost], nil
}

// RemoveRegistryAuth removes authentication for a specific registry.
// The registry host is normalized (scheme stripped) for consistent lookups.
func (s *AuthConfigStore) RemoveRegistryAuth(registryHost string) error {
	config, err := s.Load()
	if err != nil {
		return err
	}

	normalizedHost := NormalizeRegistryHost(registryHost)
	delete(config, normalizedHost)
	return s.Save(config)
}

// HasRegistryAuth checks if authentication exists for a specific registry.
// The registry host is normalized (scheme stripped) for consistent lookups.
func (s *AuthConfigStore) HasRegistryAuth(registryHost string) (bool, error) {
	config, err := s.Load()
	if err != nil {
		return false, err
	}

	normalizedHost := NormalizeRegistryHost(registryHost)
	_, exists := config[normalizedHost]
	return exists, nil
}
