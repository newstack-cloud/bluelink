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
	"golang.org/x/sys/windows/registry"
)

const (
	registryKeyName = "BluelinkDeployEngine"
	runKeyPath      = `Software\Microsoft\Windows\CurrentVersion\Run`
)

// Installs the Deploy Engine to start automatically at user login.
// Uses the Windows Registry Run key, which doesn't require admin privileges.
func Install() error {
	binPath := filepath.Join(paths.BinDir(), "deploy-engine.exe")

	// Open the Run key for the current user
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	// Check if already registered
	existingPath, _, err := key.GetStringValue(registryKeyName)
	if err == nil && existingPath == binPath {
		ui.Info("Already registered, starting...")
	} else {
		// Set the registry value
		if err := key.SetStringValue(registryKeyName, binPath); err != nil {
			return fmt.Errorf("failed to set registry value: %w", err)
		}
		ui.Success("Registered to start at login")
	}

	// Start the process immediately
	if err := Start(); err != nil {
		ui.Warn("Registered but failed to start: %v", err)
		ui.Info("The Deploy Engine will start automatically at next login")
		ui.Info("Or start it manually with: bluelink-manager start")
		return nil
	}

	ui.Success("Deploy Engine installed and started")
	ui.Info("It will start automatically at login")
	ui.Info("Manage with: bluelink-manager {start|stop|restart|status}")

	return nil
}

// Uninstall removes the Deploy Engine from auto-start.
func Uninstall() error {
	// Stop the running process first
	_ = Stop()

	// Open the Run key
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		// Key doesn't exist or can't be opened - nothing to uninstall
		return nil
	}
	defer key.Close()

	// Delete the registry value
	if err := key.DeleteValue(registryKeyName); err != nil {
		// Value doesn't exist - already uninstalled
		if err == registry.ErrNotExist {
			return nil
		}
		return fmt.Errorf("failed to delete registry value: %w", err)
	}

	return nil
}

// Start starts the Deploy Engine process.
func Start() error {
	// Check if already running
	running, _ := IsRunning()
	if running {
		return nil
	}

	binPath := filepath.Join(paths.BinDir(), "deploy-engine.exe")

	// Start the process fully detached from this console
	// DETACHED_PROCESS (0x8) - process has no console
	// CREATE_NEW_PROCESS_GROUP (0x200) - new process group for ctrl+c handling
	const DETACHED_PROCESS = 0x00000008
	cmd := exec.Command(binPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS,
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
