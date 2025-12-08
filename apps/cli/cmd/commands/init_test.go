package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/stretchr/testify/suite"
)

type InitCommandSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
}

func (s *InitCommandSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "init-cmd-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Create empty default config file to prevent load errors
	err = os.WriteFile(filepath.Join(tempDir, "bluelink.config.toml"), []byte(""), 0644)
	s.Require().NoError(err)

	s.originalDir, err = os.Getwd()
	s.Require().NoError(err)
	os.Chdir(tempDir)
}

func (s *InitCommandSuite) TearDownTest() {
	os.Chdir(s.originalDir)
	os.RemoveAll(s.tempDir)
}

// Flag tests

func (s *InitCommandSuite) Test_has_template_flag() {
	rootCmd := NewRootCmd()
	initCmd, _, _ := rootCmd.Find([]string{"init"})

	flag := initCmd.Flag("template")
	s.NotNil(flag)
	s.Equal("scaffold", flag.DefValue)
}

func (s *InitCommandSuite) Test_has_project_name_flag() {
	rootCmd := NewRootCmd()
	initCmd, _, _ := rootCmd.Find([]string{"init"})

	flag := initCmd.Flag("project-name")
	s.NotNil(flag)
	s.Equal("", flag.DefValue)
}

func (s *InitCommandSuite) Test_has_blueprint_format_flag() {
	rootCmd := NewRootCmd()
	initCmd, _, _ := rootCmd.Find([]string{"init"})

	flag := initCmd.Flag("blueprint-format")
	s.NotNil(flag)
	s.Equal("yaml", flag.DefValue)
}

func (s *InitCommandSuite) Test_has_no_git_flag() {
	rootCmd := NewRootCmd()
	initCmd, _, _ := rootCmd.Find([]string{"init"})

	flag := initCmd.Flag("no-git")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

// Help text tests

func (s *InitCommandSuite) Test_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"init", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "init")
	s.Contains(output, "project")
}

// Headless mode behavioral tests
//
// Note: Tests that pass headless validation will attempt to start the TUI,
// which hangs in a test environment. To fully test the "success" paths,
// the command would need refactoring to accept a TUI factory interface.
// For now, we test the validation error paths which fail fast before TUI init.

func (s *InitCommandSuite) Test_headless_mode_requires_project_name() {
	cleanup := headless.SetHeadlessForTesting(true)
	defer cleanup()

	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init"})

	err := rootCmd.Execute()
	s.Error(err)
	s.Contains(err.Error(), "--project-name")
	s.Contains(err.Error(), "non-interactive")
}

func TestInitCommandSuite(t *testing.T) {
	suite.Run(t, new(InitCommandSuite))
}
