//go:build windows

package service

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
)

const (
	windowsServiceName        = "BluelinkDeployEngine"
	windowsServiceDisplayName = "Bluelink Deploy Engine"
	windowsServiceDescription = "Background service for Bluelink infrastructure deployments"
)

// Install installs the Deploy Engine as a Windows service.
func Install() error {
	binPath := filepath.Join(paths.BinDir(), "deploy-engine.exe")

	// Use sc.exe to create the service
	// Running as the current user's service (not LocalSystem)
	cmd := exec.Command("sc.exe", "create", windowsServiceName,
		"binPath=", binPath,
		"DisplayName=", windowsServiceDisplayName,
		"start=", "auto",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if service already exists
		if strings.Contains(string(output), "exists") {
			ui.Info("Service already exists, updating...")
			return updateWindowsService(binPath)
		}
		return fmt.Errorf("failed to create service: %s: %w", string(output), err)
	}

	// Set the description
	descCmd := exec.Command("sc.exe", "description", windowsServiceName, windowsServiceDescription)
	_ = descCmd.Run() // Ignore error, description is optional

	// Set environment variable for BLUELINK_HOME
	// This requires modifying the registry, so we'll set it via service parameters
	setEnvCmd := exec.Command("reg", "add",
		fmt.Sprintf(`HKLM\SYSTEM\CurrentControlSet\Services\%s`, windowsServiceName),
		"/v", "Environment",
		"/t", "REG_MULTI_SZ",
		"/d", fmt.Sprintf("BLUELINK_HOME=%s", paths.InstallDir()),
		"/f",
	)
	_ = setEnvCmd.Run() // Best effort

	// Start the service
	if err := Start(); err != nil {
		ui.Warn("Service created but failed to start: %v", err)
		ui.Info("You can start it manually with: sc.exe start %s", windowsServiceName)
		return nil
	}

	ui.Success("Deploy Engine service installed and started")
	ui.Info("Manage with: sc.exe {start|stop|query} %s", windowsServiceName)

	return nil
}

func updateWindowsService(binPath string) error {
	// Stop the service first
	_ = Stop()

	// Update the binary path
	cmd := exec.Command("sc.exe", "config", windowsServiceName, "binPath=", binPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update service: %s: %w", string(output), err)
	}

	// Start the service
	return Start()
}

// Uninstall removes the Deploy Engine Windows service.
func Uninstall() error {
	// Stop the service first
	_ = Stop()

	cmd := exec.Command("sc.exe", "delete", windowsServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "does not exist") {
			return nil // Already uninstalled
		}
		return fmt.Errorf("failed to delete service: %s: %w", string(output), err)
	}

	return nil
}

// Start starts the Deploy Engine Windows service.
func Start() error {
	cmd := exec.Command("sc.exe", "start", windowsServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "already running") {
			return nil
		}
		return fmt.Errorf("failed to start service: %s: %w", string(output), err)
	}
	return nil
}

// Stop stops the Deploy Engine Windows service.
func Stop() error {
	cmd := exec.Command("sc.exe", "stop", windowsServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "not started") ||
			strings.Contains(string(output), "does not exist") {
			return nil
		}
		return fmt.Errorf("failed to stop service: %s: %w", string(output), err)
	}
	return nil
}

// Restart restarts the Deploy Engine Windows service.
func Restart() error {
	_ = Stop()
	return Start()
}

// IsRunning checks if the Deploy Engine Windows service is running.
func IsRunning() (bool, error) {
	cmd := exec.Command("sc.exe", "query", windowsServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "does not exist") {
			return false, nil
		}
		return false, err
	}

	// Check if the output contains "RUNNING"
	return strings.Contains(string(output), "RUNNING"), nil
}
