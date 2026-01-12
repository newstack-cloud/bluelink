package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CleanupCommandSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
}

func (s *CleanupCommandSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "cleanup-cmd-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	err = os.WriteFile(filepath.Join(tempDir, "bluelink.config.toml"), []byte(""), 0644)
	s.Require().NoError(err)

	s.originalDir, err = os.Getwd()
	s.Require().NoError(err)
	os.Chdir(tempDir)
}

func (s *CleanupCommandSuite) TearDownTest() {
	os.Chdir(s.originalDir)
	os.RemoveAll(s.tempDir)
}

func (s *CleanupCommandSuite) Test_has_validations_flag() {
	rootCmd := NewRootCmd()
	cleanupCmd, _, _ := rootCmd.Find([]string{"cleanup"})

	flag := cleanupCmd.Flag("validations")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *CleanupCommandSuite) Test_has_changesets_flag() {
	rootCmd := NewRootCmd()
	cleanupCmd, _, _ := rootCmd.Find([]string{"cleanup"})

	flag := cleanupCmd.Flag("changesets")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *CleanupCommandSuite) Test_has_reconciliation_results_flag() {
	rootCmd := NewRootCmd()
	cleanupCmd, _, _ := rootCmd.Find([]string{"cleanup"})

	flag := cleanupCmd.Flag("reconciliation-results")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *CleanupCommandSuite) Test_has_events_flag() {
	rootCmd := NewRootCmd()
	cleanupCmd, _, _ := rootCmd.Find([]string{"cleanup"})

	flag := cleanupCmd.Flag("events")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *CleanupCommandSuite) Test_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"cleanup", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "cleanup")
	s.Contains(output, "validations")
	s.Contains(output, "changesets")
	s.Contains(output, "reconciliation-results")
	s.Contains(output, "events")
}

func (s *CleanupCommandSuite) Test_help_contains_flag_descriptions() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"cleanup", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "retention period")
	s.Contains(output, "Cleanup change sets")
}

func (s *CleanupCommandSuite) Test_short_description() {
	rootCmd := NewRootCmd()
	cleanupCmd, _, _ := rootCmd.Find([]string{"cleanup"})

	s.Contains(cleanupCmd.Short, "Cleanup temporary resources")
}

func TestCleanupCommandSuite(t *testing.T) {
	suite.Run(t, new(CleanupCommandSuite))
}
