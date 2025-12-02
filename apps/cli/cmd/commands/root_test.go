package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RootCommandSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
}

func (s *RootCommandSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "root-cmd-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Create empty default config file to prevent load errors
	err = os.WriteFile(filepath.Join(tempDir, "bluelink.config.toml"), []byte(""), 0644)
	s.Require().NoError(err)

	s.originalDir, err = os.Getwd()
	s.Require().NoError(err)
	os.Chdir(tempDir)
}

func (s *RootCommandSuite) TearDownTest() {
	os.Chdir(s.originalDir)
	os.RemoveAll(s.tempDir)
}

func (s *RootCommandSuite) writeConfigFile(name, content string) string {
	path := filepath.Join(s.tempDir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	s.Require().NoError(err)
	return path
}

// Config loading tests

func (s *RootCommandSuite) Test_loads_config_from_yaml() {
	s.writeConfigFile("config.yaml", `
connectProtocol: "tcp"
engineEndpoint: "http://custom:9000"
`)
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"--config=config.yaml", "version"})

	err := rootCmd.Execute()
	s.NoError(err)
}

func (s *RootCommandSuite) Test_loads_config_from_json() {
	s.writeConfigFile("config.json", `{
  "connectProtocol": "tcp",
  "engineEndpoint": "http://custom:9000"
}`)
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"--config=config.json", "version"})

	err := rootCmd.Execute()
	s.NoError(err)
}

func (s *RootCommandSuite) Test_loads_config_from_toml() {
	s.writeConfigFile("config.toml", `
connectProtocol = "tcp"
engineEndpoint = "http://custom:9000"
`)
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"--config=config.toml", "version"})

	err := rootCmd.Execute()
	s.NoError(err)
}

func (s *RootCommandSuite) Test_fails_with_missing_config_file() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--config=/nonexistent/config.toml", "version"})

	err := rootCmd.Execute()
	s.Error(err)
}

// Protocol validation tests

func (s *RootCommandSuite) Test_validates_connect_protocol_tcp() {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"version", "--connect-protocol=tcp"})

	err := rootCmd.Execute()
	s.NoError(err)
}

func (s *RootCommandSuite) Test_validates_connect_protocol_unix() {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"version", "--connect-protocol=unix"})

	err := rootCmd.Execute()
	s.NoError(err)
}

func (s *RootCommandSuite) Test_rejects_invalid_connect_protocol() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version", "--connect-protocol=invalid"})

	err := rootCmd.Execute()
	s.Error(err)
	s.Contains(err.Error(), "invalid connect protocol")
}

func (s *RootCommandSuite) Test_rejects_invalid_protocol_from_config() {
	s.writeConfigFile("bluelink.config.toml", `connectProtocol = "invalid"`)

	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	s.Error(err)
	s.Contains(err.Error(), "invalid connect protocol")
}

func (s *RootCommandSuite) Test_flag_overrides_config_file() {
	s.writeConfigFile("bluelink.config.toml", `connectProtocol = "invalid"`)

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"version", "--connect-protocol=tcp"})

	err := rootCmd.Execute()
	s.NoError(err)
}

// Subcommand structure tests

func (s *RootCommandSuite) Test_has_version_subcommand() {
	rootCmd := NewRootCmd()
	versionCmd, _, err := rootCmd.Find([]string{"version"})
	s.NoError(err)
	s.Equal("version", versionCmd.Name())
}

func (s *RootCommandSuite) Test_has_init_subcommand() {
	rootCmd := NewRootCmd()
	initCmd, _, err := rootCmd.Find([]string{"init"})
	s.NoError(err)
	s.Equal("init", initCmd.Name())
}

func (s *RootCommandSuite) Test_has_validate_subcommand() {
	rootCmd := NewRootCmd()
	validateCmd, _, err := rootCmd.Find([]string{"validate"})
	s.NoError(err)
	s.Equal("validate", validateCmd.Name())
}

// Persistent flag tests

func (s *RootCommandSuite) Test_has_config_flag() {
	rootCmd := NewRootCmd()
	flag := rootCmd.Flag("config")
	s.NotNil(flag)
	s.Equal("bluelink.config.toml", flag.DefValue)
}

func (s *RootCommandSuite) Test_has_deploy_config_file_flag() {
	rootCmd := NewRootCmd()
	flag := rootCmd.PersistentFlags().Lookup("deploy-config-file")
	s.NotNil(flag)
	s.Equal("bluelink.deploy.json", flag.DefValue)
}

func (s *RootCommandSuite) Test_has_connect_protocol_flag() {
	rootCmd := NewRootCmd()
	flag := rootCmd.PersistentFlags().Lookup("connect-protocol")
	s.NotNil(flag)
	s.Equal("unix", flag.DefValue)
}

func (s *RootCommandSuite) Test_has_engine_endpoint_flag() {
	rootCmd := NewRootCmd()
	flag := rootCmd.PersistentFlags().Lookup("engine-endpoint")
	s.NotNil(flag)
	s.Equal("http://localhost:8325", flag.DefValue)
}

func (s *RootCommandSuite) Test_has_engine_auth_config_file_flag() {
	rootCmd := NewRootCmd()
	flag := rootCmd.PersistentFlags().Lookup("engine-auth-config-file")
	s.NotNil(flag)
	s.Equal("engine.auth.json", flag.DefValue)
}

func (s *RootCommandSuite) Test_has_skip_plugin_config_validation_flag() {
	rootCmd := NewRootCmd()
	flag := rootCmd.PersistentFlags().Lookup("skip-plugin-config-validation")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

// Help text tests

func (s *RootCommandSuite) Test_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "bluelink")
	s.Contains(output, "blueprint")
}

func TestRootCommandSuite(t *testing.T) {
	suite.Run(t, new(RootCommandSuite))
}
