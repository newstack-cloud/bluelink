package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"
)

type ProviderSuite struct {
	suite.Suite
	tempDir string
}

func (s *ProviderSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *ProviderSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *ProviderSuite) writeConfigFile(name, content string) string {
	path := filepath.Join(s.tempDir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	s.Require().NoError(err)
	return path
}

// Config file loading tests

func (s *ProviderSuite) Test_load_yaml_config_file() {
	path := s.writeConfigFile("config.yaml", `
apiKey: "yaml-api-key"
timeout: "30"
`)
	p := NewProvider()
	err := p.LoadConfigFile(path)
	s.NoError(err)

	val, isDefault := p.GetString("apiKey")
	s.Equal("yaml-api-key", val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_load_yml_config_file() {
	path := s.writeConfigFile("config.yml", `
apiKey: "yml-api-key"
`)
	p := NewProvider()
	err := p.LoadConfigFile(path)
	s.NoError(err)

	val, isDefault := p.GetString("apiKey")
	s.Equal("yml-api-key", val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_load_json_config_file() {
	path := s.writeConfigFile("config.json", `{
  "apiKey": "json-api-key",
  "timeout": "60"
}`)
	p := NewProvider()
	err := p.LoadConfigFile(path)
	s.NoError(err)

	val, isDefault := p.GetString("apiKey")
	s.Equal("json-api-key", val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_load_toml_config_file() {
	path := s.writeConfigFile("config.toml", `
apiKey = "toml-api-key"
timeout = "90"
`)
	p := NewProvider()
	err := p.LoadConfigFile(path)
	s.NoError(err)

	val, isDefault := p.GetString("apiKey")
	s.Equal("toml-api-key", val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_load_unsupported_format_returns_error() {
	path := s.writeConfigFile("config.xml", `<config></config>`)
	p := NewProvider()
	err := p.LoadConfigFile(path)
	s.ErrorIs(err, ErrUnsupportedConfigFileFormat)
}

func (s *ProviderSuite) Test_load_missing_file_returns_error() {
	p := NewProvider()
	err := p.LoadConfigFile("/nonexistent/path/config.yaml")
	s.Error(err)
}

func (s *ProviderSuite) Test_load_invalid_yaml_returns_error() {
	path := s.writeConfigFile("config.yaml", `
invalid: yaml: content:
  - broken
`)
	p := NewProvider()
	err := p.LoadConfigFile(path)
	s.Error(err)
}

func (s *ProviderSuite) Test_load_invalid_json_returns_error() {
	path := s.writeConfigFile("config.json", `{invalid json}`)
	p := NewProvider()
	err := p.LoadConfigFile(path)
	s.Error(err)
}

func (s *ProviderSuite) Test_load_invalid_toml_returns_error() {
	path := s.writeConfigFile("config.toml", `invalid = toml [`)
	p := NewProvider()
	err := p.LoadConfigFile(path)
	s.Error(err)
}

// Precedence tests

func (s *ProviderSuite) Test_flag_takes_precedence_over_env_var() {
	p := NewProvider()

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("api-key", "default", "API key")
	flagSet.Parse([]string{"--api-key=flag-value"})
	flag := flagSet.Lookup("api-key")
	p.BindPFlag("apiKey", flag)

	p.BindEnvVar("apiKey", "TEST_API_KEY")
	os.Setenv("TEST_API_KEY", "env-value")
	defer os.Unsetenv("TEST_API_KEY")

	val, isDefault := p.GetString("apiKey")
	s.Equal("flag-value", val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_env_var_takes_precedence_over_config_file() {
	path := s.writeConfigFile("config.yaml", `apiKey: "config-value"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	p.BindEnvVar("apiKey", "TEST_API_KEY")
	os.Setenv("TEST_API_KEY", "env-value")
	defer os.Unsetenv("TEST_API_KEY")

	val, isDefault := p.GetString("apiKey")
	s.Equal("env-value", val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_config_file_takes_precedence_over_flag_default() {
	path := s.writeConfigFile("config.yaml", `apiKey: "config-value"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("api-key", "flag-default", "API key")
	flagSet.Parse([]string{}) // No flag provided
	p.BindPFlag("apiKey", flagSet.Lookup("api-key"))

	val, isDefault := p.GetString("apiKey")
	s.Equal("config-value", val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_flag_default_takes_precedence_over_provider_default() {
	p := NewProvider()
	p.SetDefault("apiKey", "provider-default")

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("api-key", "flag-default", "API key")
	flagSet.Parse([]string{}) // No flag provided
	p.BindPFlag("apiKey", flagSet.Lookup("api-key"))

	val, isDefault := p.GetString("apiKey")
	s.Equal("flag-default", val)
	s.True(isDefault)
}

func (s *ProviderSuite) Test_provider_default_used_when_nothing_else_set() {
	p := NewProvider()
	p.SetDefault("apiKey", "provider-default")

	val, isDefault := p.GetString("apiKey")
	s.Equal("provider-default", val)
	s.True(isDefault)
}

func (s *ProviderSuite) Test_empty_string_returned_when_no_value_set() {
	p := NewProvider()

	val, isDefault := p.GetString("nonexistent")
	s.Equal("", val)
	s.True(isDefault)
}

// isDefault flag tests

func (s *ProviderSuite) Test_is_default_false_when_flag_explicitly_set() {
	p := NewProvider()

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("api-key", "default", "API key")
	flagSet.Parse([]string{"--api-key=explicit"})
	p.BindPFlag("apiKey", flagSet.Lookup("api-key"))

	_, isDefault := p.GetString("apiKey")
	s.False(isDefault)
}

func (s *ProviderSuite) Test_is_default_false_when_env_var_set() {
	p := NewProvider()
	p.BindEnvVar("apiKey", "TEST_API_KEY")
	os.Setenv("TEST_API_KEY", "env-value")
	defer os.Unsetenv("TEST_API_KEY")

	_, isDefault := p.GetString("apiKey")
	s.False(isDefault)
}

func (s *ProviderSuite) Test_is_default_false_when_config_file_value_set() {
	path := s.writeConfigFile("config.yaml", `apiKey: "config-value"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	_, isDefault := p.GetString("apiKey")
	s.False(isDefault)
}

func (s *ProviderSuite) Test_is_default_true_when_using_flag_default() {
	p := NewProvider()

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("api-key", "flag-default", "API key")
	flagSet.Parse([]string{}) // No flag provided
	p.BindPFlag("apiKey", flagSet.Lookup("api-key"))

	_, isDefault := p.GetString("apiKey")
	s.True(isDefault)
}

func (s *ProviderSuite) Test_is_default_true_when_using_provider_default() {
	p := NewProvider()
	p.SetDefault("apiKey", "provider-default")

	_, isDefault := p.GetString("apiKey")
	s.True(isDefault)
}

// Type conversion tests - GetInt32

func (s *ProviderSuite) Test_get_int32_parses_valid_number() {
	path := s.writeConfigFile("config.yaml", `timeout: "42"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, isDefault := p.GetInt32("timeout")
	s.Equal(int32(42), val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_get_int32_returns_zero_for_invalid_number() {
	path := s.writeConfigFile("config.yaml", `timeout: "not-a-number"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, isDefault := p.GetInt32("timeout")
	s.Equal(int32(0), val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_get_int32_returns_zero_for_empty_value() {
	p := NewProvider()

	val, isDefault := p.GetInt32("timeout")
	s.Equal(int32(0), val)
	s.True(isDefault)
}

// Type conversion tests - GetInt64

func (s *ProviderSuite) Test_get_int64_parses_valid_number() {
	path := s.writeConfigFile("config.yaml", `bigNumber: "9223372036854775807"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, isDefault := p.GetInt64("bigNumber")
	s.Equal(int64(9223372036854775807), val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_get_int64_returns_zero_for_invalid_number() {
	path := s.writeConfigFile("config.yaml", `bigNumber: "invalid"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, _ := p.GetInt64("bigNumber")
	s.Equal(int64(0), val)
}

// Type conversion tests - GetUint32

func (s *ProviderSuite) Test_get_uint32_parses_valid_number() {
	path := s.writeConfigFile("config.yaml", `port: "8080"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, isDefault := p.GetUint32("port")
	s.Equal(uint32(8080), val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_get_uint32_returns_zero_for_negative_number() {
	path := s.writeConfigFile("config.yaml", `port: "-1"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, _ := p.GetUint32("port")
	s.Equal(uint32(0), val)
}

// Type conversion tests - GetUint64

func (s *ProviderSuite) Test_get_uint64_parses_valid_number() {
	path := s.writeConfigFile("config.yaml", `bigUint: "18446744073709551615"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, isDefault := p.GetUint64("bigUint")
	s.Equal(uint64(18446744073709551615), val)
	s.False(isDefault)
}

// Type conversion tests - GetFloat32

func (s *ProviderSuite) Test_get_float32_parses_valid_number() {
	path := s.writeConfigFile("config.yaml", `ratio: "3.14"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, isDefault := p.GetFloat32("ratio")
	s.InDelta(float32(3.14), val, 0.001)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_get_float32_returns_zero_for_invalid_number() {
	path := s.writeConfigFile("config.yaml", `ratio: "not-a-float"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, _ := p.GetFloat32("ratio")
	s.Equal(float32(0.0), val)
}

func (s *ProviderSuite) Test_get_float32_returns_zero_for_empty_value() {
	p := NewProvider()

	val, isDefault := p.GetFloat32("ratio")
	s.Equal(float32(0.0), val)
	s.True(isDefault)
}

// Type conversion tests - GetFloat64

func (s *ProviderSuite) Test_get_float64_parses_valid_number() {
	path := s.writeConfigFile("config.yaml", `preciseRatio: "3.141592653589793"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, isDefault := p.GetFloat64("preciseRatio")
	s.InDelta(3.141592653589793, val, 0.0000000001)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_get_float64_returns_zero_for_invalid_number() {
	path := s.writeConfigFile("config.yaml", `preciseRatio: "invalid"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, _ := p.GetFloat64("preciseRatio")
	s.Equal(float64(0.0), val)
}

func (s *ProviderSuite) Test_get_float64_returns_zero_for_empty_value() {
	p := NewProvider()

	val, isDefault := p.GetFloat64("preciseRatio")
	s.Equal(float64(0.0), val)
	s.True(isDefault)
}

// Type conversion tests - GetBool

func (s *ProviderSuite) Test_get_bool_parses_true() {
	path := s.writeConfigFile("config.yaml", `enabled: "true"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, isDefault := p.GetBool("enabled")
	s.True(val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_get_bool_parses_false() {
	path := s.writeConfigFile("config.yaml", `enabled: "false"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, isDefault := p.GetBool("enabled")
	s.False(val)
	s.False(isDefault)
}

func (s *ProviderSuite) Test_get_bool_parses_numeric_one_as_true() {
	path := s.writeConfigFile("config.yaml", `enabled: "1"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, _ := p.GetBool("enabled")
	s.True(val)
}

func (s *ProviderSuite) Test_get_bool_parses_numeric_zero_as_false() {
	path := s.writeConfigFile("config.yaml", `enabled: "0"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, _ := p.GetBool("enabled")
	s.False(val)
}

func (s *ProviderSuite) Test_get_bool_returns_false_for_invalid_value() {
	path := s.writeConfigFile("config.yaml", `enabled: "yes"`)
	p := NewProvider()
	p.LoadConfigFile(path)

	val, _ := p.GetBool("enabled")
	s.False(val)
}

func (s *ProviderSuite) Test_get_bool_returns_false_for_empty_value() {
	p := NewProvider()

	val, isDefault := p.GetBool("enabled")
	s.False(val)
	s.True(isDefault)
}

// Edge cases

func (s *ProviderSuite) Test_whitespace_only_env_var_is_ignored() {
	p := NewProvider()
	p.BindEnvVar("apiKey", "TEST_API_KEY")
	p.SetDefault("apiKey", "default-value")
	os.Setenv("TEST_API_KEY", "   ")
	defer os.Unsetenv("TEST_API_KEY")

	val, isDefault := p.GetString("apiKey")
	s.Equal("default-value", val)
	s.True(isDefault)
}

func (s *ProviderSuite) Test_whitespace_only_flag_value_is_ignored() {
	p := NewProvider()
	p.SetDefault("apiKey", "default-value")

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("api-key", "   ", "API key")
	flagSet.Parse([]string{})
	p.BindPFlag("apiKey", flagSet.Lookup("api-key"))

	val, isDefault := p.GetString("apiKey")
	s.Equal("default-value", val)
	s.True(isDefault)
}

func (s *ProviderSuite) Test_nil_flag_binding_is_handled() {
	p := NewProvider()
	p.BindPFlag("apiKey", nil)
	p.SetDefault("apiKey", "default-value")

	val, isDefault := p.GetString("apiKey")
	s.Equal("default-value", val)
	s.True(isDefault)
}

func TestProviderSuite(t *testing.T) {
	suite.Run(t, new(ProviderSuite))
}
