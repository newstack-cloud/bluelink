package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Platform represents the detected OS and architecture.
type Platform struct {
	OS   string
	Arch string
}

// String returns the platform string (e.g., "darwin_arm64").
func (p Platform) String() string {
	return fmt.Sprintf("%s_%s", p.OS, p.Arch)
}

// DetectPlatform detects the current OS and architecture.
func DetectPlatform() (Platform, error) {
	p := Platform{}

	switch runtime.GOOS {
	case "linux":
		p.OS = "linux"
	case "darwin":
		p.OS = "darwin"
	case "windows":
		p.OS = "windows"
	default:
		return p, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	switch runtime.GOARCH {
	case "amd64":
		p.Arch = "amd64"
	case "arm64":
		p.Arch = "arm64"
	default:
		return p, fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	return p, nil
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// InstallDir returns the Bluelink installation directory.
// On Windows: %LOCALAPPDATA%\NewStack\Bluelink
// On Unix: ~/.bluelink
func InstallDir() string {
	if dir := os.Getenv("BLUELINK_INSTALL_DIR"); dir != "" {
		return dir
	}

	if IsWindows() {
		// Use %LOCALAPPDATA%\NewStack\Bluelink on Windows
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			// Fallback if LOCALAPPDATA is not set
			home, _ := os.UserHomeDir()
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(localAppData, "NewStack", "Bluelink")
	}

	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".bluelink")
}

// BinDir returns the directory for Bluelink binaries.
func BinDir() string {
	return filepath.Join(InstallDir(), "bin")
}

// ConfigDir returns the directory for Bluelink configuration.
func ConfigDir() string {
	return filepath.Join(InstallDir(), "config")
}

// EngineDir returns the directory for Deploy Engine data.
func EngineDir() string {
	return filepath.Join(InstallDir(), "engine")
}

// PluginsDir returns the directory for plugins.
func PluginsDir() string {
	return filepath.Join(EngineDir(), "plugins")
}

// StateDir returns the directory for state data.
func StateDir() string {
	return filepath.Join(EngineDir(), "state")
}

// EnsureDirectories creates all required directories.
func EnsureDirectories() error {
	dirs := []string{
		BinDir(),
		ConfigDir(),
		PluginsDir(),
		filepath.Join(PluginsDir(), "bin"),
		filepath.Join(PluginsDir(), "logs"),
		StateDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
