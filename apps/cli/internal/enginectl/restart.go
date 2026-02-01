package enginectl

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	managerBinary    = "bluelink-manager"
	launchdPlistName = "dev.bluelink.deploy-engine.plist"
	systemdService   = "bluelink-deploy-engine.service"
)

// RestartInstructions returns platform-specific instructions
// for restarting the deploy engine so newly installed plugins are loaded.
func RestartInstructions() string {
	return restartInstructions(runtime.GOOS)
}

func restartInstructions(goos string) string {
	if managerAvailable() {
		return "Run `bluelink-manager restart` to restart the deploy engine."
	}

	return platformFallback(goos)
}

func managerAvailable() bool {
	_, err := exec.LookPath(managerBinary)
	return err == nil
}

func platformFallback(goos string) string {
	switch goos {
	case "darwin":
		return darwinFallback()
	case "linux":
		return linuxFallback()
	default:
		return genericFallback()
	}
}

func darwinFallback() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return genericFallback()
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", launchdPlistName)
	if _, err := os.Stat(plistPath); err != nil {
		return genericFallback()
	}

	return "Run `launchctl unload " + plistPath +
		" && launchctl load " + plistPath +
		"` to restart the deploy engine."
}

func linuxFallback() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return genericFallback()
	}

	servicePath := filepath.Join(home, ".config", "systemd", "user", systemdService)
	if _, err := os.Stat(servicePath); err != nil {
		return genericFallback()
	}

	return "Run `systemctl --user restart " + systemdService +
		"` to restart the deploy engine."
}

func genericFallback() string {
	return "Restart the deploy engine to load the newly installed plugins."
}
