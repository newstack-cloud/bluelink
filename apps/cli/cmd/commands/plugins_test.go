package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"
)

type PluginsCommandSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
}

func (s *PluginsCommandSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "plugins-cmd-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Create empty default config file to prevent load errors
	err = os.WriteFile(filepath.Join(tempDir, "bluelink.config.toml"), []byte(""), 0644)
	s.Require().NoError(err)

	s.originalDir, err = os.Getwd()
	s.Require().NoError(err)
	os.Chdir(tempDir)
}

func (s *PluginsCommandSuite) TearDownTest() {
	os.Chdir(s.originalDir)
	os.RemoveAll(s.tempDir)
}

// Command structure tests

func (s *PluginsCommandSuite) Test_plugins_command_exists() {
	rootCmd := NewRootCmd()
	pluginsCmd, _, err := rootCmd.Find([]string{"plugins"})

	s.NoError(err)
	s.NotNil(pluginsCmd)
	s.Equal("plugins", pluginsCmd.Use)
}

func (s *PluginsCommandSuite) Test_plugins_login_command_exists() {
	rootCmd := NewRootCmd()
	loginCmd, _, err := rootCmd.Find([]string{"plugins", "login"})

	s.NoError(err)
	s.NotNil(loginCmd)
	s.Equal("login <registry-host>", loginCmd.Use)
}

// Help text tests

func (s *PluginsCommandSuite) Test_plugins_help_contains_description() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"plugins", "--help"})

	rootCmd.Execute()
	output := buf.String()

	s.Contains(output, "plugins")
	s.Contains(output, "login")
	s.Contains(output, "registry")
}

func (s *PluginsCommandSuite) Test_plugins_login_help_contains_usage() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"plugins", "login", "--help"})

	rootCmd.Execute()
	output := buf.String()

	s.Contains(output, "login")
	s.Contains(output, "registry-host")
	s.Contains(output, "Usage:")
}

// Argument validation tests

func (s *PluginsCommandSuite) Test_plugins_login_requires_registry_host_argument() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"plugins", "login"})

	err := rootCmd.Execute()

	s.Error(err)
	s.Contains(err.Error(), "accepts 1 arg")
}

func (s *PluginsCommandSuite) Test_plugins_login_rejects_extra_arguments() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"plugins", "login", "registry1.example.com", "registry2.example.com"})

	err := rootCmd.Execute()

	s.Error(err)
	s.Contains(err.Error(), "accepts 1 arg")
}

func (s *PluginsCommandSuite) Test_plugins_login_has_no_flags() {
	rootCmd := NewRootCmd()
	loginCmd, _, _ := rootCmd.Find([]string{"plugins", "login"})

	// Should only have the help flag
	localFlags := loginCmd.LocalFlags()
	localFlags.VisitAll(func(flag *pflag.Flag) {
		// This shouldn't iterate over any non-help flags
	})

	// Verify there's no --json flag
	s.Nil(loginCmd.Flag("json"))
	// Verify there's no --api-key flag
	s.Nil(loginCmd.Flag("api-key"))
	// Verify there's no --client-id flag
	s.Nil(loginCmd.Flag("client-id"))
}

// Uninstall command tests

func (s *PluginsCommandSuite) Test_plugins_uninstall_command_exists() {
	rootCmd := NewRootCmd()
	uninstallCmd, _, err := rootCmd.Find([]string{"plugins", "uninstall"})

	s.NoError(err)
	s.NotNil(uninstallCmd)
	s.Equal("uninstall <plugin-id> [plugin-id] ...", uninstallCmd.Use)
}

func (s *PluginsCommandSuite) Test_plugins_uninstall_requires_at_least_one_argument() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"plugins", "uninstall"})

	err := rootCmd.Execute()

	s.Error(err)
	s.Contains(err.Error(), "requires at least 1 arg")
}

func (s *PluginsCommandSuite) Test_plugins_uninstall_help_contains_usage() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"plugins", "uninstall", "--help"})

	rootCmd.Execute()
	output := buf.String()

	s.Contains(output, "uninstall")
	s.Contains(output, "plugin-id")
	s.Contains(output, "Usage:")
}

func (s *PluginsCommandSuite) Test_plugins_uninstall_accepts_multiple_arguments() {
	rootCmd := NewRootCmd()
	uninstallCmd, _, err := rootCmd.Find([]string{"plugins", "uninstall"})

	s.NoError(err)
	s.NotNil(uninstallCmd)

	// Verify the command uses MinimumNArgs(1) by checking it accepts multiple args
	// The Args field should accept more than 1 argument
	err = uninstallCmd.Args(uninstallCmd, []string{"bluelink/aws", "bluelink/gcp"})
	s.NoError(err)
}

func TestPluginsCommandSuite(t *testing.T) {
	suite.Run(t, new(PluginsCommandSuite))
}
