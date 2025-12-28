package tagging

import (
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/stretchr/testify/suite"
)

type ConfigProviderTestSuite struct {
	suite.Suite
	provider ConfigProvider
}

func (s *ConfigProviderTestSuite) SetupTest() {
	s.provider = NewConfigProvider("1.0.0")
}

func (s *ConfigProviderTestSuite) Test_CreateConfig_with_nil_request_uses_defaults() {
	config := s.provider.CreateConfig(nil)

	s.True(config.Enabled, "tagging should be enabled by default")
	s.Equal(DefaultTagPrefix, config.Prefix, "prefix should be default")
	s.Equal("1.0.0", config.DeployEngineVersion, "deploy engine version should be set")
	s.Empty(config.ProviderPluginID, "provider plugin ID should be empty (set per-resource)")
	s.Empty(config.ProviderPluginVersion, "provider plugin version should be empty (set per-resource)")
}

func (s *ConfigProviderTestSuite) Test_CreateConfig_with_empty_request_uses_defaults() {
	config := s.provider.CreateConfig(&types.TaggingOperationConfig{})

	s.True(config.Enabled, "tagging should be enabled by default")
	s.Equal(DefaultTagPrefix, config.Prefix, "prefix should be default")
	s.Equal("1.0.0", config.DeployEngineVersion, "deploy engine version should be set")
}

func (s *ConfigProviderTestSuite) Test_CreateConfig_with_enabled_false_overrides_default() {
	enabled := false
	config := s.provider.CreateConfig(&types.TaggingOperationConfig{
		Enabled: &enabled,
	})

	s.False(config.Enabled, "tagging should be disabled when explicitly set")
	s.Equal(DefaultTagPrefix, config.Prefix, "prefix should be default")
	s.Equal("1.0.0", config.DeployEngineVersion, "deploy engine version should be set")
}

func (s *ConfigProviderTestSuite) Test_CreateConfig_with_enabled_true_explicit() {
	enabled := true
	config := s.provider.CreateConfig(&types.TaggingOperationConfig{
		Enabled: &enabled,
	})

	s.True(config.Enabled, "tagging should be enabled when explicitly set")
}

func (s *ConfigProviderTestSuite) Test_CreateConfig_with_custom_prefix_overrides_default() {
	config := s.provider.CreateConfig(&types.TaggingOperationConfig{
		Prefix: "myorg:bluelink:",
	})

	s.True(config.Enabled, "tagging should be enabled by default")
	s.Equal("myorg:bluelink:", config.Prefix, "prefix should be custom value")
	s.Equal("1.0.0", config.DeployEngineVersion, "deploy engine version should be set")
}

func (s *ConfigProviderTestSuite) Test_CreateConfig_with_full_override() {
	enabled := false
	config := s.provider.CreateConfig(&types.TaggingOperationConfig{
		Enabled: &enabled,
		Prefix:  "custom:",
	})

	s.False(config.Enabled, "tagging should be disabled")
	s.Equal("custom:", config.Prefix, "prefix should be custom value")
	s.Equal("1.0.0", config.DeployEngineVersion, "deploy engine version should be set")
}

func (s *ConfigProviderTestSuite) Test_CreateConfig_deploy_engine_version_always_set() {
	provider := NewConfigProvider("2.5.3")
	config := provider.CreateConfig(nil)

	s.Equal("2.5.3", config.DeployEngineVersion, "deploy engine version should match provider version")
}

func TestConfigProviderTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigProviderTestSuite))
}
