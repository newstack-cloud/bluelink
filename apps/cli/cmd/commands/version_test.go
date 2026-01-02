package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	"github.com/stretchr/testify/suite"
)

type VersionCommandSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
}

func (s *VersionCommandSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "version-cmd-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Create empty default config file to prevent load errors
	err = os.WriteFile(filepath.Join(tempDir, "bluelink.config.toml"), []byte(""), 0644)
	s.Require().NoError(err)

	s.originalDir, err = os.Getwd()
	s.Require().NoError(err)
	os.Chdir(tempDir)
}

func (s *VersionCommandSuite) TearDownTest() {
	os.Chdir(s.originalDir)
	os.RemoveAll(s.tempDir)
}

func (s *VersionCommandSuite) Test_prints_version() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	s.NoError(err)
	s.Contains(buf.String(), "Bluelink CLI")
	// Version is "dev" by default, or set via ldflags at build time
	s.Contains(buf.String(), utils.Version)
}

func TestVersionCommandSuite(t *testing.T) {
	suite.Run(t, new(VersionCommandSuite))
}
