//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
)

const serviceName = "bluelink-deploy-engine.service"

func serviceFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", serviceName)
}

// Install installs the Deploy Engine as a systemd user service on Linux.
func Install() error {
	// Check if systemctl is available
	if _, err := exec.LookPath("systemctl"); err != nil {
		ui.Warn("systemd not found, skipping service installation")
		return nil
	}

	serviceDir := filepath.Dir(serviceFilePath())
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return err
	}

	unit := fmt.Sprintf(`[Unit]
Description=Bluelink Deploy Engine
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5
Environment=BLUELINK_HOME=%s

[Install]
WantedBy=default.target
`, filepath.Join(paths.BinDir(), "deploy-engine"), paths.InstallDir())

	if err := os.WriteFile(serviceFilePath(), []byte(unit), 0644); err != nil {
		return err
	}

	// Reload, enable, and start
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "enable", serviceName).Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "start", serviceName).Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	ui.Success("Deploy Engine service installed and started")
	ui.Info("Manage with: systemctl --user {start|stop|status} %s", serviceName)

	return nil
}

// Uninstall removes the Deploy Engine systemd service.
func Uninstall() error {
	_ = exec.Command("systemctl", "--user", "stop", serviceName).Run()
	_ = exec.Command("systemctl", "--user", "disable", serviceName).Run()
	_ = os.Remove(serviceFilePath())
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

// Start the Deploy Engine systemd service.
func Start() error {
	return exec.Command("systemctl", "--user", "start", serviceName).Run()
}

// Stop the Deploy Engine systemd service.
func Stop() error {
	return exec.Command("systemctl", "--user", "stop", serviceName).Run()
}

// Restart the Deploy Engine systemd service.
func Restart() error {
	return exec.Command("systemctl", "--user", "restart", serviceName).Run()
}

// IsRunning checks if the Deploy Engine systemd service is running.
func IsRunning() (bool, error) {
	err := exec.Command("systemctl", "--user", "is-active", "--quiet", serviceName).Run()
	if err == nil {
		return true, nil
	}
	// Exit code 3 means inactive, which is not an error
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 3 {
			return false, nil
		}
	}
	return false, err
}
