package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ValidateCommandSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
}

func (s *ValidateCommandSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "validate-cmd-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Create empty default config file to prevent load errors
	err = os.WriteFile(filepath.Join(tempDir, "bluelink.config.toml"), []byte(""), 0644)
	s.Require().NoError(err)

	s.originalDir, err = os.Getwd()
	s.Require().NoError(err)
	os.Chdir(tempDir)
}

func (s *ValidateCommandSuite) TearDownTest() {
	os.Chdir(s.originalDir)
	os.RemoveAll(s.tempDir)
}

// Flag tests

func (s *ValidateCommandSuite) Test_has_blueprint_file_flag() {
	rootCmd := NewRootCmd()
	validateCmd, _, _ := rootCmd.Find([]string{"validate"})

	flag := validateCmd.Flag("blueprint-file")
	s.NotNil(flag)
	s.Equal("project.blueprint.yaml", flag.DefValue)
}

// Help text tests

func (s *ValidateCommandSuite) Test_help_contains_usage_info() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"validate", "--help"})

	rootCmd.Execute()
	output := buf.String()
	s.Contains(output, "validate")
	s.Contains(output, "blueprint")
}

func TestValidateCommandSuite(t *testing.T) {
	suite.Run(t, new(ValidateCommandSuite))
}
