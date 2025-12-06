package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PathsSuite struct {
	suite.Suite
	originalInstallDir string
}

func (s *PathsSuite) SetupTest() {
	s.originalInstallDir = os.Getenv("BLUELINK_INSTALL_DIR")
}

func (s *PathsSuite) TearDownTest() {
	if s.originalInstallDir != "" {
		os.Setenv("BLUELINK_INSTALL_DIR", s.originalInstallDir)
	} else {
		os.Unsetenv("BLUELINK_INSTALL_DIR")
	}
}

func (s *PathsSuite) Test_DetectPlatform_returns_valid_platform() {
	platform, err := DetectPlatform()

	s.NoError(err)
	s.Contains([]string{"darwin", "linux", "windows"}, platform.OS)
	s.Contains([]string{"amd64", "arm64"}, platform.Arch)
}

func (s *PathsSuite) Test_IsWindows_returns_correct_value() {
	expected := runtime.GOOS == "windows"
	s.Equal(expected, IsWindows())
}

func (s *PathsSuite) Test_Platform_String_formats_correctly() {
	platform := Platform{OS: "darwin", Arch: "arm64"}
	s.Equal("darwin_arm64", platform.String())
}

func (s *PathsSuite) Test_InstallDir_returns_default_when_env_not_set() {
	os.Unsetenv("BLUELINK_INSTALL_DIR")

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".bluelink")

	s.Equal(expected, InstallDir())
}

func (s *PathsSuite) Test_InstallDir_returns_env_value_when_set() {
	os.Setenv("BLUELINK_INSTALL_DIR", "/custom/path")

	s.Equal("/custom/path", InstallDir())
}

func (s *PathsSuite) Test_BinDir_returns_bin_subdirectory() {
	os.Setenv("BLUELINK_INSTALL_DIR", "/test/path")

	s.Equal("/test/path/bin", BinDir())
}

func (s *PathsSuite) Test_ConfigDir_returns_config_subdirectory() {
	os.Setenv("BLUELINK_INSTALL_DIR", "/test/path")

	s.Equal("/test/path/config", ConfigDir())
}

func (s *PathsSuite) Test_EngineDir_returns_engine_subdirectory() {
	os.Setenv("BLUELINK_INSTALL_DIR", "/test/path")

	s.Equal("/test/path/engine", EngineDir())
}

func (s *PathsSuite) Test_PluginsDir_returns_plugins_subdirectory() {
	os.Setenv("BLUELINK_INSTALL_DIR", "/test/path")

	s.Equal("/test/path/engine/plugins", PluginsDir())
}

func (s *PathsSuite) Test_StateDir_returns_state_subdirectory() {
	os.Setenv("BLUELINK_INSTALL_DIR", "/test/path")

	s.Equal("/test/path/engine/state", StateDir())
}

func (s *PathsSuite) Test_EnsureDirectories_creates_all_directories() {
	tempDir, err := os.MkdirTemp("", "bluelink-test-*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	os.Setenv("BLUELINK_INSTALL_DIR", tempDir)

	err = EnsureDirectories()
	s.NoError(err)

	// Verify directories exist
	s.DirExists(BinDir())
	s.DirExists(ConfigDir())
	s.DirExists(PluginsDir())
	s.DirExists(filepath.Join(PluginsDir(), "bin"))
	s.DirExists(filepath.Join(PluginsDir(), "logs"))
	s.DirExists(StateDir())
}

func (s *PathsSuite) Test_DetectPlatform_matches_runtime() {
	platform, err := DetectPlatform()
	s.NoError(err)

	// Verify OS matches runtime
	expectedOS := runtime.GOOS
	if expectedOS == "darwin" || expectedOS == "linux" || expectedOS == "windows" {
		s.Equal(expectedOS, platform.OS)
	}

	// Verify Arch matches runtime
	expectedArch := runtime.GOARCH
	if expectedArch == "amd64" || expectedArch == "arm64" {
		s.Equal(expectedArch, platform.Arch)
	}
}

func TestPathsSuite(t *testing.T) {
	suite.Run(t, new(PathsSuite))
}
