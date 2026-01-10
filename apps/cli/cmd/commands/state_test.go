package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type StateCommandSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
}

func (s *StateCommandSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "state-cmd-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Create empty default config file to prevent load errors
	err = os.WriteFile(filepath.Join(tempDir, "bluelink.config.toml"), []byte(""), 0644)
	s.Require().NoError(err)

	s.originalDir, err = os.Getwd()
	s.Require().NoError(err)
	os.Chdir(tempDir)
}

func (s *StateCommandSuite) TearDownTest() {
	os.Chdir(s.originalDir)
	os.RemoveAll(s.tempDir)
}

// Command structure tests

func (s *StateCommandSuite) Test_state_command_exists() {
	rootCmd := NewRootCmd()
	stateCmd, _, err := rootCmd.Find([]string{"state"})

	s.Require().NoError(err)
	s.NotNil(stateCmd)
	s.Equal("state", stateCmd.Name())
}

func (s *StateCommandSuite) Test_import_subcommand_exists() {
	rootCmd := NewRootCmd()
	importCmd, _, err := rootCmd.Find([]string{"state", "import"})

	s.Require().NoError(err)
	s.NotNil(importCmd)
	s.Equal("import", importCmd.Name())
}

// Flag tests

func (s *StateCommandSuite) Test_import_has_file_flag() {
	rootCmd := NewRootCmd()
	importCmd, _, _ := rootCmd.Find([]string{"state", "import"})

	flag := importCmd.Flag("file")
	s.NotNil(flag)
	s.Equal("", flag.DefValue)
}

func (s *StateCommandSuite) Test_import_has_json_flag() {
	rootCmd := NewRootCmd()
	importCmd, _, _ := rootCmd.Find([]string{"state", "import"})

	flag := importCmd.Flag("json")
	s.NotNil(flag)
	s.Equal("false", flag.DefValue)
}

func (s *StateCommandSuite) Test_import_has_engine_config_file_flag() {
	rootCmd := NewRootCmd()
	importCmd, _, _ := rootCmd.Find([]string{"state", "import"})

	flag := importCmd.Flag("engine-config-file")
	s.NotNil(flag)
	s.Equal("", flag.DefValue)
}

// Help text tests

func (s *StateCommandSuite) Test_state_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"state", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "bluelink state")
	s.Contains(output, "import")
}

func (s *StateCommandSuite) Test_import_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"state", "import", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "bluelink state import")
	s.Contains(output, "--file")
	s.Contains(output, "s3://")
	s.Contains(output, "gcs://")
	s.Contains(output, "azureblob://")
}

func TestStateCommandSuite(t *testing.T) {
	suite.Run(t, new(StateCommandSuite))
}
