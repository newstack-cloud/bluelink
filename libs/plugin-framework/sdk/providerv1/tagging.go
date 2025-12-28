package providerv1

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// TagKeyValue represents a single tag key-value pair.
type TagKeyValue struct {
	Key   string
	Value string
}

// GetBluelinkTags extracts the Bluelink tags from the resource deploy input.
// Returns nil if tagging is not configured or not enabled.
func GetBluelinkTags(input *provider.ResourceDeployInput) *provider.BluelinkTags {
	if input == nil || input.ProviderContext == nil {
		return nil
	}

	taggingConfig := input.ProviderContext.TaggingConfig()
	if taggingConfig == nil || !taggingConfig.Enabled {
		return nil
	}

	resourceName := ""
	resourceType := ""
	if input.Changes != nil {
		resourceName = input.Changes.AppliedResourceInfo.ResourceName
		if input.Changes.AppliedResourceInfo.ResourceWithResolvedSubs != nil &&
			input.Changes.AppliedResourceInfo.ResourceWithResolvedSubs.Type != nil {
			resourceType = input.Changes.AppliedResourceInfo.ResourceWithResolvedSubs.Type.Value
		}
	}

	return &provider.BluelinkTags{
		InstanceID:            input.InstanceID,
		InstanceName:          input.InstanceName,
		ResourceName:          resourceName,
		ResourceType:          resourceType,
		ProvisionedBy:         "bluelink",
		DeployEngineVersion:   taggingConfig.DeployEngineVersion,
		ProviderPluginID:      taggingConfig.ProviderPluginID,
		ProviderPluginVersion: taggingConfig.ProviderPluginVersion,
		Prefix:                taggingConfig.Prefix,
	}
}

// ToKeyValuePairs converts BluelinkTags to a slice of TagKeyValue pairs.
// Returns nil if tags is nil.
func ToKeyValuePairs(tags *provider.BluelinkTags) []TagKeyValue {
	if tags == nil {
		return nil
	}

	prefix := tags.Prefix
	if prefix == "" {
		prefix = "bluelink:"
	}

	return []TagKeyValue{
		{Key: fmt.Sprintf("%sinstance-id", prefix), Value: tags.InstanceID},
		{Key: fmt.Sprintf("%sinstance-name", prefix), Value: tags.InstanceName},
		{Key: fmt.Sprintf("%sresource-name", prefix), Value: tags.ResourceName},
		{Key: fmt.Sprintf("%sresource-type", prefix), Value: tags.ResourceType},
		{Key: fmt.Sprintf("%sprovisioned-by", prefix), Value: tags.ProvisionedBy},
		{Key: fmt.Sprintf("%sdeploy-engine-version", prefix), Value: tags.DeployEngineVersion},
		{Key: fmt.Sprintf("%sprovider-plugin-id", prefix), Value: tags.ProviderPluginID},
		{Key: fmt.Sprintf("%sprovider-plugin-version", prefix), Value: tags.ProviderPluginVersion},
	}
}

// ToMap converts BluelinkTags to a map of tag keys to values.
// Returns nil if tags is nil.
func ToMap(tags *provider.BluelinkTags) map[string]string {
	if tags == nil {
		return nil
	}

	prefix := tags.Prefix
	if prefix == "" {
		prefix = "bluelink:"
	}

	return map[string]string{
		fmt.Sprintf("%sinstance-id", prefix):             tags.InstanceID,
		fmt.Sprintf("%sinstance-name", prefix):           tags.InstanceName,
		fmt.Sprintf("%sresource-name", prefix):           tags.ResourceName,
		fmt.Sprintf("%sresource-type", prefix):           tags.ResourceType,
		fmt.Sprintf("%sprovisioned-by", prefix):          tags.ProvisionedBy,
		fmt.Sprintf("%sdeploy-engine-version", prefix):   tags.DeployEngineVersion,
		fmt.Sprintf("%sprovider-plugin-id", prefix):      tags.ProviderPluginID,
		fmt.Sprintf("%sprovider-plugin-version", prefix): tags.ProviderPluginVersion,
	}
}
