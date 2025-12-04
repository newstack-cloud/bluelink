package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type VersionCommandSuite struct {
	suite.Suite
}

func (s *VersionCommandSuite) Test_version_command_outputs_version() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	s.NoError(err)

	output := buf.String()
	s.Contains(output, "bluelink-manager")
	s.Contains(output, "OS/Arch")
	s.Contains(output, "Built")
}

func (s *VersionCommandSuite) Test_help_contains_version_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "version")
}

func TestVersionCommandSuite(t *testing.T) {
	suite.Run(t, new(VersionCommandSuite))
}
