package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type InstallCommandSuite struct {
	suite.Suite
}

func (s *InstallCommandSuite) Test_has_cli_version_flag() {
	rootCmd := NewRootCmd()
	installCmd, _, _ := rootCmd.Find([]string{"install"})

	flag := installCmd.Flag("cli-version")
	s.NotNil(flag)
	s.Equal("", flag.DefValue)
}

func (s *InstallCommandSuite) Test_has_engine_version_flag() {
	rootCmd := NewRootCmd()
	installCmd, _, _ := rootCmd.Find([]string{"install"})

	flag := installCmd.Flag("engine-version")
	s.NotNil(flag)
	s.Equal("", flag.DefValue)
}

func (s *InstallCommandSuite) Test_has_ls_version_flag() {
	rootCmd := NewRootCmd()
	installCmd, _, _ := rootCmd.Find([]string{"install"})

	flag := installCmd.Flag("ls-version")
	s.NotNil(flag)
	s.Equal("", flag.DefValue)
}

func (s *InstallCommandSuite) Test_has_no_modify_path_flag() {
	rootCmd := NewRootCmd()
	installCmd, _, _ := rootCmd.Find([]string{"install"})

	flag := installCmd.Flag("no-modify-path")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *InstallCommandSuite) Test_has_no_service_flag() {
	rootCmd := NewRootCmd()
	installCmd, _, _ := rootCmd.Find([]string{"install"})

	flag := installCmd.Flag("no-service")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *InstallCommandSuite) Test_has_no_plugins_flag() {
	rootCmd := NewRootCmd()
	installCmd, _, _ := rootCmd.Find([]string{"install"})

	flag := installCmd.Flag("no-plugins")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *InstallCommandSuite) Test_has_core_plugins_flag() {
	rootCmd := NewRootCmd()
	installCmd, _, _ := rootCmd.Find([]string{"install"})

	flag := installCmd.Flag("core-plugins")
	s.NotNil(flag)
	s.Equal("newstack-cloud/aws", flag.DefValue)
}

func (s *InstallCommandSuite) Test_has_force_flag() {
	rootCmd := NewRootCmd()
	installCmd, _, _ := rootCmd.Find([]string{"install"})

	flag := installCmd.Flag("force")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *InstallCommandSuite) Test_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"install", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "install")
	s.Contains(output, "cli-version")
}

func TestInstallCommandSuite(t *testing.T) {
	suite.Run(t, new(InstallCommandSuite))
}
