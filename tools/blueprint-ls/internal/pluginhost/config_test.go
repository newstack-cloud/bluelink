package pluginhost

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
	originalEnv map[string]string
}

func (s *ConfigSuite) SetupTest() {
	s.originalEnv = map[string]string{
		envPluginPath:             os.Getenv(envPluginPath),
		envPluginsEnabled:         os.Getenv(envPluginsEnabled),
		envLaunchWaitTimeoutMS:    os.Getenv(envLaunchWaitTimeoutMS),
		envTotalLaunchWaitTimeout: os.Getenv(envTotalLaunchWaitTimeout),
	}

	os.Unsetenv(envPluginPath)
	os.Unsetenv(envPluginsEnabled)
	os.Unsetenv(envLaunchWaitTimeoutMS)
	os.Unsetenv(envTotalLaunchWaitTimeout)
}

func (s *ConfigSuite) TearDownTest() {
	for key, val := range s.originalEnv {
		if val == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, val)
		}
	}
}

func (s *ConfigSuite) TestDefaultConfig_defaults() {
	config := NewDefaultConfig()

	s.Assert().Equal("", config.GetPluginPath())
	s.Assert().False(config.IsEnabled())
	s.Assert().Equal(defaultLaunchWaitTimeoutMS, config.GetLaunchWaitTimeoutMS())
	s.Assert().Equal(defaultTotalLaunchWaitTimeoutMS, config.GetTotalLaunchWaitTimeoutMS())
}

func (s *ConfigSuite) TestDefaultConfig_reads_env_vars() {
	os.Setenv(envPluginPath, "/path/to/plugins")
	os.Setenv(envPluginsEnabled, "true")
	os.Setenv(envLaunchWaitTimeoutMS, "10000")
	os.Setenv(envTotalLaunchWaitTimeout, "120000")

	config := NewDefaultConfig()

	s.Assert().Equal("/path/to/plugins", config.GetPluginPath())
	s.Assert().True(config.IsEnabled())
	s.Assert().Equal(10000, config.GetLaunchWaitTimeoutMS())
	s.Assert().Equal(120000, config.GetTotalLaunchWaitTimeoutMS())
}

func (s *ConfigSuite) TestDefaultConfig_enabled_requires_exact_true() {
	os.Setenv(envPluginsEnabled, "TRUE")
	config := NewDefaultConfig()
	s.Assert().False(config.IsEnabled())

	os.Setenv(envPluginsEnabled, "1")
	config = NewDefaultConfig()
	s.Assert().False(config.IsEnabled())

	os.Setenv(envPluginsEnabled, "true")
	config = NewDefaultConfig()
	s.Assert().True(config.IsEnabled())
}

func (s *ConfigSuite) TestDefaultConfig_invalid_int_uses_default() {
	os.Setenv(envLaunchWaitTimeoutMS, "invalid")
	os.Setenv(envTotalLaunchWaitTimeout, "not-a-number")

	config := NewDefaultConfig()

	s.Assert().Equal(defaultLaunchWaitTimeoutMS, config.GetLaunchWaitTimeoutMS())
	s.Assert().Equal(defaultTotalLaunchWaitTimeoutMS, config.GetTotalLaunchWaitTimeoutMS())
}

func (s *ConfigSuite) TestWithInitOptions_nil_options() {
	os.Setenv(envPluginPath, "/env/path")
	os.Setenv(envPluginsEnabled, "true")

	config := NewDefaultConfig().WithInitOptions(nil)

	s.Assert().Equal("/env/path", config.GetPluginPath())
	s.Assert().True(config.IsEnabled())
}

func (s *ConfigSuite) TestWithInitOptions_nil_plugins() {
	os.Setenv(envPluginPath, "/env/path")
	os.Setenv(envPluginsEnabled, "true")

	config := NewDefaultConfig().WithInitOptions(&InitializationOptions{
		Plugins: nil,
	})

	s.Assert().Equal("/env/path", config.GetPluginPath())
	s.Assert().True(config.IsEnabled())
}

func (s *ConfigSuite) TestWithInitOptions_overrides_env() {
	os.Setenv(envPluginPath, "/env/path")
	os.Setenv(envPluginsEnabled, "true")

	enabled := false
	pluginPath := "/client/path"
	config := NewDefaultConfig().WithInitOptions(&InitializationOptions{
		Plugins: &PluginInitOptions{
			Enabled:    &enabled,
			PluginPath: &pluginPath,
		},
	})

	s.Assert().Equal("/client/path", config.GetPluginPath())
	s.Assert().False(config.IsEnabled())
}

func (s *ConfigSuite) TestWithInitOptions_empty_path_falls_back_to_env() {
	os.Setenv(envPluginPath, "/env/path")

	emptyPath := ""
	config := NewDefaultConfig().WithInitOptions(&InitializationOptions{
		Plugins: &PluginInitOptions{
			PluginPath: &emptyPath,
		},
	})

	s.Assert().Equal("/env/path", config.GetPluginPath())
}

func (s *ConfigSuite) TestWithInitOptions_partial_override() {
	os.Setenv(envPluginPath, "/env/path")
	os.Setenv(envPluginsEnabled, "false")

	enabled := true
	config := NewDefaultConfig().WithInitOptions(&InitializationOptions{
		Plugins: &PluginInitOptions{
			Enabled: &enabled,
			// PluginPath not set, should use env
		},
	})

	s.Assert().Equal("/env/path", config.GetPluginPath())
	s.Assert().True(config.IsEnabled())
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
