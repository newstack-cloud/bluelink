package plugins

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tailscale/hujson"
)

// DeployConfig represents the structure of bluelink.deploy.json.
type DeployConfig struct {
	Dependencies map[string]string `json:"dependencies"`
}

// LoadDeployConfig loads a deploy config from the specified path.
func LoadDeployConfig(path string) (*DeployConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read deploy config: %w", err)
	}

	data, err = hujson.Standardize(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deploy config: %w", err)
	}

	var config DeployConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse deploy config: %w", err)
	}

	if config.Dependencies == nil {
		config.Dependencies = make(map[string]string)
	}

	return &config, nil
}

// GetPluginIDs parses the dependencies and returns a list of PluginIDs.
func (c *DeployConfig) GetPluginIDs() ([]*PluginID, error) {
	var ids []*PluginID

	for pluginKey, version := range c.Dependencies {
		id, err := ParsePluginID(pluginKey)
		if err != nil {
			return nil, fmt.Errorf("invalid plugin dependency %q: %w", pluginKey, err)
		}

		// Set the version from the dependency map
		id.Version = version
		ids = append(ids, id)
	}

	return ids, nil
}
