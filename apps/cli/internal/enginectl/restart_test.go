package enginectl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RestartSuite struct {
	suite.Suite
	tempDir string
}

func TestRestartSuite(t *testing.T) {
	suite.Run(t, new(RestartSuite))
}

func (s *RestartSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "restart-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *RestartSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *RestartSuite) TestGenericFallback() {
	msg := genericFallback()
	s.Equal("Restart the deploy engine to load the newly installed plugins.", msg)
}

func (s *RestartSuite) TestPlatformFallback_unknown_os() {
	msg := platformFallback("freebsd")
	s.Equal(genericFallback(), msg)
}

func (s *RestartSuite) TestDarwinFallback_plist_exists() {
	home, err := os.UserHomeDir()
	s.Require().NoError(err)

	plistPath := filepath.Join(home, "Library", "LaunchAgents", launchdPlistName)

	// Only test if the plist actually exists on this machine
	if _, err := os.Stat(plistPath); err != nil {
		s.T().Skip("launchd plist not present, skipping")
	}

	msg := darwinFallback()
	s.Contains(msg, "launchctl unload")
	s.Contains(msg, "launchctl load")
	s.Contains(msg, launchdPlistName)
}

func (s *RestartSuite) TestLinuxFallback_service_not_found() {
	// On macOS or systems without systemd, this should return generic fallback
	msg := linuxFallback()
	home, _ := os.UserHomeDir()
	servicePath := filepath.Join(home, ".config", "systemd", "user", systemdService)
	if _, err := os.Stat(servicePath); err != nil {
		s.Equal(genericFallback(), msg)
	} else {
		s.Contains(msg, "systemctl --user restart")
	}
}

func (s *RestartSuite) TestRestartInstructions_returns_string() {
	// Just verify it returns a non-empty string on any platform
	msg := RestartInstructions()
	s.NotEmpty(msg)
}

func (s *RestartSuite) TestRestartInstructions_internal_unknown_os() {
	msg := restartInstructions("plan9")
	s.Equal(genericFallback(), msg)
}
