package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type DeployConfigSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
}

func TestDeployConfigSuite(t *testing.T) {
	suite.Run(t, new(DeployConfigSuite))
}

func (s *DeployConfigSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "deploy-config-test-*")
	s.Require().NoError(err)

	// Resolve symlinks for consistent path comparison (macOS /var -> /private/var)
	s.tempDir, err = filepath.EvalSymlinks(tempDir)
	s.Require().NoError(err)

	s.originalDir, err = os.Getwd()
	s.Require().NoError(err)
}

func (s *DeployConfigSuite) TearDownTest() {
	os.Chdir(s.originalDir)
	os.RemoveAll(s.tempDir)
}

func (s *DeployConfigSuite) TestLoadDeployConfig_success() {
	configContent := `{
  "dependencies": {
    "bluelink/aws": "1.0.0",
    "registry.example.com/my-org/custom": "2.0.0"
  }
}`
	configPath := filepath.Join(s.tempDir, "bluelink.deploy.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	s.Require().NoError(err)

	config, err := LoadDeployConfig(configPath)
	s.NoError(err)
	s.Require().NotNil(config)
	s.Len(config.Dependencies, 2)
	s.Equal("1.0.0", config.Dependencies["bluelink/aws"])
	s.Equal("2.0.0", config.Dependencies["registry.example.com/my-org/custom"])
}

func (s *DeployConfigSuite) TestLoadDeployConfig_empty_dependencies() {
	configContent := `{}`
	configPath := filepath.Join(s.tempDir, "bluelink.deploy.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	s.Require().NoError(err)

	config, err := LoadDeployConfig(configPath)
	s.NoError(err)
	s.Require().NotNil(config)
	s.Len(config.Dependencies, 0)
}

func (s *DeployConfigSuite) TestLoadDeployConfig_with_jsonc_comments() {
	configContent := `{
  // This is a line comment
  "dependencies": {
    "bluelink/aws": "1.0.0", // inline comment
    /* block comment */
    "bluelink/gcp": "2.0.0"
  }
}`
	configPath := filepath.Join(s.tempDir, "bluelink.deploy.jsonc")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	s.Require().NoError(err)

	config, err := LoadDeployConfig(configPath)
	s.NoError(err)
	s.Require().NotNil(config)
	s.Len(config.Dependencies, 2)
	s.Equal("1.0.0", config.Dependencies["bluelink/aws"])
	s.Equal("2.0.0", config.Dependencies["bluelink/gcp"])
}

func (s *DeployConfigSuite) TestLoadDeployConfig_file_not_found() {
	_, err := LoadDeployConfig(filepath.Join(s.tempDir, "nonexistent.json"))
	s.Error(err)
}

func (s *DeployConfigSuite) TestLoadDeployConfig_invalid_json() {
	configContent := `{invalid json}`
	configPath := filepath.Join(s.tempDir, "bluelink.deploy.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	s.Require().NoError(err)

	_, err = LoadDeployConfig(configPath)
	s.Error(err)
	s.Contains(err.Error(), "failed to parse")
}

func (s *DeployConfigSuite) TestGetPluginIDs_success() {
	config := &DeployConfig{
		Dependencies: map[string]string{
			"bluelink/aws":                          "1.0.0",
			"bluelink/gcp":                          "2.0.0",
			"registry.example.com/my-org/custom":    "3.0.0",
			"localhost:8080/bluelink/test-provider": "4.0.0",
		},
	}

	ids, err := config.GetPluginIDs()
	s.NoError(err)
	s.Len(ids, 4)

	// Find each plugin by name
	idMap := make(map[string]*PluginID)
	for _, id := range ids {
		idMap[id.Name] = id
	}

	// Default registry plugins
	awsID := idMap["aws"]
	s.Require().NotNil(awsID)
	s.Equal(DefaultRegistryHost, awsID.RegistryHost)
	s.Equal("bluelink", awsID.Namespace)
	s.Equal("1.0.0", awsID.Version)

	gcpID := idMap["gcp"]
	s.Require().NotNil(gcpID)
	s.Equal(DefaultRegistryHost, gcpID.RegistryHost)
	s.Equal("bluelink", gcpID.Namespace)
	s.Equal("2.0.0", gcpID.Version)

	// Custom registry plugin
	customID := idMap["custom"]
	s.Require().NotNil(customID)
	s.Equal("registry.example.com", customID.RegistryHost)
	s.Equal("my-org", customID.Namespace)
	s.Equal("3.0.0", customID.Version)

	// Localhost with port
	testID := idMap["test-provider"]
	s.Require().NotNil(testID)
	s.Equal("localhost:8080", testID.RegistryHost)
	s.Equal("bluelink", testID.Namespace)
	s.Equal("4.0.0", testID.Version)
}

func (s *DeployConfigSuite) TestGetPluginIDs_empty() {
	config := &DeployConfig{
		Dependencies: map[string]string{},
	}

	ids, err := config.GetPluginIDs()
	s.NoError(err)
	s.Len(ids, 0)
}

func (s *DeployConfigSuite) TestGetPluginIDs_invalid_plugin_id() {
	config := &DeployConfig{
		Dependencies: map[string]string{
			"invalid-plugin-id": "1.0.0",
		},
	}

	_, err := config.GetPluginIDs()
	s.Error(err)
	s.Contains(err.Error(), "invalid plugin dependency")
}

func (s *DeployConfigSuite) TestLoadDeployConfig_with_trailing_commas() {
	// hujson supports trailing commas
	configContent := `{
  "dependencies": {
    "bluelink/aws": "1.0.0",
    "bluelink/gcp": "2.0.0",
  },
}`
	configPath := filepath.Join(s.tempDir, "bluelink.deploy.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	s.Require().NoError(err)

	config, err := LoadDeployConfig(configPath)
	s.NoError(err)
	s.Require().NotNil(config)
	s.Len(config.Dependencies, 2)
}
