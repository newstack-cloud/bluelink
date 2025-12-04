//go:build !windows

package shell

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProfileSuite struct {
	suite.Suite
	tempDir            string
	originalInstallDir string
	originalHome       string
	originalShell      string
}

func (s *ProfileSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "bluelink-profile-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	s.originalInstallDir = os.Getenv("BLUELINK_INSTALL_DIR")
	s.originalHome = os.Getenv("HOME")
	s.originalShell = os.Getenv("SHELL")

	os.Setenv("BLUELINK_INSTALL_DIR", tempDir)
	os.Setenv("HOME", tempDir)

	// Create bin directory
	os.MkdirAll(filepath.Join(tempDir, "bin"), 0755)
}

func (s *ProfileSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)

	if s.originalInstallDir != "" {
		os.Setenv("BLUELINK_INSTALL_DIR", s.originalInstallDir)
	} else {
		os.Unsetenv("BLUELINK_INSTALL_DIR")
	}

	os.Setenv("HOME", s.originalHome)
	os.Setenv("SHELL", s.originalShell)
}

func (s *ProfileSuite) Test_SetupPath_adds_to_bashrc_for_bash() {
	os.Setenv("SHELL", "/bin/bash")

	// Create .bashrc
	bashrc := filepath.Join(s.tempDir, ".bashrc")
	os.WriteFile(bashrc, []byte("# existing content\n"), 0644)

	err := SetupPath()
	s.NoError(err)

	content, _ := os.ReadFile(bashrc)
	s.Contains(string(content), "Bluelink")
	s.Contains(string(content), filepath.Join(s.tempDir, "bin"))
}

func (s *ProfileSuite) Test_SetupPath_adds_to_zshrc_for_zsh() {
	os.Setenv("SHELL", "/bin/zsh")

	// Create .zshrc
	zshrc := filepath.Join(s.tempDir, ".zshrc")
	os.WriteFile(zshrc, []byte("# existing content\n"), 0644)

	err := SetupPath()
	s.NoError(err)

	content, _ := os.ReadFile(zshrc)
	s.Contains(string(content), "Bluelink")
	s.Contains(string(content), filepath.Join(s.tempDir, "bin"))
}

func (s *ProfileSuite) Test_SetupPath_skips_if_already_configured() {
	os.Setenv("SHELL", "/bin/bash")

	// Create .bashrc with BLUELINK already present
	bashrc := filepath.Join(s.tempDir, ".bashrc")
	os.WriteFile(bashrc, []byte("# BLUELINK config\nexport PATH=\"$PATH:/some/path\"\n"), 0644)

	err := SetupPath()
	s.NoError(err)

	// Content should be unchanged (no duplicate)
	content, _ := os.ReadFile(bashrc)
	s.Equal("# BLUELINK config\nexport PATH=\"$PATH:/some/path\"\n", string(content))
}

func (s *ProfileSuite) Test_SetupPath_uses_fish_syntax_for_fish() {
	os.Setenv("SHELL", "/usr/bin/fish")

	// Create fish config directory and file
	fishDir := filepath.Join(s.tempDir, ".config", "fish")
	os.MkdirAll(fishDir, 0755)
	fishConfig := filepath.Join(fishDir, "config.fish")
	os.WriteFile(fishConfig, []byte("# existing content\n"), 0644)

	err := SetupPath()
	s.NoError(err)

	content, _ := os.ReadFile(fishConfig)
	s.Contains(string(content), "set -gx PATH")
}

func TestProfileSuite(t *testing.T) {
	suite.Run(t, new(ProfileSuite))
}
