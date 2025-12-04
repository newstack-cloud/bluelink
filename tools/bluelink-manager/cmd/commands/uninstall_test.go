package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type UninstallCommandSuite struct {
	suite.Suite
}

func (s *UninstallCommandSuite) Test_has_all_flag() {
	rootCmd := NewRootCmd()
	uninstallCmd, _, _ := rootCmd.Find([]string{"uninstall"})

	flag := uninstallCmd.Flag("all")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *UninstallCommandSuite) Test_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"uninstall", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "uninstall")
	s.Contains(output, "--all")
}

func TestUninstallCommandSuite(t *testing.T) {
	suite.Run(t, new(UninstallCommandSuite))
}
