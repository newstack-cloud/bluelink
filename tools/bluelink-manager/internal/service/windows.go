//go:build windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
)

const (
	taskName = "BluelinkDeployEngine"
)

// Install installs the Deploy Engine as a Windows scheduled task that runs at user login.
// This approach doesn't require admin privileges unlike Windows services.
func Install() error {
	binPath := filepath.Join(paths.BinDir(), "deploy-engine.exe")

	// Check if task already exists
	queryCmd := exec.Command("schtasks.exe", "/Query", "/TN", taskName)
	queryCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := queryCmd.Run(); err == nil {
		ui.Info("Task already exists, updating...")
		return updateTask(binPath)
	}

	// Create scheduled task that runs at user logon
	// /SC ONLOGON - triggers at user login
	// /RL LIMITED - runs with limited privileges (no elevation)
	// /F - force create (overwrite if exists)
	cmd := exec.Command("schtasks.exe", "/Create",
		"/TN", taskName,
		"/TR", binPath,
		"/SC", "ONLOGON",
		"/RL", "LIMITED",
		"/F",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %s: %w", string(output), err)
	}

	// Start the task immediately
	if err := Start(); err != nil {
		ui.Warn("Task created but failed to start: %v", err)
		ui.Info("The Deploy Engine will start automatically at next login")
		ui.Info("Or start it manually with: bluelink-manager start")
		return nil
	}

	ui.Success("Deploy Engine installed and started")
	ui.Info("It will start automatically at login")
	ui.Info("Manage with: bluelink-manager {start|stop|restart|status}")

	return nil
}

func updateTask(binPath string) error {
	// Stop the running process first
	_ = Stop()

	// Update the task with new binary path
	cmd := exec.Command("schtasks.exe", "/Change",
		"/TN", taskName,
		"/TR", binPath,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update scheduled task: %s: %w", string(output), err)
	}

	// Start the task
	return Start()
}

// Uninstall removes the Deploy Engine scheduled task.
func Uninstall() error {
	// Stop the running process first
	_ = Stop()

	cmd := exec.Command("schtasks.exe", "/Delete", "/TN", taskName, "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "cannot find") ||
			strings.Contains(string(output), "does not exist") {
			return nil // Already uninstalled
		}
		return fmt.Errorf("failed to delete scheduled task: %s: %w", string(output), err)
	}

	return nil
}

// Start starts the Deploy Engine process.
// Since scheduled tasks with ONLOGON don't support on-demand start via schtasks /Run,
// we start the process directly.
func Start() error {
	// Check if already running
	running, _ := IsRunning()
	if running {
		return nil
	}

	binPath := filepath.Join(paths.BinDir(), "deploy-engine.exe")

	// Start the process detached from this console
	cmd := exec.Command(binPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	// Set environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("BLUELINK_HOME=%s", paths.InstallDir()))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start deploy-engine: %w", err)
	}

	// Detach - don't wait for the process
	_ = cmd.Process.Release()

	return nil
}

// Stop stops the Deploy Engine process.
func Stop() error {
	// Find and kill the deploy-engine process
	cmd := exec.Command("taskkill.exe", "/IM", "deploy-engine.exe", "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "not found") ||
			strings.Contains(outputStr, "not running") {
			return nil
		}
		// Ignore "Access denied" errors - process might have already exited
		if strings.Contains(outputStr, "Access is denied") {
			return nil
		}
		return fmt.Errorf("failed to stop deploy-engine: %s: %w", outputStr, err)
	}
	return nil
}

// Restart restarts the Deploy Engine process.
func Restart() error {
	_ = Stop()
	return Start()
}

// IsRunning checks if the Deploy Engine process is running.
func IsRunning() (bool, error) {
	cmd := exec.Command("tasklist.exe", "/FI", "IMAGENAME eq deploy-engine.exe", "/NH")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	// If the process is running, output will contain "deploy-engine.exe"
	return strings.Contains(string(output), "deploy-engine.exe"), nil
}
