package stateio

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
)

// StateConfig holds the state-related configuration from an engine config file.
// This is a subset of the deploy engine's StateConfig, containing only
// the fields needed for state import/export operations.
type StateConfig struct {
	StorageEngine              string `json:"storage_engine"`
	MemFileStateDir            string `json:"memfile_state_dir"`
	PostgresUser               string `json:"postgres_user"`
	PostgresPassword           string `json:"postgres_password"`
	PostgresHost               string `json:"postgres_host"`
	PostgresPort               int    `json:"postgres_port"`
	PostgresDatabase           string `json:"postgres_database"`
	PostgresSSLMode            string `json:"postgres_ssl_mode"`
	PostgresPoolMaxConns       int    `json:"postgres_pool_max_conns"`
	PostgresPoolMaxConnLifetime string `json:"postgres_pool_max_conn_lifetime"`
}

// EngineConfig represents the deploy engine configuration file structure.
// Only the state-related fields are parsed.
type EngineConfig struct {
	State StateConfig `json:"state"`
}

const (
	StorageEngineMemfile  = "memfile"
	StorageEnginePostgres = "postgres"
)

// LoadEngineConfig loads and parses a deploy engine configuration file.
func LoadEngineConfig(configFilePath string) (*EngineConfig, error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read engine config file: %w", err)
	}

	var config EngineConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse engine config file: %w", err)
	}

	applyStateConfigDefaults(&config.State)

	return &config, nil
}

func applyStateConfigDefaults(config *StateConfig) {
	if config.StorageEngine == "" {
		config.StorageEngine = StorageEngineMemfile
	}
	if config.MemFileStateDir == "" {
		config.MemFileStateDir = GetDefaultStateDir()
	}
	if config.PostgresHost == "" {
		config.PostgresHost = "localhost"
	}
	if config.PostgresPort == 0 {
		config.PostgresPort = 5432
	}
	if config.PostgresSSLMode == "" {
		config.PostgresSSLMode = "disable"
	}
	if config.PostgresPoolMaxConns == 0 {
		config.PostgresPoolMaxConns = 100
	}
	if config.PostgresPoolMaxConnLifetime == "" {
		config.PostgresPoolMaxConnLifetime = "1h30m"
	}
}

// GetDefaultStateDir returns the default state directory based on the OS.
func GetDefaultStateDir() string {
	if runtime.GOOS == "windows" {
		return os.ExpandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\engine\\state")
	}
	return os.ExpandEnv("$HOME/.bluelink/engine/state")
}

// GetDefaultEngineConfigPath returns the default engine config file path.
func GetDefaultEngineConfigPath() string {
	if runtime.GOOS == "windows" {
		return os.ExpandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\engine\\config.json")
	}
	return os.ExpandEnv("$HOME/.bluelink/engine/config.json")
}

// BuildPostgresDatabaseURL constructs a postgres connection URL from config.
func BuildPostgresDatabaseURL(config *StateConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&pool_max_conns=%d&pool_max_conn_lifetime=%s",
		config.PostgresUser,
		config.PostgresPassword,
		config.PostgresHost,
		config.PostgresPort,
		config.PostgresDatabase,
		config.PostgresSSLMode,
		config.PostgresPoolMaxConns,
		config.PostgresPoolMaxConnLifetime,
	)
}
