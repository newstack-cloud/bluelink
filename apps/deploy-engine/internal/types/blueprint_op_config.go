package types

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// TaggingOperationConfig is the data type for tagging configuration
// that can be provided in HTTP requests for blueprint operations.
// This controls whether Bluelink provenance tags are applied to resources.
type TaggingOperationConfig struct {
	// Enabled controls whether Bluelink tags are applied to resources.
	// If nil, the default (true) is used.
	Enabled *bool `json:"enabled,omitempty"`
	// Prefix is the tag key prefix (e.g., "bluelink:" or "myorg:bluelink:").
	// If empty, the default ("bluelink:") is used.
	Prefix string `json:"prefix,omitempty"`
}

// BlueprintOperationConfig is the data type for configuration that can be provided
// in HTTP requests for actions that are carried out for blueprints.
// These values will be merged with the default values either defined in
// plugins or in the blueprint itself.
type BlueprintOperationConfig struct {
	Providers          map[string]map[string]*core.ScalarValue `json:"providers"`
	Transformers       map[string]map[string]*core.ScalarValue `json:"transformers"`
	ContextVariables   map[string]*core.ScalarValue            `json:"contextVariables"`
	BlueprintVariables map[string]*core.ScalarValue            `json:"blueprintVariables"`
	Dependencies       map[string]string                       `json:"dependencies,omitempty"`
	// Tagging is the configuration for Bluelink resource tagging.
	// If nil, default tagging configuration is used.
	Tagging *TaggingOperationConfig `json:"tagging,omitempty"`
}
