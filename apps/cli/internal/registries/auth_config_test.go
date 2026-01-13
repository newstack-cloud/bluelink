package registries

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
)

type AuthConfigStoreSuite struct {
	suite.Suite
	tempDir string
}

func (s *AuthConfigStoreSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "auth-config-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *AuthConfigStoreSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *AuthConfigStoreSuite) TestGetAuthConfigPath_returns_platform_specific_path() {
	path := GetAuthConfigPath()
	s.NotEmpty(path)

	if runtime.GOOS == "windows" {
		s.Contains(path, "NewStack")
		s.Contains(path, "Bluelink")
		s.Contains(path, "plugins.auth.json")
	} else {
		s.Contains(path, ".bluelink")
		s.Contains(path, "clients")
		s.Contains(path, "plugins.auth.json")
	}
}

func (s *AuthConfigStoreSuite) TestNewAuthConfigStore_uses_default_path() {
	store := NewAuthConfigStore()
	s.Equal(GetAuthConfigPath(), store.Path())
}

func (s *AuthConfigStoreSuite) TestNewAuthConfigStoreWithPath_uses_custom_path() {
	customPath := filepath.Join(s.tempDir, "custom.json")
	store := NewAuthConfigStoreWithPath(customPath)
	s.Equal(customPath, store.Path())
}

func (s *AuthConfigStoreSuite) TestLoad_returns_empty_map_when_file_does_not_exist() {
	store := NewAuthConfigStoreWithPath(filepath.Join(s.tempDir, "nonexistent.json"))

	config, err := store.Load()

	s.NoError(err)
	s.NotNil(config)
	s.Empty(config)
}

func (s *AuthConfigStoreSuite) TestLoad_returns_error_for_invalid_json() {
	configPath := filepath.Join(s.tempDir, "invalid.json")
	err := os.WriteFile(configPath, []byte("not valid json"), 0600)
	s.Require().NoError(err)

	store := NewAuthConfigStoreWithPath(configPath)
	_, err = store.Load()

	s.Error(err)
	s.Contains(err.Error(), "failed to parse auth config")
}

func (s *AuthConfigStoreSuite) TestLoad_parses_valid_config_with_api_key() {
	configPath := filepath.Join(s.tempDir, "config.json")
	content := `{
		"registry.example.com": {
			"apiKey": "test-api-key"
		}
	}`
	err := os.WriteFile(configPath, []byte(content), 0600)
	s.Require().NoError(err)

	store := NewAuthConfigStoreWithPath(configPath)
	config, err := store.Load()

	s.NoError(err)
	s.Require().NotNil(config["registry.example.com"])
	s.Equal("test-api-key", config["registry.example.com"].APIKey)
}

func (s *AuthConfigStoreSuite) TestLoad_parses_valid_config_with_oauth2() {
	configPath := filepath.Join(s.tempDir, "config.json")
	content := `{
		"registry.example.com": {
			"oauth2": {
				"clientId": "test-client-id",
				"clientSecret": "test-client-secret"
			}
		}
	}`
	err := os.WriteFile(configPath, []byte(content), 0600)
	s.Require().NoError(err)

	store := NewAuthConfigStoreWithPath(configPath)
	config, err := store.Load()

	s.NoError(err)
	s.Require().NotNil(config["registry.example.com"])
	s.Require().NotNil(config["registry.example.com"].OAuth2)
	s.Equal("test-client-id", config["registry.example.com"].OAuth2.ClientId)
	s.Equal("test-client-secret", config["registry.example.com"].OAuth2.ClientSecret)
}

func (s *AuthConfigStoreSuite) TestLoad_handles_null_json() {
	configPath := filepath.Join(s.tempDir, "config.json")
	err := os.WriteFile(configPath, []byte("null"), 0600)
	s.Require().NoError(err)

	store := NewAuthConfigStoreWithPath(configPath)
	config, err := store.Load()

	s.NoError(err)
	s.NotNil(config)
	s.Empty(config)
}

func (s *AuthConfigStoreSuite) TestSave_creates_parent_directories() {
	nestedPath := filepath.Join(s.tempDir, "nested", "dir", "config.json")
	store := NewAuthConfigStoreWithPath(nestedPath)

	config := make(AuthConfigFile)
	config["registry.example.com"] = &RegistryAuthConfig{APIKey: "test-key"}

	err := store.Save(config)

	s.NoError(err)
	s.FileExists(nestedPath)
}

func (s *AuthConfigStoreSuite) TestSave_writes_valid_json() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	config := make(AuthConfigFile)
	config["registry.example.com"] = &RegistryAuthConfig{APIKey: "test-key"}

	err := store.Save(config)
	s.NoError(err)

	// Verify by loading
	loaded, err := store.Load()
	s.NoError(err)
	s.Require().NotNil(loaded["registry.example.com"])
	s.Equal("test-key", loaded["registry.example.com"].APIKey)
}

func (s *AuthConfigStoreSuite) TestSave_sets_restrictive_permissions() {
	if runtime.GOOS == "windows" {
		s.T().Skip("File permissions work differently on Windows")
	}

	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	config := make(AuthConfigFile)
	config["registry.example.com"] = &RegistryAuthConfig{APIKey: "test-key"}

	err := store.Save(config)
	s.NoError(err)

	info, err := os.Stat(configPath)
	s.NoError(err)
	s.Equal(os.FileMode(0600), info.Mode().Perm())
}

func (s *AuthConfigStoreSuite) TestSaveRegistryAuth_adds_new_registry() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	auth := &RegistryAuthConfig{APIKey: "new-key"}
	err := store.SaveRegistryAuth("new-registry.example.com", auth)

	s.NoError(err)

	loaded, err := store.Load()
	s.NoError(err)
	s.Require().NotNil(loaded["new-registry.example.com"])
	s.Equal("new-key", loaded["new-registry.example.com"].APIKey)
}

func (s *AuthConfigStoreSuite) TestSaveRegistryAuth_updates_existing_registry() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	// Save initial auth
	initialAuth := &RegistryAuthConfig{APIKey: "initial-key"}
	err := store.SaveRegistryAuth("registry.example.com", initialAuth)
	s.Require().NoError(err)

	// Update with new auth
	updatedAuth := &RegistryAuthConfig{APIKey: "updated-key"}
	err = store.SaveRegistryAuth("registry.example.com", updatedAuth)
	s.NoError(err)

	loaded, err := store.Load()
	s.NoError(err)
	s.Equal("updated-key", loaded["registry.example.com"].APIKey)
}

func (s *AuthConfigStoreSuite) TestSaveRegistryAuth_preserves_other_registries() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	// Save first registry
	auth1 := &RegistryAuthConfig{APIKey: "key-1"}
	err := store.SaveRegistryAuth("registry1.example.com", auth1)
	s.Require().NoError(err)

	// Save second registry
	auth2 := &RegistryAuthConfig{APIKey: "key-2"}
	err = store.SaveRegistryAuth("registry2.example.com", auth2)
	s.NoError(err)

	loaded, err := store.Load()
	s.NoError(err)
	s.Len(loaded, 2)
	s.Equal("key-1", loaded["registry1.example.com"].APIKey)
	s.Equal("key-2", loaded["registry2.example.com"].APIKey)
}

func (s *AuthConfigStoreSuite) TestGetRegistryAuth_returns_nil_for_nonexistent_registry() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	auth, err := store.GetRegistryAuth("nonexistent.example.com")

	s.NoError(err)
	s.Nil(auth)
}

func (s *AuthConfigStoreSuite) TestGetRegistryAuth_returns_auth_for_existing_registry() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	savedAuth := &RegistryAuthConfig{APIKey: "test-key"}
	err := store.SaveRegistryAuth("registry.example.com", savedAuth)
	s.Require().NoError(err)

	auth, err := store.GetRegistryAuth("registry.example.com")

	s.NoError(err)
	s.Require().NotNil(auth)
	s.Equal("test-key", auth.APIKey)
}

func (s *AuthConfigStoreSuite) TestRemoveRegistryAuth_removes_existing_registry() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	// Save registry
	auth := &RegistryAuthConfig{APIKey: "test-key"}
	err := store.SaveRegistryAuth("registry.example.com", auth)
	s.Require().NoError(err)

	// Remove registry
	err = store.RemoveRegistryAuth("registry.example.com")
	s.NoError(err)

	// Verify removal
	loaded, err := store.Load()
	s.NoError(err)
	s.Nil(loaded["registry.example.com"])
}

func (s *AuthConfigStoreSuite) TestRemoveRegistryAuth_succeeds_for_nonexistent_registry() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	err := store.RemoveRegistryAuth("nonexistent.example.com")
	s.NoError(err)
}

func (s *AuthConfigStoreSuite) TestRemoveRegistryAuth_preserves_other_registries() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	// Save two registries
	auth1 := &RegistryAuthConfig{APIKey: "key-1"}
	err := store.SaveRegistryAuth("registry1.example.com", auth1)
	s.Require().NoError(err)

	auth2 := &RegistryAuthConfig{APIKey: "key-2"}
	err = store.SaveRegistryAuth("registry2.example.com", auth2)
	s.Require().NoError(err)

	// Remove first registry
	err = store.RemoveRegistryAuth("registry1.example.com")
	s.NoError(err)

	// Verify second registry still exists
	loaded, err := store.Load()
	s.NoError(err)
	s.Len(loaded, 1)
	s.Nil(loaded["registry1.example.com"])
	s.NotNil(loaded["registry2.example.com"])
	s.Equal("key-2", loaded["registry2.example.com"].APIKey)
}

func (s *AuthConfigStoreSuite) TestHasRegistryAuth_returns_false_for_nonexistent_registry() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	exists, err := store.HasRegistryAuth("nonexistent.example.com")

	s.NoError(err)
	s.False(exists)
}

func (s *AuthConfigStoreSuite) TestHasRegistryAuth_returns_true_for_existing_registry() {
	configPath := filepath.Join(s.tempDir, "config.json")
	store := NewAuthConfigStoreWithPath(configPath)

	auth := &RegistryAuthConfig{APIKey: "test-key"}
	err := store.SaveRegistryAuth("registry.example.com", auth)
	s.Require().NoError(err)

	exists, err := store.HasRegistryAuth("registry.example.com")

	s.NoError(err)
	s.True(exists)
}

func TestAuthConfigStoreSuite(t *testing.T) {
	suite.Run(t, new(AuthConfigStoreSuite))
}
