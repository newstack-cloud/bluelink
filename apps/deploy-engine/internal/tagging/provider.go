package tagging

import (
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

const (
	// DefaultTagPrefix is the default prefix for Bluelink tags.
	DefaultTagPrefix = "bluelink:"
)

// ConfigProvider provides tagging configuration for blueprint operations.
// It merges request-level configuration with code-level defaults.
type ConfigProvider interface {
	// CreateConfig creates a TaggingConfig by applying defaults
	// where request config values are not specified.
	// The DeployEngineVersion is always set from the engine version.
	CreateConfig(requestConfig *types.TaggingOperationConfig) *provider.TaggingConfig
}

type configProviderImpl struct {
	deployEngineVersion string
}

// NewConfigProvider creates a new ConfigProvider with the given deploy engine version.
func NewConfigProvider(deployEngineVersion string) ConfigProvider {
	return &configProviderImpl{
		deployEngineVersion: deployEngineVersion,
	}
}

func (p *configProviderImpl) CreateConfig(
	requestConfig *types.TaggingOperationConfig,
) *provider.TaggingConfig {
	enabled, prefix := extractTaggingSettings(requestConfig)

	return &provider.TaggingConfig{
		Enabled:             enabled,
		Prefix:              prefix,
		DeployEngineVersion: p.deployEngineVersion,
		// ProviderPluginID and ProviderPluginVersion are set per-resource
		// during deployment based on the resource type's provider.
	}
}

// extractTaggingSettings extracts enabled and prefix from the request config,
// applying defaults where values are not specified.
func extractTaggingSettings(requestConfig *types.TaggingOperationConfig) (enabled bool, prefix string) {
	// Default values
	enabled = true
	prefix = DefaultTagPrefix

	if requestConfig == nil {
		return enabled, prefix
	}

	if requestConfig.Enabled != nil {
		enabled = *requestConfig.Enabled
	}

	if requestConfig.Prefix != "" {
		prefix = requestConfig.Prefix
	}

	return enabled, prefix
}
