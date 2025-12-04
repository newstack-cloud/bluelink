//go:build darwin

package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
)

const (
	launchdLabel = "dev.bluelink.deploy-engine"
)

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

// Install installs the Deploy Engine as a launchd service on macOS.
func Install() error {
	plistDir := filepath.Dir(plistPath())
	if err := os.MkdirAll(plistDir, 0755); err != nil {
		return err
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>BLUELINK_HOME</key>
        <string>%s</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, launchdLabel,
		filepath.Join(paths.BinDir(), "deploy-engine"),
		paths.InstallDir(),
		filepath.Join(paths.EngineDir(), "deploy-engine.log"),
		filepath.Join(paths.EngineDir(), "deploy-engine.err"))

	if err := os.WriteFile(plistPath(), []byte(plist), 0644); err != nil {
		return err
	}

	// Unload first (ignore errors)
	_ = exec.Command("launchctl", "unload", plistPath()).Run()

	// Load the service
	if err := exec.Command("launchctl", "load", plistPath()).Run(); err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	ui.Success("Deploy Engine service installed and started")
	ui.Info("Manage with: launchctl load|unload %s", plistPath())

	return nil
}

// Uninstall removes the Deploy Engine launchd service.
func Uninstall() error {
	_ = exec.Command("launchctl", "unload", plistPath()).Run()
	return os.Remove(plistPath())
}

// Start the Deploy Engine launchd service.
func Start() error {
	return exec.Command("launchctl", "load", plistPath()).Run()
}

// Stop the Deploy Engine launchd service.
func Stop() error {
	return exec.Command("launchctl", "unload", plistPath()).Run()
}

// Restart the Deploy Engine launchd service.
func Restart() error {
	_ = Stop()
	return Start()
}

// IsRunning checks if the Deploy Engine launchd service is running.
func IsRunning() (bool, error) {
	cmd := exec.Command("launchctl", "list")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return false, err
	}

	return strings.Contains(out.String(), launchdLabel), nil
}
