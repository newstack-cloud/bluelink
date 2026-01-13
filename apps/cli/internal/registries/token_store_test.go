package registries

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TokenStoreSuite struct {
	suite.Suite
	tempDir string
}

func (s *TokenStoreSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "token-store-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *TokenStoreSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *TokenStoreSuite) TestGetTokenStorePath_returns_platform_specific_path() {
	path := GetTokenStorePath()
	s.NotEmpty(path)

	if runtime.GOOS == "windows" {
		s.Contains(path, "NewStack")
		s.Contains(path, "Bluelink")
		s.Contains(path, "plugins.tokens.json")
	} else {
		s.Contains(path, ".bluelink")
		s.Contains(path, "clients")
		s.Contains(path, "plugins.tokens.json")
	}
}

func (s *TokenStoreSuite) TestNewTokenStore_uses_default_path() {
	store := NewTokenStore()
	s.Equal(GetTokenStorePath(), store.Path())
}

func (s *TokenStoreSuite) TestNewTokenStoreWithPath_uses_custom_path() {
	customPath := filepath.Join(s.tempDir, "custom-tokens.json")
	store := NewTokenStoreWithPath(customPath)
	s.Equal(customPath, store.Path())
}

func (s *TokenStoreSuite) TestLoad_returns_empty_map_when_file_does_not_exist() {
	store := NewTokenStoreWithPath(filepath.Join(s.tempDir, "nonexistent.json"))

	tokens, err := store.Load()

	s.NoError(err)
	s.NotNil(tokens)
	s.Empty(tokens)
}

func (s *TokenStoreSuite) TestLoad_returns_error_for_invalid_json() {
	tokensPath := filepath.Join(s.tempDir, "invalid.json")
	err := os.WriteFile(tokensPath, []byte("not valid json"), 0600)
	s.Require().NoError(err)

	store := NewTokenStoreWithPath(tokensPath)
	_, err = store.Load()

	s.Error(err)
	s.Contains(err.Error(), "failed to parse token store")
}

func (s *TokenStoreSuite) TestLoad_parses_valid_tokens() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	expiry := time.Now().Add(time.Hour)
	content := `{
		"registry.example.com": {
			"clientId": "test-client-id",
			"accessToken": "test-access-token",
			"refreshToken": "test-refresh-token",
			"tokenExpiry": "` + expiry.Format(time.RFC3339Nano) + `"
		}
	}`
	err := os.WriteFile(tokensPath, []byte(content), 0600)
	s.Require().NoError(err)

	store := NewTokenStoreWithPath(tokensPath)
	tokens, err := store.Load()

	s.NoError(err)
	s.Require().NotNil(tokens["registry.example.com"])
	s.Equal("test-client-id", tokens["registry.example.com"].ClientId)
	s.Equal("test-access-token", tokens["registry.example.com"].AccessToken)
	s.Equal("test-refresh-token", tokens["registry.example.com"].RefreshToken)
	s.NotNil(tokens["registry.example.com"].TokenExpiry)
}

func (s *TokenStoreSuite) TestLoad_handles_null_json() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	err := os.WriteFile(tokensPath, []byte("null"), 0600)
	s.Require().NoError(err)

	store := NewTokenStoreWithPath(tokensPath)
	tokens, err := store.Load()

	s.NoError(err)
	s.NotNil(tokens)
	s.Empty(tokens)
}

func (s *TokenStoreSuite) TestSave_creates_parent_directories() {
	nestedPath := filepath.Join(s.tempDir, "nested", "dir", "tokens.json")
	store := NewTokenStoreWithPath(nestedPath)

	tokens := make(TokensFile)
	tokens["registry.example.com"] = &RegistryTokens{
		ClientId:    "test-client",
		AccessToken: "test-token",
	}

	err := store.Save(tokens)

	s.NoError(err)
	s.FileExists(nestedPath)
}

func (s *TokenStoreSuite) TestSave_writes_valid_json() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	expiry := time.Now().Add(time.Hour)
	tokens := make(TokensFile)
	tokens["registry.example.com"] = &RegistryTokens{
		ClientId:     "test-client",
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		TokenExpiry:  &expiry,
	}

	err := store.Save(tokens)
	s.NoError(err)

	// Verify by loading
	loaded, err := store.Load()
	s.NoError(err)
	s.Require().NotNil(loaded["registry.example.com"])
	s.Equal("test-client", loaded["registry.example.com"].ClientId)
	s.Equal("test-access", loaded["registry.example.com"].AccessToken)
	s.Equal("test-refresh", loaded["registry.example.com"].RefreshToken)
}

func (s *TokenStoreSuite) TestSave_sets_restrictive_permissions() {
	if runtime.GOOS == "windows" {
		s.T().Skip("File permissions work differently on Windows")
	}

	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	tokens := make(TokensFile)
	tokens["registry.example.com"] = &RegistryTokens{
		ClientId:    "test-client",
		AccessToken: "test-token",
	}

	err := store.Save(tokens)
	s.NoError(err)

	info, err := os.Stat(tokensPath)
	s.NoError(err)
	s.Equal(os.FileMode(0600), info.Mode().Perm())
}

func (s *TokenStoreSuite) TestSaveRegistryTokens_adds_new_registry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	tokens := &RegistryTokens{
		ClientId:    "new-client",
		AccessToken: "new-token",
	}
	err := store.SaveRegistryTokens("new-registry.example.com", tokens)

	s.NoError(err)

	loaded, err := store.Load()
	s.NoError(err)
	s.Require().NotNil(loaded["new-registry.example.com"])
	s.Equal("new-client", loaded["new-registry.example.com"].ClientId)
}

func (s *TokenStoreSuite) TestSaveRegistryTokens_updates_existing_registry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	// Save initial tokens
	initialTokens := &RegistryTokens{
		ClientId:    "initial-client",
		AccessToken: "initial-token",
	}
	err := store.SaveRegistryTokens("registry.example.com", initialTokens)
	s.Require().NoError(err)

	// Update with new tokens
	updatedTokens := &RegistryTokens{
		ClientId:    "updated-client",
		AccessToken: "updated-token",
	}
	err = store.SaveRegistryTokens("registry.example.com", updatedTokens)
	s.NoError(err)

	loaded, err := store.Load()
	s.NoError(err)
	s.Equal("updated-client", loaded["registry.example.com"].ClientId)
	s.Equal("updated-token", loaded["registry.example.com"].AccessToken)
}

func (s *TokenStoreSuite) TestSaveRegistryTokens_preserves_other_registries() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	// Save first registry
	tokens1 := &RegistryTokens{ClientId: "client-1", AccessToken: "token-1"}
	err := store.SaveRegistryTokens("registry1.example.com", tokens1)
	s.Require().NoError(err)

	// Save second registry
	tokens2 := &RegistryTokens{ClientId: "client-2", AccessToken: "token-2"}
	err = store.SaveRegistryTokens("registry2.example.com", tokens2)
	s.NoError(err)

	loaded, err := store.Load()
	s.NoError(err)
	s.Len(loaded, 2)
	s.Equal("token-1", loaded["registry1.example.com"].AccessToken)
	s.Equal("token-2", loaded["registry2.example.com"].AccessToken)
}

func (s *TokenStoreSuite) TestGetRegistryTokens_returns_nil_for_nonexistent_registry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	tokens, err := store.GetRegistryTokens("nonexistent.example.com")

	s.NoError(err)
	s.Nil(tokens)
}

func (s *TokenStoreSuite) TestGetRegistryTokens_returns_tokens_for_existing_registry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	savedTokens := &RegistryTokens{ClientId: "test-client", AccessToken: "test-token"}
	err := store.SaveRegistryTokens("registry.example.com", savedTokens)
	s.Require().NoError(err)

	tokens, err := store.GetRegistryTokens("registry.example.com")

	s.NoError(err)
	s.Require().NotNil(tokens)
	s.Equal("test-token", tokens.AccessToken)
}

func (s *TokenStoreSuite) TestRemoveRegistryTokens_removes_existing_registry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	// Save tokens
	tokens := &RegistryTokens{ClientId: "test-client", AccessToken: "test-token"}
	err := store.SaveRegistryTokens("registry.example.com", tokens)
	s.Require().NoError(err)

	// Remove tokens
	err = store.RemoveRegistryTokens("registry.example.com")
	s.NoError(err)

	// Verify removal
	loaded, err := store.Load()
	s.NoError(err)
	s.Nil(loaded["registry.example.com"])
}

func (s *TokenStoreSuite) TestRemoveRegistryTokens_succeeds_for_nonexistent_registry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	err := store.RemoveRegistryTokens("nonexistent.example.com")
	s.NoError(err)
}

func (s *TokenStoreSuite) TestHasRegistryTokens_returns_false_for_nonexistent_registry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	exists, err := store.HasRegistryTokens("nonexistent.example.com")

	s.NoError(err)
	s.False(exists)
}

func (s *TokenStoreSuite) TestHasRegistryTokens_returns_true_for_existing_registry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	tokens := &RegistryTokens{ClientId: "test-client", AccessToken: "test-token"}
	err := store.SaveRegistryTokens("registry.example.com", tokens)
	s.Require().NoError(err)

	exists, err := store.HasRegistryTokens("registry.example.com")

	s.NoError(err)
	s.True(exists)
}

func (s *TokenStoreSuite) TestGetValidTokens_returns_nil_for_nonexistent_registry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	tokens, err := store.GetValidTokens("nonexistent.example.com")

	s.NoError(err)
	s.Nil(tokens)
}

func (s *TokenStoreSuite) TestGetValidTokens_returns_nil_for_expired_tokens() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	// Save expired tokens
	expiredTime := time.Now().Add(-time.Hour)
	tokens := &RegistryTokens{
		ClientId:    "test-client",
		AccessToken: "test-token",
		TokenExpiry: &expiredTime,
	}
	err := store.SaveRegistryTokens("registry.example.com", tokens)
	s.Require().NoError(err)

	validTokens, err := store.GetValidTokens("registry.example.com")

	s.NoError(err)
	s.Nil(validTokens)
}

func (s *TokenStoreSuite) TestGetValidTokens_returns_tokens_when_not_expired() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	// Save valid tokens
	futureTime := time.Now().Add(time.Hour)
	tokens := &RegistryTokens{
		ClientId:    "test-client",
		AccessToken: "test-token",
		TokenExpiry: &futureTime,
	}
	err := store.SaveRegistryTokens("registry.example.com", tokens)
	s.Require().NoError(err)

	validTokens, err := store.GetValidTokens("registry.example.com")

	s.NoError(err)
	s.Require().NotNil(validTokens)
	s.Equal("test-token", validTokens.AccessToken)
}

func (s *TokenStoreSuite) TestGetValidTokens_returns_tokens_when_no_expiry() {
	tokensPath := filepath.Join(s.tempDir, "tokens.json")
	store := NewTokenStoreWithPath(tokensPath)

	// Save tokens without expiry
	tokens := &RegistryTokens{
		ClientId:    "test-client",
		AccessToken: "test-token",
		TokenExpiry: nil,
	}
	err := store.SaveRegistryTokens("registry.example.com", tokens)
	s.Require().NoError(err)

	validTokens, err := store.GetValidTokens("registry.example.com")

	s.NoError(err)
	s.Require().NotNil(validTokens)
	s.Equal("test-token", validTokens.AccessToken)
}

func TestTokenStoreSuite(t *testing.T) {
	suite.Run(t, new(TokenStoreSuite))
}
