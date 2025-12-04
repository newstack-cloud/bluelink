//go:build windows

package shell

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
	"golang.org/x/sys/windows/registry"
)

// SetupPath adds the Bluelink bin directory to the user's PATH on Windows.
// This modifies the user's PATH environment variable in the registry.
func SetupPath() error {
	binDir := paths.BinDir()

	// Open the user environment registry key
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	// Get current PATH
	currentPath, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to read PATH: %w", err)
	}

	// Check if already in PATH
	if strings.Contains(strings.ToLower(currentPath), strings.ToLower(binDir)) {
		ui.Info("Bin directory already in PATH")
		return nil
	}

	// Append bin directory to PATH
	var newPath string
	if currentPath == "" {
		newPath = binDir
	} else {
		newPath = currentPath + ";" + binDir
	}

	// Write the new PATH
	if err := key.SetStringValue("Path", newPath); err != nil {
		return fmt.Errorf("failed to update PATH: %w", err)
	}

	// Broadcast WM_SETTINGCHANGE to notify other applications
	broadcastSettingChange()

	ui.Success("Added %s to user PATH", binDir)
	ui.Info("You may need to restart your terminal or sign out and back in for changes to take effect")

	return nil
}

// broadcastSettingChange notifies Windows that environment variables have changed.
// This allows new command prompts to see the updated PATH immediately.
func broadcastSettingChange() {
	// Use PowerShell to broadcast the setting change
	// This is more reliable than calling the Windows API directly
	cmd := exec.Command("powershell", "-Command",
		`Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition '[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);'; `+
			`$HWND_BROADCAST = [IntPtr]0xffff; $WM_SETTINGCHANGE = 0x1a; $result = [UIntPtr]::Zero; `+
			`[Win32.NativeMethods]::SendMessageTimeout($HWND_BROADCAST, $WM_SETTINGCHANGE, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null`)
	_ = cmd.Run() // Best effort, ignore errors
}
