package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type AuthConfigSuite struct {
	suite.Suite
	tempDir            string
	originalInstallDir string
}

func (s *AuthConfigSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "bluelink-auth-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	s.originalInstallDir = os.Getenv("BLUELINK_INSTALL_DIR")
	os.Setenv("BLUELINK_INSTALL_DIR", tempDir)

	// Create required directories
	os.MkdirAll(filepath.Join(tempDir, "config"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "engine"), 0755)
}

func (s *AuthConfigSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
	if s.originalInstallDir != "" {
		os.Setenv("BLUELINK_INSTALL_DIR", s.originalInstallDir)
	} else {
		os.Unsetenv("BLUELINK_INSTALL_DIR")
	}
}

func (s *AuthConfigSuite) Test_ConfigureAuth_creates_cli_auth_file() {
	err := ConfigureAuth(false)
	s.NoError(err)

	cliAuthPath := filepath.Join(s.tempDir, "config", "engine.auth.json")
	s.FileExists(cliAuthPath)

	// Verify content
	data, err := os.ReadFile(cliAuthPath)
	s.NoError(err)

	var config CLIAuthConfig
	err = json.Unmarshal(data, &config)
	s.NoError(err)
	s.Equal("apiKey", config.Method)
	s.NotEmpty(config.APIKey)
	s.Len(config.APIKey, 64) // 32 bytes = 64 hex chars
}

func (s *AuthConfigSuite) Test_ConfigureAuth_creates_engine_config_file() {
	err := ConfigureAuth(false)
	s.NoError(err)

	engineConfigPath := filepath.Join(s.tempDir, "engine", "config.json")
	s.FileExists(engineConfigPath)

	// Verify content
	data, err := os.ReadFile(engineConfigPath)
	s.NoError(err)

	var config EngineConfig
	err = json.Unmarshal(data, &config)
	s.NoError(err)
	s.True(config.LoopbackOnly)
	s.Len(config.Auth.BluelinkAPIKeys, 1)
	s.Len(config.Auth.BluelinkAPIKeys[0], 64)
}

func (s *AuthConfigSuite) Test_ConfigureAuth_uses_same_api_key_for_both_configs() {
	err := ConfigureAuth(false)
	s.NoError(err)

	// Read CLI config
	cliData, _ := os.ReadFile(filepath.Join(s.tempDir, "config", "engine.auth.json"))
	var cliConfig CLIAuthConfig
	json.Unmarshal(cliData, &cliConfig)

	// Read engine config
	engineData, _ := os.ReadFile(filepath.Join(s.tempDir, "engine", "config.json"))
	var engineConfig EngineConfig
	json.Unmarshal(engineData, &engineConfig)

	s.Equal(cliConfig.APIKey, engineConfig.Auth.BluelinkAPIKeys[0])
}

func (s *AuthConfigSuite) Test_ConfigureAuth_skips_if_exists_without_force() {
	// Create initial config
	err := ConfigureAuth(false)
	s.NoError(err)

	// Read original API key
	cliData, _ := os.ReadFile(filepath.Join(s.tempDir, "config", "engine.auth.json"))
	var originalConfig CLIAuthConfig
	json.Unmarshal(cliData, &originalConfig)

	// Run again without force
	err = ConfigureAuth(false)
	s.NoError(err)

	// Verify key unchanged
	cliData, _ = os.ReadFile(filepath.Join(s.tempDir, "config", "engine.auth.json"))
	var newConfig CLIAuthConfig
	json.Unmarshal(cliData, &newConfig)

	s.Equal(originalConfig.APIKey, newConfig.APIKey)
}

func (s *AuthConfigSuite) Test_ConfigureAuth_regenerates_with_force() {
	// Create initial config
	err := ConfigureAuth(false)
	s.NoError(err)

	// Read original API key
	cliData, _ := os.ReadFile(filepath.Join(s.tempDir, "config", "engine.auth.json"))
	var originalConfig CLIAuthConfig
	json.Unmarshal(cliData, &originalConfig)

	// Run again with force
	err = ConfigureAuth(true)
	s.NoError(err)

	// Verify key changed
	cliData, _ = os.ReadFile(filepath.Join(s.tempDir, "config", "engine.auth.json"))
	var newConfig CLIAuthConfig
	json.Unmarshal(cliData, &newConfig)

	s.NotEqual(originalConfig.APIKey, newConfig.APIKey)
}

func TestAuthConfigSuite(t *testing.T) {
	suite.Run(t, new(AuthConfigSuite))
}
