package provider

// TaggingSupport indicates how a resource type supports external tagging.
type TaggingSupport int

const (
	// TaggingSupportNone indicates the resource type does not support tags.
	// Provenance will be stored in Bluelink state only.
	TaggingSupportNone TaggingSupport = iota
	// TaggingSupportFull indicates the resource supports arbitrary key-value tags (AWS, Azure).
	TaggingSupportFull
	// TaggingSupportLabels indicates the resource supports labels with restrictions (GCP, K8s).
	TaggingSupportLabels
)

// BluelinkTags contains the standard tags that Bluelink applies to provisioned resources.
type BluelinkTags struct {
	// InstanceID is the unique identifier for the blueprint instance.
	InstanceID string
	// InstanceName is the user-defined name for the blueprint instance.
	InstanceName string
	// ResourceName is the logical name of the resource in the blueprint.
	ResourceName string
	// ResourceType is the fully-qualified resource type (e.g., "aws/lambda/function").
	ResourceType string
	// ProvisionedBy is a marker indicating the resource was provisioned by Bluelink.
	ProvisionedBy string
	// DeployEngineVersion is the version of the deploy engine.
	DeployEngineVersion string
	// ProviderPluginID is the identifier for the provider plugin (e.g., "bluelink/aws").
	ProviderPluginID string
	// ProviderPluginVersion is the version of the provider plugin.
	ProviderPluginVersion string
	// Prefix is the configurable tag key prefix (e.g., "bluelink:" or "myorg:bluelink:").
	Prefix string
}

// TaggingConfig provides configuration for resource tagging behavior.
type TaggingConfig struct {
	// Prefix is the prefix to apply to all Bluelink tag keys.
	// Default is "bluelink:".
	Prefix string
	// DeployEngineVersion is the version of the deploy engine.
	DeployEngineVersion string
	// ProviderPluginID is the identifier for the provider plugin.
	ProviderPluginID string
	// ProviderPluginVersion is the version of the provider plugin.
	ProviderPluginVersion string
	// Enabled controls whether tagging is enabled.
	Enabled bool
}

// DefaultTaggingConfig returns the default tagging configuration.
func DefaultTaggingConfig() *TaggingConfig {
	return &TaggingConfig{
		Prefix:  "bluelink:",
		Enabled: true,
	}
}
