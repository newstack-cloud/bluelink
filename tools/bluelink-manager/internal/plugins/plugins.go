package plugins

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
)

const (
	// Default Deploy Engine health check endpoint
	defaultHealthEndpoint = "http://127.0.0.1:8325/v1/health"
)

// InstallCore installs the core plugins.
func InstallCore(pluginsList string) error {
	if pluginsList == "" {
		return nil
	}

	ui.Println()
	ui.Info("Installing core plugins...")

	// Wait for deploy engine to be ready
	if err := waitForEngine(); err != nil {
		ui.Warn("Deploy Engine may not be ready: %v", err)
	}

	bluelinkBin := filepath.Join(paths.BinDir(), "bluelink")

	var lastErr error
	for plugin := range strings.SplitSeq(pluginsList, ",") {
		plugin = strings.TrimSpace(plugin)
		if plugin == "" {
			continue
		}

		ui.Info("Installing plugin: %s", plugin)

		cmd := exec.Command(bluelinkBin, "plugins", "install", plugin)
		cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s:%s", paths.BinDir(), os.Getenv("PATH")))

		if output, err := cmd.CombinedOutput(); err != nil {
			ui.Warn("Failed to install %s: %v\n%s", plugin, err, string(output))
			ui.Warn("You can install it later with: bluelink plugins install %s", plugin)
			lastErr = err
		} else {
			ui.Success("Installed %s", plugin)
		}
	}

	return lastErr
}

func waitForEngine() error {
	ui.Info("Waiting for Deploy Engine to start...")

	maxAttempts := 30
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for range maxAttempts {
		resp, err := client.Get(defaultHealthEndpoint)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				ui.Success("Deploy Engine is ready")
				return nil
			}
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("deploy engine did not start within %ds", maxAttempts)
}
