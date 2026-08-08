package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
)

// CLIAuthConfig represents the CLI's auth configuration.
type CLIAuthConfig struct {
	Method string `json:"method"`
	APIKey string `json:"apiKey"`
}

// EngineConfig represents the Deploy Engine configuration.
type EngineConfig struct {
	Auth         EngineAuthConfig `json:"auth"`
	LoopbackOnly bool             `json:"loopback_only"`
	// UseUnixSocket makes the engine listen on a unix domain socket instead of
	// a loopback TCP port. Omitted (engine defaults to TCP) on Windows, which
	// cannot use a unix socket outside WSL.
	UseUnixSocket bool `json:"use_unix_socket,omitempty"`
}

// EngineAuthConfig represents the auth section of engine config.
type EngineAuthConfig struct {
	BluelinkAPIKeys []string `json:"bluelink_api_keys"`
}

// ConfigureAuth sets up authentication between CLI and Deploy Engine.
func ConfigureAuth(force bool) error {
	ui.Info("Configuring authentication...")

	cliAuthPath := filepath.Join(paths.ConfigDir(), "engine.auth.json")
	engineConfigPath := filepath.Join(paths.EngineDir(), "config.json")

	// Check if already configured
	if !force {
		if _, err := os.Stat(cliAuthPath); err == nil {
			ui.Info("Configuration already exists, skipping (use --force to regenerate)")
			return nil
		}
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}

	// Write CLI auth config
	cliAuth := CLIAuthConfig{
		Method: "apiKey",
		APIKey: apiKey,
	}

	if err := writeJSON(cliAuthPath, cliAuth); err != nil {
		return fmt.Errorf("failed to write CLI auth config: %w", err)
	}

	// Write engine config
	engineConfig := EngineConfig{
		Auth: EngineAuthConfig{
			BluelinkAPIKeys: []string{apiKey},
		},
		LoopbackOnly: true,
		// Prefer a unix domain socket on unix-like systems: it can be locked to
		// the owning user via filesystem permissions (the engine chmods it
		// 0600), unlike a loopback TCP port that any local user can reach. The
		// CLIs default to a unix socket on these systems too, so the two align
		// out of the box at the engine's default socket path.
		UseUnixSocket: runtime.GOOS != "windows",
	}

	if err := writeJSON(engineConfigPath, engineConfig); err != nil {
		return fmt.Errorf("failed to write engine config: %w", err)
	}

	ui.Success("Generated API key for CLI <-> Deploy Engine communication")
	return nil
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func writeJSON(path string, v any) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
