package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RootCommandSuite struct {
	suite.Suite
}

func (s *RootCommandSuite) Test_has_expected_subcommands() {
	rootCmd := NewRootCmd()

	// These are the commands we explicitly add
	expectedCommands := []string{
		"install",
		"update",
		"uninstall",
		"status",
		"start",
		"stop",
		"restart",
		"self-update",
		"version",
	}

	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		s.True(found, "expected subcommand %q not found", expected)
	}
}

func (s *RootCommandSuite) Test_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	rootCmd.Execute()
	output := buf.String()

	s.Contains(output, "bluelink-manager")
	s.Contains(output, "install")
	s.Contains(output, "Available Commands")
}

func TestRootCommandSuite(t *testing.T) {
	suite.Run(t, new(RootCommandSuite))
}
