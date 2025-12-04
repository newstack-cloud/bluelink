package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type UpdateCommandSuite struct {
	suite.Suite
}

func (s *UpdateCommandSuite) Test_has_cli_version_flag() {
	rootCmd := NewRootCmd()
	updateCmd, _, _ := rootCmd.Find([]string{"update"})

	flag := updateCmd.Flag("cli-version")
	s.NotNil(flag)
	s.Equal("", flag.DefValue)
}

func (s *UpdateCommandSuite) Test_has_engine_version_flag() {
	rootCmd := NewRootCmd()
	updateCmd, _, _ := rootCmd.Find([]string{"update"})

	flag := updateCmd.Flag("engine-version")
	s.NotNil(flag)
	s.Equal("", flag.DefValue)
}

func (s *UpdateCommandSuite) Test_has_ls_version_flag() {
	rootCmd := NewRootCmd()
	updateCmd, _, _ := rootCmd.Find([]string{"update"})

	flag := updateCmd.Flag("ls-version")
	s.NotNil(flag)
	s.Equal("", flag.DefValue)
}

func (s *UpdateCommandSuite) Test_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"update", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "update")
	s.Contains(output, "cli-version")
}

func TestUpdateCommandSuite(t *testing.T) {
	suite.Run(t, new(UpdateCommandSuite))
}
