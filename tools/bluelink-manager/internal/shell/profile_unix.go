//go:build !windows

package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
)

// SetupPath adds the Bluelink bin directory to the user's PATH.
func SetupPath() error {
	binDir := paths.BinDir()

	// Check if already in PATH
	pathEnv := os.Getenv("PATH")
	if strings.Contains(pathEnv, binDir) {
		ui.Info("Bin directory already in PATH")
		return nil
	}

	// Detect shell and profile
	profile, shellName := detectProfile()

	// Check if already configured in profile
	if profile != "" {
		content, err := os.ReadFile(profile)
		if err == nil && strings.Contains(string(content), "BLUELINK") {
			ui.Info("PATH already configured in %s", profile)
			return nil
		}
	}

	if profile == "" {
		ui.Warn("Could not detect shell profile, please add %s to your PATH manually", binDir)
		return nil
	}

	ui.Info("Adding %s to PATH in %s", binDir, profile)

	// Build the line to add
	var line string
	if shellName == "fish" {
		line = fmt.Sprintf("\n# Bluelink\nset -gx PATH $PATH %s\n", binDir)
	} else {
		line = fmt.Sprintf("\n# Bluelink\nexport PATH=\"$PATH:%s\"\n", binDir)
	}

	// Append to profile
	f, err := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", profile, err)
	}
	defer f.Close()

	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("failed to write to %s: %w", profile, err)
	}

	ui.Success("Added to PATH in %s", profile)
	ui.Warn("Run 'source %s' or start a new shell to use bluelink commands", profile)

	return nil
}

func detectProfile() (string, string) {
	shell := os.Getenv("SHELL")
	shellName := filepath.Base(shell)
	home, _ := os.UserHomeDir()

	switch shellName {
	case "bash":
		// Prefer .bashrc if it exists, otherwise .bash_profile
		bashrc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc, shellName
		}
		return filepath.Join(home, ".bash_profile"), shellName

	case "zsh":
		return filepath.Join(home, ".zshrc"), shellName

	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), shellName

	default:
		return filepath.Join(home, ".profile"), shellName
	}
}
